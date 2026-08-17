// Package postgres provides a PostgreSQL implementation of the durability Store.
//
// Usage:
//
//	store, err := postgres.NewStore(ctx, "postgres://user:pass@localhost:5432/aetheris")
//	runner := core.NewRunner(store)
//
// The store creates tables automatically on first use (if they don't exist).
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Colin4k1024/Aetheris/durability/core"
)

// Store implements core.Store using PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new PostgreSQL store.
// It automatically creates the required tables if they don't exist.
func NewStore(ctx context.Context, connString string) (*Store, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("durability/postgres: connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("durability/postgres: ping: %w", err)
	}

	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("durability/postgres: migrate: %w", err)
	}

	return s, nil
}

// NewStoreWithPool creates a store with an existing pgxpool.Pool.
func NewStoreWithPool(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Close closes the underlying connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// migrate creates the required tables if they don't exist.
func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS durability_jobs (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			state       TEXT NOT NULL DEFAULT 'created',
			version     INT NOT NULL DEFAULT 0,
			input       JSONB,
			result      JSONB,
			error       TEXT,
			completed_steps JSONB DEFAULT '{}',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS durability_events (
			id          TEXT NOT NULL,
			job_id      TEXT NOT NULL REFERENCES durability_jobs(id),
			type        TEXT NOT NULL,
			step_id     TEXT,
			payload     JSONB,
			version     INT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			prev_hash   TEXT,
			hash        TEXT,
			PRIMARY KEY (job_id, version)
		);

		CREATE INDEX IF NOT EXISTS idx_durability_events_job_id ON durability_events(job_id);

		CREATE TABLE IF NOT EXISTS durability_checkpoints (
			job_id      TEXT PRIMARY KEY REFERENCES durability_jobs(id),
			step_id     TEXT NOT NULL,
			state       JSONB,
			version     INT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func (s *Store) AppendEvent(ctx context.Context, jobID string, expectedVersion int, event core.Event) (int, error) {
	// Use a transaction for atomic version check + append
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("durability/postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock the job row to prevent concurrent appends
	var currentVersion int
	err = tx.QueryRow(ctx,
		`SELECT version FROM durability_jobs WHERE id = $1 FOR UPDATE`,
		jobID,
	).Scan(&currentVersion)
	if err == pgx.ErrNoRows {
		return 0, core.ErrJobNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("durability/postgres: lock job: %w", err)
	}

	if currentVersion != expectedVersion {
		return currentVersion, core.ErrVersionMismatch
	}

	newVersion := currentVersion + 1
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s-%d", jobID, newVersion)
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO durability_events (id, job_id, type, step_id, payload, version, created_at, prev_hash, hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		event.ID, jobID, string(event.Type), event.StepID, event.Payload,
		newVersion, event.CreatedAt, event.PrevHash, event.Hash,
	)
	if err != nil {
		return 0, fmt.Errorf("durability/postgres: insert event: %w", err)
	}

	// Update job version
	_, err = tx.Exec(ctx,
		`UPDATE durability_jobs SET version = $1, updated_at = NOW() WHERE id = $2`,
		newVersion, jobID,
	)
	if err != nil {
		return 0, fmt.Errorf("durability/postgres: update version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("durability/postgres: commit: %w", err)
	}

	return newVersion, nil
}

func (s *Store) ListEvents(ctx context.Context, jobID string) ([]core.Event, int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, job_id, type, step_id, payload, version, created_at, prev_hash, hash
		 FROM durability_events WHERE job_id = $1 ORDER BY version`,
		jobID,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("durability/postgres: list events: %w", err)
	}
	defer rows.Close()

	var events []core.Event
	for rows.Next() {
		var e core.Event
		if err := rows.Scan(&e.ID, &e.JobID, &e.Type, &e.StepID, &e.Payload,
			&e.Version, &e.CreatedAt, &e.PrevHash, &e.Hash); err != nil {
			return nil, 0, fmt.Errorf("durability/postgres: scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, len(events), nil
}

func (s *Store) SaveJob(ctx context.Context, job *core.Job) error {
	completedStepsJSON, err := json.Marshal(job.CompletedSteps)
	if err != nil {
		return fmt.Errorf("durability/postgres: marshal completed steps: %w", err)
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO durability_jobs (id, name, state, version, input, result, error, completed_steps, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (id) DO UPDATE SET
			state = EXCLUDED.state,
			version = EXCLUDED.version,
			result = EXCLUDED.result,
			error = EXCLUDED.error,
			completed_steps = EXCLUDED.completed_steps,
			updated_at = EXCLUDED.updated_at`,
		job.ID, job.Name, string(job.State), job.Version,
		job.Input, job.Result, job.Error, completedStepsJSON,
		job.CreatedAt, job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("durability/postgres: upsert job: %w", err)
	}
	return nil
}

func (s *Store) LoadJob(ctx context.Context, jobID string) (*core.Job, error) {
	var job core.Job
	var stateStr string
	var completedStepsJSON []byte

	err := s.pool.QueryRow(ctx,
		`SELECT id, name, state, version, input, result, error, completed_steps, created_at, updated_at
		 FROM durability_jobs WHERE id = $1`,
		jobID,
	).Scan(&job.ID, &job.Name, &stateStr, &job.Version,
		&job.Input, &job.Result, &job.Error, &completedStepsJSON,
		&job.CreatedAt, &job.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("durability/postgres: load job: %w", err)
	}

	job.State = core.JobState(stateStr)
	if len(completedStepsJSON) > 0 {
		json.Unmarshal(completedStepsJSON, &job.CompletedSteps)
	}
	if job.CompletedSteps == nil {
		job.CompletedSteps = make(map[string]json.RawMessage)
	}
	return &job, nil
}

func (s *Store) SaveCheckpoint(ctx context.Context, jobID string, stepID string, state []byte, version int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO durability_checkpoints (job_id, step_id, state, version, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (job_id) DO UPDATE SET
			step_id = EXCLUDED.step_id,
			state = EXCLUDED.state,
			version = EXCLUDED.version,
			created_at = EXCLUDED.created_at`,
		jobID, stepID, state, version,
	)
	if err != nil {
		return fmt.Errorf("durability/postgres: save checkpoint: %w", err)
	}
	return nil
}

func (s *Store) LoadCheckpoint(ctx context.Context, jobID string) (*core.Checkpoint, error) {
	var cp core.Checkpoint
	err := s.pool.QueryRow(ctx,
		`SELECT job_id, step_id, state, version, created_at
		 FROM durability_checkpoints WHERE job_id = $1`,
		jobID,
	).Scan(&cp.JobID, &cp.StepID, &cp.State, &cp.Version, &cp.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("durability/postgres: load checkpoint: %w", err)
	}
	return &cp, nil
}
