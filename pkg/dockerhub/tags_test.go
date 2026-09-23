package dockerhub

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestGetImageTags(t *testing.T) {
	// An inactive tag is dropped by name but not by position: the list is
	// sized to every tag and written only at the active ones, so an inactive
	// tag leaves an empty name behind. Recorded as it stands — the callers
	// offer this list in completion and recommend a tag from it.
	t.Run("leaves an empty name where an inactive tag was", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "node")
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(tags, []string{"20.11", "22.1", "latest", ""}))
	})

	// One request, and no page size: Docker Hub's default page is small, so
	// this reports the first handful of tags of a repository that has many.
	t.Run("reads one page, at whatever size Docker Hub defaults to", func(t *testing.T) {
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

		_, asksForASize := requests[0].Query["page_size"]
		assert.Check(t, !asksForASize, "no page size is requested")
	})

	// The status is never read, so an error body decodes into a response with
	// no results and a repository nobody has reads as one with no tags.
	t.Run("reports a repository Docker Hub does not have as having no tags", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "nope")
		assert.Check(t, cmp.Nil(err), "a 404 is not reported as an error")
		assert.Check(t, cmp.Len(tags, 0))
	})

	// Same for a rate limit, which is the failure a busy editor actually hits.
	t.Run("reports a rate limit as having no tags", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus(cimgNodeTagsRoute, http.StatusTooManyRequests)
		api := apiFor(fake)

		tags, err := api.GetImageTags("cimg", "node")
		assert.Check(t, cmp.Nil(err), "a 429 is not reported as an error")
		assert.Check(t, cmp.Len(tags, 0))
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

		hasTag := api.ImageHasTag("cimg", "node", "22.1")
		assert.Check(t, hasTag)
	})

	t.Run("denies a tag the repository does not have", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		hasTag := api.ImageHasTag("cimg", "node", "99.9")
		assert.Check(t, !hasTag)
	})

	// An inactive tag is still a tag, and this route answers for one, so the
	// caller asking "does this tag exist" gets a yes — unlike GetImageTags,
	// which drops the name.
	t.Run("confirms an inactive tag", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		hasTag := api.ImageHasTag("cimg", "node", "18.0")
		assert.Check(t, hasTag)
	})

	t.Run("denies when Docker Hub is unreachable", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)
		fake.Close()

		hasTag := api.ImageHasTag("cimg", "node", "22.1")
		assert.Check(t, !hasTag)
	})

	// Any non-200 is a no, a rate limit included.
	t.Run("denies when Docker Hub rate limits", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus("GET /v2/namespaces/cimg/repositories/node/tags/22.1", http.StatusTooManyRequests)
		api := apiFor(fake)

		hasTag := api.ImageHasTag("cimg", "node", "22.1")
		assert.Check(t, !hasTag)
	})
}
