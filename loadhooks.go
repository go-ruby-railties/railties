// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import "sync"

// LoadHook is a deferred callback registered against a named load event, mirroring
// the block passed to ActiveSupport.on_load. It receives the base object the event
// fired with.
type LoadHook func(base any) error

// LazyLoadHooks is a faithful port of ActiveSupport::LazyLoadHooks: callbacks are
// registered against a symbolic event name with OnLoad and later fired, in
// registration order, by RunLoadHooks. If an event has already fired, a hook
// registered afterwards runs immediately against the remembered base(s) — the
// lazy-load contract Rails relies on so that `ActiveSupport.on_load(:action_controller)`
// works whether it is written before or after the framework loads.
type LazyLoadHooks struct {
	mu       sync.Mutex
	hooks    map[string][]LoadHook
	fired    map[string][]any // bases each event has already fired with
	firstErr error
}

// NewLazyLoadHooks returns an empty hook registry.
func NewLazyLoadHooks() *LazyLoadHooks {
	return &LazyLoadHooks{hooks: map[string][]LoadHook{}, fired: map[string][]any{}}
}

// OnLoad registers hook against name. If name has already fired, hook runs
// immediately against every base the event fired with (in fire order), mirroring
// on_load's replay-against-past-executions behaviour. Any error from an immediate
// run is returned.
func (h *LazyLoadHooks) OnLoad(name string, hook LoadHook) error {
	h.mu.Lock()
	h.hooks[name] = append(h.hooks[name], hook)
	bases := append([]any(nil), h.fired[name]...)
	h.mu.Unlock()
	for _, base := range bases {
		if err := hook(base); err != nil {
			return err
		}
	}
	return nil
}

// RunLoadHooks fires name against base: it remembers base (so later OnLoad calls
// replay against it) and runs every hook registered so far, in registration
// order, mirroring run_load_hooks(name, base). The first hook error stops the run
// and is returned.
func (h *LazyLoadHooks) RunLoadHooks(name string, base any) error {
	h.mu.Lock()
	h.fired[name] = append(h.fired[name], base)
	hooks := append([]LoadHook(nil), h.hooks[name]...)
	h.mu.Unlock()
	for _, hook := range hooks {
		if err := hook(base); err != nil {
			return err
		}
	}
	return nil
}
