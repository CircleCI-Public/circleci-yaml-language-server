package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// keepForever is a lifetime long enough that nothing expires during a test.
func keepForever(string) time.Duration { return 24 * time.Hour }

// fakeClock is a clock that moves only when a test advances it.
type fakeClock struct {
	now atomic.Int64
}

func (c *fakeClock) NowNano() int64 { return c.now.Load() }

// Tick never ticks: expiry is checked on read, which is all a test needs.
func (c *fakeClock) Tick(time.Duration) <-chan time.Time { return nil }

func (c *fakeClock) advance(d time.Duration) { c.now.Add(int64(d)) }

func TestMemo(t *testing.T) {
	t.Run("concurrent callers share one fetch", func(t *testing.T) {
		m := newMemo(keepForever, nil)
		var fetches atomic.Int32
		release := make(chan struct{})

		const callers = 10
		results := make([]string, callers)
		var started, done sync.WaitGroup

		t.Run("start the callers", func(t *testing.T) {
			for i := range callers {
				started.Add(1)
				done.Go(func() {
					started.Done()
					results[i], _ = m.get("key", func() (string, error) {
						fetches.Add(1)
						<-release
						return "value", nil
					})
				})
			}
			started.Wait()
		})

		t.Run("let the fetch finish", func(t *testing.T) {
			close(release)
			done.Wait()
		})

		t.Run("check one fetch served them all", func(t *testing.T) {
			assert.Check(t, cmp.Equal(fetches.Load(), int32(1)))
			for i, result := range results {
				assert.Check(t, cmp.Equal(result, "value"), "caller %d", i)
			}
		})
	})

	t.Run("a remembered value is not fetched again", func(t *testing.T) {
		m := newMemo(keepForever, nil)
		_, err := m.get("key", func() (string, error) { return "value", nil })
		assert.NilError(t, err)

		got, err := m.get("key", func() (string, error) {
			t.Error("fetched a value that was remembered")
			return "", nil
		})
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(got, "value"))
	})

	t.Run("an error is returned but not remembered", func(t *testing.T) {
		m := newMemo(keepForever, nil)
		errFetch := errors.New("unavailable")

		_, err := m.get("key", func() (string, error) { return "", errFetch })
		assert.Check(t, cmp.ErrorIs(err, errFetch))

		_, known := m.peek("key")
		assert.Check(t, !known, "a failed fetch must not be remembered")

		got, err := m.get("key", func() (string, error) { return "value", nil })
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(got, "value"))
	})

	t.Run("a fetch in flight across clear is not remembered", func(t *testing.T) {
		m := newMemo(keepForever, nil)
		fetching := make(chan struct{})
		release := make(chan struct{})
		var done sync.WaitGroup

		t.Run("start a fetch", func(t *testing.T) {
			done.Go(func() {
				_, _ = m.get("key", func() (string, error) {
					close(fetching)
					<-release
					return "stale", nil
				})
			})
			<-fetching
		})

		t.Run("clear, then let the fetch finish", func(t *testing.T) {
			m.clear()
			close(release)
			done.Wait()
		})

		t.Run("check the value was dropped", func(t *testing.T) {
			_, known := m.peek("key")
			assert.Check(t, !known, "a value read before clear must not survive it")
		})
	})
	t.Run("an answer is kept for as long as its lifetime", func(t *testing.T) {
		clock := &fakeClock{}
		m := newMemo(existenceLifetime, clock)

		t.Run("remember that one thing exists and one does not", func(t *testing.T) {
			m.put("found", true)
			m.put("missing", false)
		})

		t.Run("check a missing answer expires first", func(t *testing.T) {
			clock.advance(notFoundLifetime + time.Second)

			_, known := m.peek("missing")
			assert.Check(t, !known, "a not-found answer must expire after notFoundLifetime")
			_, known = m.peek("found")
			assert.Check(t, known, "a found answer must outlive notFoundLifetime")
		})

		t.Run("check a found answer expires after its own lifetime", func(t *testing.T) {
			clock.advance(foundLifetime)

			_, known := m.peek("found")
			assert.Check(t, !known, "a found answer must expire after foundLifetime")
		})

		t.Run("check an expired answer is fetched again", func(t *testing.T) {
			got, err := m.get("missing", func() (bool, error) { return true, nil })
			assert.NilError(t, err)
			assert.Check(t, got, "the answer fetched after expiry")
		})
	})
}
