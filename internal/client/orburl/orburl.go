// Package orburl fetches the source of an orb that a config references by URL,
// such as `https://raw.githubusercontent.com/acme/orbs/main/go.yml`.
//
// The compiler fetches these itself (orb-service's url_orb_resolver.clj), but
// only from the https prefixes in an organization's URL orb allow-list, with
// the credential each prefix is given there. That list isn't something the
// server can read, so it fetches a URL as it stands, without credentials.
package orburl

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

// ErrNotHTTPS reports an orb URL that isn't https. The compiler only fetches
// from https prefixes, so neither does the server.
var ErrNotHTTPS = errors.New("orbs are only fetched over https")

// Config configures Fetch. It exists so that a test can point it at a fake
// host; production has no reason to set any of it.
type Config struct {
	// Transport carries every request. Nil means http.DefaultTransport.
	Transport http.RoundTripper
}

// Fetch returns the source at an orb URL. found is false when the host
// answers that there is nothing there it will serve: 401, 403, 404 or 410.
// GitHub answers 404 for a file in a private repository. Anything else that
// goes wrong is an error, because the host didn't say.
func Fetch(ctx context.Context, cfg Config, address string) (source string, found bool, err error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return "", false, err
	}
	if parsed.Scheme != "https" {
		return "", false, ErrNotHTTPS
	}

	cl := client.New(httpcl.Config{BaseURL: address, Transport: cfg.Transport})
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
