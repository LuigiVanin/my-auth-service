# Plugins

A plugin is a cross cutting feature packaged as an `fx.Option`, wired into the
container next to the modules but owning no domain. `plugins/` holds one today:
the Swagger UI.

## Plugin, module, or bootstrap?

| The feature…                                            | Goes to        |
| -------------------------------------------------------- | -------------- |
| owns a table, a service, or HTTP routes of its own domain | a module       |
| is a single infrastructure dependency the app injects     | `infra/bootstrap/` |
| attaches behaviour to the running app, across all modules | a **plugin**   |

`bootstrap` builds *things* — the database handle, the logger, the fiber app,
the email manager — as plain constructors that `main.go` provides. A plugin
builds *and attaches*: it supplies its own providers and its own `fx.Invoke`,
so adding it to `appOptions()` is the only wiring step.

Examples that would be plugins: request tracing, metrics export, a rate limiter
mounted on every route, a dev only admin endpoint. Examples that would not: a
second database connection (bootstrap), OTP delivery (a module).

## Anatomy

A plugin is a function returning `fx.Option`, with an options struct carrying
defaults:

```go
type SwaggerPluginOptions struct {
    Path string
    Info openapi.Info
}

func SwaggerPlugin(options ...SwaggerPluginOptions) fx.Option {
    option := SwaggerPluginOptions{
        Path: "/docs/",
        Info: openapi.Info{Title: "Auth Service", ...},
    }

    if len(options) > 0 {
        option = options[0]
    }

    return fx.Options(
        // what the rest of the app may inject
        fx.Provide(func() *openapi.Builder {
            return openapi.NewBuilder(option.Info.Title, option.Info.Description, option.Info.Version)
        }),

        // what the plugin does to the running app
        fx.Invoke(func(server *fiber.App, lifecycle fx.Lifecycle, logger *zap.Logger, swagger *openapi.Builder) {
            document := swagger.Build()
            url := strings.TrimSuffix(option.Path, "/")

            server.Get(fmt.Sprintf("%s/*", url), NewSwagger(url, *document).Handler)
        }),
    )
}
```

The shape to follow:

- **Variadic options with a default.** `SwaggerPlugin()` must work with no
  arguments.
- **`fx.Provide` for what others consume.** Every controller injects
  `*openapi.Builder`; that provider is the plugin's public surface.
- **`fx.Invoke` for the side effect.** Mounting routes, registering lifecycle
  hooks, wrapping the server.
- **No domain imports.** A plugin may import `fiber`, `fx`, `zap` and its own
  library. It must not import a module, a service or a repository — if it needs
  one, it is a module.

## Ordering

**`fx.Invoke` runs in the order the options are supplied, and the swagger plugin
depends on it.**

`swagger.Build()` snapshots whatever routes have been added to the builder at
that moment. Controllers call `this.swagger.Add(...)` inside `Register`, which
runs in the `fx.Invoke` that iterates the `group:"controllers"` slice. That
invoke is supplied **before** `plugins.SwaggerPlugin(swaggerOptions)` in
`appOptions()`:

```go
fx.Invoke(fx.Annotate(
    func(server *fiber.App, controllers []interfaces.IController) {
        for _, controller := range controllers {
            controller.Register(server)
        }
    },
    fx.ParamTags(``, `group:"controllers"`),
)),
plugins.SwaggerPlugin(swaggerOptions),
```

Move the plugin above that invoke, or insert a new invoke between them that
registers routes, and the document silently comes out empty or partial. There is
no error — the endpoints simply vanish from `/api/docs`.

A new plugin that reads state produced by another invoke has the same
constraint. Say so in a comment at its call site in `appOptions()`.

## Adding one

1. `plugins/{name}.go`, package `plugins`.
2. Export `{Name}Plugin(options ...{Name}PluginOptions) fx.Option`.
3. Add it to `appOptions()` in [cmd/main.go](../cmd/main.go), placed according
   to its ordering needs.
4. Run `go test ./cmd/` — `fx.ValidateApp` type checks the new providers.

If the plugin mounts routes, document them with the `app/docs` constructors the
same way a controller does, so they appear in the OpenAPI document alongside
everything else.
