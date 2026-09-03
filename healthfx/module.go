package healthfx

import "go.uber.org/fx"

// Module registers GET /health and GET /ready on the shared HTTP mux.
var Module = fx.Module(
	"health",
	fx.Invoke(registerRoutes),
)
