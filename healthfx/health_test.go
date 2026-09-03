package healthfx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestHealthAndReadyRoutes(t *testing.T) {
	mux := http.NewServeMux()
	err := registerRoutes(routeParams{
		Mux: mux,
		Checks: []Check{
			{Name: "postgres", Run: func(context.Context) error { return nil }},
		},
	})
	if err != nil {
		t.Fatalf("registerRoutes() error = %v", err)
	}

	health := httptest.NewRecorder()
	mux.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want %d", health.Code, http.StatusOK)
	}

	ready := httptest.NewRecorder()
	mux.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("/ready status = %d, want %d", ready.Code, http.StatusOK)
	}
	if !strings.Contains(ready.Body.String(), "\"postgres\":\"ok\"") {
		t.Fatalf("/ready body = %s", ready.Body.String())
	}
}

func TestReadyReportsFailedCheck(t *testing.T) {
	mux := http.NewServeMux()
	core, logs := observer.New(zap.WarnLevel)
	err := registerRoutes(routeParams{
		Mux:    mux,
		Logger: zap.New(core),
		Checks: []Check{
			{Name: "postgres", Run: func(context.Context) error { return errors.New("down") }},
		},
	})
	if err != nil {
		t.Fatalf("registerRoutes() error = %v", err)
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("/ready status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	entries := logs.FilterMessage("readiness check failed").All()
	if len(entries) != 1 || entries[0].ContextMap()["check"] != "postgres" {
		t.Fatalf("readiness failure logs = %#v, want postgres failure", entries)
	}
}

func TestReadyChecksRunConcurrently(t *testing.T) {
	mux := http.NewServeMux()
	started := make(chan string, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseChecks := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseChecks()

	err := registerRoutes(routeParams{
		Mux:    mux,
		Config: Config{CheckTimeout: time.Second},
		Checks: []Check{
			{Name: "first", Run: blockingCheck(started, release, "first")},
			{Name: "second", Run: blockingCheck(started, release, "second")},
		},
	})
	if err != nil {
		t.Fatalf("registerRoutes() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("readiness checks did not start concurrently")
		}
	}
	releaseChecks()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("readiness request did not finish")
	}
}

func TestReadyTimesOutHungCheck(t *testing.T) {
	mux := http.NewServeMux()
	err := registerRoutes(routeParams{
		Mux:    mux,
		Config: Config{CheckTimeout: 10 * time.Millisecond},
		Checks: []Check{
			{Name: "hung", Run: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			}},
		},
	})
	if err != nil {
		t.Fatalf("registerRoutes() error = %v", err)
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("/ready status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(response.Body.String(), `"hung":"failed"`) {
		t.Fatalf("/ready body = %s, want timed-out check", response.Body.String())
	}
}

func TestRegisterRoutesRejectsInvalidChecks(t *testing.T) {
	tests := []struct {
		name   string
		checks []Check
	}{
		{name: "empty name", checks: []Check{{Run: func(context.Context) error { return nil }}}},
		{name: "nil function", checks: []Check{{Name: "nil"}}},
		{name: "duplicate name", checks: []Check{
			{Name: "same", Run: func(context.Context) error { return nil }},
			{Name: "same", Run: func(context.Context) error { return nil }},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := registerRoutes(routeParams{Mux: http.NewServeMux(), Checks: test.checks})
			if err == nil {
				t.Fatal("registerRoutes() error = nil, want validation error")
			}
		})
	}
}

func TestConfigRejectsNegativeTimeout(t *testing.T) {
	if err := (Config{CheckTimeout: -time.Second}).Validate(); err == nil {
		t.Fatal("Validate() error = nil, want negative timeout error")
	}
}

func blockingCheck(started chan<- string, release <-chan struct{}, name string) func(context.Context) error {
	return func(context.Context) error {
		started <- name
		<-release
		return nil
	}
}
