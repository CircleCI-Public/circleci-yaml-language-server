package circleci

import (
	"context"
	"log/slog"
	"sync"
)

// OrbRegistry reads orb and namespace metadata.
//
// The interface is shaped around what the language server needs rather than
// around the API's routes.
type OrbRegistry interface {
	// FetchOrb returns an orb and its released versions, newest first. It
	// reports ErrNotFound when no such orb is visible.
	FetchOrb(ctx context.Context, fullName string) (*OrbPackage, error)
	// ResolveVersion resolves a reference such as "circleci/go@1.7",
	// "circleci/go@volatile" or "circleci/go@dev:alpha" to a concrete version,
	// with its source and its siblings. It reports ErrNotFound when the
	// reference does not resolve.
	ResolveVersion(ctx context.Context, ref string) (*ResolvedOrbVersion, error)
	// FetchNamespace reports whether a registry namespace exists. It reports
	// ErrNotFound when it does not.
	FetchNamespace(ctx context.Context, name string) (*Namespace, error)
	// ListNamespaceOrbs returns the orbs in a namespace, each with its
	// versions. It reports ErrNotFound when the namespace does not exist.
	ListNamespaceOrbs(ctx context.Context, name string) ([]OrbPackage, error)
}

// ResolvedOrbVersion is an orb reference resolved to a concrete version.
type ResolvedOrbVersion struct {
	ID           string
	Version      string
	Source       string
	OrbPackageID string
	// Versions holds the releases published by the same orb, newest first.
	// Upgrade hints are computed from it. It excludes development tags.
	Versions []OrbPackageVersion
}

// NewOrbRegistry returns the registry served by a V3 client.
func NewOrbRegistry(client *V3Client) OrbRegistry {
	return v3OrbRegistry{client: client}
}

type v3OrbRegistry struct {
	client *V3Client
}

func (registry v3OrbRegistry) FetchOrb(ctx context.Context, fullName string) (*OrbPackage, error) {
	return FetchOrbPackage(ctx, registry.client, fullName)
}

func (registry v3OrbRegistry) FetchNamespace(ctx context.Context, name string) (*Namespace, error) {
	return FetchNamespace(ctx, registry.client, name)
}

func (registry v3OrbRegistry) ListNamespaceOrbs(ctx context.Context, name string) ([]OrbPackage, error) {
	namespace, err := FetchNamespace(ctx, registry.client, name)
	if err != nil {
		return nil, err
	}

	return ListNamespaceOrbs(ctx, registry.client, namespace.ID)
}

// ResolveVersion takes three requests. Resolving the reference and listing the
// orb's versions are independent, so they overlap; only the source fetch has to
// wait, because it is addressed by the resolved version's id.
func (registry v3OrbRegistry) ResolveVersion(ctx context.Context, ref string) (*ResolvedOrbVersion, error) {
	var (
		resolved   *OrbVersionRef
		resolveErr error
		orbPackage *OrbPackage
		packageErr error
		wait       sync.WaitGroup
	)

	wait.Add(2)
	go func() {
		defer wait.Done()
		resolved, resolveErr = ResolveOrbRef(ctx, registry.client, ref, "")
	}()
	go func() {
		defer wait.Done()
		orbPackage, packageErr = FetchOrbPackage(ctx, registry.client, OrbPackageName(ref))
	}()
	wait.Wait()

	if resolveErr != nil {
		return nil, resolveErr
	}

	source, err := FetchOrbSource(ctx, registry.client, resolved.ID)
	if err != nil {
		return nil, err
	}

	version := &ResolvedOrbVersion{
		ID:           resolved.ID,
		Version:      resolved.Version,
		Source:       source,
		OrbPackageID: resolved.OrbPackageID,
	}

	// The version list only drives upgrade hints, so failing to fetch it costs
	// those hints rather than the whole orb. Say so in the log: an empty list
	// is otherwise indistinguishable from an orb that has never released.
	switch {
	case packageErr != nil:
		slog.Warn("listing orb versions for upgrade hints",
			"orb", OrbPackageName(ref), "err", packageErr,
		)
	case orbPackage != nil:
		version.Versions = orbPackage.Versions
	}

	return version, nil
}
