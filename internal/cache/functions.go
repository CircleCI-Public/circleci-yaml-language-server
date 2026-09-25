package cache

import (
	"context"
	"errors"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

// Functions remembers the functions catalog: each function by its name, with
// its versions, and each version's descriptor.
//
// What is returned is shared between callers, so it must not be changed.
type Functions struct {
	// packages holds nil for a function that isn't published.
	packages *memo.Memo[*circleci.FunctionPackage]
	// descriptors are keyed by version id. A version never changes once
	// published.
	descriptors *memo.Memo[*circleci.FunctionDescriptor]
}

func functionPackageLifetime(function *circleci.FunctionPackage) time.Duration {
	return memo.Existence(function != nil)
}

// Function returns the function a name such as
// "github.com/circleci-functions/setup-go" names, or nil when none is
// published. An error, such as from a host that doesn't serve the catalog, is
// returned but not remembered.
func (c *Functions) Function(client *circleci.V3Client, name string) (*circleci.FunctionPackage, error) {
	return c.packages.Get(name, func() (*circleci.FunctionPackage, error) {
		function, err := circleci.FetchFunction(context.Background(), client, name)
		if errors.Is(err, circleci.ErrFunctionNotPublished) {
			return nil, nil
		}
		return function, err
	})
}

// Descriptor returns the descriptor of a version of a function. The version
// must be one Function listed.
func (c *Functions) Descriptor(client *circleci.V3Client, version circleci.FunctionVersion) (*circleci.FunctionDescriptor, error) {
	return c.descriptors.Get(version.ID, func() (*circleci.FunctionDescriptor, error) {
		return circleci.FetchFunctionDescriptor(context.Background(), client, version.ID)
	})
}
