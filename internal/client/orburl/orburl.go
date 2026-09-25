// Package orburl fetches the source of an orb that a config references by URL,
// such as `https://raw.githubusercontent.com/acme/orbs/main/go.yml`.
//
// The compiler fetches these itself (orb-service's url_orb_resolver.clj), but
// only from the https prefixes in an organization's URL orb allow-list, with
// the credential each prefix is given there. That list isn't something the
// server can read, so it fetches a URL as it stands, and a file on GitHub that
// isn't public with the user's GitHub token, when it has one.
package orburl

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

// ErrNotHTTPS reports an orb URL that isn't https. The compiler only fetches
// from https prefixes, so neither does the server.
var ErrNotHTTPS = errors.New("orbs are only fetched over https")

// gitHubHosts are the hosts a GitHub token is sent to. It is never sent
// anywhere else: a config can name any URL.
var gitHubHosts = []string{"raw.githubusercontent.com", "github.com"}

// Config configures Fetch.
type Config struct {
	// GitHubToken, when set, is used to fetch a file on GitHub that can't be
	// fetched without it, as for a private repository.
	GitHubToken string
	// Transport carries every request. Nil means http.DefaultTransport. It
	// exists so that a test can point Fetch at a fake host.
	Transport http.RoundTripper
}

// NeedsGitHubToken reports whether an orb URL that couldn't be fetched might
// be with a GitHub token, because it is on GitHub and there is none.
func NeedsGitHubToken(cfg Config, address string) bool {
	parsed, err := url.Parse(address)
	return err == nil && cfg.GitHubToken == "" && slices.Contains(gitHubHosts, parsed.Hostname())
}

// Fetch returns the source at an orb URL. found is false when the host
// answers that there is nothing there it will serve: 401, 403, 404 or 410.
// GitHub answers 404 for a file in a private repository. Anything else that
// goes wrong is an error, because the host didn't say.
//
// A URL is fetched without credentials first, and only then, if it is on
// GitHub, with the GitHub token. The compiler does the same with its
// allow-list, because GitHub answers 404 for a public file when a token is
// sent that can't see it.
func Fetch(ctx context.Context, cfg Config, address string) (source string, found bool, err error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return "", false, err
	}
	if parsed.Scheme != "https" {
		return "", false, ErrNotHTTPS
	}

	source, found, err = get(ctx, cfg, address, "")
	if err != nil || found || cfg.GitHubToken == "" || !slices.Contains(gitHubHosts, parsed.Hostname()) {
		return source, found, err
	}

	return get(ctx, cfg, address, cfg.GitHubToken)
}

// get fetches an orb URL, sending token, when it isn't empty, as a GitHub
// token. The client drops the header on a redirect to another host.
func get(ctx context.Context, cfg Config, address, token string) (source string, found bool, err error) {
	httpConfig := httpcl.Config{BaseURL: address, Transport: cfg.Transport}
	if token != "" {
		httpConfig.AuthHeader = "Authorization"
		httpConfig.AuthToken = "token " + token
	}

	cl := client.New(httpConfig)
	_, err = cl.Call(ctx, httpcl.NewRequest(http.MethodGet, "", httpcl.StringDecoder(&source)))

	switch {
	case err == nil:
		return source, true, nil
	case httpcl.HasStatusCode(err, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusGone):
		return "", false, nil
	default:
		return "", false, err
	}
}
