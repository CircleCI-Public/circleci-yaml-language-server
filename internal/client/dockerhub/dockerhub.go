package dockerhub

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

// API is what validation asks Docker Hub.
//
// DoesImageExist and ImageHasTag answer false only when Docker Hub says the
// image or tag is not there. When it cannot say — a rate limit, an outage, a
// host that is not answering — they return an error instead, which a caller
// must not turn into "absent".
type API interface {
	DoesImageExist(namespace, image string) (bool, error)
	GetImageTags(namespace, image string) ([]string, error)
	ImageHasTag(namespace, image, tag string) (bool, error)
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
	// point at a fake. Completion requests are served on goroutines of their
	// own, all sharing one API, so it is only reached through namespace and
	// knownNamespace, which hold namespacesMutex.
	namespacesMutex sync.Mutex
	namespaces      map[string]*HubNamespace
}

// defaultAPI backs the package-level Search and SearchTags for the zero
// Config, which is what the server runs with, so that every completion
// request shares its cache of namespaces.
var defaultAPI = newAPI(Config{})

// searchAPI is the API the package-level Search and SearchTags read through
// for cfg: the shared default for the zero Config, and one of its own for any
// other, such as a test's fake.
func searchAPI(cfg Config) *dockerHubAPI {
	if cfg.BaseURL == "" && cfg.Transport == nil {
		return defaultAPI
	}
	return newAPI(cfg)
}

func NewAPI() API {
	return newAPI(Config{})
}

// NewAPIWithConfig returns an API configured by cfg, defaulting anything cfg
// leaves empty to what NewAPI uses.
func NewAPIWithConfig(cfg Config) API {
	return newAPI(cfg)
}

// newAPI is NewAPIWithConfig with the concrete type, which this package's own
// functions need and callers do not.
func newAPI(cfg Config) *dockerHubAPI {
	api := &dockerHubAPI{
		baseURL:    baseURL,
		client:     client.New(httpcl.Config{Transport: cfg.Transport}),
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

// namespace returns the cache for a namespace, creating it on first use.
func (me *dockerHubAPI) namespace(name string) *HubNamespace {
	me.namespacesMutex.Lock()
	defer me.namespacesMutex.Unlock()

	ns := me.namespaces[name]
	if ns == nil {
		ns = &HubNamespace{api: me, namespace: name}
		me.namespaces[name] = ns
	}

	return ns
}

// knownNamespace returns the cache for a namespace, or nil when nothing has
// asked for it yet.
func (me *dockerHubAPI) knownNamespace(name string) *HubNamespace {
	me.namespacesMutex.Lock()
	defer me.namespacesMutex.Unlock()

	return me.namespaces[name]
}

// exists asks whether an absolute URL names something Docker Hub has: a 2xx
// is yes, a 404 is no, and anything else is an error, because Docker Hub did
// not say.
func (me *dockerHubAPI) exists(address string) (bool, error) {
	_, err := me.get(address, nil)
	switch {
	case err == nil:
		return true, nil
	case httpcl.HasStatusCode(err, http.StatusNotFound):
		return false, nil
	default:
		return false, err
	}
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
