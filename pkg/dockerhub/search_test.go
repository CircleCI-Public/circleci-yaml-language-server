package dockerhub

import (
	"net/http"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func TestSearch(t *testing.T) {
	t.Run("finds a repository", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		cursor := api.Search("cimg/base")
		assert.Assert(t, cursor.HasNext())

		repository := cursor.Next()
		assert.Assert(t, repository != nil)
		assert.Check(t, cmp.Equal(repository.Name, "base"))
		assert.Check(t, cmp.Equal(repository.Namespace, "cimg"))
	})

	// Docker Hub reports the next page as an absolute URL, and the cursor
	// follows it, so a repository the first page did not carry is still found.
	t.Run("finds a repository on a later page", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetPageLimit(1)
		api := apiFor(fake)

		cursor := api.Search("cimg/python")
		assert.Assert(t, cursor.HasNext())

		repository := cursor.Next()
		assert.Assert(t, repository != nil)
		assert.Check(t, cmp.Equal(repository.Name, "python"))
	})

	// Asking whether anything matches reads only as far as the answer: the
	// repository is on the first of four pages, so the other three are left.
	t.Run("stops reading at the first match", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetPageLimit(1)
		api := apiFor(fake)

		assert.Assert(t, api.Search("cimg/base").HasNext())

		requestCount := fake.RequestCount(http.MethodGet, cimgReposPath)
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	// Having read it to the end, the next keystroke costs nothing.
	t.Run("reuses a namespace it has already read", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		assert.Assert(t, api.Search("cimg/python").HasNext())
		before := fake.RequestCount(http.MethodGet, cimgReposPath)

		assert.Check(t, api.Search("cimg/go").HasNext())

		after := fake.RequestCount(http.MethodGet, cimgReposPath)
		assert.Check(t, cmp.Equal(after, before))
	})

	t.Run("walks every match in order", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetPageLimit(2)
		api := apiFor(fake)

		cursor := api.Search("cimg/")

		names := []string{}
		for cursor.HasNext() {
			repository := cursor.Next()
			assert.Assert(t, repository != nil)
			names = append(names, repository.Name)
		}

		assert.Check(t, cmp.DeepEqual(names, []string{"base", "go", "node", "python"}))
	})

	t.Run("reports no match for a repository that is not there", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		cursor := api.Search("cimg/nope")
		assert.Check(t, !cursor.HasNext())
	})

	t.Run("reports no match in a namespace Docker Hub does not have", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		cursor := api.Search("nobody/anything")
		assert.Check(t, !cursor.HasNext())
	})

	// A page that fails to load used to leave the search asking for it again
	// in a tight loop, so each case gives up after a deadline rather than
	// hanging the test binary.
	t.Run("ends the search when a page will not load", func(t *testing.T) {
		failures := []struct {
			name string
			fail func(fake *fakes.DockerHub)
		}{
			{"rate limited", func(fake *fakes.DockerHub) {
				fake.SetStatus(cimgReposRoute, http.StatusTooManyRequests)
			}},
			{"a body that is not JSON", func(fake *fakes.DockerHub) {
				fake.SetBody(cimgReposRoute, "<html>502 Bad Gateway</html>")
			}},
			{"unreachable", func(fake *fakes.DockerHub) {
				fake.Close()
			}},
		}

		for _, failure := range failures {
			t.Run(failure.name, func(t *testing.T) {
				fake := cimgFake(t)
				api := apiFor(fake)
				failure.fail(fake)

				hasNext := make(chan bool, 1)
				go func() { hasNext <- api.Search("cimg/node").HasNext() }()

				select {
				case got := <-hasNext:
					assert.Check(t, !got)
				case <-time.After(5 * time.Second):
					t.Fatal("the search is still loading after 5s")
				}
			})
		}
	})

	// An unqualified name is a library image, which is the namespace the API
	// starts life knowing about.
	t.Run("searches the library namespace for an unqualified name", func(t *testing.T) {
		fake := cimgFake(t)
		fake.AddRepository("library", "node")
		api := apiFor(fake)

		cursor := api.Search("node")
		assert.Assert(t, cursor.HasNext())

		repository := cursor.Next()
		assert.Assert(t, repository != nil)
		assert.Check(t, cmp.Equal(repository.Namespace, "library"))

		requestCount := fake.RequestCount(http.MethodGet, "/v2/namespaces/library/repositories")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})
}

func TestSearchCursorPrev(t *testing.T) {
	fake := cimgFake(t)
	api := apiFor(fake)

	cursor := api.Search("cimg/")
	assert.Assert(t, cursor.HasNext())
	assert.Assert(t, cursor.Next() != nil) // base
	assert.Assert(t, cursor.Next() != nil) // go

	t.Run("reports nothing before the first match", func(t *testing.T) {
		fresh := api.Search("cimg/")
		assert.Check(t, cmp.Nil(fresh.Prev()))
	})

	// Prev does not step back one: it re-matches from the start of what has
	// been walked, so it reports the first match rather than the previous one.
	t.Run("reports the first match before the cursor", func(t *testing.T) {
		repository := cursor.Prev()
		assert.Assert(t, repository != nil)
		assert.Check(t, cmp.Equal(repository.Name, "base"))
	})
}
