package dockerhub

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

func TestDoesImageExist(t *testing.T) {
	t.Run("confirms a repository Docker Hub has", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		exists, err := api.DoesImageExist("cimg", "node")
		assert.NilError(t, err)
		assert.Check(t, exists)

		requestCount := fake.RequestCount(http.MethodGet, cimgNodePath)
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("denies a repository Docker Hub does not have", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		exists, err := api.DoesImageExist("cimg", "nope")
		assert.NilError(t, err)
		assert.Check(t, !exists)
	})

	// Completion reads a namespace a page at a time and validation runs over
	// the same document, so a repository already read is answered locally.
	t.Run("answers from what a search already read", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		cursor := api.Search("cimg/node")
		assert.Assert(t, cursor.HasNext(), "the fixture has a cimg/node repository")

		exists, err := api.DoesImageExist("cimg", "node")
		assert.NilError(t, err)
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

		exists, err := api.DoesImageExist("circleci", "node")
		assert.NilError(t, err)
		assert.Check(t, exists)

		requestCount := fake.RequestCount(http.MethodGet, "/v2/namespaces/circleci/repositories/node")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	// Only a 404 is a no. Anything else means Docker Hub did not say, which
	// a caller must be able to tell apart from an image that is not there.
	t.Run("reports an unreachable Docker Hub", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)
		fake.Close()

		exists, err := api.DoesImageExist("cimg", "node")
		assert.Check(t, !exists)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})

	t.Run("reports a Docker Hub failure", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus(cimgNodeRoute, http.StatusInternalServerError)
		api := apiFor(fake)

		exists, err := api.DoesImageExist("cimg", "node")
		assert.Check(t, !exists)
		assert.Check(t, httpcl.HasStatusCode(err, http.StatusInternalServerError), "got %v", err)
	})

	t.Run("reports a rate limit", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus(cimgNodeRoute, http.StatusTooManyRequests)
		api := apiFor(fake)

		exists, err := api.DoesImageExist("cimg", "node")
		assert.Check(t, !exists)
		assert.Check(t, httpcl.HasStatusCode(err, http.StatusTooManyRequests), "got %v", err)
	})
}
