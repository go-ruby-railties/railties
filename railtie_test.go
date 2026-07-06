// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"errors"
	"reflect"
	"testing"
)

func TestConfigurationBag(t *testing.T) {
	c := newConfiguration()
	if _, ok := c.Get("missing"); ok {
		t.Fatal("unset key should not be present")
	}
	c.Set("a", 1).Set("b", "two")
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("get a: %v %v", v, ok)
	}
	if v, ok := c.Get("b"); !ok || v != "two" {
		t.Fatalf("get b: %v %v", v, ok)
	}
}

func TestRailtieNameInflection(t *testing.T) {
	cases := map[string]string{
		"MyApp::Application":    "my_app_application",
		"Foo::Railtie":          "foo",
		"Sprockets::Railtie":    "sprockets",
		"ActionMailer::Railtie": "action_mailer",
	}
	for in, want := range cases {
		r := NewRailtie(in)
		if got := r.RailtieName(); got != want {
			t.Fatalf("RailtieName(%q) = %q, want %q", in, got, want)
		}
		// second call hits the memoised branch
		if got := r.RailtieName(); got != want {
			t.Fatalf("memoised RailtieName(%q) = %q", in, got)
		}
	}
	if NewRailtie("Foo").Name() != "Foo" {
		t.Fatal("Name accessor wrong")
	}
	if NewRailtie("Foo").Config() == nil {
		t.Fatal("Config should be non-nil")
	}
}

func TestInitializerRegistrationDefaultAfterChaining(t *testing.T) {
	r := NewRailtie("X")
	// first: no chaining (list empty)
	r.Initializer("first", InitOpts{}, nil)
	// second: default after -> "first"
	r.Initializer("second", InitOpts{}, nil)
	// third: explicit after wins over chaining
	r.Initializer("third", InitOpts{After: "first"}, nil)
	// fourth: has Before matching an existing initializer -> no default-after
	r.Initializer("fourth", InitOpts{Before: "second"}, nil)
	ins := r.Initializers()
	byName := map[string]*Initializer{}
	for _, in := range ins {
		byName[in.Name] = in
	}
	if byName["first"].After != "" {
		t.Fatalf("first should have no after, got %q", byName["first"].After)
	}
	if byName["second"].After != "first" {
		t.Fatalf("second should chain after first, got %q", byName["second"].After)
	}
	if byName["third"].After != "first" {
		t.Fatalf("third explicit after wrong, got %q", byName["third"].After)
	}
	if byName["fourth"].After != "" {
		t.Fatalf("fourth has matching before, should not chain, got %q", byName["fourth"].After)
	}
	// context binding
	if byName["first"].Context() != r {
		t.Fatal("railtie initializers should be bound to the railtie")
	}
}

func TestHasInitializerBlankNeverMatches(t *testing.T) {
	r := NewRailtie("X")
	if r.hasInitializer("") {
		t.Fatal("blank name should never match")
	}
	r.Initializer("z", InitOpts{}, nil)
	if !r.hasInitializer("z") {
		t.Fatal("registered name should match")
	}
	if r.hasInitializer("nope") {
		t.Fatal("unknown name should not match")
	}
}

func TestRailtieHooks(t *testing.T) {
	r := NewRailtie("X")
	var log []string
	mk := func(tag string) Hook {
		return func(ctx any, args ...any) error { log = append(log, tag); return nil }
	}
	r.RakeTasks(mk("rake")).Console(mk("console")).Generators(mk("gen")).
		Runner(mk("runner")).Server(mk("server"))
	ctx := "c"
	if err := r.RunRakeTasks(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.RunConsole(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.RunGenerators(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.RunRunner(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.RunServer(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(log, []string{"rake", "console", "gen", "runner", "server"}) {
		t.Fatalf("hook order wrong: %v", log)
	}
}

func TestRailtieHookError(t *testing.T) {
	r := NewRailtie("X")
	boom := errors.New("boom")
	r.RakeTasks(func(ctx any, args ...any) error { return boom })
	if err := r.RunRakeTasks(nil); !errors.Is(err, boom) {
		t.Fatalf("hook error should propagate, got %v", err)
	}
}
