package dockerhub

// These tests record what this package does today, against a fake Docker Hub.
// Where the behaviour looks wrong the test says so and pins it anyway: the
// point is a spec to change the code against, not the code the spec implies.
//
// They live inside the package because the searches are methods on the
// concrete API: the exported interface carries only the three calls validation
// makes, and the package-level Search and SearchTags read through the default
// API, which points at the real Docker Hub.

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

// Routes, as the fake names them for SetStatus, SetBody and FailAfter.
const (
	cimgReposRoute    = "GET /v2/namespaces/cimg/repositories"
	cimgNodeRoute     = "GET /v2/namespaces/cimg/repositories/node"
	cimgNodeTagsRoute = "GET /v2/namespaces/cimg/repositories/node/tags"
)

// Paths, as the fake records them for RequestCount.
const (
	cimgReposPath    = "/v2/namespaces/cimg/repositories"
	cimgNodePath     = "/v2/namespaces/cimg/repositories/node"
	cimgNodeTagsPath = "/v2/namespaces/cimg/repositories/node/tags"
)

// cimgFake builds a fake carrying the cimg namespace: four repositories in a
// known order, and a spread of tags on cimg/node including an inactive one,
// which the API reports and callers are meant to leave out.
func cimgFake(t *testing.T) *fakes.DockerHub {
	t.Helper()

	fake := fakes.NewDockerHub(t)

	for _, repository := range []string{"base", "go", "node", "python"} {
		fake.AddRepository("cimg", repository)
	}

	fake.AddTag("cimg", "node", "20.11", "")
	fake.AddTag("cimg", "node", "22.1", "")
	fake.AddTag("cimg", "node", "latest", "")
	fake.AddTag("cimg", "node", "18.0", "inactive")

	return fake
}

// apiFor is an API pointed at a fake.
func apiFor(fake *fakes.DockerHub) *dockerHubAPI {
	return newAPI(Config{BaseURL: fake.URL()})
}

// TestNewAPIWithConfig covers the seam itself, which is what a caller outside
// this package configures: everything else here builds the API directly.
func TestNewAPIWithConfig(t *testing.T) {
	fake := cimgFake(t)

	var api DockerHubAPI = NewAPIWithConfig(Config{BaseURL: fake.URL()})

	exists := api.DoesImageExist("cimg", "node")
	assert.Check(t, exists)

	t.Run("identifies the language server", func(t *testing.T) {
		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.Equal(requests[0].UserAgent, utils.UserAgent))
	})

	// What one API read is its own, so a test cannot leave a namespace behind
	// for the next one — which the package-level cache this replaced did.
	t.Run("shares no cache with another API", func(t *testing.T) {
		searched := apiFor(fake)
		searched.Search("cimg/node").HasNext()

		fresh := apiFor(fake)
		exists := fresh.DoesImageExist("cimg", "node")
		assert.Check(t, exists)

		requestCount := fake.RequestCount(http.MethodGet, cimgNodePath)
		assert.Check(t, cmp.Equal(requestCount, 2))
	})
}
