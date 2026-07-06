<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-railties/brand/main/social/go-ruby-railties-railties.png" alt="go-ruby-railties/railties" width="720"></p>

# railties — go-ruby-railties

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-railties.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the core of Ruby on Rails'
[`railties`](https://github.com/rails/rails/tree/main/railties)** — the app-boot
framework: `Rails::Railtie`, `Rails::Engine`, `Rails::Application`, the
`Initializable` topological-sort that orders initializers across every railtie,
and the paths / configuration DSL — faithful to MRI Rails 4.0.5, **without any Ruby
runtime**.

It is the railties backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of
[go-ruby-activesupport](https://github.com/go-ruby-activesupport/activesupport)
(whose inflector it reuses for `railtie_name` / `isolate_namespace`) and
[go-ruby-set](https://github.com/go-ruby-set/set).

> **v0.1 is the foundation.** The structural core — registration, the initializer
> ordering, the boot sequence, configuration and the `Rails.*` globals — is
> complete and exact. The initializer *bodies* (the work each initializer does in a
> live Rails process) are **seams**: the host supplies them. See
> [Roadmap](#roadmap).

## The package is `rails`

The gem is named `railties` (hence the module path) but its Ruby namespace is
`Rails::`, so the Go package is `rails` — call sites read like the Ruby:

```go
import "github.com/go-ruby-railties/railties" // package rails

app := rails.NewApplication("MyApp::Application", "/srv/app")
_ = rails.Root()  // Rails.root
_ = rails.Env()   // Rails.env  (a StringInquirer: rails.Env().Is("production"))
```

## The initializer ordering — the fidelity-critical piece

Rails orders initializers by a **topological sort** over their `before:` / `after:`
constraints, and this port reproduces
`ActiveSupport::Initializable::Collection#tsort_each` exactly:

- **Registration order is the base order.** `Collection.TSort` visits nodes in the
  order they were registered (`tsort_each_node`), so unconstrained initializers keep
  their registration order.
- **Children run first.** The predecessors of an initializer `n` are, exactly as in
  Rails, `select { |i| i.before == n.name || i.name == n.after }` — every
  initializer declaring `before: n.name`, plus the one named by `n.after`.
- **Default-after chaining.** When you register an initializer with no `after:`, it
  defaults to the previously-registered initializer's name — *unless* the list is
  empty or an existing initializer already matches its `before:` — reproducing
  `opts[:after] ||= initializers.last.name unless …`. This is what keeps a
  railtie's initializers in declaration order by default.
- **Ruby's TSort, faithfully.** The sort is Ruby's own `TSort`: nodes visited in
  registration order, Tarjan's strongly-connected-component algorithm yielding
  components in post-order. A component of one node is emitted; a component of two
  or more is a genuine cycle → `*CyclicError` (mirroring `TSort::Cyclic`). Because
  it is SCC-based, a lone self-referential node is **not** a cycle — matching
  `TSort#tsort`.

`Application.Initialize` (`initialize!`) assembles the full chain — **bootstrap**
initializers first (the canonical `load_environment_hook … set_secrets_root`, in the
`all` group so they run in every pass), then each registered railtie's initializers,
then the application's own, then the **finisher** initializers — topologically sorts
it, and runs those belonging to the group, once.

## The `RunInitializer` seam

Initializer bodies are deferred. Each `Initializer` may carry a Go `Block`
(`func(ctx any, args ...any) error`); those without one dispatch to the
host-supplied **`RunInitializer`** seam:

```go
type InitializerSeam func(name string, ctx any) error
```

`Application.RunInitializer` is where a host (e.g. go-embedded-ruby, or a real Rails
boot) provides the deferred bodies of the bootstrap/finisher initializers and any
registered without a block. `ctx` is the initializer's bound context (the
railtie/engine/app). Leaving the seam `nil` makes body-less initializers no-ops —
so the ordering machinery is fully testable in isolation with fake seams.

## Usage

```go
package main

import (
	"fmt"

	rails "github.com/go-ruby-railties/railties"
)

func main() {
	app := rails.NewApplication("Blog::Application", "/srv/blog")
	app.Config().LoadDefaults("8.0")
	app.Config().EagerLoad = true
	app.Config().Paths().Add("app/models", rails.PathOpts{Autoload: true, EagerLoad: true})
	app.Config().Middleware().Use("Rack::Runtime")

	// A railtie contributing an ordered initializer.
	rt := rails.NewRailtie("Blog::Railtie")
	rt.Initializer("blog.load_config", rails.InitOpts{Before: "build_middleware_stack"}, nil)
	app.AddRailtie(rt)

	// Host supplies the deferred bodies.
	app.RunInitializer = func(name string, ctx any) error {
		fmt.Println("running:", name)
		return nil
	}

	if err := app.Initialize(); err != nil { // initialize!
		panic(err)
	}
	fmt.Println(rails.Root(), rails.Env()) // Rails.root, Rails.env
}
```

## API surface (v0.1)

- **`Railtie`** — `NewRailtie`, `Config`, `RailtieName` (via ActiveSupport's
  inflector), `Initializer(name, InitOpts{Before,After,Group}, block)`,
  `Initializers`, and the `RakeTasks` / `Console` / `Generators` / `Runner` /
  `Server` hooks with their `Run…` executors.
- **`Initializable`** — `Initializer` (`BelongsTo`, `Bind`, `Run`), `Collection`
  (`TSort`, `RunInitializers`), `CyclicError`, the `Runner` and `InitializerSeam`
  seams.
- **`Engine`** (embeds `*Railtie`) — `NewEngine`, `Paths`, `Endpoint`/`SetEndpoint`,
  `Routes` (`RouteSet.Draw`/`Run`), `IsolateNamespace`, `EngineName`, `Config`
  (`EngineConfiguration`).
- **`Application`** (embeds `*Engine`) — `NewApplication`, `AddRailtie`,
  `Initializers`, `Initialize` (the boot sequence), `RunInitializer` seam,
  `OnLoad`/`RunLoadHooks`, `Secrets`/`Credentials` seams, `Config`
  (`ApplicationConfiguration`: `LoadDefaults`, `EagerLoad`, `TimeZone`, `I18n`,
  `Generators`, `Middleware`).
- **`Paths::Root`/`Path`** — `NewPathsRoot`, `Add`, `Get`, `Keys`, `Values`, the
  per-`Path` flags (`EagerLoad`/`Autoload`/`AutoloadOnce`/`LoadPath` + `…Q`
  predicates), `Push`/`Unshift`, `Expanded`, and the root aggregators
  `AutoloadPaths` / `AutoloadOnce` / `EagerLoad` / `LoadPaths`.
- **Configuration** — `MiddlewareStack` + `MiddlewareStackProxy` (the deferred-op
  stack: `Use`/`Unshift`/`InsertBefore`/`InsertAfter`/`Swap`/`Delete` → `MergeInto`),
  `Generators`, `I18nConfig`.
- **`Rails.*` globals** — `App`/`SetApp`, `Root`/`SetRoot`, `Env`/`SetEnv`
  (`StringInquirer`, derived from `RAILS_ENV`→`RACK_ENV`→`development`),
  `Logger`/`SetLogger`, `ResetGlobals`.
- **Load hooks** — `LazyLoadHooks` (`OnLoad`/`RunLoadHooks`), the pure-Go
  `ActiveSupport::LazyLoadHooks`.

## Roadmap

Deferred to later versions (tracked so the boundary is explicit):

- **Initializer bodies** — the real work behind the bootstrap/finisher initializers
  (load the environment, build the middleware stack, eager-load, hook the routes
  reloader, …). Today they are seams.
- **The `rails` CLI** — `bin/rails`, `rails new`, `server`, `console`, `runner`,
  `rails stats`, and the command dispatcher.
- **Generators & templates** — the generator base classes, generator resolution,
  and application/plugin templates.
- **Middleware implementations** — the actual Rack middlewares; v0.1 models only the
  stack-as-a-list and its operations.
- **Routing DSL** — `RouteSet` records and replays `draw` blocks; the concrete
  route matching / URL helpers / mounting are deferred.
- **Full `load_defaults` matrix** — the complete per-version framework-defaults flag
  table (v0.1 ships a representative subset for 7.0 / 7.1 / 8.0).
- **Plugin / engine mounting** wiring, `secrets` / `credentials` encrypted stores,
  code statistics, and the paths `existent` / glob expansion.

## Tests & coverage

Deterministic, ruby-free tests hold coverage at **100%** — the initializer
registration and default-after chaining, the TSort ordering (before/after,
default-chaining, self-loop, 2- and 3-node cycle detection), the engine paths DSL,
the application boot running groups in order (via fake `RunInitializer` seams), the
`load_defaults` table, the middleware stack operations, and the `Rails.*` globals.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

CGO-free, `gofmt` + `go vet` clean, and green across the six 64-bit Go targets
(amd64, arm64, riscv64, loong64, ppc64le, s390x — including big-endian s390x) and
three OSes (Linux, macOS, Windows).

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-railties/railties authors.
