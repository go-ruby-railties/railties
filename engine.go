// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"strings"

	"github.com/go-ruby-activesupport/activesupport/inflector"
)

// RouteBlock is a routing definition block, mirroring the block passed to
// `Engine.routes.draw do … end`. The actual routing DSL is deferred; a block is
// recorded and can be replayed by a host.
type RouteBlock func(r *RouteSet) error

// RouteSet is an engine's route set, mirroring the relevant surface of
// ActionDispatch::Routing::RouteSet: draw blocks are recorded in order, plus the
// default scope module set by isolate_namespace. The concrete route matching /
// dispatch is deferred (see the roadmap).
type RouteSet struct {
	draws        []RouteBlock
	defaultScope string // the module namespace routes are drawn under
}

func newRouteSet() *RouteSet { return &RouteSet{} }

// Draw records a routing block and returns the receiver, mirroring routes.draw.
func (rs *RouteSet) Draw(block RouteBlock) *RouteSet { rs.draws = append(rs.draws, block); return rs }

// Draws returns the number of recorded routing blocks.
func (rs *RouteSet) Draws() int { return len(rs.draws) }

// DefaultScope returns the default routing module set by isolate_namespace.
func (rs *RouteSet) DefaultScope() string { return rs.defaultScope }

// Run replays the recorded routing blocks in order against this set, stopping on
// the first error. Hosts use it to materialise the routes once the DSL is wired.
func (rs *RouteSet) Run() error {
	for _, d := range rs.draws {
		if err := d(rs); err != nil {
			return err
		}
	}
	return nil
}

// Engine is a Railtie with paths, routes, an endpoint and namespace isolation,
// mirroring Rails::Engine < Rails::Railtie. It embeds *Railtie so every railtie
// facility (initializers, hooks, railtie_name) is available.
type Engine struct {
	*Railtie

	config     *EngineConfiguration
	routes     *RouteSet
	endpoint   any
	namespace  string
	isolated   bool
	engineName string // memoised
}

// NewEngine returns an Engine identified by name, rooted at root (the empty root
// is valid — paths simply expand relative to it).
func NewEngine(name, root string) *Engine {
	return &Engine{
		Railtie: NewRailtie(name),
		config:  newEngineConfiguration(root),
		routes:  newRouteSet(),
	}
}

// Config returns the engine's configuration, mirroring Engine#config. It shadows
// the embedded Railtie.Config so callers get the richer EngineConfiguration.
func (e *Engine) Config() *EngineConfiguration { return e.config }

// Paths returns the engine's paths DSL, mirroring Engine#paths (the same object as
// config.paths).
func (e *Engine) Paths() *PathsRoot { return e.config.paths }

// Endpoint returns the engine's Rack endpoint (Engine.endpoint reader).
func (e *Engine) Endpoint() any { return e.endpoint }

// SetEndpoint sets the engine's Rack endpoint and returns the receiver
// (Engine.endpoint writer). The endpoint itself is an opaque seam.
func (e *Engine) SetEndpoint(app any) *Engine { e.endpoint = app; return e }

// Routes returns the engine's route set, mirroring Engine#routes.
func (e *Engine) Routes() *RouteSet { return e.routes }

// IsolateNamespace marks the engine as namespace-isolated under mod, mirroring
// Engine.isolate_namespace: it sets the engine name (underscored module name) and
// the routes' default scope, so the engine's models/controllers/routes live in
// their own namespace. It returns the receiver.
func (e *Engine) IsolateNamespace(mod string) *Engine {
	e.isolated = true
	e.namespace = mod
	e.engineName = inflector.Underscore(strings.ReplaceAll(mod, "::", "/"))
	e.engineName = strings.ReplaceAll(e.engineName, "/", "_")
	e.routes.defaultScope = mod
	return e
}

// Isolated reports whether isolate_namespace was called (Engine.isolated?).
func (e *Engine) Isolated() bool { return e.isolated }

// Namespace returns the isolated module name, or "" (Engine.railtie_namespace-ish).
func (e *Engine) Namespace() string { return e.namespace }

// EngineName returns the engine's short name: the isolated namespace underscored
// once isolate_namespace has run, otherwise the railtie name. Mirrors Engine.engine_name.
func (e *Engine) EngineName() string {
	if e.engineName != "" {
		return e.engineName
	}
	return e.RailtieName()
}
