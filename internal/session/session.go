package session

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
)

type Settings struct {
	Api                circleci.Config
	UserIdForTelemetry string
	IsCciExtension     bool

	// DockerHub configures where the server asks Docker Hub. The zero value is
	// the public Docker Hub; it exists so that a test can point the server at
	// a fake.
	DockerHub dockerhub.Config
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
