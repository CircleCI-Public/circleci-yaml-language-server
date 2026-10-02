package dockerhub

import "context"

// TagsSearchCursor reads the pages of tags as it is walked, under the context
// of the search it was made for.
type TagsSearchCursor struct {
	ctx          context.Context
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

// SearchTags searches the tags of a repository on the Docker Hub cfg
// configures; see searchAPI.
func SearchTags(ctx context.Context, cfg Config, namespace, repo string, query string) (TagsResultsCursor, error) {
	return searchAPI(cfg).SearchTags(ctx, namespace, repo, query)
}

func (me *dockerHubAPI) SearchTags(ctx context.Context, namespace, repo string, query string) (TagsResultsCursor, error) {
	results, err := me.fetchTags(ctx, namespace, repo, query)

	if err != nil {
		return nil, err
	}

	return &TagsSearchCursor{
		ctx:          ctx,
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
		nextPage, err := t.lastResponse.loadNext(t.ctx, t.api)

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
