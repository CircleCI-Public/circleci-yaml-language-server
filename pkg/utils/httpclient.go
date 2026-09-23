package utils

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

// NewHTTPClient returns an httpcl client carrying what every request the
// language server makes has in common: its User-Agent, and one shared
// transport so that clients made per call still reuse connections.
//
// Retries are off, as they were before these clients moved onto httpcl: an
// editor is waiting on the answer, and retrying is a change in behaviour to
// make on purpose rather than as part of a refactor.
func NewHTTPClient(cfg httpcl.Config) *httpcl.Client {
	cfg.UserAgent = UserAgent
	cfg.DisableRetries = true
	if cfg.Transport == nil {
		cfg.Transport = http.DefaultTransport
	}

	return httpcl.New(cfg)
}

// newV2Client returns a client for the CircleCI V2 REST API on the configured
// host. V2 authenticates with a Circle-Token header rather than a bearer token.
func newV2Client(apiContext ApiContext) *httpcl.Client {
	return NewHTTPClient(httpcl.Config{
		BaseURL:    apiContext.HostUrl + "/api/v2",
		AuthToken:  apiContext.Token,
		AuthHeader: "Circle-Token",
	})
}

// debugTransport logs each response body, which is what the Debug flag on the
// V3 and GraphQL clients adds. httpcl already logs every request's method,
// address, status and duration at debug level, so this logs only the rest.
type debugTransport struct {
	next http.RoundTripper
}

func newDebugTransport(next http.RoundTripper) *debugTransport {
	return &debugTransport{next: next}
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := d.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, err
	}
	// The body can only be read once, so hand the caller a fresh reader over
	// what was just logged.
	res.Body = io.NopCloser(bytes.NewReader(body))

	slog.DebugContext(req.Context(), "response body",
		"url.full", req.URL.String(),
		"http.response.status_code", res.StatusCode,
		"request_id", res.Header.Get("X-Request-Id"),
		"body", string(body),
	)

	return res, nil
}
