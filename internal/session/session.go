package session

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/orburl"
)

type Settings struct {
	Api                circleci.Config
	UserIdForTelemetry string
	IsCciExtension     bool

	// SchemaHovers is whether hovering a key shows what the schema says of
	// it. A client that shows the schema's descriptions itself turns it off,
	// so that each isn't shown twice.
	SchemaHovers bool

	// DockerHub configures where the server asks Docker Hub. The zero value is
	// the public Docker Hub; it exists so that a test can point the server at
	// a fake.
	DockerHub dockerhub.Config

	// OrbURLs configures how orbs referenced by URL are fetched. The zero
	// value fetches them from wherever they are; it exists so that a test can
	// point the server at a fake.
	OrbURLs orburl.Config

	// GitHubSignInCommand is a command of the client's that signs the user in
	// to GitHub, and so gives the server a GitHub token. Empty when the
	// client has none.
	GitHubSignInCommand string

	// DefinitionLinks is whether the client reads a definition as a link,
	// which says what text it was found from as well as where it is.
	DefinitionLinks bool
}

// OrbRegistry returns the orb registry for the configured host and token.
func (lsContext *Settings) OrbRegistry() circleci.OrbRegistry {
	return circleci.NewOrbRegistry(
		lsContext.Api.HostUrl,
		lsContext.Api.Token,
		lsContext.UserIdForTelemetry,
		false,
	)
}

// V3Client returns a client for the V3 API on the configured host.
func (lsContext *Settings) V3Client() *circleci.V3Client {
	return circleci.NewV3Client(
		lsContext.Api.HostUrl,
		lsContext.Api.Token,
		lsContext.UserIdForTelemetry,
		false,
	)
}
