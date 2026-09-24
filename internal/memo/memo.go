// Package memo is how the server remembers what it has looked up.
//
// Every lookup worth remembering goes through a Memo, so that each follows the
// same rules: concurrent callers share one fetch, an answer is kept for a
// lifetime chosen by its value, and an error is never remembered.
package memo

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/maypok86/otter/v2"
)

const (
	// FoundLifetime is how long an answer that something exists is kept.
	// Images, tags, orbs and namespaces are rarely deleted, so this can be
	// long.
	FoundLifetime = time.Hour
	// NotFoundLifetime is how long an answer that something does not exist is
	// kept. It is short because the usual fix for such a diagnostic is to
	// publish the missing thing, which should show without a restart.
	NotFoundLifetime = 5 * time.Minute
)

// Existence is the lifetime of an answer to "does this exist?".
func Existence(exists bool) time.Duration {
	if exists {
		return FoundLifetime
	}
	return NotFoundLifetime
}

// Fixed is a lifetime of d for every value.
func Fixed[V any](d time.Duration) func(V) time.Duration {
	return func(V) time.Duration { return d }
}

// Memo remembers the answer fetched for each key for a while, and has
// concurrent callers asking for the same key share one fetch rather than each
// making their own.
//
// A single open file is validated several times over as the client starts up
// — on open, on each configuration change, for code actions — each on a
// goroutine of its own. Checking a map and then fetching left every one of
// them to find nothing and ask the same question at once.
//
// An error is handed to every caller sharing the fetch but is not remembered,
// so the next call asks again: a request that failed is not an answer.
type Memo[V any] struct {
	// entries is replaced by Clear rather than emptied. otter leaves undefined
	// what InvalidateAll does to a load in flight; a load on a replaced cache
	// stores into one nothing reads any more.
	entries atomic.Pointer[otter.Cache[string, V]]
	options otter.Options[string, V]
}

// New returns a Memo keeping each value for lifetime(value). A nil clock
// means the wall clock.
func New[V any](lifetime func(V) time.Duration, clock otter.Clock) *Memo[V] {
	m := &Memo[V]{
		options: otter.Options[string, V]{
			ExpiryCalculator: otter.ExpiryWritingFunc(func(entry otter.Entry[string, V]) time.Duration {
				return lifetime(entry.Value)
			}),
			Clock: clock,
		},
	}
	m.Clear()

	return m
}

// Get returns the value remembered for key, calling fetch for it when there
// is none.
func (m *Memo[V]) Get(key string, fetch func() (V, error)) (V, error) {
	return m.entries.Load().Get(context.Background(), key, otter.LoaderFunc[string, V](
		func(context.Context, string) (V, error) {
			return fetch()
		},
	))
}

// Peek returns the value remembered for key without fetching it.
func (m *Memo[V]) Peek(key string) (V, bool) {
	return m.entries.Load().GetIfPresent(key)
}

// Put remembers a value for key.
func (m *Memo[V]) Put(key string, value V) {
	m.entries.Load().Set(key, value)
}

// Clear forgets every value, including any a fetch still in flight would
// otherwise store.
func (m *Memo[V]) Clear() {
	m.entries.Store(otter.Must(&m.options))
}
