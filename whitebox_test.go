// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

import (
	"errors"
	"testing"
)

// TestInsertAtClamps exercises the defensive index clamps in insertAt directly:
// the public InsertBefore/InsertAfter always pass an in-range index, so these
// out-of-range cases are only reachable white-box.
func TestInsertAtClamps(t *testing.T) {
	s := NewMiddlewareStack()
	s.Use("A")
	s.insertAt(-5, Middleware{Klass: "head"})  // clamps to 0
	s.insertAt(999, Middleware{Klass: "tail"}) // clamps to len
	got := klasses(s.Entries())
	if got[0] != "head" || got[len(got)-1] != "tail" {
		t.Fatalf("insertAt clamps wrong: %v", got)
	}
}

// TestTSortSecondCycleIgnored ensures that once a cycle is recorded, a second
// independent cycle does not overwrite it — exercising the `cyclic == nil` guard's
// false branch. Two disjoint 2-node cycles.
func TestTSortSecondCycleIgnored(t *testing.T) {
	c := Collection{
		{Name: "a", After: "b"},
		{Name: "b", After: "a"},
		{Name: "c", After: "d"},
		{Name: "d", After: "c"},
	}
	_, err := c.TSort()
	var ce *CyclicError
	if !errors.As(err, &ce) {
		t.Fatalf("want CyclicError, got %v", err)
	}
	// The first cycle discovered is kept.
	if len(ce.Component) != 2 {
		t.Fatalf("component size wrong: %d", len(ce.Component))
	}
}

// TestTSortThreeNodeCycle drives a 3-node cycle (a→b→c→a) so the recursive
// low-link update (a deeper visit returning a strictly smaller id) is exercised,
// and the whole SCC is reported as a single cyclic component.
func TestTSortThreeNodeCycle(t *testing.T) {
	c := Collection{
		{Name: "a", After: "b"},
		{Name: "b", After: "c"},
		{Name: "c", After: "a"},
	}
	_, err := c.TSort()
	var ce *CyclicError
	if !errors.As(err, &ce) {
		t.Fatalf("want CyclicError, got %v", err)
	}
	if len(ce.Component) != 3 {
		t.Fatalf("3-node cycle component size wrong: %d", len(ce.Component))
	}
}

// TestTSortSharedChildAlreadyInSCC exercises the branch where a node's child has
// already been placed in a finished component (inSCC true): two initializers that
// both depend on the same already-emitted base.
func TestTSortSharedChildAlreadyInSCC(t *testing.T) {
	c := Collection{
		{Name: "base"},
		{Name: "x", After: "base"},
		{Name: "y", After: "base"},
	}
	ord, err := c.TSort()
	if err != nil {
		t.Fatal(err)
	}
	got := names(ord)
	if got[0] != "base" {
		t.Fatalf("base should come first: %v", got)
	}
}
