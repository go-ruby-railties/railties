// Copyright (c) the go-ruby-railties/railties authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rails

// StringInquirer wraps a string so it can be queried by value, mirroring
// ActiveSupport::StringInquirer — the type Rails.env returns so that
// `Rails.env.production?` reads naturally. In Go the dynamic `foo?` predicate
// becomes Is("foo").
type StringInquirer string

// String returns the underlying value.
func (s StringInquirer) String() string { return string(s) }

// Is reports whether the value equals name, mirroring the dynamic `name?`
// predicate (env.development? ⇒ env.Is("development")).
func (s StringInquirer) Is(name string) bool { return string(s) == name }

// Any reports whether the value equals any of the given names, mirroring
// StringInquirer#any?-style checks (env.local? style helpers build on this).
func (s StringInquirer) Any(names ...string) bool {
	for _, n := range names {
		if string(s) == n {
			return true
		}
	}
	return false
}
