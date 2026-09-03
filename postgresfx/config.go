package postgresfx

import (
	"errors"
	"time"
)

// Config controls the pgx connection pool.
type Config struct {
	DSN               string        `koanf:"dsn"`
	MaxConns          int32         `koanf:"max_conns"`
	MinConns          int32         `koanf:"min_conns"`
	MaxConnLifetime   time.Duration `koanf:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `koanf:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `koanf:"health_check_period"`
	ConnectTimeout    time.Duration `koanf:"connect_timeout"`
	PingTimeout       time.Duration `koanf:"ping_timeout"`
}

func (config Config) withDefaults() Config {
	if config.PingTimeout == 0 {
		config.PingTimeout = 5 * time.Second
	}
	return config
}

// Validate checks the pool configuration before opening it.
func (config Config) Validate() error {
	if config.DSN == "" {
		return errors.New("PostgreSQL DSN must not be empty")
	}
	if config.MaxConns < 0 || config.MinConns < 0 {
		return errors.New("PostgreSQL connection limits must not be negative")
	}
	if config.MaxConns > 0 && config.MinConns > config.MaxConns {
		return errors.New("PostgreSQL min_conns must not exceed max_conns")
	}
	if config.MaxConnLifetime < 0 ||
		config.MaxConnIdleTime < 0 ||
		config.HealthCheckPeriod < 0 ||
		config.ConnectTimeout < 0 ||
		config.PingTimeout < 0 {
		return errors.New("PostgreSQL timeouts must not be negative")
	}
	return nil
}
