package dockerhub

type SearchCursor struct {
	hub   *HubNamespace
	index int
	query string
}

type DockerResultsCursor interface {
	HasNext() bool
	Next() *Repository
	Prev() *Repository
}

// Search reads through the process-wide default API; a caller with an API of
// its own searches through that.
func Search(query string) DockerResultsCursor {
	return defaultAPI.Search(query)
}

func (me *dockerHubAPI) Search(query string) DockerResultsCursor {
	namespace := getQueryNamespace(query)
	imageName := getQueryImageName(query)

	return me.namespace(namespace).createSearchCursor(imageName)
}

// --
// Implement Cursor for search results
// --

func (s *SearchCursor) HasNext() bool {
	s.hub.mutex.Lock()
	defer s.hub.mutex.Unlock()

	start := s.index + 1

	for {
		searchItems := s.hub.allRepositories[start:]
		if _, index := findFirstMatch(&searchItems, s.query); index >= 0 {
			return true
		}

		if s.hub.hasLoaded && s.hub.nextURL == "" {
			return false
		}

		// A failed load sets neither nextURL nor hasLoaded, so going round
		// again would ask for the same page for ever.
		if _, err := s.hub.loadNext(); err != nil {
			return false
		}
	}
}

func (s *SearchCursor) Next() *Repository {
	s.hub.mutex.Lock()
	defer s.hub.mutex.Unlock()

	// index is -1 or the position of the last match, so start is never past
	// the end.
	start := s.index + 1
	searchDomain := s.hub.allRepositories[start:]
	repo, index := findFirstMatch(&searchDomain, s.query)

	if index >= 0 {
		s.index += index + 1
	}

	return repo
}

func (s *SearchCursor) Prev() *Repository {
	s.hub.mutex.Lock()
	defer s.hub.mutex.Unlock()

	// Before the first Next there is nothing behind the cursor, and index is
	// -1, which is no bound to slice to.
	if s.index <= 0 {
		return nil
	}

	searchDomain := s.hub.allRepositories[:s.index]
	repo, index := findFirstMatch(&searchDomain, s.query)

	if index >= 0 {
		s.index -= len(searchDomain) - index
	}

	return repo
}
