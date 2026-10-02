package methods

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bep/debounce"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// Methods is the language server: protocol.ServerHandler decodes each request
// and calls the method for it. A request the server does not handle is left to
// UnimplementedServer, which answers method-not-found, or ignores it if it is
// a notification.
//
// Methods run concurrently with one another and with the validation they start
// in the background, so what they share is safe for that: the cache locks, and
// the settings are replaced rather than changed.
type Methods struct {
	protocol.UnimplementedServer

	// Ctx is the session's. A request's context ends once it is answered,
	// so what runs on after that, such as validation, runs under this one.
	Ctx            context.Context
	Client         protocol.Client
	Cache          *cache.Cache
	SchemaLocation string

	settings        atomic.Pointer[session.Settings]
	settingsUpdates sync.Mutex

	// debounceEdit and debounceRevalidation each run only the last of a burst
	// of calls, once the burst is over. How long an edit waits is the
	// client's to choose (see Initialize), so it is set before any document
	// can change.
	editDebounce         time.Duration
	debounceEdit         func(func())
	debounceRevalidation func(func())

	// workspaceFolders are the client's, as it gave them when initializing,
	// which comes before any request that reads them.
	workspaceFolders []uri.URI
	// inlayHintRefresh is whether the client can be asked to ask for inlay
	// hints again. It is set when initializing, like workspaceFolders.
	inlayHintRefresh bool

	exited   chan struct{}
	exitOnce sync.Once
}

var _ protocol.Server = (*Methods)(nil)

// defaultEditDebounce is how long a document is left after a change before it
// is checked, unless the client chooses otherwise. It is sized for typing, so
// that a burst of keystrokes is checked once.
const defaultEditDebounce = time.Second

func New(ctx context.Context, client protocol.Client, cache *cache.Cache, settings session.Settings, schemaLocation string) *Methods {
	methods := &Methods{
		Ctx:                  ctx,
		Client:               client,
		Cache:                cache,
		SchemaLocation:       schemaLocation,
		editDebounce:         defaultEditDebounce,
		debounceEdit:         debounce.New(defaultEditDebounce),
		debounceRevalidation: debounce.New(1000 * time.Millisecond),
		exited:               make(chan struct{}),
	}
	methods.settings.Store(&settings)

	return methods
}

// Settings are the session's settings as they are now. They are never changed
// once returned, so a caller reading several of them sees one consistent set.
func (methods *Methods) Settings() *session.Settings {
	return methods.settings.Load()
}

// updateSettings replaces the settings with a copy that change has been
// applied to.
func (methods *Methods) updateSettings(change func(*session.Settings)) {
	methods.settingsUpdates.Lock()
	defer methods.settingsUpdates.Unlock()

	updated := *methods.settings.Load()
	change(&updated)
	methods.settings.Store(&updated)
}

func (methods *Methods) Shutdown(context.Context) error {
	methods.Cache.Close()
	return nil
}

// Exit ends the session. Ending the process is left to whoever runs the
// session, so that it can clean up after it first.
func (methods *Methods) Exit(context.Context) error {
	methods.exitOnce.Do(func() { close(methods.exited) })
	return nil
}

// Exited is closed once the client has sent exit.
func (methods *Methods) Exited() <-chan struct{} {
	return methods.exited
}
