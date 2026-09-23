package circleci

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
)

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
