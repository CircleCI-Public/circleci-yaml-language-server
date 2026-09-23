package dockerhub

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestSearchTags(t *testing.T) {
	t.Run("narrows the tags to the query", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		cursor, err := api.SearchTags("cimg", "node", "22")
		assert.NilError(t, err)

		tag := cursor.Next()
		assert.Assert(t, tag != nil)
		assert.Check(t, cmp.Equal(tag.Name, "22.1"))

		// The narrowing is the API's, and the page is asked for at its
		// largest — unlike GetImageTags, which asks for neither.
		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.Equal(requests[0].Query["name"], "22"))
		assert.Check(t, cmp.Equal(requests[0].Query["page_size"], "100"))
	})

	// Inactive tags are walked too: this cursor reports what the API reported,
	// where GetImageTags drops the name of an inactive tag.
	t.Run("walks every tag the repository has", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)

		cursor, err := api.SearchTags("cimg", "node", "")
		assert.NilError(t, err)

		names := []string{}
		for cursor.HasNext() {
			tag := cursor.Next()
			assert.Assert(t, tag != nil)
			names = append(names, tag.Name)
		}

		assert.Check(t, cmp.DeepEqual(names, []string{"20.11", "22.1", "latest", "18.0"}))
	})

	// Tags are the collection that really does run to several pages, and the
	// next URL is absolute, so this exercises following it.
	t.Run("follows the next page", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetPageLimit(1)
		api := apiFor(fake)

		cursor, err := api.SearchTags("cimg", "node", "")
		assert.NilError(t, err)

		count := 0
		for cursor.HasNext() {
			assert.Assert(t, cursor.Next() != nil)
			count++
		}

		assert.Check(t, cmp.Equal(count, 4))

		requestCount := fake.RequestCount(http.MethodGet, cimgNodeTagsPath)
		assert.Check(t, cmp.Equal(requestCount, 4))
	})

	// A request that reached Docker Hub is never checked for its status, so an
	// error body decodes into a page with no tags on it: a rate-limited search
	// is an empty cursor rather than a failure. Recorded as it stands.
	t.Run("hands back an empty cursor when Docker Hub rate limits", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetStatus(cimgNodeTagsRoute, http.StatusTooManyRequests)
		api := apiFor(fake)

		cursor, err := api.SearchTags("cimg", "node", "")
		assert.NilError(t, err, "a 429 is not reported as an error")
		assert.Assert(t, cursor != nil)
		assert.Check(t, !cursor.HasNext())
	})

	// Same for a later page: the walk ends where the failure was, as though
	// the tags had run out.
	t.Run("treats a failing later page as the end of the tags", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetPageLimit(1)
		fake.FailAfter(cimgNodeTagsRoute, 1, http.StatusInternalServerError)
		api := apiFor(fake)

		cursor, err := api.SearchTags("cimg", "node", "")
		assert.NilError(t, err)

		count := 0
		for cursor.HasNext() {
			assert.Assert(t, cursor.Next() != nil)
			count++
		}

		assert.Check(t, cmp.Equal(count, 1), "the tag from the page that loaded is still walked")
	})

	// A request that could not be made at all is a real error, and there the
	// cursor drops the page it is holding: it answers no while the tag it read
	// is still unwalked. Recorded as it stands.
	t.Run("abandons the tag it read when Docker Hub goes away", func(t *testing.T) {
		fake := cimgFake(t)
		fake.SetPageLimit(1)
		api := apiFor(fake)

		cursor, err := api.SearchTags("cimg", "node", "")
		assert.NilError(t, err)

		fake.Close()

		hasNext := cursor.HasNext()
		assert.Check(t, !hasNext, "the tag from the page that loaded is not offered")
	})

	t.Run("reports an unreachable Docker Hub", func(t *testing.T) {
		fake := cimgFake(t)
		api := apiFor(fake)
		fake.Close()

		cursor, err := api.SearchTags("cimg", "node", "")
		assert.Check(t, cmp.Nil(cursor))
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})
}

func TestTagsSearchCursorPrev(t *testing.T) {
	fake := cimgFake(t)
	api := apiFor(fake)

	cursor, err := api.SearchTags("cimg", "node", "")
	assert.NilError(t, err)

	t.Run("reports nothing before the first tag", func(t *testing.T) {
		tag := cursor.Prev()
		assert.Check(t, cmp.Nil(tag))
	})

	t.Run("steps back over the tags already walked", func(t *testing.T) {
		assert.Assert(t, cursor.Next() != nil) // 20.11
		assert.Assert(t, cursor.Next() != nil) // 22.1

		tag := cursor.Prev()
		assert.Assert(t, tag != nil)
		assert.Check(t, cmp.Equal(tag.Name, "20.11"))
	})
}
