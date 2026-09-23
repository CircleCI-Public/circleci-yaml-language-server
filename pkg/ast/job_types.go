package ast

func HasStoreTestResultStep(step []Step) bool {
	for _, s := range step {
		switch s := s.(type) {
		case NamedStep:
			if s.Name == "store_test_results" {
				return true
			}
		case StoreTestResults:
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
	"lock",
	"unlock",
}

// CheckoutMethods is a list of all valid checkout methods that are supported by CircleCI
var CheckoutMethods = []string{
	"blobless",
	"full", // default
	"shallow",
}
