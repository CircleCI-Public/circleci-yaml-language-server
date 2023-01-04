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

var hubNamespaces = map[string]*HubNamespace{
	"library": {
		namespace: "library",
	},
}

func Search(query string) DockerResultsCursor {
	namespace := getQueryNamespace(query)
	imageName := getQueryImageName(query)

	if hubNamespaces[namespace] == nil {
		hubNamespaces[namespace] = &HubNamespace{
			namespace: namespace,
		}
	}

	ns := hubNamespaces[namespace]

	return ns.createSearchCursor(imageName)
}

// --
// Implement Cursor for search results
// --

func (s *SearchCursor) HasNext() bool {
	start := s.index + 1
	searchItems := s.hub.allRepositories[start:]
	_, index := findFirstMatch(&searchItems, s.query)

	if index >= 0 {
		return true
	}

	for s.hub.nextURL != "" || !s.hub.hasLoaded {
		s.hub.loadNext()
		searchItems := s.hub.allRepositories[start:]

		_, index = findFirstMatch(
			&searchItems,
			s.query,
		)
	}

	return index >= 0
}

func (s *SearchCursor) Next() *Repository {
	start := s.index + 1
	searchDomain := s.hub.allRepositories[start:] // TODO: Check Bounds
	repo, index := findFirstMatch(&searchDomain, s.query)

	if index >= 0 {
		s.index += index + 1
	}

	return repo
}

func (s *SearchCursor) Prev() *Repository {
	searchDomain := s.hub.allRepositories[:s.index] // TODO: Check bounds
	repo, index := findFirstMatch(&searchDomain, s.query)

	if index >= 0 {
		s.index -= len(searchDomain) - index
	}

	return repo
}
