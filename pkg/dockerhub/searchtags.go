package dockerhub

type TagsSearchCursor struct {
	api          *dockerHubAPI
	query        string
	index        int
	results      []RepoTag
	lastResponse TagResponse
}

type TagsResultsCursor interface {
	HasNext() bool
	Next() *RepoTag
	Prev() *RepoTag
}

// SearchTags reads through the process-wide default API; a caller with an API
// of its own searches through that.
func SearchTags(namespace, repo string, query string) (TagsResultsCursor, error) {
	return defaultAPI.SearchTags(namespace, repo, query)
}

func (me *dockerHubAPI) SearchTags(namespace, repo string, query string) (TagsResultsCursor, error) {
	results, err := me.fetchTags(namespace, repo, query)

	if err != nil {
		return nil, err
	}

	return &TagsSearchCursor{
		api:          me,
		query:        query,
		index:        0,
		results:      results.Results,
		lastResponse: results,
	}, nil
}

// --
// Implement
// --

func (t *TagsSearchCursor) HasNext() bool {
	if t.index >= len(t.results)-1 && t.lastResponse.Next != "" {
		nextPage, err := t.lastResponse.loadNext(t.api)

		if err != nil {
			// The walk ends at the page that failed, rather than the cursor
			// with it: tags already read are still offered, and forgetting the
			// failed page stops the next call asking for it again.
			t.lastResponse.Next = ""
		} else {
			t.results = append(t.results, nextPage.Results...)
			t.lastResponse = nextPage
		}
	}

	return t.index < len(t.results)
}

func (t *TagsSearchCursor) Next() *RepoTag {
	forwards := t.forwards()
	if len(forwards) == 0 {
		return nil
	}

	t.index += 1
	return &forwards[0]
}

func (t *TagsSearchCursor) Prev() *RepoTag {
	back := t.backwards()

	if len(back) < 2 {
		return nil
	}

	t.index -= 1

	return &back[len(back)-2]
}

func (t *TagsSearchCursor) forwards() []RepoTag {
	if t.index < len(t.results) {
		return t.results[t.index:]
	}

	return []RepoTag{}
}

func (t *TagsSearchCursor) backwards() []RepoTag {
	if t.index > len(t.results)-1 {
		return t.results[:]
	}

	return t.results[:t.index]
}
