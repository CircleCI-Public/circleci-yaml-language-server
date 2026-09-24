package cache

import (
	"context"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

// OrbPackages remembers orbs by their "namespace/orb" name, with the versions
// each has published, and the orbs each namespace publishes. Validation asks
// whether an orb exists; completion asks for its versions, and for the orbs of
// a namespace as a name is typed.
//
// What is returned is shared between callers, so it must not be changed.
type OrbPackages struct {
	// packages holds nil for an orb that does not exist.
	packages *memo.Memo[*circleci.OrbPackage]
	// namespaces holds nil for a namespace that does not exist.
	namespaces *memo.Memo[[]circleci.OrbPackage]
}

func orbPackageLifetime(orb *circleci.OrbPackage) time.Duration {
	return memo.Existence(orb != nil)
}

func namespaceOrbsLifetime(orbs []circleci.OrbPackage) time.Duration {
	return memo.Existence(len(orbs) != 0)
}

// Orb returns the orb a "namespace/orb" name names, or nil when there is no
// such orb or it is private to an organization the token cannot see. It asks
// the registry only when no answer is remembered, and an error from the
// registry is returned but not remembered.
func (c *OrbPackages) Orb(registry circleci.OrbRegistry, name string) (*circleci.OrbPackage, error) {
	return c.packages.Get(name, func() (*circleci.OrbPackage, error) {
		orb, err := registry.FetchOrb(context.Background(), name)
		if circleci.IsNotFound(err) {
			return nil, nil
		}
		return orb, err
	})
}

// InNamespace returns the orbs a namespace publishes, or nil when there is no
// such namespace. It asks the registry only when no answer is remembered, and
// an error from the registry is returned but not remembered.
//
// The listing carries each orb's versions, so each orb it finds is remembered
// for Orb too.
func (c *OrbPackages) InNamespace(registry circleci.OrbRegistry, namespace string) ([]circleci.OrbPackage, error) {
	return c.namespaces.Get(namespace, func() ([]circleci.OrbPackage, error) {
		orbs, err := registry.ListNamespaceOrbs(context.Background(), namespace)
		if circleci.IsNotFound(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		for i := range orbs {
			c.packages.Put(orbs[i].Name, &orbs[i])
		}

		return orbs, nil
	})
}
