package cache

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/maypok86/otter/v2"
)

const (
	// foundLifetime is how long an answer that something exists is kept.
	// Images, tags and namespaces are rarely deleted, so this can be long.
	foundLifetime = time.Hour
	// notFoundLifetime is how long an answer that something does not exist is
	// kept. It is short because the usual fix for such a diagnostic is to
	// publish the missing thing, which should show without a restart.
	notFoundLifetime = 5 * time.Minute
)

// existenceLifetime is the lifetime of an answer to "does this exist?".
func existenceLifetime(exists bool) time.Duration {
	if exists {
		return foundLifetime
	}
	return notFoundLifetime
}

// memo remembers the answer fetched for each key for a while, and has
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
type memo[V any] struct {
	// entries is replaced by clear rather than emptied. otter leaves undefined
	// what InvalidateAll does to a load in flight; a load on a replaced cache
	// stores into one nothing reads any more.
	entries atomic.Pointer[otter.Cache[string, V]]
	options otter.Options[string, V]
}

// newMemo returns a memo keeping each value for lifetime(value). A nil clock
// means the wall clock.
func newMemo[V any](lifetime func(V) time.Duration, clock otter.Clock) *memo[V] {
	m := &memo[V]{
		options: otter.Options[string, V]{
			ExpiryCalculator: otter.ExpiryWritingFunc(func(entry otter.Entry[string, V]) time.Duration {
				return lifetime(entry.Value)
			}),
			Clock: clock,
		},
	}
	m.clear()

	return m
}

// get returns the value remembered for key, calling fetch for it when there
// is none.
func (m *memo[V]) get(key string, fetch func() (V, error)) (V, error) {
	return m.entries.Load().Get(context.Background(), key, otter.LoaderFunc[string, V](
		func(context.Context, string) (V, error) {
			return fetch()
		},
	))
}

// peek returns the value remembered for key without fetching it.
func (m *memo[V]) peek(key string) (V, bool) {
	return m.entries.Load().GetIfPresent(key)
}

// put remembers a value for key.
func (m *memo[V]) put(key string, value V) {
	m.entries.Load().Set(key, value)
}

// clear forgets every value, including any a fetch still in flight would
// otherwise store.
func (m *memo[V]) clear() {
	m.entries.Store(otter.Must(&m.options))
}
