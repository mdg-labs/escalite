package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

const defaultQueueWorkers = 10

// Options configures a River queue client for the engine service.
type Options struct {
	DatabaseURL           string
	Logger                *slog.Logger
	Workers               *river.Workers
	HeartbeatScanInterval time.Duration
}

// Client wraps a pgx pool and River worker client.
type Client struct {
	pool   *pgxpool.Pool
	river  *river.Client[pgx.Tx]
	logger *slog.Logger
}

// New connects to Postgres, applies River migrations, and prepares a worker client.
func New(ctx context.Context, opts Options) (*Client, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	pool, err := pgxpool.New(ctx, opts.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}

	if err := MigrateUp(ctx, pool, opts.Logger); err != nil {
		pool.Close()
		return nil, err
	}

	client := &Client{
		pool:   pool,
		logger: opts.Logger,
	}

	workers := opts.Workers
	if workers == nil {
		workers = NewWorkers(opts.Logger, pool, client)
	}

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: defaultQueueWorkers},
		},
		Workers:      workers,
		PeriodicJobs: NewPeriodicJobs(opts.HeartbeatScanInterval),
		Middleware: []rivertype.Middleware{
			NewWorkerLoggingMiddleware(opts.Logger),
		},
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create river client: %w", err)
	}

	client.river = riverClient
	return client, nil
}

// Pool exposes the underlying pgx pool for tests and callers.
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Start begins processing jobs.
func (c *Client) Start(ctx context.Context) error {
	if err := c.river.Start(ctx); err != nil {
		return fmt.Errorf("start river client: %w", err)
	}
	c.logger.Info("river worker started")
	return nil
}

// Stop waits for in-flight jobs to finish.
func (c *Client) Stop(ctx context.Context) error {
	if err := c.river.Stop(ctx); err != nil {
		return fmt.Errorf("stop river client: %w", err)
	}
	c.logger.Info("river worker stopped")
	return nil
}

// Close releases the database pool.
func (c *Client) Close() {
	c.pool.Close()
}
