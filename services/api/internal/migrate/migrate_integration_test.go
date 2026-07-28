package migrate_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/testutil"
	"github.com/mdg-labs/escalite/services/api/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const pgSchemaDiffVersion = "v1.0.7"

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}

func canonicalSchemaDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "services", "api", "schema", "sql")
}

func copyCanonicalSchema(t *testing.T) string {
	t.Helper()

	src := canonicalSchemaDir(t)
	dst := filepath.Join(t.TempDir(), "schema")
	require.NoError(t, os.MkdirAll(dst, 0o755))
	for _, name := range []string{"schema.sql", "realtime_notify.sql"} {
		data, err := os.ReadFile(filepath.Join(src, name))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dst, name), data, 0o644))
	}
	return dst
}

func prepareSchemaDiffDirs(t *testing.T, schemaDir string) (tablesDir, triggersDir string) {
	t.Helper()

	tablesDir = filepath.Join(t.TempDir(), "tables")
	triggersDir = filepath.Join(t.TempDir(), "triggers")
	require.NoError(t, os.MkdirAll(tablesDir, 0o755))
	require.NoError(t, os.MkdirAll(triggersDir, 0o755))

	schemaSQL, err := os.ReadFile(filepath.Join(schemaDir, "schema.sql"))
	require.NoError(t, err)
	notifySQL, err := os.ReadFile(filepath.Join(schemaDir, "realtime_notify.sql"))
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(tablesDir, "schema.sql"), schemaSQL, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(triggersDir, "realtime_notify.sql"), notifySQL, 0o644))
	return tablesDir, triggersDir
}

func resolvePgSchemaDiff(t *testing.T) string {
	t.Helper()

	if path, err := exec.LookPath("pg-schema-diff"); err == nil {
		return path
	}

	install := exec.Command("go", "install", "github.com/stripe/pg-schema-diff/cmd/pg-schema-diff@"+pgSchemaDiffVersion)
	install.Env = append(os.Environ(), goBuildEnv(t)...)
	install.Stdout = os.Stderr
	install.Stderr = os.Stderr
	require.NoError(t, install.Run())

	path, err := exec.LookPath("pg-schema-diff")
	require.NoError(t, err)
	return path
}

func goBuildEnv(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	tmpDir := filepath.Join(root, ".tmp-go-build", "tmp")
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))
	return []string{
		"GOCACHE=" + filepath.Join(root, ".tmp-go-build", "cache"),
		"GOTMPDIR=" + tmpDir,
		"TMPDIR=" + tmpDir,
	}
}

func pgSchemaDiffPlan(t *testing.T, serverDSN string, fromEmpty bool, schemaDir string) string {
	t.Helper()

	bin := resolvePgSchemaDiff(t)
	tablesDir, triggersDir := prepareSchemaDiffDirs(t, schemaDir)

	args := []string{
		"plan",
		"--output-format", "sql",
		"--no-concurrent-index-ops",
		"--disable-plan-validation",
		"--temp-db-dsn", serverDSN,
		"--to-dir", tablesDir,
		"--to-dir", triggersDir,
	}
	if fromEmpty {
		args = append(args, "--from-empty-dsn")
	} else {
		args = append(args, "--from-dsn", serverDSN)
	}

	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), goBuildEnv(t)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), "pg-schema-diff stderr: %s", stderr.String())
	return stdout.String()
}

func stripPlanNoise(plan string) string {
	return filterIgnoredPlanStatements(plan)
}

func filterIgnoredPlanStatements(planSQL string) string {
	var kept []string
	var block []string

	flush := func() {
		if len(block) == 0 {
			return
		}
		text := strings.Join(block, "\n")
		if !strings.Contains(strings.ToLower(text), "goose_db_version") {
			kept = append(kept, text)
		}
		block = nil
	}

	for _, line := range strings.Split(planSQL, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "/*") {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "Statement") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "  - ") {
			continue
		}
		if trimmed == "" {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if strings.HasPrefix(trimmed, "SET SESSION") {
			continue
		}
		block = append(block, line)
		if strings.HasSuffix(trimmed, ";") {
			flush()
		}
	}
	flush()

	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func planHasDrift(plan string) bool {
	return strings.TrimSpace(stripPlanNoise(plan)) != ""
}

func schemaFingerprint(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()

	var parts []string

	appendQuery := func(label, query string) {
		rows, err := db.QueryContext(ctx, query)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			parts = append(parts, label+":"+line)
		}
		require.NoError(t, rows.Err())
	}

	appendQuery("column",
		`SELECT table_name || '.' || column_name || ':' || data_type || ':' || is_nullable || ':' || COALESCE(column_default::text, '')
		 FROM information_schema.columns
		 WHERE table_schema = 'public'
		 ORDER BY table_name, ordinal_position`)

	appendQuery("index",
		`SELECT tablename || '.' || indexname || ':' || indexdef
		 FROM pg_indexes
		 WHERE schemaname = 'public'
		 ORDER BY tablename, indexname`)

	appendQuery("function",
		`SELECT p.proname || ':' || md5(pg_get_functiondef(p.oid))
		 FROM pg_proc p
		 JOIN pg_namespace n ON n.oid = p.pronamespace
		 WHERE n.nspname = 'public' AND p.prokind = 'f'
		 ORDER BY p.proname`)

	appendQuery("trigger",
		`SELECT c.relname || '.' || t.tgname || ':' || pg_get_triggerdef(t.oid, true)
		 FROM pg_trigger t
		 JOIN pg_class c ON c.oid = t.tgrelid
		 JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE NOT t.tgisinternal AND n.nspname = 'public'
		 ORDER BY c.relname, t.tgname`)

	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func openDB(t *testing.T, databaseURL string) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(context.Background()))
	return db
}

func gooseUpFromDir(t *testing.T, databaseURL, migrationsDir string) {
	t.Helper()

	ctx := context.Background()
	db := openDB(t, databaseURL)

	goose.SetBaseFS(nil)
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.UpContext(ctx, db, migrationsDir))
}

func runSchemaDiff(t *testing.T, databaseURL, migrationName, schemaDir, migrationsDir string, fromEmpty bool) {
	t.Helper()

	root := repoRoot(t)
	args := []string{
		"run", "./tools/schema-diff",
		"--name", migrationName,
		"--dsn", databaseURL,
		"--schema-dir", schemaDir,
		"--migrations-dir", migrationsDir,
		"--skip-validation",
	}
	if fromEmpty {
		args = append(args, "--from-empty")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), goBuildEnv(t)...)
	cmd.Env = append(cmd.Env, "DATABASE_URL="+databaseURL)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), "schema-diff failed: %s", stderr.String())
}

func TestIntegration_BaselineMigrationsApplyOnEmptyPostgres(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		ctx := context.Background()
		require.NoError(a, migrate.Up(ctx, databaseURL, slog.Default()))

		db := openDB(t, databaseURL)

		var tableCount int
		require.NoError(a, db.QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
	`).Scan(&tableCount))
		require.Greater(a, tableCount, 30)
	})
}

func TestIntegration_SchemaSQLFingerprintMatchesDBAfterGooseUp(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		ctx := context.Background()
		require.NoError(a, migrate.Up(ctx, databaseURL, slog.Default()))

		schemaDir := canonicalSchemaDir(t)
		plan := pgSchemaDiffPlan(t, databaseURL, false, schemaDir)
		require.False(a, planHasDrift(plan), "expected no drift between DB and canonical schema/sql, plan:\n%s", plan)

		db := openDB(t, databaseURL)
		fp1 := schemaFingerprint(t, ctx, db)
		fp2 := schemaFingerprint(t, ctx, db)
		require.Equal(a, fp1, fp2)
		require.NotEmpty(a, fp1)

		var notifyFunctions, notifyTriggers int
		require.NoError(a, db.QueryRowContext(ctx, `
		SELECT count(*) FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = 'public' AND p.proname LIKE 'notify_%'
	`).Scan(&notifyFunctions))
		require.NoError(a, db.QueryRowContext(ctx, `
		SELECT count(*) FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		WHERE NOT t.tgisinternal AND c.relnamespace = 'public'::regnamespace
		  AND t.tgname LIKE '%notify%'
	`).Scan(&notifyTriggers))
		require.Equal(a, 2, notifyFunctions)
		require.Equal(a, 5, notifyTriggers)
	})
}

func TestIntegration_SchemaDiffGeneratesAndAppliesIncrementalMigration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		ctx := context.Background()
		require.NoError(a, migrate.Up(ctx, databaseURL, slog.Default()))

		schemaDir := copyCanonicalSchema(t)
		schemaPath := filepath.Join(schemaDir, "schema.sql")
		schemaSQL, err := os.ReadFile(schemaPath)
		require.NoError(a, err)

		const marker = "migration_itest_marker"
		needle := `"name" text NOT NULL,`
		replacement := fmt.Sprintf(`"name" text NOT NULL,
  "%s" text NULL,`, marker)
		require.Contains(a, string(schemaSQL), needle)
		require.NoError(a, os.WriteFile(schemaPath, []byte(strings.Replace(string(schemaSQL), needle, replacement, 1)), 0o644))

		migrationsDir := filepath.Join(t.TempDir(), "migrations")
		require.NoError(a, os.MkdirAll(migrationsDir, 0o755))
		runSchemaDiff(t, databaseURL, "itest_add_marker", schemaDir, migrationsDir, false)

		entries, err := os.ReadDir(migrationsDir)
		require.NoError(a, err)
		require.Len(a, entries, 1)
		require.True(a, strings.HasSuffix(entries[0].Name(), "_itest_add_marker.sql"))

		gooseUpFromDir(t, databaseURL, migrationsDir)

		db := openDB(t, databaseURL)
		var columnName string
		require.NoError(a, db.QueryRowContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'organizations' AND column_name = $1
	`, marker).Scan(&columnName))
		require.Equal(a, marker, columnName)

		plan := pgSchemaDiffPlan(t, databaseURL, false, schemaDir)
		require.False(a, planHasDrift(plan), "expected DB to match modified canonical schema/sql, plan:\n%s", plan)
	})
}

func TestIntegration_DriftDetectionFailsWhenMigrationHandEdited(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		migrationsDir := filepath.Join(t.TempDir(), "migrations")
		require.NoError(a, os.MkdirAll(migrationsDir, 0o755))

		entries, err := migrations.Files.ReadDir(".")
		require.NoError(a, err)
		require.NotEmpty(a, entries)

		var bootstrapName string
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".sql") {
				bootstrapName = entry.Name()
				break
			}
		}
		require.NotEmpty(a, bootstrapName)

		bootstrapSQL, err := migrations.Files.ReadFile(bootstrapName)
		require.NoError(a, err)

		const driftColumn = "drift_smoke_col"
		handEdited := strings.Replace(
			string(bootstrapSQL),
			"-- +goose StatementEnd",
			fmt.Sprintf("ALTER TABLE organizations ADD COLUMN %s text;\n-- +goose StatementEnd", driftColumn),
			1,
		)
		require.NoError(a, os.WriteFile(filepath.Join(migrationsDir, bootstrapName), []byte(handEdited), 0o644))

		gooseUpFromDir(t, databaseURL, migrationsDir)

		db := openDB(t, databaseURL)
		var columnExists bool
		require.NoError(a, db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'organizations' AND column_name = $1
		)
	`, driftColumn).Scan(&columnExists))
		require.True(a, columnExists, "hand-edited migration should have created drift column")

		plan := pgSchemaDiffPlan(t, databaseURL, false, canonicalSchemaDir(t))
		require.True(a, planHasDrift(plan), "expected drift detection when migration was hand-edited without schema/sql change")
	})
}
