package loggingfx

import "go.uber.org/fx"

// Module provides one shared Zap logger and registers its shutdown hook.
var Module = fx.Module(
	"logging",
	fx.Provide(New),
	fx.Invoke(registerLifecycle),
)
