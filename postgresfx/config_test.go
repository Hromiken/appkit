package postgresfx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/puddle/v2"
	"go.uber.org/fx/fxtest"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{name: "empty DSN", config: Config{}},
		{name: "negative max", config: Config{DSN: "postgres://localhost/db", MaxConns: -1}},
		{name: "min exceeds max", config: Config{DSN: "postgres://localhost/db", MinConns: 3, MaxConns: 2}},
		{name: "negative timeout", config: Config{DSN: "postgres://localhost/db", PingTimeout: -time.Second}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want validation error")
			}
		})
	}
}

func TestNewPoolClosesPoolWhenStartupPingFails(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	pool, err := NewPool(lifecycle, Config{
		DSN:            "postgres://postgres@127.0.0.1:1/test?sslmode=disable",
		ConnectTimeout: 50 * time.Millisecond,
		PingTimeout:    100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := lifecycle.Start(ctx); err == nil {
		t.Fatal("Lifecycle.Start() error = nil, want ping error")
	}

	if err := pool.Ping(context.Background()); !errors.Is(err, puddle.ErrClosedPool) {
		t.Fatalf("Ping() error after failed start = %v, want closed pool", err)
	}
}

func TestNewPoolAppliesConfig(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	pool, err := NewPool(lifecycle, Config{
		DSN:            "postgres://postgres@localhost/test?sslmode=disable",
		MinConns:       2,
		MaxConns:       7,
		ConnectTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()

	config := pool.Config()
	if config.MinConns != 2 || config.MaxConns != 7 {
		t.Fatalf("connection limits = (%d, %d), want (2, 7)", config.MinConns, config.MaxConns)
	}
	if config.ConnConfig.ConnectTimeout != 3*time.Second {
		t.Fatalf("ConnectTimeout = %s, want 3s", config.ConnConfig.ConnectTimeout)
	}
}

func TestConfigValidationAcceptsValidConfig(t *testing.T) {
	config := Config{
		DSN:      "postgres://localhost/db",
		MinConns: 1,
		MaxConns: 4,
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
