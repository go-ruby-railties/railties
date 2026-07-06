// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"errors"
	"reflect"
	"testing"
)

// names extracts the ordered initializer names from a collection.
func names(c Collection) []string {
	out := make([]string, len(c))
	for i, in := range c {
		out[i] = in.Name
	}
	return out
}

func TestGroupNormalisationAndBelongsTo(t *testing.T) {
	def := &Initializer{Name: "a"}
	if def.group() != "default" {
		t.Fatalf("empty group should normalise to default, got %q", def.group())
	}
	if !def.BelongsTo("default") {
		t.Fatal("default initializer should belong to default group")
	}
	if def.BelongsTo("other") {
		t.Fatal("default initializer should not belong to other group")
	}
	all := &Initializer{Name: "b", Group: "all"}
	if !all.BelongsTo("default") || !all.BelongsTo("anything") {
		t.Fatal("all-group initializer should belong to every group")
	}
	named := &Initializer{Name: "c", Group: "assets"}
	if !named.BelongsTo("assets") || named.BelongsTo("default") {
		t.Fatal("named group membership wrong")
	}
}

func TestBindAndContext(t *testing.T) {
	orig := &Initializer{Name: "a"}
	if orig.Context() != nil {
		t.Fatal("unbound initializer should have nil context")
	}
	ctx := "ctx"
	bound := orig.Bind(ctx)
	if bound.Context() != ctx {
		t.Fatal("bound context not set")
	}
	if orig.Context() != nil {
		t.Fatal("Bind must not mutate the original")
	}
	if bound == orig {
		t.Fatal("Bind must return a copy")
	}
}

func TestRunBlockSeamAndNoop(t *testing.T) {
	// Block path.
	var got any
	blk := &Initializer{Name: "b", Block: func(ctx any, args ...any) error {
		got = args
		return nil
	}}
	blk = blk.Bind("c")
	if err := blk.Run(nil, 1, 2); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []any{1, 2}) {
		t.Fatalf("block args not threaded: %v", got)
	}
	// Block error.
	boom := errors.New("boom")
	errBlk := &Initializer{Name: "e", Block: func(ctx any, args ...any) error { return boom }}
	if err := errBlk.Run(nil); !errors.Is(err, boom) {
		t.Fatalf("want block error, got %v", err)
	}
	// Seam path (no block).
	var seamName string
	var seamCtx any
	seam := func(name string, ctx any) error { seamName, seamCtx = name, ctx; return nil }
	(&Initializer{Name: "s"}).Bind("ctx").Run(seam)
	if seamName != "s" || seamCtx != "ctx" {
		t.Fatalf("seam not invoked with name/ctx: %q %v", seamName, seamCtx)
	}
	// No block, no seam -> no-op.
	if err := (&Initializer{Name: "n"}).Run(nil); err != nil {
		t.Fatalf("noop run should be nil, got %v", err)
	}
}

func TestTSortRegistrationOrder(t *testing.T) {
	c := Collection{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	ord, err := c.TSort()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(ord); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("unconstrained order should be registration order, got %v", got)
	}
}

func TestTSortBeforeAfter(t *testing.T) {
	// b must run before a; c must run after a.
	c := Collection{
		{Name: "a"},
		{Name: "b", Before: "a"},
		{Name: "c", After: "a"},
	}
	ord, err := c.TSort()
	if err != nil {
		t.Fatal(err)
	}
	got := names(ord)
	pos := map[string]int{}
	for i, n := range got {
		pos[n] = i
	}
	if !(pos["b"] < pos["a"] && pos["a"] < pos["c"]) {
		t.Fatalf("before/after ordering violated: %v", got)
	}
}

func TestTSortChainDefaultsAfter(t *testing.T) {
	// The registration default-after chaining should keep sequential order.
	r := NewRailtie("X")
	r.Initializer("one", InitOpts{}, nil)
	r.Initializer("two", InitOpts{}, nil)
	r.Initializer("three", InitOpts{}, nil)
	ord, err := r.Initializers().TSort()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(ord); !reflect.DeepEqual(got, []string{"one", "two", "three"}) {
		t.Fatalf("chained order wrong: %v", got)
	}
}

func TestTSortCycle(t *testing.T) {
	// a after b, b after a -> genuine 2-node cycle.
	c := Collection{
		{Name: "a", After: "b"},
		{Name: "b", After: "a"},
	}
	_, err := c.TSort()
	var ce *CyclicError
	if !errors.As(err, &ce) {
		t.Fatalf("want CyclicError, got %v", err)
	}
	if len(ce.Component) != 2 {
		t.Fatalf("cycle component should be 2 nodes, got %d", len(ce.Component))
	}
	if ce.Error() == "" {
		t.Fatal("CyclicError should render a message")
	}
	// RunInitializers must surface the cycle before running anything.
	if err := c.RunInitializers("default", nil); !errors.As(err, &ce) {
		t.Fatalf("RunInitializers should propagate cycle, got %v", err)
	}
}

func TestTSortSelfLoopNotCyclic(t *testing.T) {
	// A lone self-referential node (before == its own name) is a size-1 SCC and,
	// like Ruby's TSort#tsort, must NOT be reported as a cycle.
	c := Collection{{Name: "a", Before: "a"}}
	ord, err := c.TSort()
	if err != nil {
		t.Fatalf("self-loop should not be cyclic, got %v", err)
	}
	if got := names(ord); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("self-loop node should be emitted once, got %v", got)
	}
}

func TestRunInitializersGroupFilterAndSeam(t *testing.T) {
	var ran []string
	seam := func(name string, ctx any) error { ran = append(ran, name); return nil }
	c := Collection{
		{Name: "boot", Group: "all"},
		{Name: "main"},                    // default
		{Name: "assets", Group: "assets"}, // filtered out of default
	}
	if err := c.RunInitializers("default", seam); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ran, []string{"boot", "main"}) {
		t.Fatalf("group filtering wrong: %v", ran)
	}
}

func TestRunInitializersBodyError(t *testing.T) {
	boom := errors.New("kaboom")
	c := Collection{{Name: "x", Block: func(ctx any, args ...any) error { return boom }}}
	err := c.RunInitializers("default", nil)
	if !errors.Is(err, boom) {
		t.Fatalf("want wrapped body error, got %v", err)
	}
}
