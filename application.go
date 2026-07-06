// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import "errors"

// ErrAlreadyInitialized is returned by Application.Initialize when the application
// has already booted, mirroring the "Application has been already initialized."
// guard.
var ErrAlreadyInitialized = errors.New("rails: application has already been initialized")

// Application is the top-level Rails application, mirroring Rails::Application <
// Rails::Engine. It embeds *Engine (hence *Railtie), assembles the full
// bootstrap → railtie → finisher initializer chain and runs it in dependency
// order during Initialize.
type Application struct {
	*Engine

	config      *ApplicationConfiguration
	initialized bool
	railties    []*Railtie
	hooks       *LazyLoadHooks

	secrets     any
	credentials any

	// RunInitializer is the host-supplied seam that supplies the deferred body of
	// every initializer that carries no Go Block — the bootstrap and finisher
	// initializers, and any registered without a block. It is called with the
	// initializer's name and its bound context (the railtie/engine/app). Leaving
	// it nil makes those initializers no-ops.
	RunInitializer InitializerSeam
}

// NewApplication returns an Application named name, rooted at root. It registers
// itself as the current Rails application (Rails.application) and sets Rails.root,
// mirroring the side effects of defining a Rails::Application subclass.
func NewApplication(name, root string) *Application {
	eng := NewEngine(name, root)
	appCfg := newApplicationConfiguration(root)
	// Share one EngineConfiguration so engine.paths and config.paths are the same
	// object, exactly as in Rails.
	eng.config = appCfg.EngineConfiguration
	app := &Application{
		Engine: eng,
		config: appCfg,
		hooks:  NewLazyLoadHooks(),
	}
	SetApp(app)
	SetRoot(root)
	return app
}

// Config returns the application configuration, shadowing Engine.Config so callers
// get the richer ApplicationConfiguration (Application#config).
func (a *Application) Config() *ApplicationConfiguration { return a.config }

// Hooks returns the application's lazy load-hook registry (backing
// ActiveSupport.on_load / run_load_hooks). See RunLoadHooks / OnLoad.
func (a *Application) Hooks() *LazyLoadHooks { return a.hooks }

// OnLoad registers a load hook on the application's registry (ActiveSupport.on_load).
func (a *Application) OnLoad(name string, hook LoadHook) error { return a.hooks.OnLoad(name, hook) }

// RunLoadHooks fires a load event on the application's registry (run_load_hooks).
func (a *Application) RunLoadHooks(name string, base any) error {
	return a.hooks.RunLoadHooks(name, base)
}

// AddRailtie registers a railtie (or engine, via its embedded *Railtie) whose
// initializers are collected into the boot chain, mirroring how Rails collects its
// railties. It returns the receiver.
func (a *Application) AddRailtie(r *Railtie) *Application {
	a.railties = append(a.railties, r)
	return a
}

// Railties returns the registered railties in registration order.
func (a *Application) Railties() []*Railtie { return append([]*Railtie(nil), a.railties...) }

// Initialized reports whether Initialize has run (Application#initialized?).
func (a *Application) Initialized() bool { return a.initialized }

// Secrets returns the application secrets (a seam; the real store is deferred).
func (a *Application) Secrets() any { return a.secrets }

// SetSecrets sets the application secrets seam and returns the receiver.
func (a *Application) SetSecrets(v any) *Application { a.secrets = v; return a }

// Credentials returns the application credentials (a seam; the real encrypted
// store is deferred).
func (a *Application) Credentials() any { return a.credentials }

// SetCredentials sets the application credentials seam and returns the receiver.
func (a *Application) SetCredentials(v any) *Application { a.credentials = v; return a }

// Env returns Rails.env (the process environment string inquirer).
func (a *Application) Env() StringInquirer { return Env() }

// ownInitializers returns the application's own registered initializers bound to
// the application (so their context is the app, not the base Railtie).
func (a *Application) ownInitializers() Collection {
	src := a.Engine.Railtie.initializers
	out := make(Collection, len(src))
	for i, in := range src {
		out[i] = in.Bind(a)
	}
	return out
}

// boundNamed builds a collection of body-less initializers (bodies deferred to the
// RunInitializer seam) with the given names and group, bound to ctx.
func boundNamed(ctx any, names []string, group string) Collection {
	c := make(Collection, len(names))
	for i, n := range names {
		c[i] = (&Initializer{Name: n, Group: group}).Bind(ctx)
	}
	return c
}

// bootstrapInitializers are the canonical Rails bootstrap initializers, in order,
// in the "all" group so they run in every pass. Their bodies are seams. The full
// bodies (loading the environment, the logger, the cache, …) are deferred.
func bootstrapInitializers(a *Application) Collection {
	return boundNamed(a, []string{
		"load_environment_hook",
		"load_active_support",
		"set_eager_load",
		"initialize_logger",
		"initialize_cache",
		"bootstrap_hook",
		"set_secrets_root",
	}, "all")
}

// finisherInitializers are a representative subset of the canonical Rails finisher
// initializers, in order, in the default group. Their bodies are seams. The
// complete finisher list is deferred (see the roadmap).
func finisherInitializers(a *Application) Collection {
	return boundNamed(a, []string{
		"add_generator_templates",
		"build_middleware_stack",
		"define_main_app_helper",
		"add_builtin_route",
		"eager_load!",
		"finisher_hook",
		"set_routes_reloader_hook",
	}, "default")
}

// Initializers returns the fully-assembled boot chain — bootstrap, then each
// registered railtie's initializers (in registration order), then the
// application's own initializers, then the finishers — every initializer bound to
// its context. This is the collection Initialize topologically sorts and runs,
// mirroring Application#initializers.
func (a *Application) Initializers() Collection {
	var c Collection
	c = append(c, bootstrapInitializers(a)...)
	for _, rt := range a.railties {
		c = append(c, rt.Initializers()...)
	}
	c = append(c, a.ownInitializers()...)
	c = append(c, finisherInitializers(a)...)
	return c
}

// Initialize boots the application: it fires the before_initialize load hooks,
// topologically sorts the assembled initializers and runs those belonging to the
// group (default "default"; bootstrap initializers are in "all" and so always
// run), then fires after_initialize. It mirrors Rails::Application#initialize! and
// is guarded so it runs at most once (ErrAlreadyInitialized otherwise).
//
// Initializer bodies are supplied by RunInitializer (for body-less initializers)
// or their attached Block.
func (a *Application) Initialize(group ...string) error {
	g := "default"
	if len(group) > 0 {
		g = group[0]
	}
	if a.initialized {
		return ErrAlreadyInitialized
	}
	if err := a.hooks.RunLoadHooks("before_initialize", a); err != nil {
		return err
	}
	if err := a.Initializers().RunInitializers(g, a.RunInitializer); err != nil {
		return err
	}
	a.initialized = true
	if err := a.hooks.RunLoadHooks("after_initialize", a); err != nil {
		return err
	}
	return nil
}
