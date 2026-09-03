// Package healthfx provides standard liveness and readiness endpoints.
package healthfx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Check is a named readiness check. Run must honor context cancellation.
type Check struct {
	Name string
	Run  func(context.Context) error
}

// Result publishes a readiness check to the Fx value group consumed by Module.
type Result struct {
	fx.Out

	Check Check `group:"appkit_health_checks"`
}

type routeParams struct {
	fx.In

	Mux    *http.ServeMux
	Checks []Check     `group:"appkit_health_checks"`
	Config Config      `optional:"true"`
	Logger *zap.Logger `optional:"true"`
}

type response struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

func registerRoutes(params routeParams) error {
	if err := params.Config.Validate(); err != nil {
		return err
	}
	config := params.Config.withDefaults()

	checks := append([]Check(nil), params.Checks...)
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Name < checks[j].Name
	})

	seen := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		if check.Name == "" {
			return errors.New("health check name must not be empty")
		}
		if check.Run == nil {
			return errors.New("health check function must not be nil")
		}
		if _, exists := seen[check.Name]; exists {
			return errors.New("duplicate health check: " + check.Name)
		}
		seen[check.Name] = struct{}{}
	}

	params.Mux.HandleFunc("GET /health", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, response{Status: "ok"})
	})

	params.Mux.HandleFunc("GET /ready", func(writer http.ResponseWriter, request *http.Request) {
		result := response{
			Status: "ok",
			Checks: make(map[string]string, len(checks)),
		}
		statusCode := http.StatusOK

		for name, err := range runChecks(request.Context(), checks, config.CheckTimeout) {
			if err != nil {
				result.Status = "unavailable"
				result.Checks[name] = "failed"
				statusCode = http.StatusServiceUnavailable
				if params.Logger != nil {
					params.Logger.Warn("readiness check failed", zap.String("check", name), zap.Error(err))
				}
				continue
			}
			result.Checks[name] = "ok"
		}

		writeJSON(writer, statusCode, result)
	})

	return nil
}

type checkOutcome struct {
	name string
	err  error
}

func runChecks(parent context.Context, checks []Check, timeout time.Duration) map[string]error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	outcomes := make(chan checkOutcome, len(checks))
	for _, check := range checks {
		check := check
		go func() {
			outcomes <- checkOutcome{name: check.Name, err: check.Run(ctx)}
		}()
	}

	results := make(map[string]error, len(checks))
	for len(results) < len(checks) {
		select {
		case outcome := <-outcomes:
			results[outcome.name] = outcome.err
		case <-ctx.Done():
			for _, check := range checks {
				if _, ok := results[check.Name]; !ok {
					results[check.Name] = ctx.Err()
				}
			}
		}
	}

	return results
}

func writeJSON(writer http.ResponseWriter, statusCode int, value response) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(value)
}
