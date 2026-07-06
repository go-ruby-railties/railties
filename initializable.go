// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"fmt"
	"strings"
)

// Runner is the body of an initializer — the seam that replaces the Ruby block
// passed to `initializer "name" do … end`. It receives the bound context (the
// railtie/engine/application the initializer belongs to) and the extra arguments
// threaded through run_initializers.
type Runner func(ctx any, args ...any) error

// InitializerSeam is the host-supplied dispatcher for initializers that carry no
// Go Block — the mechanism by which a host (e.g. go-embedded-ruby) provides the
// real, deferred initializer bodies. It mirrors Rails' `initializer.run`: given
// the initializer's name and its bound context, do the work.
//
// This is the RunInitializer seam referenced throughout railties.
type InitializerSeam func(name string, ctx any) error

// Initializer is a single registered initializer, mirroring
// ActiveSupport::Initializable::Initializer. Ordering is decided by Name together
// with the Before/After constraints; Group selects which run_initializers pass it
// belongs to; Block (or the collection seam, when Block is nil) is its body.
type Initializer struct {
	Name   string
	Before string
	After  string
	Group  string // "" is normalised to "default"
	Block  Runner

	ctx any // the bound context (railtie/engine/app); set by Bind
}

// group returns the effective group, normalising the empty string to "default"
// exactly as Rails' `options[:group] ||= :default`.
func (i *Initializer) group() string {
	if i.Group == "" {
		return "default"
	}
	return i.Group
}

// BelongsTo reports whether this initializer runs in the given group, mirroring
// Initializable::Initializer#belongs_to?: it runs in its own group or when its
// group is "all".
func (i *Initializer) BelongsTo(group string) bool {
	g := i.group()
	return g == group || g == "all"
}

// Bind returns a copy of the initializer bound to ctx, mirroring
// Initializer#bind — the collection binds every initializer to the object whose
// run_initializers is executing before running it.
func (i *Initializer) Bind(ctx any) *Initializer {
	c := *i
	c.ctx = ctx
	return &c
}

// Context returns the bound context (nil until Bind is called).
func (i *Initializer) Context() any { return i.ctx }

// Run executes the initializer body: its Block if set, otherwise the supplied
// seam with (Name, boundContext). It mirrors Initializer#run. If the initializer
// has neither a Block nor a seam, Run is a no-op (the body is deferred).
func (i *Initializer) Run(seam InitializerSeam, args ...any) error {
	if i.Block != nil {
		return i.Block(i.ctx, args...)
	}
	if seam != nil {
		return seam(i.Name, i.ctx)
	}
	return nil
}

// Collection is an ordered list of initializers, mirroring
// ActiveSupport::Initializable::Collection. Its defining behaviour is the
// topological sort (TSort) that orders initializers by their Before/After
// constraints while preserving registration order as the tie-break.
type Collection []*Initializer

// CyclicError reports that the Before/After constraints form a cycle, mirroring
// TSort::Cyclic. It names the initializers in the offending strongly-connected
// component.
type CyclicError struct {
	Component []*Initializer
}

func (e *CyclicError) Error() string {
	names := make([]string, len(e.Component))
	for i, in := range e.Component {
		names[i] = in.Name
	}
	return "rails: initializer cycle detected: " + strings.Join(names, " -> ")
}

// children returns the initializers that must run *before* n — the direct
// predecessors — exactly as Collection#tsort_each_child does in Rails:
//
//	select { |i| i.before == n.name || i.name == n.after }
//
// i.e. every initializer that declares `before: n.name`, plus the single
// initializer named by n.after. Registration order is preserved.
func (c Collection) children(n *Initializer) []*Initializer {
	var out []*Initializer
	for _, i := range c {
		if i.Before == n.Name || i.Name == n.After {
			out = append(out, i)
		}
	}
	return out
}

// TSort returns the initializers in dependency order (each initializer after all
// of its children), reproducing ActiveSupport::Initializable::Collection#tsort_each.
//
// It is a faithful port of Ruby's TSort: nodes are visited in registration order
// and Tarjan's strongly-connected-component algorithm yields components in
// post-order. A component of a single node is emitted (DAG case); a component of
// two or more nodes is a genuine cycle and returns a *CyclicError. Because it is
// SCC-based rather than back-edge-based, a lone self-referential node (before ==
// its own name) is *not* treated as a cycle — matching Ruby's TSort#tsort.
func (c Collection) TSort() (Collection, error) {
	ids := make(map[*Initializer]int)    // assigned index during DFS
	inSCC := make(map[*Initializer]bool) // already emitted in a finished component
	var stack []*Initializer
	var out Collection
	var cyclic *CyclicError

	var visit func(n *Initializer) int
	visit = func(n *Initializer) int {
		id := len(ids)
		ids[n] = id
		minID := id
		stackLen := len(stack)
		stack = append(stack, n)

		for _, child := range c.children(n) {
			if cid, seen := ids[child]; seen {
				// Already visited: only tighten minID if it is still on the
				// live stack (not yet placed in a finished component).
				if !inSCC[child] && cid < minID {
					minID = cid
				}
			} else {
				if sub := visit(child); sub < minID {
					minID = sub
				}
			}
		}

		if id == minID {
			component := append(Collection(nil), stack[stackLen:]...)
			stack = stack[:stackLen]
			for _, m := range component {
				inSCC[m] = true
			}
			if len(component) == 1 {
				out = append(out, component[0])
			} else if cyclic == nil {
				cyclic = &CyclicError{Component: component}
			}
		}
		return minID
	}

	for _, n := range c {
		if _, seen := ids[n]; !seen {
			visit(n)
		}
	}
	if cyclic != nil {
		return nil, cyclic
	}
	return out, nil
}

// RunInitializers topologically sorts the collection and runs, in order, every
// initializer that belongs to the given group, mirroring Initializable#run_initializers.
// Initializers with a Block run it; those without dispatch to seam. args are
// threaded to each body. A cycle (see TSort) is returned before anything runs.
func (c Collection) RunInitializers(group string, seam InitializerSeam, args ...any) error {
	ordered, err := c.TSort()
	if err != nil {
		return err
	}
	for _, in := range ordered {
		if in.BelongsTo(group) {
			if err := in.Run(seam, args...); err != nil {
				return fmt.Errorf("rails: initializer %q failed: %w", in.Name, err)
			}
		}
	}
	return nil
}
