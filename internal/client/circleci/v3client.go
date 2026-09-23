package circleci

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
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

	httpClient *httpcl.Client
}

// NewV3Client returns a client for the V3 API on the given host.
func NewV3Client(host, token, userId string, debug bool) *V3Client {
	var transport http.RoundTripper
	if debug {
		transport = newDebugTransport(http.DefaultTransport)
	}

	return &V3Client{
		Host:   host,
		Token:  token,
		UserId: userId,
		Debug:  debug,
		// The host is joined onto each route rather than set as the base URL,
		// so that a missing or relative host is reported by the request that
		// needs it, as it was before this client moved onto httpcl.
		//
		// An empty token means "anonymous": httpcl sends no Authorization
		// header at all, which is served for public orbs, where "Bearer " with
		// nothing after it would be rejected.
		httpClient: client.New(httpcl.Config{
			AuthToken: token,
			Transport: transport,
		}),
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

	// response is the error httpcl reported, kept so that
	// httpcl.HasStatusCode works on an APIError too.
	response *httpcl.HTTPError
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

func (err *APIError) Unwrap() error {
	if err.response == nil {
		return nil
	}

	return err.response
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
	address, err := cl.address(path)
	if err != nil {
		return nil, err
	}

	opts := []func(*httpcl.Request){}
	if cl.UserId != "" {
		opts = append(opts, httpcl.Header("user_id", cl.UserId))
	}
	for key, values := range query {
		for _, value := range values {
			// url.Values.Encode percent-encodes the brackets in filter[name]
			// and page[cursor], which the API accepts.
			opts = append(opts, httpcl.QueryParam(key, value))
		}
	}

	var body []byte
	opts = append(opts, httpcl.BytesDecoder(&body))

	_, err = cl.httpClient.Call(ctx, httpcl.NewRequest(http.MethodGet, address, opts...))
	if httpErr, ok := errors.AsType[*httpcl.HTTPError](err); ok {
		return nil, parseAPIError(httpErr)
	}
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (cl *V3Client) address(path string) (string, error) {
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

	return strings.TrimSuffix(host.String(), "/") + "/api/v3/" + strings.TrimPrefix(path, "/"), nil
}

func parseAPIError(httpErr *httpcl.HTTPError) error {
	apiErr := &APIError{Status: httpErr.StatusCode, response: httpErr}

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

	if err := json.Unmarshal(bytes.TrimSpace(httpErr.Body), &envelope); err == nil {
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
