package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/escalite"
	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/goalert"
	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/importer"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		sourceURL     = flag.String("source-url", os.Getenv("GOALERT_DATABASE_URL"), "GoAlert Postgres URL")
		targetURL     = flag.String("target-url", os.Getenv("ESCALITE_DATABASE_URL"), "Escalite Postgres URL")
		orgIDStr      = flag.String("organization-id", "", "Target Escalite organization UUID (required)")
		teamIDStr     = flag.String("team-id", "", "Target Escalite team UUID")
		teamName      = flag.String("team-name", "GoAlert Import", "Team name when --create-team is set")
		createTeam    = flag.Bool("create-team", false, "Create a new team in the target organization")
		dryRun        = flag.Bool("dry-run", false, "Print mapping summary without writing to Escalite")
		includeAlerts = flag.Bool("include-alerts", false, "Import historical alerts (optional)")
		mappingPath   = flag.String("mapping-file", "goalert-id-mapping.json", "Path for audit ID mapping output on import")
	)
	flag.Parse()

	if *sourceURL == "" {
		fmt.Fprintln(os.Stderr, "source-url or GOALERT_DATABASE_URL is required")
		return 2
	}
	if *targetURL == "" {
		fmt.Fprintln(os.Stderr, "target-url or ESCALITE_DATABASE_URL is required")
		return 2
	}
	if *orgIDStr == "" {
		fmt.Fprintln(os.Stderr, "organization-id is required")
		return 2
	}
	orgID, err := uuid.Parse(*orgIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid organization-id: %v\n", err)
		return 2
	}

	var teamID uuid.UUID
	if *teamIDStr != "" {
		teamID, err = uuid.Parse(*teamIDStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid team-id: %v\n", err)
			return 2
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sourcePool, err := pgxpool.New(ctx, *sourceURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect source database: %v\n", err)
		return 1
	}
	defer sourcePool.Close()

	targetPool, err := pgxpool.New(ctx, *targetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect target database: %v\n", err)
		return 1
	}
	defer targetPool.Close()

	reader := goalert.NewReader(sourcePool)
	writer := escalite.NewWriter(targetPool)
	service := importer.NewService(reader, writer)

	opts := importer.Options{
		OrganizationID: orgID,
		TeamID:         teamID,
		TeamName:       *teamName,
		CreateTeam:     *createTeam,
		IncludeAlerts:  *includeAlerts,
		MappingPath:    *mappingPath,
	}

	if *dryRun {
		summary, err := service.DryRun(ctx, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dry-run failed: %v\n", err)
			return 1
		}
		if *createTeam {
			summary.TeamID = uuid.Nil
			fmt.Fprintf(os.Stdout, "Team will be created: %q\n\n", *teamName)
		}
		if err := importer.FormatSummary(os.Stdout, summary); err != nil {
			fmt.Fprintf(os.Stderr, "print summary: %v\n", err)
			return 1
		}
		return 0
	}

	started := time.Now()
	store, err := service.Import(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import failed: %v\n", err)
		return 1
	}
	elapsed := time.Since(started)

	fmt.Fprintf(os.Stdout, "Import complete in %s\n", elapsed.Round(time.Millisecond))
	fmt.Fprintf(os.Stdout, "  users: %d\n", len(store.Users))
	fmt.Fprintf(os.Stdout, "  schedules: %d\n", len(store.Schedules))
	fmt.Fprintf(os.Stdout, "  rotations: %d\n", len(store.Rotations))
	fmt.Fprintf(os.Stdout, "  services: %d\n", len(store.Services))
	fmt.Fprintf(os.Stdout, "  escalation policies: %d\n", len(store.EscalationPolicies))
	if *includeAlerts {
		fmt.Fprintf(os.Stdout, "  alerts: %d\n", len(store.Alerts))
	}
	if len(store.Skipped) > 0 {
		fmt.Fprintf(os.Stdout, "  skipped: %d (see mapping file)\n", len(store.Skipped))
	}
	fmt.Fprintf(os.Stdout, "ID mapping written to %s\n", *mappingPath)
	return 0
}
