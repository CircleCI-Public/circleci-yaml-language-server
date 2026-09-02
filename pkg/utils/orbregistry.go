package utils

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
)

// OrbRegistry reads orb and namespace metadata.
//
// There are two implementations. The V3 REST API is used on circleci.com, and
// GraphQL is used on CircleCI Server, which does not serve the V3 orb and
// namespace routes. NewOrbRegistry picks between them per host and remembers
// the answer, so callers never have to care which is in play.
//
// The interface is shaped around what the language server needs rather than
// around either API's routes: GraphQL answers ResolveVersion in one request,
// where V3 needs three.
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
	//
	// The list is best-effort: a non-nil error means nothing was read, and a
	// nil error means the orbs returned are usable but not guaranteed to be
	// the whole namespace. The GraphQL implementation reads a single page of
	// orbs and logs if a namespace outgrows it, on the grounds that
	// autocomplete over most of a namespace beats autocomplete over none of
	// it.
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

// orbVersionCount is the ceiling put on GraphQL version lists. The V3 routes
// need no equivalent, because they return every version.
const orbVersionCount = 10000

// NewOrbRegistry returns the registry for a host.
func NewOrbRegistry(hostUrl, token, userId string, debug bool) OrbRegistry {
	return &fallbackOrbRegistry{
		host: hostUrl,
		v3:   v3OrbRegistry{client: NewV3Client(hostUrl, token, userId, debug)},
		gql:  graphqlOrbRegistry{client: NewClient(hostUrl, "graphql-unstable", token, debug), userId: userId},
	}
}

// NewOrbRegistryFromContext returns the registry configured from the language
// server context.
func NewOrbRegistryFromContext(lsContext *LsContext) OrbRegistry {
	return NewOrbRegistry(
		lsContext.Api.HostUrl,
		lsContext.Api.Token,
		lsContext.UserIdForTelemetry,
		false,
	)
}

// NewGraphQLOrbRegistry returns a registry that always uses GraphQL, skipping
// capability detection.
//
// The fallback cannot otherwise be exercised against real infrastructure
// without a CircleCI Server instance, and circleci.com serves GraphQL as well
// as V3 — so this makes the GraphQL path checkable against production. See
// cmd/orb.
func NewGraphQLOrbRegistry(hostUrl, token, userId string, debug bool) OrbRegistry {
	return graphqlOrbRegistry{
		client: NewClient(hostUrl, "graphql-unstable", token, debug),
		userId: userId,
	}
}

// --- Capability detection ---

// v3OrbRoutes caches, per host, whether the V3 orb routes are served. Hosts are
// long-lived and the answer cannot change under a running process, so one
// answer per host is enough.
var v3OrbRoutes = struct {
	mutex sync.RWMutex
	known map[string]bool
}{known: map[string]bool{}}

func lookupV3OrbRoutes(host string) (available, known bool) {
	v3OrbRoutes.mutex.RLock()
	defer v3OrbRoutes.mutex.RUnlock()

	available, known = v3OrbRoutes.known[host]

	return available, known
}

func recordV3OrbRoutes(host string, available bool) {
	v3OrbRoutes.mutex.Lock()
	defer v3OrbRoutes.mutex.Unlock()

	v3OrbRoutes.known[host] = available
}

// resetV3OrbRoutes forgets what is known about every host. It exists for tests,
// which stand up a fresh server per case on a fresh address, and reaches them
// through export_test.go rather than this package's public API.
func resetV3OrbRoutes() {
	v3OrbRoutes.mutex.Lock()
	defer v3OrbRoutes.mutex.Unlock()

	v3OrbRoutes.known = map[string]bool{}
}

// fallbackOrbRegistry dispatches to V3 where it is available and to GraphQL
// where it is not.
type fallbackOrbRegistry struct {
	host string
	v3   v3OrbRegistry
	gql  graphqlOrbRegistry
}

// orbRoutesProbeName is a name no orb can have, so the probe never matches a
// real orb and never has a side effect.
const orbRoutesProbeName = "circleci-yaml-language-server/v3-capability-probe"

// useV3 reports whether the V3 orb routes are available on this host,
// discovering it once and remembering the answer.
//
// The probe asks orb/packages for a name that cannot exist. That route answers
// a missing orb with 200 and an empty list, so a 404 from it can only mean the
// route itself is absent — which is exactly the CircleCI Server case. Every
// other orb and namespace route uses 404 for missing data, so none of them can
// tell "no such orb" from "no such route".
//
// The probe deliberately goes through the HTTP client rather than FetchOrb,
// because FetchOrb folds an empty list and a 404 into the same ErrNotFound and
// so cannot distinguish the two either.
func (registry *fallbackOrbRegistry) useV3(ctx context.Context) bool {
	// circleci.com serves V3, so the probe is skipped for it. This keeps the
	// common case at zero extra requests.
	if v3OrbRoutesAssumed(registry.host) {
		return true
	}

	if available, known := lookupV3OrbRoutes(registry.host); known {
		return available
	}

	available, err := v3OrbRoutesRespond(ctx, registry.v3.client)
	if err != nil {
		// The request failed for a reason that says nothing about whether the
		// route exists. Assume V3 so that circleci.com-like hosts keep working,
		// and do not cache, so a later call can settle the question.
		return true
	}

	recordV3OrbRoutes(registry.host, available)

	return available
}

// v3OrbRoutesAssumed reports whether a host is known to serve the V3 orb routes
// without having to ask. Only circleci.com qualifies; every other host,
// including a CircleCI Server instance, has to be probed.
func v3OrbRoutesAssumed(host string) bool {
	return host == CIRCLE_CI_APP_HOST_URL
}

// v3OrbRoutesRespond reports whether GET /api/v3/orb/packages is served.
func v3OrbRoutesRespond(ctx context.Context, client *V3Client) (bool, error) {
	query := url.Values{}
	query.Set("filter[name]", orbRoutesProbeName)

	// A served route answers 200 with an empty list, which is a nil error here.
	_, err := GetPaged[struct{}](ctx, client, "orb/packages", query)
	switch {
	case err == nil:
		return true, nil
	case IsNotFound(err):
		return false, nil
	default:
		return false, err
	}
}

func (registry *fallbackOrbRegistry) FetchOrb(ctx context.Context, fullName string) (*OrbPackage, error) {
	if !registry.useV3(ctx) {
		return registry.gql.FetchOrb(ctx, fullName)
	}

	return registry.v3.FetchOrb(ctx, fullName)
}

func (registry *fallbackOrbRegistry) ResolveVersion(ctx context.Context, ref string) (*ResolvedOrbVersion, error) {
	if !registry.useV3(ctx) {
		return registry.gql.ResolveVersion(ctx, ref)
	}

	return registry.v3.ResolveVersion(ctx, ref)
}

func (registry *fallbackOrbRegistry) FetchNamespace(ctx context.Context, name string) (*Namespace, error) {
	if !registry.useV3(ctx) {
		return registry.gql.FetchNamespace(ctx, name)
	}

	return registry.v3.FetchNamespace(ctx, name)
}

func (registry *fallbackOrbRegistry) ListNamespaceOrbs(ctx context.Context, name string) ([]OrbPackage, error) {
	if !registry.useV3(ctx) {
		return registry.gql.ListNamespaceOrbs(ctx, name)
	}

	return registry.v3.ListNamespaceOrbs(ctx, name)
}

// --- V3 ---

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

// ResolveVersion takes three requests where GraphQL takes one. Resolving the
// reference and listing the orb's versions are independent, so they overlap;
// only the source fetch has to wait, because it is addressed by the resolved
// version's id.
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
	// those hints rather than the whole orb. Say so on stderr: an empty list is
	// otherwise indistinguishable from an orb that has never released.
	switch {
	case packageErr != nil:
		fmt.Fprintf(os.Stderr,
			"listing versions of %s for upgrade hints: %s\n",
			OrbPackageName(ref), packageErr,
		)
	case orbPackage != nil:
		version.Versions = orbPackage.Versions
	}

	return version, nil
}

// --- GraphQL ---

type graphqlOrbRegistry struct {
	client *Client
	userId string
}

func (registry graphqlOrbRegistry) request(query string) *Request {
	request := NewRequest(query)
	// An empty token means anonymous, and an empty Authorization header is
	// worse than none at all.
	if registry.client.Token != "" {
		request.SetToken(registry.client.Token)
	}
	if registry.userId != "" {
		request.SetUserId(registry.userId)
	}

	return request
}

// gqlOrb is the orb shape shared by the queries below.
type gqlOrb struct {
	Id       string
	Name     string
	Versions []struct {
		Version string
	}
}

func (orb gqlOrb) toOrbPackage() OrbPackage {
	versions := make([]OrbPackageVersion, 0, len(orb.Versions))
	for _, version := range orb.Versions {
		versions = append(versions, OrbPackageVersion{Version: version.Version})
	}
	SortOrbVersionsDesc(versions)

	return OrbPackage{
		ID:       orb.Id,
		Name:     orb.Name,
		Versions: versions,
	}
}

// FetchOrb asks for the orb's versions as well as its identity, so that one
// query serves both an existence check and a version listing.
func (registry graphqlOrbRegistry) FetchOrb(ctx context.Context, fullName string) (*OrbPackage, error) {
	query := `query($orbName: String!, $versionCount: Int!) {
		orb(name: $orbName) {
			id
			name
			versions(count: $versionCount) {
				version
			}
		}
	}`

	request := registry.request(query)
	request.Var("orbName", fullName)
	request.Var("versionCount", orbVersionCount)

	var response struct {
		// A missing orb comes back as a null member of data rather than as an
		// error, so this has to be a pointer to be distinguishable.
		Orb *gqlOrb
	}
	if err := registry.client.RunWithContext(ctx, request, &response); err != nil {
		return nil, err
	}

	if response.Orb == nil {
		return nil, ErrNotFound
	}

	orb := response.Orb.toOrbPackage()

	return &orb, nil
}

func (registry graphqlOrbRegistry) ResolveVersion(ctx context.Context, ref string) (*ResolvedOrbVersion, error) {
	query := `query($orbVersionRef: String!, $versionCount: Int!) {
		orbVersion(orbVersionRef: $orbVersionRef) {
			id
			version
			source
			orb {
				id
				versions(count: $versionCount) {
					version
				}
			}
		}
	}`

	request := registry.request(query)
	request.Var("orbVersionRef", ref)
	request.Var("versionCount", orbVersionCount)

	var response struct {
		OrbVersion *struct {
			Id      string
			Version string
			Source  string
			Orb     gqlOrb
		}
	}
	if err := registry.client.RunWithContext(ctx, request, &response); err != nil {
		return nil, err
	}

	if response.OrbVersion == nil {
		return nil, ErrNotFound
	}

	orb := response.OrbVersion.Orb.toOrbPackage()

	return &ResolvedOrbVersion{
		ID:           response.OrbVersion.Id,
		Version:      response.OrbVersion.Version,
		Source:       response.OrbVersion.Source,
		OrbPackageID: response.OrbVersion.Orb.Id,
		Versions:     orb.Versions,
	}, nil
}

func (registry graphqlOrbRegistry) FetchNamespace(ctx context.Context, name string) (*Namespace, error) {
	query := `query($name: String!) {
		registryNamespace(name: $name) {
			id
			name
		}
	}`

	request := registry.request(query)
	request.Var("name", name)

	var response struct {
		RegistryNamespace *struct {
			Id   string
			Name string
		}
	}
	if err := registry.client.RunWithContext(ctx, request, &response); err != nil {
		return nil, err
	}

	if response.RegistryNamespace == nil {
		return nil, ErrNotFound
	}

	return &Namespace{
		ID:   response.RegistryNamespace.Id,
		Name: response.RegistryNamespace.Name,
	}, nil
}

func (registry graphqlOrbRegistry) ListNamespaceOrbs(ctx context.Context, name string) ([]OrbPackage, error) {
	query := `query($name: String!, $versionCount: Int!) {
		registryNamespace(name: $name) {
			id
			name
			orbs(first: 1000) {
				totalCount
				pageInfo {
					hasNextPage
				}
				edges {
					cursor
					node {
						id
						name
						versions(count: $versionCount) {
							version
						}
					}
				}
			}
		}
	}`

	request := registry.request(query)
	request.Var("name", name)
	request.Var("versionCount", orbVersionCount)

	var response struct {
		RegistryNamespace *struct {
			Id   string
			Name string
			Orbs struct {
				TotalCount int
				PageInfo   struct {
					HasNextPage bool
				}
				Edges []struct {
					Cursor string
					Node   gqlOrb
				}
			}
		}
	}
	if err := registry.client.RunWithContext(ctx, request, &response); err != nil {
		return nil, err
	}

	if response.RegistryNamespace == nil {
		return nil, ErrNotFound
	}

	namespace := response.RegistryNamespace
	orbs := make([]OrbPackage, 0, len(namespace.Orbs.Edges))
	for _, edge := range namespace.Orbs.Edges {
		orb := edge.Node.toOrbPackage()
		orb.NamespaceID = namespace.Id
		orbs = append(orbs, orb)
	}

	// orbs(first:) has no cursor-following here: the largest namespace on
	// circleci.com holds 79 orbs against a ceiling of 1000. A namespace that
	// outgrows it still autocompletes from the page that was read, but say so
	// on stderr rather than truncating in silence.
	if namespace.Orbs.PageInfo.HasNextPage {
		fmt.Fprintf(os.Stderr,
			"namespace %s holds %d orbs; only the first %d were read\n",
			name, namespace.Orbs.TotalCount, len(namespace.Orbs.Edges),
		)
	}

	return orbs, nil
}
