package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// V3Client is an HTTP client for the CircleCI V3 REST API
// (https://circleci.com/docs/api/v3).
//
// Every route is served under /api/v3 on the configured host. Single entities
// come back as {"data": {...}} and collections as
// {"data": [...], "page": {"next": ..., "prev": ...}}, so this client unwraps
// the envelope and leaves callers to describe only the payload.
type V3Client struct {
	// Host is the scheme and authority of the CircleCI instance, e.g.
	// "https://circleci.com".
	Host string
	// Token is a personal API token. When empty, no Authorization header is
	// sent: the orb and namespace routes answer unauthenticated requests for
	// public orbs, which is how the language server serves users who have not
	// logged in.
	Token string
	// UserId is sent as the user_id header for telemetry, mirroring what the
	// GraphQL requests used to do.
	UserId string
	Debug  bool

	httpClient *http.Client
}

// NewV3Client returns a client for the V3 API on the given host.
func NewV3Client(host, token, userId string, debug bool) *V3Client {
	return &V3Client{
		Host:       host,
		Token:      token,
		UserId:     userId,
		Debug:      debug,
		httpClient: GetHTTPClient(),
	}
}

// NewV3ClientFromContext returns a client configured from the language server
// context.
func NewV3ClientFromContext(lsContext *LsContext) *V3Client {
	return NewV3Client(
		lsContext.Api.HostUrl,
		lsContext.Api.Token,
		lsContext.UserIdForTelemetry,
		false,
	)
}

// GetHTTPClient returns the HTTP client both API clients use.
func GetHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          10,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}
}

// ErrNotFound reports that the API answered 404. Callers that turn "absent"
// into a diagnostic need to tell it apart from a transport failure, where
// staying silent is the right behaviour.
var ErrNotFound = errors.New("not found")

// ErrHostNotDefined reports that no CircleCI host has been configured.
var ErrHostNotDefined = errors.New("host URL not defined")

// APIError is a non-2xx V3 response. The V3 error envelope is a single
// {"error": {...}} object, never a list.
type APIError struct {
	Status int
	Type   string
	ID     string
	Title  string
	Detail string
}

func (err *APIError) Error() string {
	parts := []string{}
	if err.Title != "" {
		parts = append(parts, err.Title)
	}
	if err.Detail != "" {
		parts = append(parts, err.Detail)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("CircleCI API returned %d", err.Status)
	}

	return fmt.Sprintf("CircleCI API returned %d: %s", err.Status, strings.Join(parts, ": "))
}

// Is lets errors.Is(err, ErrNotFound) match a 404 APIError, so callers can
// branch on absence without unwrapping the concrete type.
func (err *APIError) Is(target error) bool {
	return target == ErrNotFound && err.Status == http.StatusNotFound
}

// page is the cursor pagination envelope shared by every V3 collection.
type page struct {
	Next *string `json:"next"`
	Prev *string `json:"prev"`
}

// Get requests a single entity and decodes the "data" member into out.
func (cl *V3Client) Get(ctx context.Context, path string, query url.Values, out interface{}) error {
	body, err := cl.get(ctx, path, query)
	if err != nil {
		return err
	}

	envelope := struct {
		Data interface{} `json:"data"`
	}{Data: out}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decoding response from %s: %w", path, err)
	}

	return nil
}

// GetText requests a text/plain body, such as an orb version's YAML source.
func (cl *V3Client) GetText(ctx context.Context, path string, query url.Values) (string, error) {
	body, err := cl.get(ctx, path, query)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// GetPaged requests a collection and follows page.next until it is null,
// returning every item. Cursors are opaque and are echoed back verbatim.
//
// This is a function rather than a method because Go does not allow methods to
// introduce their own type parameters.
func GetPaged[T any](ctx context.Context, cl *V3Client, path string, query url.Values) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}

	// Copy so that following cursors does not mutate the caller's values.
	next := url.Values{}
	for key, values := range query {
		next[key] = append([]string(nil), values...)
	}

	items := []T{}

	for {
		body, err := cl.get(ctx, path, next)
		if err != nil {
			return nil, err
		}

		var envelope struct {
			Data []T   `json:"data"`
			Page *page `json:"page"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, fmt.Errorf("decoding response from %s: %w", path, err)
		}

		items = append(items, envelope.Data...)

		if envelope.Page == nil || envelope.Page.Next == nil || *envelope.Page.Next == "" {
			return items, nil
		}

		next.Set("page[cursor]", *envelope.Page.Next)
	}
}

func (cl *V3Client) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	address, err := cl.address(path, query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	// An empty token means "anonymous": sending "Bearer " with nothing after it
	// is rejected, while sending no header at all is served for public orbs.
	if cl.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cl.Token)
	}
	if cl.UserId != "" {
		req.Header.Set("user_id", cl.UserId)
	}

	logger := log.New(os.Stderr, "", 0)
	if cl.Debug {
		logger.Printf(">> GET %s", address)
	}

	res, err := cl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			logger.Printf("%s", closeErr.Error())
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", path, err)
	}

	if cl.Debug {
		logger.Printf("<< request id: %s", res.Header.Get("X-Request-Id"))
		logger.Printf("<< %s: %s", res.Status, string(body))
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, parseAPIError(res.StatusCode, body)
	}

	return body, nil
}

func (cl *V3Client) address(path string, query url.Values) (string, error) {
	if cl.Host == "" {
		return "", ErrHostNotDefined
	}

	host, err := url.Parse(cl.Host)
	if err != nil {
		return "", fmt.Errorf("parsing host %q: %w", cl.Host, err)
	}
	if !host.IsAbs() {
		return "", fmt.Errorf("host (%s) must be an absolute URL, including scheme", cl.Host)
	}

	address := strings.TrimSuffix(host.String(), "/") + "/api/v3/" + strings.TrimPrefix(path, "/")
	if len(query) > 0 {
		// url.Values.Encode percent-encodes the brackets in filter[name] and
		// page[cursor], which the API accepts.
		address += "?" + query.Encode()
	}

	return address, nil
}

func parseAPIError(status int, body []byte) error {
	apiErr := &APIError{Status: status}

	var envelope struct {
		Error struct {
			Type   string `json:"type"`
			ID     string `json:"id"`
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"error"`
		// Some routes in front of the V3 handlers (notably token validation)
		// answer with a bare {"message": ...} instead of the V3 envelope.
		Message string `json:"message"`
	}

	if err := json.Unmarshal(bytes.TrimSpace(body), &envelope); err == nil {
		apiErr.Type = envelope.Error.Type
		apiErr.ID = envelope.Error.ID
		apiErr.Title = envelope.Error.Title
		apiErr.Detail = envelope.Error.Detail
		if apiErr.Title == "" && envelope.Message != "" {
			apiErr.Title = envelope.Message
		}
	}

	return apiErr
}
