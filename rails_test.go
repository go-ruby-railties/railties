// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"errors"
	"testing"
)

func TestStringInquirer(t *testing.T) {
	s := StringInquirer("production")
	if s.String() != "production" {
		t.Fatal("String wrong")
	}
	if !s.Is("production") || s.Is("development") {
		t.Fatal("Is wrong")
	}
	if !s.Any("test", "production") || s.Any("dev", "stage") {
		t.Fatal("Any wrong")
	}
}

func TestGlobalsAppRootLogger(t *testing.T) {
	ResetGlobals()
	if App() != nil || Root() != "" || Logger() != nil {
		t.Fatal("globals should start empty after reset")
	}
	SetRoot("/x")
	if Root() != "/x" {
		t.Fatal("SetRoot failed")
	}
	SetLogger("logger")
	if Logger() != "logger" {
		t.Fatal("SetLogger failed")
	}
	a := &Application{}
	SetApp(a)
	if App() != a {
		t.Fatal("SetApp failed")
	}
}

func TestEnvExplicitSet(t *testing.T) {
	ResetGlobals()
	SetEnv("staging")
	if !Env().Is("staging") {
		t.Fatal("SetEnv should win")
	}
	// memoised: a second read returns the same without re-deriving
	if !Env().Is("staging") {
		t.Fatal("memoised env wrong")
	}
}

func TestEnvFromRailsEnv(t *testing.T) {
	ResetGlobals()
	t.Setenv("RAILS_ENV", "production")
	t.Setenv("RACK_ENV", "ignored")
	if !Env().Is("production") {
		t.Fatalf("RAILS_ENV should win, got %q", Env())
	}
}

func TestEnvFromRackEnv(t *testing.T) {
	ResetGlobals()
	t.Setenv("RAILS_ENV", "")
	t.Setenv("RACK_ENV", "test")
	if !Env().Is("test") {
		t.Fatalf("RACK_ENV should be used, got %q", Env())
	}
}

func TestEnvDefaultDevelopment(t *testing.T) {
	ResetGlobals()
	t.Setenv("RAILS_ENV", "")
	t.Setenv("RACK_ENV", "")
	if !Env().Is("development") {
		t.Fatalf("default env should be development, got %q", Env())
	}
}

func TestLazyLoadHooksReplayAndOrder(t *testing.T) {
	h := NewLazyLoadHooks()
	var log []string
	// register before firing
	h.OnLoad("evt", func(base any) error { log = append(log, "first:"+base.(string)); return nil })
	if err := h.RunLoadHooks("evt", "A"); err != nil {
		t.Fatal(err)
	}
	// register after firing -> replays immediately against remembered base
	h.OnLoad("evt", func(base any) error { log = append(log, "late:"+base.(string)); return nil })
	want := []string{"first:A", "late:A"}
	for i := range want {
		if log[i] != want[i] {
			t.Fatalf("hook log wrong: %v", log)
		}
	}
}

func TestLazyLoadHooksErrors(t *testing.T) {
	boom := errors.New("boom")
	// error during RunLoadHooks
	h := NewLazyLoadHooks()
	h.OnLoad("evt", func(base any) error { return boom })
	if err := h.RunLoadHooks("evt", nil); !errors.Is(err, boom) {
		t.Fatalf("run error should surface, got %v", err)
	}
	// error during immediate replay in OnLoad
	h2 := NewLazyLoadHooks()
	if err := h2.RunLoadHooks("evt", "base"); err != nil {
		t.Fatal(err)
	}
	err := h2.OnLoad("evt", func(base any) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("on_load immediate error should surface, got %v", err)
	}
}
