package circleci

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// ErrFunctionNotPublished reports that the catalog has no function by a name.
// It is not ErrNotFound, which a host that doesn't serve the route answers
// with too.
var ErrFunctionNotPublished = errors.New("function not published")

// FunctionPackage is a function published to the functions catalog, with its
// versions, from GET /api/v3/function/packages.
type FunctionPackage struct {
	Name          string
	Description   string
	LatestVersion string
	Versions      []FunctionVersion
}

// FunctionVersion is one published version of a function.
type FunctionVersion struct {
	ID      string
	Version string
}

// FunctionDescriptor is a function version's function.yaml: what it does and
// the flags it takes, which a step passes under `with`.
type FunctionDescriptor struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Version     string                     `json:"version"`
	Flags       []FunctionFlag             `json:"flags"`
	Commands    map[string]FunctionCommand `json:"commands"`
}

// FunctionCommand is a subcommand of a function, which a step names as
// `alias/command`.
type FunctionCommand struct {
	Description string         `json:"description"`
	Flags       []FunctionFlag `json:"flags"`
}

// FunctionFlag is a flag a function or one of its commands takes.
type FunctionFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     any    `json:"default"`
	Description string `json:"description"`
}

// FetchFunction returns the function a name such as
// "github.com/circleci-functions/setup-go" names, with its versions. It
// reports ErrFunctionNotPublished when no such function is published.
//
// A host that doesn't serve the route, such as CircleCI Server, answers 404,
// which is returned as an APIError: that says nothing about the function.
func FetchFunction(ctx context.Context, cl *V3Client, name string) (*FunctionPackage, error) {
	query := url.Values{}
	query.Set("filter[name]", name)

	type entity struct {
		Attributes struct {
			Name          string `json:"name"`
			Description   string `json:"description"`
			LatestVersion string `json:"latest_version"`
		} `json:"attributes"`
		References struct {
			Versions []struct {
				ID         string `json:"id"`
				Attributes struct {
					Version string `json:"version"`
				} `json:"attributes"`
			} `json:"versions"`
		} `json:"references"`
	}

	entities, err := GetPaged[entity](ctx, cl, "function/packages", query)
	if err != nil {
		return nil, err
	}

	for _, found := range entities {
		if found.Attributes.Name != name {
			continue
		}

		function := &FunctionPackage{
			Name:          found.Attributes.Name,
			Description:   found.Attributes.Description,
			LatestVersion: found.Attributes.LatestVersion,
		}
		for _, version := range found.References.Versions {
			function.Versions = append(function.Versions, FunctionVersion{ID: version.ID, Version: version.Attributes.Version})
		}

		return function, nil
	}

	return nil, fmt.Errorf("function %s: %w", name, ErrFunctionNotPublished)
}

// FetchFunctionDescriptor returns the descriptor of a function version, by
// the id FetchFunction gave it.
func FetchFunctionDescriptor(ctx context.Context, cl *V3Client, versionID string) (*FunctionDescriptor, error) {
	var entity struct {
		Attributes struct {
			Descriptor FunctionDescriptor `json:"descriptor"`
		} `json:"attributes"`
	}

	if err := cl.Get(ctx, "function/versions/"+url.PathEscape(versionID), nil, &entity); err != nil {
		return nil, err
	}

	return &entity.Attributes.Descriptor, nil
}
