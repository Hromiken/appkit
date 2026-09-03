// Package postgresfx provides a pgx pool managed by the Fx lifecycle.
package postgresfx

import (
	"context"
	"fmt"

	"github.com/Hromiken/appkit/healthfx"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// NewPool builds a pool, verifies it on application start, and closes it on
// application stop.
func NewPool(lifecycle fx.Lifecycle, config Config) (*pgxpool.Pool, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.withDefaults()

	poolConfig, err := pgxpool.ParseConfig(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL DSN: %w", err)
	}

	if config.MaxConns > 0 {
		poolConfig.MaxConns = config.MaxConns
	}
	if config.MinConns > 0 {
		poolConfig.MinConns = config.MinConns
	}
	if config.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = config.MaxConnLifetime
	}
	if config.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	}
	if config.HealthCheckPeriod > 0 {
		poolConfig.HealthCheckPeriod = config.HealthCheckPeriod
	}
	if config.ConnectTimeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if config.PingTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, config.PingTimeout)
				defer cancel()
			}

			if err := pool.Ping(ctx); err != nil {
				pool.Close()
				return fmt.Errorf("ping PostgreSQL: %w", err)
			}
			return nil
		},
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}

// NewHealthCheck publishes PostgreSQL as a readiness dependency.
func NewHealthCheck(pool *pgxpool.Pool) healthfx.Result {
	return healthfx.Result{
		Check: healthfx.Check{
			Name: "postgres",
			Run:  pool.Ping,
		},
	}
}

func requirePool(*pgxpool.Pool) {}
