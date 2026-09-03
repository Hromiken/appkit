package postgresfx

import "go.uber.org/fx"

// Module provides one shared pgx pool and a PostgreSQL readiness check.
var Module = fx.Module(
	"postgres",
	fx.Provide(
		NewPool,
		NewHealthCheck,
	),
	fx.Invoke(requirePool),
)
