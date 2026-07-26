// schema-diff generates goose-format SQL migrations from pg-schema-diff plan output.
//
// pg-schema-diff needs a running Postgres server to load canonical SQL and introspect
// catalog metadata. It creates temporary databases on that server; it does not start
// Docker containers itself.
//
// Usage:
//
//	schema-diff --name add_foo_column --dsn "$DATABASE_URL"
//	schema-diff --name bootstrap --from-empty --dsn "$DATABASE_URL"
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const pgSchemaDiffVersion = "v1.0.7"

func main() {
	name := flag.String("name", "", "migration name (snake_case, required)")
	serverDSN := flag.String("dsn", "", "Postgres server URL (default: DATABASE_URL, ESCALITE_DATABASE_URL, or compose localhost)")
	fromEmpty := flag.Bool("from-empty", false, "diff from an empty database; use for rare baseline squashes")
	skipValidation := flag.Bool("skip-validation", false, "skip pg-schema-diff plan validation (faster local runs)")
	schemaDir := flag.String("schema-dir", "", "directory containing schema.sql and realtime_notify.sql")
	migrationsDir := flag.String("migrations-dir", "", "directory to write the generated goose migration")
	pgSchemaDiffBin := flag.String("pg-schema-diff", "", "path to pg-schema-diff binary (default: PATH or go run)")
	flag.Parse()

	if strings.TrimSpace(*name) == "" {
		fatal("missing required --name")
	}
	if !validMigrationName(*name) {
		fatal("invalid --name %q: use snake_case letters, digits, and underscores", *name)
	}

	dsn, err := resolveServerDSN(*serverDSN)
	if err != nil {
		fatal("%v", err)
	}

	root, err := repoRoot()
	if err != nil {
		fatal("%v", err)
	}

	if *schemaDir == "" {
		*schemaDir = filepath.Join(root, "services", "api", "schema", "sql")
	}
	if *migrationsDir == "" {
		*migrationsDir = filepath.Join(root, "services", "api", "migrations")
	}

	schemaSQL := filepath.Join(*schemaDir, "schema.sql")
	notifySQL := filepath.Join(*schemaDir, "realtime_notify.sql")
	for _, path := range []string{schemaSQL, notifySQL} {
		if _, err := os.Stat(path); err != nil {
			fatal("canonical schema file missing: %s", path)
		}
	}

	tablesDir, triggersDir, cleanup, err := prepareSchemaDirs(schemaSQL, notifySQL)
	if err != nil {
		fatal("%v", err)
	}
	defer cleanup()

	bin, err := resolvePgSchemaDiff(*pgSchemaDiffBin)
	if err != nil {
		fatal("%v", err)
	}

	planSQL, err := runPlan(bin, *fromEmpty, *skipValidation, dsn, tablesDir, triggersDir)
	if err != nil {
		fatal("%v", err)
	}
	if strings.TrimSpace(planSQL) == "" {
		fatal("no schema changes detected; edit services/api/schema/sql/*.sql first")
	}

	filename := fmt.Sprintf("%s_%s.sql", time.Now().UTC().Format("20060102150405"), *name)
	target := filepath.Join(*migrationsDir, filename)
	body := formatGooseMigration(planSQL)

	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		fatal("write migration: %v", err)
	}

	fmt.Printf("wrote migration %s\n", target)
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not find repo root (go.work)")
		}
		dir = parent
	}
}

func validMigrationName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
		if i == 0 && r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}

func prepareSchemaDirs(schemaSQL, notifySQL string) (tablesDir, triggersDir string, cleanup func(), err error) {
	tablesDir, err = os.MkdirTemp("", "escalite-schema-tables-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("create temp tables dir: %w", err)
	}
	triggersDir, err = os.MkdirTemp("", "escalite-schema-triggers-*")
	if err != nil {
		os.RemoveAll(tablesDir)
		return "", "", nil, fmt.Errorf("create temp triggers dir: %w", err)
	}

	if err := copyFile(schemaSQL, filepath.Join(tablesDir, "schema.sql")); err != nil {
		os.RemoveAll(tablesDir)
		os.RemoveAll(triggersDir)
		return "", "", nil, err
	}
	if err := copyFile(notifySQL, filepath.Join(triggersDir, "realtime_notify.sql")); err != nil {
		os.RemoveAll(tablesDir)
		os.RemoveAll(triggersDir)
		return "", "", nil, err
	}

	cleanup = func() {
		os.RemoveAll(tablesDir)
		os.RemoveAll(triggersDir)
	}
	return tablesDir, triggersDir, cleanup, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func resolveServerDSN(explicit string) (string, error) {
	for _, candidate := range []string{explicit, os.Getenv("DATABASE_URL"), os.Getenv("ESCALITE_DATABASE_URL")} {
		if strings.TrimSpace(candidate) != "" {
			return strings.TrimSpace(candidate), nil
		}
	}
	// Compose dev default when postgres is published on localhost:5432.
	return "postgres://escalite:escalite@127.0.0.1:5432/escalite?sslmode=disable", nil
}

func pqEnvFromDSN(dsn string) ([]string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported dsn scheme %q", u.Scheme)
	}

	user := u.User.Username()
	pass, _ := u.User.Password()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	db := strings.TrimPrefix(u.Path, "/")
	if db == "" {
		return nil, errors.New("dsn missing database name")
	}

	env := []string{
		"PGHOST=" + host,
		"PGPORT=" + port,
		"PGUSER=" + user,
		"PGPASSWORD=" + pass,
		"PGDATABASE=" + db,
	}
	if ssl := u.Query().Get("sslmode"); ssl != "" {
		env = append(env, "PGSSLMODE="+ssl)
	}
	return env, nil
}

func resolvePgSchemaDiff(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if path, err := exec.LookPath("pg-schema-diff"); err == nil {
		return path, nil
	}
	// Fall back to pinned module install.
	install := exec.Command("go", "install", "github.com/stripe/pg-schema-diff/cmd/pg-schema-diff@"+pgSchemaDiffVersion)
	install.Stdout = os.Stderr
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("pg-schema-diff not found and install failed: %w", err)
	}
	path, err := exec.LookPath("pg-schema-diff")
	if err != nil {
		return "", fmt.Errorf("pg-schema-diff not found after install: %w", err)
	}
	return path, nil
}

func runPlan(bin string, fromEmpty, skipValidation bool, serverDSN, tablesDir, triggersDir string) (string, error) {
	args := []string{
		"plan",
		"--output-format", "sql",
		"--no-concurrent-index-ops",
		"--temp-db-dsn", serverDSN,
		"--to-dir", tablesDir,
		"--to-dir", triggersDir,
	}
	if skipValidation {
		args = append(args, "--disable-plan-validation")
	}
	if fromEmpty {
		args = append(args, "--from-empty-dsn")
	} else {
		args = append(args, "--from-dsn", serverDSN)
	}

	cmd := exec.Command(bin, args...)
	if fromEmpty {
		pqEnv, err := pqEnvFromDSN(serverDSN)
		if err != nil {
			return "", err
		}
		cmd.Env = append(os.Environ(), pqEnv...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("pg-schema-diff plan failed: %s", msg)
	}
	return filterIgnoredPlanStatements(stdout.String()), nil
}

// filterIgnoredPlanStatements removes pg-schema-diff output for objects that are
// runtime-managed and intentionally absent from canonical schema/sql.
func filterIgnoredPlanStatements(planSQL string) string {
	var kept []string
	var block []string

	flush := func() {
		if len(block) == 0 {
			return
		}
		text := strings.Join(block, "\n")
		if !containsIgnoredPlanObject(text) {
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
		block = append(block, line)
		if strings.HasSuffix(trimmed, ";") {
			flush()
		}
	}
	flush()

	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func containsIgnoredPlanObject(stmt string) bool {
	lower := strings.ToLower(stmt)
	return strings.Contains(lower, "goose_db_version")
}

func extractSQLStatements(planSQL string) []string {
	filtered := filterIgnoredPlanStatements(planSQL)
	if strings.TrimSpace(filtered) == "" {
		return nil
	}

	var statements []string
	var block []string

	flush := func() {
		if len(block) == 0 {
			return
		}
		text := strings.TrimSpace(strings.Join(block, "\n"))
		if text != "" {
			statements = append(statements, text)
		}
		block = nil
	}

	for _, line := range strings.Split(filtered, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		block = append(block, line)
		if strings.HasSuffix(trimmed, ";") {
			flush()
		}
	}
	flush()

	return statements
}

func formatGooseMigration(planSQL string) string {
	statements := extractSQLStatements(planSQL)
	var b strings.Builder
	b.WriteString("-- Code generated by tools/schema-diff; DO NOT EDIT.\n")
	b.WriteString("-- +goose Up\n")
	for _, stmt := range statements {
		b.WriteString("-- +goose StatementBegin\n")
		b.WriteString(stmt)
		b.WriteString("\n-- +goose StatementEnd\n")
	}
	b.WriteString("\n-- +goose Down\n")
	b.WriteString("-- generated migration; manual rollback required\n")
	return b.String()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "schema-diff: "+format+"\n", args...)
	os.Exit(1)
}
