// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"errors"
	"testing"
)

func TestEngineBasics(t *testing.T) {
	e := NewEngine("Blog::Engine", "/srv/blog")
	if e.Config().Root() != "/srv/blog" {
		t.Fatal("engine root wrong")
	}
	if e.Paths() != e.Config().Paths() {
		t.Fatal("engine.paths and config.paths must be the same object")
	}
	// railtie facilities available via embedding
	if e.RailtieName() != "blog_engine" {
		t.Fatalf("engine railtie_name wrong: %q", e.RailtieName())
	}
	e.Initializer("blog.setup", InitOpts{}, nil)
	if len(e.Initializers()) != 1 {
		t.Fatal("engine should expose railtie initializers")
	}
}

func TestEngineEndpointAndRoutes(t *testing.T) {
	e := NewEngine("Blog::Engine", "")
	if e.Endpoint() != nil {
		t.Fatal("endpoint should start nil")
	}
	e.SetEndpoint("rack-app")
	if e.Endpoint() != "rack-app" {
		t.Fatal("endpoint setter failed")
	}
	rs := e.Routes()
	if rs.Draws() != 0 {
		t.Fatal("routes should start empty")
	}
	var ran int
	rs.Draw(func(r *RouteSet) error { ran++; return nil }).
		Draw(func(r *RouteSet) error { ran++; return nil })
	if rs.Draws() != 2 {
		t.Fatal("draws not recorded")
	}
	if err := rs.Run(); err != nil {
		t.Fatal(err)
	}
	if ran != 2 {
		t.Fatalf("draw blocks not replayed: %d", ran)
	}
	// draw error propagates
	boom := errors.New("boom")
	rs2 := newRouteSet()
	rs2.Draw(func(r *RouteSet) error { return boom })
	if err := rs2.Run(); !errors.Is(err, boom) {
		t.Fatalf("draw error should propagate, got %v", err)
	}
}

func TestEngineIsolateNamespace(t *testing.T) {
	e := NewEngine("Blog::Engine", "")
	if e.Isolated() {
		t.Fatal("should not be isolated initially")
	}
	if e.Namespace() != "" {
		t.Fatal("namespace should start empty")
	}
	// before isolation, EngineName falls back to railtie name
	if e.EngineName() != "blog_engine" {
		t.Fatalf("pre-isolation engine_name wrong: %q", e.EngineName())
	}
	e.IsolateNamespace("Blog")
	if !e.Isolated() {
		t.Fatal("should be isolated")
	}
	if e.Namespace() != "Blog" {
		t.Fatal("namespace wrong")
	}
	if e.EngineName() != "blog" {
		t.Fatalf("engine_name after isolate wrong: %q", e.EngineName())
	}
	if e.Routes().DefaultScope() != "Blog" {
		t.Fatal("isolate_namespace should set routes default scope")
	}
	// multi-segment module
	e2 := NewEngine("X", "")
	e2.IsolateNamespace("My::Blog")
	if e2.EngineName() != "my_blog" {
		t.Fatalf("multi-segment engine_name wrong: %q", e2.EngineName())
	}
}
