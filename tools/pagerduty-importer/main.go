package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/escalite"
	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/importer"
	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/pagerduty"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		apiToken    = flag.String("api-token", os.Getenv("PAGERDUTY_API_TOKEN"), "PagerDuty API token (prefer PAGERDUTY_API_TOKEN env)")
		exportFile  = flag.String("export-file", "", "PagerDuty export JSON file (offline import; no API token required)")
		targetURL   = flag.String("target-url", os.Getenv("ESCALITE_DATABASE_URL"), "Escalite Postgres URL")
		orgIDStr    = flag.String("organization-id", "", "Target Escalite organization UUID (required)")
		teamIDStr   = flag.String("team-id", "", "Target Escalite team UUID")
		teamName    = flag.String("team-name", "PagerDuty Import", "Team name when --create-team is set")
		createTeam  = flag.Bool("create-team", false, "Create a new team in the target organization")
		dryRun      = flag.Bool("dry-run", false, "Print mapping summary without writing to Escalite")
		mappingPath = flag.String("mapping-file", "pagerduty-id-mapping.json", "Path for audit ID mapping output on import")
	)
	flag.Parse()

	if *exportFile == "" && *apiToken == "" {
		fmt.Fprintln(os.Stderr, "PAGERDUTY_API_TOKEN or --export-file is required")
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

	source, err := openSource(*apiToken, *exportFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open source: %v\n", err)
		return 1
	}

	targetPool, err := pgxpool.New(ctx, *targetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect target database: %v\n", err)
		return 1
	}
	defer targetPool.Close()

	writer := escalite.NewWriter(targetPool)
	service := importer.NewService(source, writer)

	opts := importer.Options{
		OrganizationID: orgID,
		TeamID:         teamID,
		TeamName:       *teamName,
		CreateTeam:     *createTeam,
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

	if err := importer.FormatImportReport(os.Stdout, store, elapsed, *mappingPath); err != nil {
		fmt.Fprintf(os.Stderr, "print report: %v\n", err)
		return 1
	}
	return 0
}

func openSource(apiToken, exportFile string) (pagerduty.Source, error) {
	if exportFile != "" {
		return pagerduty.LoadExport(exportFile)
	}
	return pagerduty.NewClient(apiToken, &http.Client{Timeout: 60 * time.Second}), nil
}
