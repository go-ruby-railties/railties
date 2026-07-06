// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"os"
	"sync"
)

// The Rails.* process globals. In MRI these are singleton methods on the Rails
// module; here they are guarded package state with getters/setters mirroring
// Rails.application / Rails.root / Rails.env / Rails.logger.
var (
	globalMu       sync.RWMutex
	currentApp     *Application
	currentRoot    string
	currentEnv     StringInquirer
	envInitialized bool
	currentLogger  any
)

// App returns the current Rails application, mirroring Rails.application (nil
// until an Application is created or SetApp is called).
func App() *Application {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return currentApp
}

// SetApp sets the current Rails application (Rails.application=).
func SetApp(a *Application) {
	globalMu.Lock()
	currentApp = a
	globalMu.Unlock()
}

// Root returns the application root path, mirroring Rails.root.
func Root() string {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return currentRoot
}

// SetRoot sets the application root path (Rails.root=).
func SetRoot(p string) {
	globalMu.Lock()
	currentRoot = p
	globalMu.Unlock()
}

// Env returns the current environment as a StringInquirer, mirroring Rails.env.
// On first use it is derived from RAILS_ENV, then RACK_ENV, defaulting to
// "development"; the result is memoised until SetEnv or ResetGlobals.
func Env() StringInquirer {
	globalMu.Lock()
	defer globalMu.Unlock()
	if !envInitialized {
		env := os.Getenv("RAILS_ENV")
		if env == "" {
			env = os.Getenv("RACK_ENV")
		}
		if env == "" {
			env = "development"
		}
		currentEnv = StringInquirer(env)
		envInitialized = true
	}
	return currentEnv
}

// SetEnv sets the current environment (Rails.env=).
func SetEnv(env string) {
	globalMu.Lock()
	currentEnv = StringInquirer(env)
	envInitialized = true
	globalMu.Unlock()
}

// Logger returns the current logger seam, mirroring Rails.logger (nil until set).
func Logger() any {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return currentLogger
}

// SetLogger sets the current logger seam (Rails.logger=).
func SetLogger(l any) {
	globalMu.Lock()
	currentLogger = l
	globalMu.Unlock()
}

// ResetGlobals clears all Rails.* process globals. It exists so tests (and hosts
// that re-boot) start from a clean slate; it has no MRI counterpart.
func ResetGlobals() {
	globalMu.Lock()
	currentApp = nil
	currentRoot = ""
	currentEnv = ""
	envInitialized = false
	currentLogger = nil
	globalMu.Unlock()
}
