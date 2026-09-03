# appkit

`appkit` is a small application foundation for Go services built with Uber Fx.
It provides typed TOML/environment configuration, Zap logging, a pgx pool, a
managed `net/http` server, and health endpoints.

## Packages

- `configfx`: typed TOML config with environment overrides and validation.
- `loggingfx`: one shared Zap logger and Fx event logging.
- `postgresfx`: pgx pool with startup ping, shutdown, and readiness check.
- `httpfx`: `net/http` server with graceful shutdown.
- `healthfx`: `GET /health` and `GET /ready`.

## Usage

Define the application-owned config:

```go
type Config struct {
	Logging  loggingfx.Config  `koanf:"logging"`
	Postgres postgresfx.Config `koanf:"postgres"`
	HTTP     httpfx.Config     `koanf:"http"`
	Health   healthfx.Config   `koanf:"health"`
}

func (c *Config) Validate() error {
	if err := c.Postgres.Validate(); err != nil {
		return err
	}
	if err := c.HTTP.Validate(); err != nil {
		return err
	}
	return c.Health.Validate()
}
```

Compose it with appkit:

```go
func main() {
	app := appkit.New(
		configfx.Provide[Config](configfx.DefaultOptions()),
		fx.Provide(
			func(c *Config) loggingfx.Config { return c.Logging },
			func(c *Config) postgresfx.Config { return c.Postgres },
			func(c *Config) httpfx.Config { return c.HTTP },
			func(c *Config) healthfx.Config { return c.Health },
			newRepository,
			newService,
		),
		fx.Invoke(registerRoutes),
	)

	app.Run()
}
```

Application routes use the shared standard-library mux:

```go
func registerRoutes(mux *http.ServeMux, service *Service) {
	mux.HandleFunc("GET /drivers", service.ListDrivers)
}
```

Example `config.toml`:

```toml
[logging]
development = true
level = "debug"
service = "drivers"

[postgres]
dsn = "postgres://postgres:postgres@localhost:5432/drivers?sslmode=disable"
max_conns = 10
ping_timeout = "5s"

[http]
address = ":8080"
shutdown_timeout = "10s"

[health]
check_timeout = "5s"
```

Environment variables override the file. Double underscores separate nested
sections:

```text
APP_POSTGRES__DSN=postgres://...
APP_LOGGING__LEVEL=info
APP_HTTP__ADDRESS=:9000
```

Unknown TOML keys and environment variables with the configured prefix are
rejected, so configuration typos fail at startup. Set `AllowUnknownFields` in
`configfx.Options` only when intentionally sharing a larger configuration.

Use the individual modules when an application needs only part of the stack.
`loggingfx.Module` needs `loggingfx.Config`; `postgresfx.Module` needs
`postgresfx.Config`; `httpfx.Module` needs `httpfx.Config` and a `*zap.Logger`;
`healthfx.Module` needs a `*http.ServeMux` and accepts optional
`healthfx.Config` and `*zap.Logger` values.
