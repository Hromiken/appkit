// Package appkit provides a small, reusable application foundation built on Fx.
package appkit

import (
	"github.com/Hromiken/appkit/healthfx"
	"github.com/Hromiken/appkit/httpfx"
	"github.com/Hromiken/appkit/loggingfx"
	"github.com/Hromiken/appkit/postgresfx"

	"go.uber.org/fx"
)

// Module combines the standard infrastructure modules. Applications may use
// the individual modules instead when they do not need the full stack.
var Module = fx.Module(
	"appkit",
	loggingfx.Module,
	postgresfx.Module,
	httpfx.Module,
	healthfx.Module,
)

// New creates an Fx application with the standard appkit modules and routes
// Fx's own events through the application Zap logger.
func New(options ...fx.Option) *fx.App {
	base := []fx.Option{
		Module,
		fx.WithLogger(loggingfx.NewFxEventLogger),
	}

	return fx.New(append(base, options...)...)
}
