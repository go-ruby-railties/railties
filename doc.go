// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package rails is a pure-Go (no cgo) reimplementation of the core of Ruby on
// Rails' railties gem — the Rails::Railtie / Rails::Engine / Rails::Application
// boot framework — faithful to MRI Rails 4.0.5 (the railties the go-embedded-ruby
// world targets).
//
// railties is the framework that boots a Rails application: it defines the
// Railtie base every framework hangs off, the Initializable topological-sort that
// orders initializers across every railtie, the Engine that layers a paths DSL
// and routing/namespace on top, and the Application that assembles the whole
// bootstrap → railtie → finisher initializer chain and runs it.
//
// # Fidelity boundary — bodies are seams
//
// This is the v0.1 foundation: the structural core (registration, ordering, the
// boot sequence, configuration and the Rails globals) is complete and exact. The
// initializer *bodies* — the real work each initializer does inside a running
// Rails process (loading the environment, building the middleware stack, eager
// loading, hooking the routes reloader, …) — are seams. A host supplies them
// through the RunInitializer seam (see Application.RunInitializer) or by attaching
// a Block to an Initializer. Everything that concerns the *order* initializers run
// in, and the machinery that runs them, is implemented and tested here.
//
// # Package name
//
// The gem is named "railties" (hence the module path) but its Ruby namespace is
// Rails::, so the Go package is named rails: rails.Railtie, rails.Engine,
// rails.Application, rails.Root, rails.Env — mirroring Rails.root / Rails.env.
//
// # Reuse
//
// Names are inflected with go-ruby-activesupport's inflector (railtie_name,
// isolate_namespace), exactly as MRI railties leans on ActiveSupport::Inflector.
package rails
