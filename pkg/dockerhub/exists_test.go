package dockerhub

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestDoesImageExist(t *testing.T) {
	t.Run("confirms a repository Docker Hub has", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		exists := api.DoesImageExist("cimg", "node")
		assert.Check(t, exists)

		requestCount := fake.RequestCount(http.MethodGet, cimgNodePath)
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("denies a repository Docker Hub does not have", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		exists := api.DoesImageExist("cimg", "nope")
		assert.Check(t, !exists)
	})

	// Completion reads a namespace a page at a time and validation runs over
	// the same document, so a repository already read is answered locally.
	t.Run("answers from what a search already read", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		cursor := api.Search("cimg/node")
		assert.Assert(t, cursor.HasNext(), "the fixture has a cimg/node repository")

		exists := api.DoesImageExist("cimg", "node")
		assert.Check(t, exists)

		requestCount := fake.RequestCount(http.MethodGet, cimgNodePath)
		assert.Check(t, cmp.Equal(requestCount, 0), "a repository already read is not fetched again")
	})

	// The short-circuit only covers the namespace that was searched, so a
	// repository in another one is still a request.
	t.Run("asks about a repository no search has read", func(t *testing.T) {
		fake := cimgFake(t)
		fake.AddRepository("circleci", "node")
		api := apiFor(fake)

		assert.Assert(t, api.Search("cimg/node").HasNext())

		exists := api.DoesImageExist("circleci", "node")
		assert.Check(t, exists)

		requestCount := fake.RequestCount(http.MethodGet, "/v2/namespaces/circleci/repositories/node")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	// An unreachable Docker Hub reads as "no such image", which the caller
	// turns into a valid image rather than a diagnostic.
	t.Run("denies when Docker Hub is unreachable", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)
		fake.Close()

		exists := api.DoesImageExist("cimg", "node")
		assert.Check(t, !exists)
	})

	t.Run("denies when Docker Hub fails", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus(cimgNodeRoute, http.StatusInternalServerError)
		api := apiFor(fake)

		exists := api.DoesImageExist("cimg", "node")
		assert.Check(t, !exists)
	})

	// A rate limit is indistinguishable from a missing image here, so an
	// editor that has been busy starts reporting images as unknown.
	t.Run("denies when Docker Hub rate limits", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus(cimgNodeRoute, http.StatusTooManyRequests)
		api := apiFor(fake)

		exists := api.DoesImageExist("cimg", "node")
		assert.Check(t, !exists)
	})
}
