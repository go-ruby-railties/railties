// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"errors"
	"testing"
)

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func TestApplicationBootOrder(t *testing.T) {
	ResetGlobals()
	app := NewApplication("MyApp::Application", "/srv/app")
	if App() != app {
		t.Fatal("NewApplication should register Rails.application")
	}
	if Root() != "/srv/app" {
		t.Fatal("NewApplication should set Rails.root")
	}
	if app.Config().Root() != "/srv/app" {
		t.Fatal("app config root wrong")
	}
	// engine.paths and app.config.paths are the same object
	if app.Paths() != app.Config().Paths() {
		t.Fatal("app.paths must equal config.paths")
	}

	// a railtie with two chained initializers
	rt := NewRailtie("Widget::Railtie")
	rt.Initializer("widget.one", InitOpts{}, nil)
	rt.Initializer("widget.two", InitOpts{}, nil)
	app.AddRailtie(rt)
	if len(app.Railties()) != 1 {
		t.Fatal("railtie not registered")
	}

	// an app-own initializer, one with a Go block
	var blockRan bool
	app.Initializer("app.custom", InitOpts{}, func(ctx any, args ...any) error {
		if ctx != app {
			t.Error("app initializer should be bound to the app")
		}
		blockRan = true
		return nil
	})

	var order []string
	app.RunInitializer = func(name string, ctx any) error {
		order = append(order, name)
		return nil
	}

	var beforeFired, afterFired bool
	app.OnLoad("before_initialize", func(base any) error { beforeFired = true; return nil })
	app.OnLoad("after_initialize", func(base any) error { afterFired = true; return nil })

	if err := app.Initialize(); err != nil {
		t.Fatal(err)
	}
	if !app.Initialized() {
		t.Fatal("app should be initialized")
	}
	if !blockRan {
		t.Fatal("app initializer block should have run")
	}
	if !beforeFired || !afterFired {
		t.Fatal("load hooks should have fired")
	}

	// bootstrap (all group) came first, finisher last, railtie+app in between,
	// in registration order.
	need := []string{
		"load_environment_hook", // first bootstrap
		"set_secrets_root",      // last bootstrap
		"widget.one", "widget.two",
		"add_generator_templates",  // first finisher
		"set_routes_reloader_hook", // last finisher
	}
	for i := 1; i < len(need); i++ {
		if indexOf(order, need[i-1]) >= indexOf(order, need[i]) {
			t.Fatalf("boot order violated: %q not before %q in %v", need[i-1], need[i], order)
		}
	}
	// the block-carrying app initializer is NOT dispatched through the seam
	if indexOf(order, "app.custom") != -1 {
		t.Fatal("block initializer should not go through the seam")
	}

	// second Initialize is a no-op error
	if err := app.Initialize(); !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("re-init should error, got %v", err)
	}
}

func TestApplicationInitializeGroupArg(t *testing.T) {
	ResetGlobals()
	app := NewApplication("A", "")
	var ran []string
	app.RunInitializer = func(name string, ctx any) error { ran = append(ran, name); return nil }
	// custom group: only bootstrap ("all") initializers run, finishers ("default") do not
	if err := app.Initialize("assets"); err != nil {
		t.Fatal(err)
	}
	if indexOf(ran, "load_environment_hook") == -1 {
		t.Fatal("all-group bootstrap should run in any group")
	}
	if indexOf(ran, "add_generator_templates") != -1 {
		t.Fatal("default-group finisher should not run in assets group")
	}
}

func TestApplicationSeamError(t *testing.T) {
	ResetGlobals()
	app := NewApplication("A", "")
	boom := errors.New("seam boom")
	app.RunInitializer = func(name string, ctx any) error { return boom }
	if err := app.Initialize(); !errors.Is(err, boom) {
		t.Fatalf("seam error should propagate, got %v", err)
	}
	if app.Initialized() {
		t.Fatal("failed boot should not mark initialized")
	}
}

func TestApplicationHookErrors(t *testing.T) {
	boom := errors.New("hook boom")

	// before_initialize error stops the boot
	ResetGlobals()
	app := NewApplication("A", "")
	app.OnLoad("before_initialize", func(base any) error { return boom })
	if err := app.Initialize(); !errors.Is(err, boom) {
		t.Fatalf("before_initialize error should stop boot, got %v", err)
	}

	// after_initialize error surfaces after a successful init pass
	ResetGlobals()
	app2 := NewApplication("A", "")
	app2.OnLoad("after_initialize", func(base any) error { return boom })
	if err := app2.Initialize(); !errors.Is(err, boom) {
		t.Fatalf("after_initialize error should surface, got %v", err)
	}
}

func TestApplicationSeamsAndAccessors(t *testing.T) {
	ResetGlobals()
	app := NewApplication("A", "")
	if app.Secrets() != nil || app.Credentials() != nil {
		t.Fatal("secrets/credentials should start nil")
	}
	app.SetSecrets("s").SetCredentials("c")
	if app.Secrets() != "s" || app.Credentials() != "c" {
		t.Fatal("secrets/credentials seam setters failed")
	}
	if app.Hooks() == nil {
		t.Fatal("hooks registry should be non-nil")
	}
	if err := app.RunLoadHooks("evt", app); err != nil {
		t.Fatal(err)
	}
	// Env delegates to the global
	SetEnv("production")
	if !app.Env().Is("production") {
		t.Fatal("app.Env should delegate to Rails.env")
	}
	// Config shadows to ApplicationConfiguration
	if app.Config().TimeZone != "UTC" {
		t.Fatal("app.Config should be the ApplicationConfiguration")
	}
}
