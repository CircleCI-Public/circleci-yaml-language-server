package dockerhub

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

func TestGetImageTags(t *testing.T) {
	// The callers offer this list in completion and recommend a tag from it,
	// so an inactive tag is left out altogether rather than leaving a gap.
	t.Run("leaves out an inactive tag", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "node")
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(tags, []string{"20.11", "22.1", "latest"}))
	})

	// One request, at the largest page Docker Hub serves, so that the
	// recommendation is chosen from more than the default handful.
	t.Run("reads one page, of a hundred tags", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetPageLimit(2)
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "node")
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(tags, []string{"20.11", "22.1"}))

		requestCount := fake.RequestCount(http.MethodGet, cimgNodeTagsPath)
		assert.Check(t, cmp.Equal(requestCount, 1))

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))

		assert.Check(t, cmp.Equal(requests[0].Query["page_size"], "100"))
	})

	t.Run("reports a repository Docker Hub does not have", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "nope")
		assert.Check(t, cmp.Nil(tags))
		assert.Check(t, httpcl.HasStatusCode(err, http.StatusNotFound), "got %v", err)
	})

	// A rate limit is the failure a busy editor actually hits, and it must not
	// read as a repository with no tags.
	t.Run("reports a rate limit", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus(cimgNodeTagsRoute, http.StatusTooManyRequests)
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "node")
		assert.Check(t, cmp.Nil(tags))
		assert.Check(t, httpcl.HasStatusCode(err, http.StatusTooManyRequests), "got %v", err)
	})

	// A body that is not JSON at all is reported, because decoding it fails.
	t.Run("reports a malformed body", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetBody(cimgNodeTagsRoute, "{")
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "node")
		assert.Check(t, cmp.Nil(tags))
		assert.Check(t, err != nil, "a body that will not decode must be reported")
	})

	t.Run("reports an unreachable Docker Hub", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)
		fake.Close()

		tags, err := api.GetImageTags("cimg", "node")
		assert.Check(t, cmp.Nil(tags))
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})
}

func TestImageHasTag(t *testing.T) {
	t.Run("confirms a tag the repository has", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		hasTag, err := api.ImageHasTag("cimg", "node", "22.1")
		assert.NilError(t, err)
		assert.Check(t, hasTag)
	})

	t.Run("denies a tag the repository does not have", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		hasTag, err := api.ImageHasTag("cimg", "node", "99.9")
		assert.NilError(t, err)
		assert.Check(t, !hasTag)
	})

	// An inactive tag is still a tag, and this route answers for one, so the
	// caller asking "does this tag exist" gets a yes — unlike GetImageTags,
	// which drops the name.
	t.Run("confirms an inactive tag", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		hasTag, err := api.ImageHasTag("cimg", "node", "18.0")
		assert.NilError(t, err)
		assert.Check(t, hasTag)
	})

	// As for DoesImageExist, only a 404 is a no.
	t.Run("reports an unreachable Docker Hub", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)
		fake.Close()

		hasTag, err := api.ImageHasTag("cimg", "node", "22.1")
		assert.Check(t, !hasTag)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})

	t.Run("reports a rate limit", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus("GET /v2/namespaces/cimg/repositories/node/tags/22.1", http.StatusTooManyRequests)
		api := apiFor(fake)

		hasTag, err := api.ImageHasTag("cimg", "node", "22.1")
		assert.Check(t, !hasTag)
		assert.Check(t, httpcl.HasStatusCode(err, http.StatusTooManyRequests), "got %v", err)
	})
}
