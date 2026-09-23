package dockerhub

import (
	"context"
	"net/http"
	"net/url"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

type DockerHubAPI interface {
	DoesImageExist(namespace, image string) bool
	GetImageTags(namespace, image string) ([]string, error)
	ImageHasTag(namespace, image, tag string) bool
}

// Config configures an API. It exists so that a test can point this package at
// a fake Docker Hub; production has no reason to set any of it.
//
// Its fields are the subset of internal/httpcl's Config — a base URL and an
// injectable transport — that a test needs, so the two read alike.
type Config struct {
	// BaseURL is the API root including its version path, for example
	// "https://hub.docker.com/v2". Empty means the public Docker Hub.
	BaseURL string
	// Transport carries every request. Nil means http.DefaultTransport.
	Transport http.RoundTripper
}

type dockerHubAPI struct {
	baseURL url.URL
	// client has no base URL of its own: Docker Hub hands back each next page
	// as an absolute URL, so every request is made to one.
	client *httpcl.Client

	// namespaces caches the repositories read for a namespace. It used to be a
	// package-level map, which every test in the binary shared and none could
	// point at a fake.
	namespaces map[string]*HubNamespace
}

// defaultAPI backs the package-level Search and SearchTags, which the
// completion path calls without an API of its own. Tests build their own with
// NewAPIWithConfig rather than reaching for this one.
var defaultAPI = newAPI(Config{})

func NewAPI() DockerHubAPI {
	return newAPI(Config{})
}

// NewAPIWithConfig returns an API configured by cfg, defaulting anything cfg
// leaves empty to what NewAPI uses.
func NewAPIWithConfig(cfg Config) DockerHubAPI {
	return newAPI(cfg)
}

// newAPI is NewAPIWithConfig with the concrete type, which this package's own
// functions need and callers do not.
func newAPI(cfg Config) *dockerHubAPI {
	api := &dockerHubAPI{
		baseURL:    baseURL,
		client:     utils.NewHTTPClient(httpcl.Config{Transport: cfg.Transport}),
		namespaces: map[string]*HubNamespace{},
	}

	if cfg.BaseURL != "" {
		if parsed, err := url.Parse(cfg.BaseURL); err == nil {
			api.baseURL = *parsed
		}
	}

	api.namespaces["library"] = &HubNamespace{namespace: "library", api: api}

	return api
}

// get requests an absolute URL, decoding a 2xx body into out when out is not
// nil. Any other status is an *httpcl.HTTPError.
func (me *dockerHubAPI) get(address string, out any) (int, error) {
	opts := []func(*httpcl.Request){}
	if out != nil {
		opts = append(opts, httpcl.JSONDecoder(out))
	}

	return me.client.Call(context.Background(), httpcl.NewRequest(http.MethodGet, address, opts...))
}
