package client

import (
	"net/http"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/version"
)

// New returns an httpcl client carrying what every request the
// language server makes has in common: its User-Agent, and one shared
// transport so that clients made per call still reuse connections.
//
// Retries are off, as they were before these clients moved onto httpcl: an
// editor is waiting on the answer, and retrying is a change in behaviour to
// make on purpose rather than as part of a refactor.
func New(cfg httpcl.Config) *httpcl.Client {
	cfg.UserAgent = version.UserAgent
	cfg.DisableRetries = true
	if cfg.Transport == nil {
		cfg.Transport = http.DefaultTransport
	}

	return httpcl.New(cfg)
}
