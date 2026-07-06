// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import "path"

// PathOpts are the options accepted by PathsRoot.Add, mirroring the keyword
// arguments of Rails::Paths::Root#add (with:, eager_load:, autoload:,
// autoload_once:, load_path:, glob:).
type PathOpts struct {
	// With overrides the relative sub-paths this entry maps to. When empty the
	// entry maps to its own key, mirroring `add "app", with: [...]` defaulting to
	// [key].
	With []string
	// Glob is the glob appended when expanding (informational in v0.1; globbing
	// itself is deferred).
	Glob string

	EagerLoad    bool
	Autoload     bool
	AutoloadOnce bool
	LoadPath     bool
}

// Path is one entry in a PathsRoot — a labelled set of relative sub-paths plus the
// autoload / eager-load / load-path flags that classify it, mirroring
// Rails::Paths::Path.
type Path struct {
	root    *PathsRoot
	current string   // the key this path was added under
	paths   []string // the relative sub-paths (defaults to [current])
	glob    string

	eagerLoad    bool
	autoload     bool
	autoloadOnce bool
	loadPath     bool
}

// EagerLoad flags this path for eager loading and returns it for chaining,
// mirroring Path#eager_load!.
func (p *Path) EagerLoad() *Path { p.eagerLoad = true; return p }

// Autoload flags this path for autoloading and returns it, mirroring Path#autoload!.
func (p *Path) Autoload() *Path { p.autoload = true; return p }

// AutoloadOnce flags this path for autoload-once and returns it, mirroring
// Path#autoload_once!.
func (p *Path) AutoloadOnce() *Path { p.autoloadOnce = true; return p }

// LoadPath flags this path for the $LOAD_PATH and returns it, mirroring
// Path#load_path!.
func (p *Path) LoadPath() *Path { p.loadPath = true; return p }

// EagerLoadQ reports whether this path is flagged for eager loading (Path#eager_load?).
func (p *Path) EagerLoadQ() bool { return p.eagerLoad }

// AutoloadQ reports whether this path is flagged for autoloading (Path#autoload?).
func (p *Path) AutoloadQ() bool { return p.autoload }

// AutoloadOnceQ reports whether this path is flagged autoload-once (Path#autoload_once?).
func (p *Path) AutoloadOnceQ() bool { return p.autoloadOnce }

// LoadPathQ reports whether this path is flagged for the load path (Path#load_path?).
func (p *Path) LoadPathQ() bool { return p.loadPath }

// Push appends relative sub-paths, mirroring Path#<< / Path#push.
func (p *Path) Push(paths ...string) *Path { p.paths = append(p.paths, paths...); return p }

// Unshift prepends relative sub-paths, mirroring Path#unshift.
func (p *Path) Unshift(paths ...string) *Path {
	p.paths = append(append([]string(nil), paths...), p.paths...)
	return p
}

// To returns the raw relative sub-paths in order, mirroring Path#to_a.
func (p *Path) To() []string { return append([]string(nil), p.paths...) }

// Expanded returns each sub-path resolved against the root, mirroring
// Path#expanded. Absolute sub-paths are returned unchanged; relative ones are
// joined onto the root (existence checking and globbing are deferred — see the
// roadmap).
func (p *Path) Expanded() []string {
	out := make([]string, 0, len(p.paths))
	for _, sp := range p.paths {
		if path.IsAbs(sp) {
			out = append(out, sp)
		} else {
			out = append(out, path.Join(p.root.root, sp))
		}
	}
	return out
}

// PathsRoot is the paths DSL rooted at a directory, mirroring Rails::Paths::Root:
// it maps labels ("app/models", "config", …) to Path entries and aggregates the
// expanded paths by classification (autoload, eager-load, load-path).
type PathsRoot struct {
	root  string
	paths map[string]*Path
	order []string // insertion order of keys, so aggregation is deterministic
}

// NewPathsRoot returns a PathsRoot anchored at root, mirroring Root.new(path).
func NewPathsRoot(root string) *PathsRoot {
	return &PathsRoot{root: root, paths: map[string]*Path{}}
}

// Root returns the anchor directory (Root#path).
func (r *PathsRoot) Root() string { return r.root }

// SetRoot changes the anchor directory (Root#path=).
func (r *PathsRoot) SetRoot(p string) { r.root = p }

// Add registers path under its own key with the given options, mirroring
// Root#add. Re-adding a key replaces it. It returns the created Path for chaining.
func (r *PathsRoot) Add(path string, opts PathOpts) *Path {
	subs := opts.With
	if len(subs) == 0 {
		subs = []string{path}
	}
	p := &Path{
		root:         r,
		current:      path,
		paths:        append([]string(nil), subs...),
		glob:         opts.Glob,
		eagerLoad:    opts.EagerLoad,
		autoload:     opts.Autoload,
		autoloadOnce: opts.AutoloadOnce,
		loadPath:     opts.LoadPath,
	}
	if _, exists := r.paths[path]; !exists {
		r.order = append(r.order, path)
	}
	r.paths[path] = p
	return p
}

// Get returns the Path registered under key, or nil, mirroring Root#[].
func (r *PathsRoot) Get(key string) *Path { return r.paths[key] }

// Keys returns the registered keys in insertion order (Root#keys).
func (r *PathsRoot) Keys() []string { return append([]string(nil), r.order...) }

// Values returns the registered Paths in insertion order (Root#values).
func (r *PathsRoot) Values() []*Path {
	out := make([]*Path, 0, len(r.order))
	for _, k := range r.order {
		out = append(out, r.paths[k])
	}
	return out
}

// filterExpanded gathers the expanded sub-paths of every entry for which keep
// returns true, in insertion order.
func (r *PathsRoot) filterExpanded(keep func(*Path) bool) []string {
	var out []string
	for _, k := range r.order {
		p := r.paths[k]
		if keep(p) {
			out = append(out, p.Expanded()...)
		}
	}
	return out
}

// AutoloadPaths returns the expanded paths of every autoload entry (Root#autoload_paths).
func (r *PathsRoot) AutoloadPaths() []string {
	return r.filterExpanded(func(p *Path) bool { return p.autoload })
}

// AutoloadOnce returns the expanded paths of every autoload-once entry
// (Root#autoload_once).
func (r *PathsRoot) AutoloadOnce() []string {
	return r.filterExpanded(func(p *Path) bool { return p.autoloadOnce })
}

// EagerLoad returns the expanded paths of every eager-load entry (Root#eager_load).
func (r *PathsRoot) EagerLoad() []string {
	return r.filterExpanded(func(p *Path) bool { return p.eagerLoad })
}

// LoadPaths returns the expanded paths of every load-path entry (Root#load_paths).
func (r *PathsRoot) LoadPaths() []string {
	return r.filterExpanded(func(p *Path) bool { return p.loadPath })
}
