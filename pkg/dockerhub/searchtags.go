package dockerhub

type TagsSearchCursor struct {
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

func SearchTags(namespace, repo string, query string) (TagsResultsCursor, error) {
	results, err := fetchTags(namespace, repo, query)

	if err != nil {
		return nil, err
	}

	return &TagsSearchCursor{
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
		nextPage, err := t.lastResponse.loadNext()

		if err != nil {
			return false
		}

		t.results = append(t.results, nextPage.Results...)
		t.lastResponse = nextPage
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
