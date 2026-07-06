// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestPathsRootAddDefaultsAndGet(t *testing.T) {
	r := NewPathsRoot("/app")
	if r.Root() != "/app" {
		t.Fatal("root wrong")
	}
	p := r.Add("config", PathOpts{})
	if !reflect.DeepEqual(p.To(), []string{"config"}) {
		t.Fatalf("default sub-path should be the key, got %v", p.To())
	}
	if r.Get("config") != p {
		t.Fatal("Get should return the added path")
	}
	if r.Get("missing") != nil {
		t.Fatal("Get of missing key should be nil")
	}
	// With override
	p2 := r.Add("app", PathOpts{With: []string{"app/models", "app/controllers"}})
	if !reflect.DeepEqual(p2.To(), []string{"app/models", "app/controllers"}) {
		t.Fatalf("with override wrong: %v", p2.To())
	}
	// Re-add same key replaces, does not duplicate the key ordering.
	r.Add("config", PathOpts{With: []string{"cfg"}})
	if got := r.Keys(); !reflect.DeepEqual(got, []string{"config", "app"}) {
		t.Fatalf("keys order wrong after re-add: %v", got)
	}
	if vals := r.Values(); len(vals) != 2 {
		t.Fatalf("values length wrong: %d", len(vals))
	}
}

func TestPathExpandedAbsoluteAndRelative(t *testing.T) {
	r := NewPathsRoot("/app")
	p := r.Add("mixed", PathOpts{With: []string{"lib", "/abs/path"}})
	got := p.Expanded()
	want := []string{filepath.Join("/app", "lib"), "/abs/path"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded wrong: %v want %v", got, want)
	}
	// SetRoot repoints expansion.
	r.SetRoot("/other")
	if p.Expanded()[0] != filepath.Join("/other", "lib") {
		t.Fatalf("SetRoot should repoint expansion, got %v", p.Expanded())
	}
}

func TestPathFlagsAndPushUnshift(t *testing.T) {
	r := NewPathsRoot("/app")
	p := r.Add("app/models", PathOpts{})
	if p.EagerLoadQ() || p.AutoloadQ() || p.AutoloadOnceQ() || p.LoadPathQ() {
		t.Fatal("flags should default false")
	}
	p.EagerLoad().Autoload().AutoloadOnce().LoadPath()
	if !p.EagerLoadQ() || !p.AutoloadQ() || !p.AutoloadOnceQ() || !p.LoadPathQ() {
		t.Fatal("flag setters did not take")
	}
	p.Push("extra")
	p.Unshift("front")
	if !reflect.DeepEqual(p.To(), []string{"front", "app/models", "extra"}) {
		t.Fatalf("push/unshift order wrong: %v", p.To())
	}
}

func TestPathsRootAggregators(t *testing.T) {
	r := NewPathsRoot("/app")
	r.Add("app/models", PathOpts{Autoload: true, EagerLoad: true, LoadPath: true})
	r.Add("app/mailers", PathOpts{Autoload: true, EagerLoad: true})
	r.Add("lib", PathOpts{AutoloadOnce: true, LoadPath: true})
	r.Add("config", PathOpts{}) // classified nowhere

	j := func(p string) string { return filepath.Join("/app", p) }
	if got := r.AutoloadPaths(); !reflect.DeepEqual(got, []string{j("app/models"), j("app/mailers")}) {
		t.Fatalf("autoload_paths wrong: %v", got)
	}
	if got := r.EagerLoad(); !reflect.DeepEqual(got, []string{j("app/models"), j("app/mailers")}) {
		t.Fatalf("eager_load wrong: %v", got)
	}
	if got := r.AutoloadOnce(); !reflect.DeepEqual(got, []string{j("lib")}) {
		t.Fatalf("autoload_once wrong: %v", got)
	}
	if got := r.LoadPaths(); !reflect.DeepEqual(got, []string{j("app/models"), j("lib")}) {
		t.Fatalf("load_paths wrong: %v", got)
	}
}
