package methods

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bep/debounce"
	"go.lsp.dev/protocol"

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

	Ctx            context.Context
	Client         protocol.Client
	Cache          *cache.Cache
	SchemaLocation string

	settings        atomic.Pointer[session.Settings]
	settingsUpdates sync.Mutex

	// debounceEdit and debounceRevalidation each run only the last of a burst
	// of calls, once the burst is over.
	debounceEdit         func(func())
	debounceRevalidation func(func())
}

var _ protocol.Server = (*Methods)(nil)

func New(ctx context.Context, client protocol.Client, cache *cache.Cache, settings session.Settings, schemaLocation string) *Methods {
	methods := &Methods{
		Ctx:                  ctx,
		Client:               client,
		Cache:                cache,
		SchemaLocation:       schemaLocation,
		debounceEdit:         debounce.New(1000 * time.Millisecond),
		debounceRevalidation: debounce.New(1000 * time.Millisecond),
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
	return nil
}

func (methods *Methods) Exit(context.Context) error {
	os.Exit(0)
	return nil
}
