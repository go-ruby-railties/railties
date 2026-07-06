// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"strings"

	"github.com/go-ruby-activesupport/activesupport/inflector"
)

// Hook is the body of a railtie extension point (rake_tasks, console, runner,
// generators, server), mirroring the block passed to those class methods. It is a
// seam: the host attaches real behaviour.
type Hook func(ctx any, args ...any) error

// InitOpts are the options accepted when registering an initializer, mirroring the
// keyword options of `initializer name, before:, after:, group:`.
type InitOpts struct {
	Before string
	After  string
	Group  string
}

// Configuration is the per-railtie configuration bag, mirroring
// Rails::Railtie::Configuration. Rails' configuration objects are radically
// dynamic (`config.anything = value` via method_missing), so the base is a typed
// key/value bag; Engine and Application layer strongly-typed fields on top.
type Configuration struct {
	opts map[string]any
}

func newConfiguration() *Configuration { return &Configuration{opts: map[string]any{}} }

// Set assigns an arbitrary configuration value and returns the receiver for
// chaining, mirroring dynamic `config.key = value` assignment.
func (c *Configuration) Set(key string, value any) *Configuration {
	c.opts[key] = value
	return c
}

// Get returns a configuration value and whether it was set, mirroring reading
// `config.key`.
func (c *Configuration) Get(key string) (any, bool) {
	v, ok := c.opts[key]
	return v, ok
}

// Railtie is the base every Rails framework hangs off, mirroring Rails::Railtie:
// it owns a Configuration, a registry of initializers, and the rake_tasks /
// console / generators / runner / server extension hooks.
type Railtie struct {
	name         string
	config       *Configuration
	initializers Collection

	rakeTasksHooks  []Hook
	consoleHooks    []Hook
	generatorsHooks []Hook
	runnerHooks     []Hook
	serverHooks     []Hook

	railtieName string // memoised
}

// NewRailtie returns a Railtie identified by name (the Ruby class name, e.g.
// "MyEngine::Railtie" or "MyApp::Application").
func NewRailtie(name string) *Railtie {
	return &Railtie{name: name, config: newConfiguration()}
}

// Name returns the railtie's identifying class name.
func (r *Railtie) Name() string { return r.name }

// Config returns the railtie's Configuration, mirroring Railtie#config.
func (r *Railtie) Config() *Configuration { return r.config }

// RailtieName returns the railtie's short name, mirroring Railtie.railtie_name:
// the underscored class name with "::" flattened to "_" and a trailing
// "_railtie" stripped (memoised on first call). It uses ActiveSupport's inflector,
// exactly as MRI railties does.
func (r *Railtie) RailtieName() string {
	if r.railtieName == "" {
		n := inflector.Underscore(strings.ReplaceAll(r.name, "::", "/"))
		n = strings.ReplaceAll(n, "/", "_")
		r.railtieName = strings.TrimSuffix(n, "_railtie")
	}
	return r.railtieName
}

// Initializer registers an initializer, mirroring the class-level
// `initializer name, opts, &block`. When After is unset it defaults to the name
// of the previously-registered initializer — unless this is the first one or an
// already-registered initializer matches this one's Before — reproducing Rails'
// default-chaining:
//
//	opts[:after] ||= initializers.last.name unless
//	  initializers.empty? || initializers.find { |i| i.name == opts[:before] }
//
// The block may be nil, in which case the body is supplied later through the
// RunInitializer seam. It returns the receiver for chaining.
func (r *Railtie) Initializer(name string, opts InitOpts, block Runner) *Railtie {
	after := opts.After
	if after == "" && len(r.initializers) > 0 && !r.hasInitializer(opts.Before) {
		after = r.initializers[len(r.initializers)-1].Name
	}
	r.initializers = append(r.initializers, &Initializer{
		Name:   name,
		Before: opts.Before,
		After:  after,
		Group:  opts.Group,
		Block:  block,
	})
	return r
}

// hasInitializer reports whether an initializer named name is already registered.
// A blank name (no Before constraint) never matches.
func (r *Railtie) hasInitializer(name string) bool {
	if name == "" {
		return false
	}
	for _, i := range r.initializers {
		if i.Name == name {
			return true
		}
	}
	return false
}

// Initializers returns the railtie's initializers bound to it, mirroring
// Railtie#initializers (initializers_for(self)). The returned initializers carry
// this railtie as their context.
func (r *Railtie) Initializers() Collection {
	out := make(Collection, len(r.initializers))
	for i, in := range r.initializers {
		out[i] = in.Bind(r)
	}
	return out
}

// RakeTasks registers a rake_tasks hook and returns the receiver (Railtie.rake_tasks).
func (r *Railtie) RakeTasks(h Hook) *Railtie {
	r.rakeTasksHooks = append(r.rakeTasksHooks, h)
	return r
}

// Console registers a console hook and returns the receiver (Railtie.console).
func (r *Railtie) Console(h Hook) *Railtie { r.consoleHooks = append(r.consoleHooks, h); return r }

// Generators registers a generators hook and returns the receiver (Railtie.generators).
func (r *Railtie) Generators(h Hook) *Railtie {
	r.generatorsHooks = append(r.generatorsHooks, h)
	return r
}

// Runner registers a runner hook and returns the receiver (Railtie.runner).
func (r *Railtie) Runner(h Hook) *Railtie { r.runnerHooks = append(r.runnerHooks, h); return r }

// Server registers a server hook and returns the receiver (Railtie.server).
func (r *Railtie) Server(h Hook) *Railtie { r.serverHooks = append(r.serverHooks, h); return r }

// runHooks runs a hook slice in registration order, stopping on the first error.
func runHooks(hooks []Hook, ctx any, args ...any) error {
	for _, h := range hooks {
		if err := h(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

// RunRakeTasks runs the registered rake_tasks hooks in order.
func (r *Railtie) RunRakeTasks(ctx any, args ...any) error {
	return runHooks(r.rakeTasksHooks, ctx, args...)
}

// RunConsole runs the registered console hooks in order.
func (r *Railtie) RunConsole(ctx any, args ...any) error {
	return runHooks(r.consoleHooks, ctx, args...)
}

// RunGenerators runs the registered generators hooks in order.
func (r *Railtie) RunGenerators(ctx any, args ...any) error {
	return runHooks(r.generatorsHooks, ctx, args...)
}

// RunRunner runs the registered runner hooks in order.
func (r *Railtie) RunRunner(ctx any, args ...any) error {
	return runHooks(r.runnerHooks, ctx, args...)
}

// RunServer runs the registered server hooks in order.
func (r *Railtie) RunServer(ctx any, args ...any) error {
	return runHooks(r.serverHooks, ctx, args...)
}
