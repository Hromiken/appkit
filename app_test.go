package appkit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Hromiken/appkit/httpfx"
	"github.com/Hromiken/appkit/loggingfx"
	"github.com/Hromiken/appkit/postgresfx"

	"go.uber.org/fx"
)

func TestModuleDependencyGraph(t *testing.T) {
	err := fx.ValidateApp(
		Module,
		fx.WithLogger(loggingfx.NewFxEventLogger),
		fx.Supply(
			loggingfx.Config{},
			httpfx.Config{},
			postgresfx.Config{},
		),
	)
	if err != nil {
		t.Fatalf("ValidateApp() error = %v", err)
	}
}

func TestModuleStartsPostgresBeforeHTTP(t *testing.T) {
	app := New(fx.Supply(
		loggingfx.Config{},
		httpfx.Config{Address: "]"},
		postgresfx.Config{
			DSN:            "postgres://postgres@127.0.0.1:1/test?sslmode=disable",
			ConnectTimeout: 50 * time.Millisecond,
			PingTimeout:    100 * time.Millisecond,
		},
	))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := app.Start(ctx)
	if err == nil {
		t.Fatal("Start() error = nil, want PostgreSQL connection error")
	}
	if !strings.Contains(err.Error(), "ping PostgreSQL") {
		t.Fatalf("Start() error = %q, want PostgreSQL to start before HTTP", err)
	}
}
