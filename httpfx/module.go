package httpfx

import "go.uber.org/fx"

// Module provides a net/http server and manages it through the Fx lifecycle.
var Module = fx.Module(
	"http",
	fx.Provide(
		NewMux,
		NewServer,
		newListenFunc,
	),
	fx.Invoke(registerLifecycle),
)
