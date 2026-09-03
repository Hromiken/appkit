// Package httpfx provides a standard net/http server managed by Fx.
package httpfx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewMux returns the shared application router.
func NewMux() *http.ServeMux {
	return http.NewServeMux()
}

// NewServer creates the shared HTTP server.
func NewServer(config Config, mux *http.ServeMux) (*http.Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.withDefaults()

	return &http.Server{
		Addr:              config.Address,
		Handler:           mux,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
	}, nil
}

type listenFunc func(network, address string) (net.Listener, error)

func newListenFunc() listenFunc {
	return net.Listen
}

type lifecycleParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Shutdown  fx.Shutdowner
	Config    Config
	Server    *http.Server
	Logger    *zap.Logger
	Listen    listenFunc
}

func registerLifecycle(params lifecycleParams) {
	config := params.Config.withDefaults()

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := params.Listen("tcp", params.Server.Addr)
			if err != nil {
				return fmt.Errorf("listen on %s: %w", params.Server.Addr, err)
			}

			params.Logger.Info("HTTP server started", zap.String("address", listener.Addr().String()))

			go func() {
				err := params.Server.Serve(listener)
				if err == nil || errors.Is(err, http.ErrServerClosed) {
					return
				}

				params.Logger.Error("HTTP server stopped unexpectedly", zap.Error(err))
				_ = params.Shutdown.Shutdown(fx.ExitCode(1))
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if config.ShutdownTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, config.ShutdownTimeout)
				defer cancel()
			}

			if err := params.Server.Shutdown(ctx); err != nil {
				return fmt.Errorf("shutdown HTTP server: %w", err)
			}

			params.Logger.Info("HTTP server stopped")
			return nil
		},
	})
}
