package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig captures the minimal knobs required to bootstrap a pgxpool-backed persistence layer.
// Values map 1:1 with envconfig-driven configuration (see docs/project-requirements-document.md).
type PoolConfig struct {
	ConnString          string        // full DSN or URL, e.g. postgres://user:pass@host:5432/db
	MaxConns            int32         // optional cap for concurrent connections
	MinConns            int32         // optional floor for warm pool size
	MaxConnLifetime     time.Duration // recycle connections after this duration (0 leaves pgx default)
	MaxConnIdleTime     time.Duration // close idle connections after this duration (0 leaves pgx default)
	HealthCheckInterval time.Duration // override pgx health check period (0 leaves pgx default)
}

// NewPool builds a pgxpool.Pool using the shared configuration and eagerly verifies connectivity.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	if cfg.ConnString == "" {
		return nil, fmt.Errorf("conn string is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("parse pgx pool config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolConfig.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolConfig.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckInterval > 0 {
		poolConfig.HealthCheckPeriod = cfg.HealthCheckInterval
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// ClosePool shuts down the pool gracefully; safe to call with nil.
func ClosePool(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
