package utils

import "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"

func HasStoreTestResultStep(step []ast.Step) bool {
	for _, s := range step {
		switch s := s.(type) {
		case ast.NamedStep:
			if s.Name == "store_test_results" {
				return true
			}
		case ast.StoreTestResults:
			return true
		}
	}
	return false
}

// JobTypes is a list of all valid job types that are supported by CircleCI
var JobTypes = []string{
	"approval",
	"build", // default
	"no-op",
	"release",
}
