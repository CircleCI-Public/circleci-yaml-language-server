package dockerhub

import (
	"net/http"
	"net/url"
)

type DockerHubAPI interface {
	DoesImageExist(namespace, image string) bool
	GetImageTags(namespace, image string) ([]string, error)
	ImageHasTag(namespace, image, tag string) bool
}

// Config configures an API. It exists so that a test can point this package at
// a fake Docker Hub; production has no reason to set any of it.
//
// It follows the shape of the CLI's internal/httpcl Config — a base URL and an
// injectable client — so the two read alike.
type Config struct {
	// BaseURL is the API root including its version path, for example
	// "https://hub.docker.com/v2". Empty means the public Docker Hub.
	BaseURL string
	// HTTPClient issues every request. Nil means http.DefaultClient, which is
	// what this package has always used.
	HTTPClient *http.Client
}

type dockerHubAPI struct {
	baseURL    url.URL
	httpClient *http.Client

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
		httpClient: cfg.HTTPClient,
		namespaces: map[string]*HubNamespace{},
	}

	if cfg.BaseURL != "" {
		if parsed, err := url.Parse(cfg.BaseURL); err == nil {
			api.baseURL = *parsed
		}
	}

	if api.httpClient == nil {
		api.httpClient = http.DefaultClient
	}

	api.namespaces["library"] = &HubNamespace{namespace: "library", api: api}

	return api
}
