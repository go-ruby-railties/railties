// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import "fmt"

// Generators is the code-generator configuration, mirroring
// Rails::Configuration::Generators as a typed key/value bag (the actual
// generators and templates are deferred — see the roadmap).
type Generators struct {
	opts map[string]any
}

func newGenerators() *Generators { return &Generators{opts: map[string]any{}} }

// Set records a generator option (e.g. Set("orm", "active_record")) and returns
// the receiver, mirroring `config.generators { |g| g.orm :active_record }`.
func (g *Generators) Set(key string, value any) *Generators { g.opts[key] = value; return g }

// Get returns a generator option and whether it was set.
func (g *Generators) Get(key string) (any, bool) { v, ok := g.opts[key]; return v, ok }

// I18nConfig is the internationalisation configuration, mirroring the relevant
// slice of Rails' config.i18n.
type I18nConfig struct {
	DefaultLocale    string
	AvailableLocales []string
	LoadPath         []string
	Fallbacks        bool
}

func newI18nConfig() *I18nConfig { return &I18nConfig{DefaultLocale: "en"} }

// Middleware is a single opaque middleware entry — its class/name plus
// construction args. The actual middleware implementations are deferred; here a
// middleware is just an identity carried through the stack.
type Middleware struct {
	Klass any
	Args  []any
}

// MiddlewareStack is the ordered list of middlewares, mirroring
// ActionDispatch::MiddlewareStack as a plain list. MiddlewareStackProxy replays
// its recorded operations onto one of these.
type MiddlewareStack struct {
	entries []Middleware
}

// NewMiddlewareStack returns an empty stack.
func NewMiddlewareStack() *MiddlewareStack { return &MiddlewareStack{} }

// Entries returns the middlewares in order.
func (s *MiddlewareStack) Entries() []Middleware { return append([]Middleware(nil), s.entries...) }

// Len returns the number of middlewares.
func (s *MiddlewareStack) Len() int { return len(s.entries) }

// index returns the position of klass, or -1.
func (s *MiddlewareStack) index(klass any) int {
	for i, e := range s.entries {
		if e.Klass == klass {
			return i
		}
	}
	return -1
}

// Use appends a middleware (MiddlewareStack#use).
func (s *MiddlewareStack) Use(klass any, args ...any) {
	s.entries = append(s.entries, Middleware{Klass: klass, Args: args})
}

// Unshift prepends a middleware (MiddlewareStack#unshift).
func (s *MiddlewareStack) Unshift(klass any, args ...any) {
	s.entries = append([]Middleware{{Klass: klass, Args: args}}, s.entries...)
}

// insertAt inserts m at position i (clamped to the ends).
func (s *MiddlewareStack) insertAt(i int, m Middleware) {
	if i < 0 {
		i = 0
	}
	if i > len(s.entries) {
		i = len(s.entries)
	}
	s.entries = append(s.entries, Middleware{})
	copy(s.entries[i+1:], s.entries[i:])
	s.entries[i] = m
}

// InsertBefore inserts klass before target, mirroring insert_before. If target is
// absent it prepends. It reports whether target was found.
func (s *MiddlewareStack) InsertBefore(target, klass any, args ...any) bool {
	i := s.index(target)
	if i < 0 {
		s.insertAt(0, Middleware{Klass: klass, Args: args})
		return false
	}
	s.insertAt(i, Middleware{Klass: klass, Args: args})
	return true
}

// InsertAfter inserts klass after target, mirroring insert_after. If target is
// absent it appends. It reports whether target was found.
func (s *MiddlewareStack) InsertAfter(target, klass any, args ...any) bool {
	i := s.index(target)
	if i < 0 {
		s.insertAt(len(s.entries), Middleware{Klass: klass, Args: args})
		return false
	}
	s.insertAt(i+1, Middleware{Klass: klass, Args: args})
	return true
}

// Swap replaces target in place with klass, mirroring swap. It reports whether
// target was found (no-op if not).
func (s *MiddlewareStack) Swap(target, klass any, args ...any) bool {
	i := s.index(target)
	if i < 0 {
		return false
	}
	s.entries[i] = Middleware{Klass: klass, Args: args}
	return true
}

// Delete removes target, mirroring delete. It reports whether it was found.
func (s *MiddlewareStack) Delete(target any) bool {
	i := s.index(target)
	if i < 0 {
		return false
	}
	s.entries = append(s.entries[:i], s.entries[i+1:]...)
	return true
}

// mwOp is one recorded, deferred middleware operation.
type mwOp struct {
	kind   string
	target any
	klass  any
	args   []any
}

// MiddlewareStackProxy records middleware operations to be applied later, mirroring
// Rails::Configuration::MiddlewareStackProxy: config.middleware.use / insert_before
// / insert_after / swap / delete / unshift are recorded now and merged into the
// real stack during boot.
type MiddlewareStackProxy struct {
	ops []mwOp
}

func newMiddlewareStackProxy() *MiddlewareStackProxy { return &MiddlewareStackProxy{} }

// Use records a use operation.
func (p *MiddlewareStackProxy) Use(klass any, args ...any) *MiddlewareStackProxy {
	p.ops = append(p.ops, mwOp{kind: "use", klass: klass, args: args})
	return p
}

// Unshift records an unshift operation.
func (p *MiddlewareStackProxy) Unshift(klass any, args ...any) *MiddlewareStackProxy {
	p.ops = append(p.ops, mwOp{kind: "unshift", klass: klass, args: args})
	return p
}

// InsertBefore records an insert_before operation.
func (p *MiddlewareStackProxy) InsertBefore(target, klass any, args ...any) *MiddlewareStackProxy {
	p.ops = append(p.ops, mwOp{kind: "insert_before", target: target, klass: klass, args: args})
	return p
}

// InsertAfter records an insert_after operation.
func (p *MiddlewareStackProxy) InsertAfter(target, klass any, args ...any) *MiddlewareStackProxy {
	p.ops = append(p.ops, mwOp{kind: "insert_after", target: target, klass: klass, args: args})
	return p
}

// Swap records a swap operation.
func (p *MiddlewareStackProxy) Swap(target, klass any, args ...any) *MiddlewareStackProxy {
	p.ops = append(p.ops, mwOp{kind: "swap", target: target, klass: klass, args: args})
	return p
}

// Delete records a delete operation.
func (p *MiddlewareStackProxy) Delete(target any) *MiddlewareStackProxy {
	p.ops = append(p.ops, mwOp{kind: "delete", target: target})
	return p
}

// Ops returns the number of recorded operations.
func (p *MiddlewareStackProxy) Ops() int { return len(p.ops) }

// MergeInto replays the recorded operations onto stack in order and returns it,
// mirroring MiddlewareStackProxy#merge_into.
func (p *MiddlewareStackProxy) MergeInto(stack *MiddlewareStack) *MiddlewareStack {
	for _, op := range p.ops {
		switch op.kind {
		case "use":
			stack.Use(op.klass, op.args...)
		case "unshift":
			stack.Unshift(op.klass, op.args...)
		case "insert_before":
			stack.InsertBefore(op.target, op.klass, op.args...)
		case "insert_after":
			stack.InsertAfter(op.target, op.klass, op.args...)
		case "swap":
			stack.Swap(op.target, op.klass, op.args...)
		case "delete":
			stack.Delete(op.target)
		}
	}
	return stack
}

// EngineConfiguration is an Engine's configuration, mirroring
// Rails::Engine::Configuration: it adds a root, the paths DSL and a generators
// bag on top of the base Configuration.
type EngineConfiguration struct {
	*Configuration
	root       string
	paths      *PathsRoot
	generators *Generators
}

func newEngineConfiguration(root string) *EngineConfiguration {
	return &EngineConfiguration{
		Configuration: newConfiguration(),
		root:          root,
		paths:         NewPathsRoot(root),
		generators:    newGenerators(),
	}
}

// Root returns the engine root directory (Engine::Configuration#root).
func (c *EngineConfiguration) Root() string { return c.root }

// SetRoot changes the engine root and repoints the paths DSL at it.
func (c *EngineConfiguration) SetRoot(root string) {
	c.root = root
	c.paths.SetRoot(root)
}

// Paths returns the engine's paths DSL (Engine::Configuration#paths).
func (c *EngineConfiguration) Paths() *PathsRoot { return c.paths }

// Generators returns the engine's generators configuration.
func (c *EngineConfiguration) Generators() *Generators { return c.generators }

// EagerLoadPaths returns the eager-load paths from the paths DSL, mirroring
// Engine::Configuration#eager_load_paths.
func (c *EngineConfiguration) EagerLoadPaths() []string { return c.paths.EagerLoad() }

// AutoloadPaths returns the autoload paths from the paths DSL, mirroring
// Engine::Configuration#autoload_paths.
func (c *EngineConfiguration) AutoloadPaths() []string { return c.paths.AutoloadPaths() }

// defaultsTable maps a load_defaults version to the framework-default flags it
// turns on. It is a representative subset (the complete per-version flag matrix is
// deferred — see the roadmap); the versions themselves and the load_defaults
// mechanism are faithful.
var defaultsTable = map[string]map[string]any{
	"7.0": {
		"action_view.button_to_generates_button_tag": true,
		"action_dispatch.cookies_serializer":         "json",
	},
	"7.1": {
		"active_support.cache_format_version":         7.1,
		"action_dispatch.default_headers.x_permitted": true,
		"add_autoload_paths_to_load_path":             false,
	},
	"8.0": {
		"active_support.to_time_preserves_timezone": "zone",
		"action_dispatch.strict_freshness":          true,
	},
}

// ApplicationConfiguration is the top-level application configuration, mirroring
// Rails::Application::Configuration: load_defaults plus the eager-load, time-zone,
// i18n, generators and middleware settings.
type ApplicationConfiguration struct {
	*EngineConfiguration

	loadedDefaults string
	defaults       map[string]any

	EagerLoad  bool
	TimeZone   string
	i18n       *I18nConfig
	appGens    *Generators
	middleware *MiddlewareStackProxy
}

func newApplicationConfiguration(root string) *ApplicationConfiguration {
	return &ApplicationConfiguration{
		EngineConfiguration: newEngineConfiguration(root),
		defaults:            map[string]any{},
		TimeZone:            "UTC",
		i18n:                newI18nConfig(),
		appGens:             newGenerators(),
		middleware:          newMiddlewareStackProxy(),
	}
}

// LoadDefaults applies the framework defaults for the given version and records
// it, mirroring config.load_defaults(version). An unknown version is an error, as
// Rails raises for one.
func (c *ApplicationConfiguration) LoadDefaults(version string) error {
	flags, ok := defaultsTable[version]
	if !ok {
		return fmt.Errorf("rails: unknown load_defaults version %q", version)
	}
	for k, v := range flags {
		c.defaults[k] = v
	}
	c.loadedDefaults = version
	return nil
}

// LoadedDefaults returns the version passed to the most recent LoadDefaults, or ""
// (config.loaded_config_version-style accessor).
func (c *ApplicationConfiguration) LoadedDefaults() string { return c.loadedDefaults }

// Default returns a flag applied by LoadDefaults and whether it was set.
func (c *ApplicationConfiguration) Default(key string) (any, bool) {
	v, ok := c.defaults[key]
	return v, ok
}

// I18n returns the i18n configuration (config.i18n).
func (c *ApplicationConfiguration) I18n() *I18nConfig { return c.i18n }

// Generators returns the application's generators configuration, mirroring
// Application::Configuration#generators (distinct from the engine bag).
func (c *ApplicationConfiguration) Generators() *Generators { return c.appGens }

// Middleware returns the middleware stack proxy (config.middleware).
func (c *ApplicationConfiguration) Middleware() *MiddlewareStackProxy { return c.middleware }
