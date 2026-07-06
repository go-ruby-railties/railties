// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"reflect"
	"testing"
)

func TestGeneratorsBag(t *testing.T) {
	g := newGenerators()
	if _, ok := g.Get("orm"); ok {
		t.Fatal("unset generator option present")
	}
	g.Set("orm", "active_record").Set("test_framework", "minitest")
	if v, ok := g.Get("orm"); !ok || v != "active_record" {
		t.Fatalf("orm: %v %v", v, ok)
	}
}

func TestI18nDefaults(t *testing.T) {
	c := newI18nConfig()
	if c.DefaultLocale != "en" {
		t.Fatalf("default locale should be en, got %q", c.DefaultLocale)
	}
}

func klasses(entries []Middleware) []any {
	out := make([]any, len(entries))
	for i, e := range entries {
		out[i] = e.Klass
	}
	return out
}

func TestMiddlewareStackDirect(t *testing.T) {
	s := NewMiddlewareStack()
	s.Use("A", 1)
	s.Use("B")
	s.Unshift("Z")
	if got := klasses(s.Entries()); !reflect.DeepEqual(got, []any{"Z", "A", "B"}) {
		t.Fatalf("use/unshift order wrong: %v", got)
	}
	if s.Len() != 3 {
		t.Fatal("len wrong")
	}
	if s.Entries()[1].Args[0] != 1 {
		t.Fatal("args not carried")
	}
	// insert before/after existing
	if !s.InsertBefore("A", "beforeA") {
		t.Fatal("insert before existing should report found")
	}
	if !s.InsertAfter("B", "afterB") {
		t.Fatal("insert after existing should report found")
	}
	if got := klasses(s.Entries()); !reflect.DeepEqual(got, []any{"Z", "beforeA", "A", "B", "afterB"}) {
		t.Fatalf("insert order wrong: %v", got)
	}
	// insert before/after missing target (clamps to ends)
	if s.InsertBefore("nope", "head") {
		t.Fatal("insert before missing should report not found")
	}
	if s.InsertAfter("nope", "tail") {
		t.Fatal("insert after missing should report not found")
	}
	if got := klasses(s.Entries()); got[0] != "head" || got[len(got)-1] != "tail" {
		t.Fatalf("missing-target clamp wrong: %v", got)
	}
	// swap
	if !s.Swap("A", "A2") {
		t.Fatal("swap existing should report found")
	}
	if s.Swap("nope", "x") {
		t.Fatal("swap missing should report not found")
	}
	// delete
	if !s.Delete("A2") {
		t.Fatal("delete existing should report found")
	}
	if s.Delete("nope") {
		t.Fatal("delete missing should report not found")
	}
}

func TestMiddlewareProxyMergeInto(t *testing.T) {
	p := newMiddlewareStackProxy()
	p.Use("A").Unshift("Z").InsertBefore("A", "beforeA").
		InsertAfter("A", "afterA").Swap("Z", "Z2").Delete("afterA")
	if p.Ops() != 6 {
		t.Fatalf("recorded ops wrong: %d", p.Ops())
	}
	stack := NewMiddlewareStack()
	p.MergeInto(stack)
	if got := klasses(stack.Entries()); !reflect.DeepEqual(got, []any{"Z2", "beforeA", "A"}) {
		t.Fatalf("merge_into replay wrong: %v", got)
	}
}

func TestEngineConfiguration(t *testing.T) {
	c := newEngineConfiguration("/app")
	if c.Root() != "/app" {
		t.Fatal("root wrong")
	}
	c.Paths().Add("app/models", PathOpts{Autoload: true, EagerLoad: true})
	if len(c.EagerLoadPaths()) != 1 || len(c.AutoloadPaths()) != 1 {
		t.Fatal("engine config path aggregators wrong")
	}
	c.Generators().Set("orm", "ar")
	if v, _ := c.Generators().Get("orm"); v != "ar" {
		t.Fatal("engine generators wrong")
	}
	// SetRoot repoints the paths DSL
	c.SetRoot("/new")
	if c.Paths().Root() != "/new" {
		t.Fatal("SetRoot should repoint paths root")
	}
	// base Configuration still reachable via embedding
	c.Set("x", 1)
	if v, _ := c.Get("x"); v != 1 {
		t.Fatal("embedded configuration bag broken")
	}
}

func TestApplicationConfigurationLoadDefaults(t *testing.T) {
	c := newApplicationConfiguration("/app")
	if c.TimeZone != "UTC" {
		t.Fatalf("default time zone should be UTC, got %q", c.TimeZone)
	}
	if c.LoadedDefaults() != "" {
		t.Fatal("loaded defaults should start empty")
	}
	if _, ok := c.Default("anything"); ok {
		t.Fatal("no defaults before load")
	}
	if err := c.LoadDefaults("7.1"); err != nil {
		t.Fatal(err)
	}
	if c.LoadedDefaults() != "7.1" {
		t.Fatalf("loaded defaults version wrong: %q", c.LoadedDefaults())
	}
	if v, ok := c.Default("add_autoload_paths_to_load_path"); !ok || v != false {
		t.Fatalf("7.1 default flag wrong: %v %v", v, ok)
	}
	// unknown version errors
	if err := c.LoadDefaults("0.1"); err == nil {
		t.Fatal("unknown version should error")
	}
	// other known versions apply
	for _, v := range []string{"7.0", "8.0"} {
		if err := c.LoadDefaults(v); err != nil {
			t.Fatalf("LoadDefaults(%q): %v", v, err)
		}
	}
	// sub-configs
	c.EagerLoad = true
	if c.I18n().DefaultLocale != "en" {
		t.Fatal("i18n default wrong")
	}
	c.Generators().Set("orm", "ar")
	if v, _ := c.Generators().Get("orm"); v != "ar" {
		t.Fatal("app generators wrong")
	}
	c.Middleware().Use("A")
	if c.Middleware().Ops() != 1 {
		t.Fatal("middleware proxy wrong")
	}
}
