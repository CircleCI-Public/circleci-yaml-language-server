package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

// A Client is an HTTP client for our GraphQL endpoint.
//
// The V3 REST API has replaced GraphQL for orb and namespace metadata on
// circleci.com, but CircleCI Server does not serve the V3 orb routes, so this
// client remains as the fallback. See OrbRegistry.
type Client struct {
	Debug      bool
	Endpoint   string
	Host       string
	Token      string
	httpClient *httpcl.Client
}

// NewClient returns a reference to a Client.
func NewClient(host, endpoint, token string, debug bool) *Client {
	var transport http.RoundTripper
	if debug {
		transport = newDebugTransport(http.DefaultTransport)
	}

	return &Client{
		// The address is resolved per request by getServerAddress, so the
		// client carries no base URL, and no token either: GraphQL requests
		// set their own raw Authorization header through Request.SetToken.
		httpClient: NewHTTPClient(httpcl.Config{Transport: transport}),
		Endpoint:   endpoint,
		Host:       host,
		Token:      token,
		Debug:      debug,
	}
}

// NewRequest returns a new GraphQL request.
func NewRequest(query string) *Request {
	request := &Request{
		Query:     query,
		Variables: make(map[string]interface{}),
		Header:    make(map[string][]string),
	}

	return request
}

// Request is a GraphQL request.
type Request struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`

	// Header represent any request headers that will be set
	// when the request is made.
	Header http.Header `json:"-"`
}

// SetToken sets the Authorization header for the request with the given token.
func (request *Request) SetToken(token string) {
	request.Header.Set("Authorization", token)
}

// SetUserId sets the Authorization header for the request with the given user id.
func (request *Request) SetUserId(userId string) {
	request.Header.Set("user_id", userId)
}

// Var sets a variable.
func (request *Request) Var(key string, value interface{}) {
	request.Variables[key] = value
}

// Encode will return a buffer of the JSON encoded request body
func (request *Request) Encode() (bytes.Buffer, error) {
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(request)
	return body, err
}

// Response wraps the result from our GraphQL server response including out-of-band errors and the data itself.
type Response struct {
	Data   interface{}
	Errors ResponseErrorsCollection
}

// ResponseErrorsCollection represents a slice of errors returned by the GraphQL server out-of-band from the actual data.
type ResponseErrorsCollection []ResponseError

// ResponseError represents the key-value pair of data returned by the GraphQL server to represent errors.
type ResponseError struct {
	Message   string
	Locations []struct {
		Line   int
		Column int
	}
	Extensions struct {
		Field         string
		Argument      string
		Value         string
		AllowedValues []string `json:"allowed-values"`
		EnumType      string   `json:"enum-type"`
	}
}

// Error turns a ResponseErrorsCollection into an acceptable error string that can be printed to the user.
func (errs ResponseErrorsCollection) Error() string {
	messages := []string{}

	for i := range errs {
		messages = append(messages, errs[i].Message)
	}

	return strings.Join(messages, "\n")
}

// getServerAddress returns the full address to the server
func getServerAddress(host, endpoint string) (string, error) {
	// 1. Parse the endpoint
	e, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parsing endpoint '%s': %w", endpoint, err)
	}

	// 2. Parse the host
	h, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("parsing host '%s': %w", host, err)
	}
	if !h.IsAbs() {
		return h.String(), fmt.Errorf("Host (%s) must be absolute URL, including scheme", host)
	}

	// 3. Resolve the two URLs using host as the base
	// We use ResolveReference which has specific behavior we can rely for
	// older configurations which included the absolute path for the endpoint flag.
	//
	// https://golang.org/pkg/net/url/#URL.ResolveReference
	//
	// Specifically this function always returns the reference (endpoint) if provided an absolute URL.
	// This way we can safely introduce --host and merge the two.
	return h.ResolveReference(e).String(), err
}

// Run sends an HTTP request to the GraphQL server and deserializes the response or returns an error.
func (cl *Client) Run(request *Request, resp interface{}) error {
	return cl.RunWithContext(context.Background(), request, resp)
}

// RunWithContext sends an HTTP request to the GraphQL server and deserializes
// the response or returns an error.
func (cl *Client) RunWithContext(ctx context.Context, request *Request, resp interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	address, err := getServerAddress(cl.Host, cl.Endpoint)
	if err != nil {
		return err
	}

	if cl.Debug {
		l := log.New(os.Stderr, "", 0)
		l.Printf(">> variables: %v", request.Variables)
		l.Printf(">> query: %s", request.Query)
	}

	wrappedResponse := &Response{
		Data: resp,
	}

	opts := []func(*httpcl.Request){
		httpcl.Body(request),
		httpcl.JSONDecoder(&wrappedResponse),
	}
	for key, values := range request.Header {
		for _, value := range values {
			opts = append(opts, httpcl.Header(key, value))
		}
	}

	status, err := cl.httpClient.Call(ctx, httpcl.NewRequest(http.MethodPost, address, opts...))
	var httpErr *httpcl.HTTPError
	if errors.As(err, &httpErr) || (err == nil && status != http.StatusOK) {
		return fmt.Errorf("failure calling GraphQL API: %d %s", status, http.StatusText(status))
	}
	if err != nil {
		return err
	}

	if len(wrappedResponse.Errors) > 0 {
		return wrappedResponse.Errors
	}

	return nil
}
