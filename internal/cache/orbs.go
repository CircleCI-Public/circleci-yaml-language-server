package cache

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

// Orbs remembers each remote orb resolved, by the reference that names it
// ("circleci/go@1.7.1"), for memo.FoundLifetime. A reference to a partial
// version or to volatile resolves to whatever is newest, so even a pinned
// reference is asked about again after that long rather than kept for good.
//
// An orb is shared between callers, so it is never changed once cached: it is
// replaced.
type Orbs struct {
	orbs *memo.Memo[*ast.OrbInfo]
}

// Load returns the orb a reference names, calling fetch only when none is
// remembered. An error from fetch is returned but not remembered.
func (c *Orbs) Load(orbID string, fetch func() (*ast.OrbInfo, error)) (*ast.OrbInfo, error) {
	return c.orbs.Get(orbID, fetch)
}

func (c *Orbs) HasOrb(orbID string) bool {
	_, ok := c.orbs.Peek(orbID)
	return ok
}

func (c *Orbs) SetOrb(orb *ast.OrbInfo, orbID string) ast.OrbInfo {
	c.orbs.Put(orbID, orb)
	return *orb
}

// UpdateOrbParsedAttributes replaces what is known of an orb's contents, for
// when its source is edited in the editor. An orb no longer remembered is left
// alone.
func (c *Orbs) UpdateOrbParsedAttributes(orbID string, parsedOrbAttributes ast.OrbParsedAttributes) {
	orb, ok := c.orbs.Peek(orbID)
	if !ok {
		return
	}
	// The orb is replaced rather than changed: whoever got it from GetOrb may
	// still be reading it.
	updated := *orb
	updated.OrbParsedAttributes = parsedOrbAttributes
	c.orbs.Put(orbID, &updated)
}

func (c *Orbs) GetOrb(orbID string) *ast.OrbInfo {
	orb, _ := c.orbs.Peek(orbID)
	return orb
}

// orbSources is the directory remote orb sources are written to. They are
// kept in memory, and written out only so that go-to-definition has a file to
// open.
//
// The directory belongs to one run of the server: it is made on first use,
// removed by Close and by ClearHostData, and never read back. Sources used to
// be kept in a cache directory shared by every run, which served a source
// long after the reference it was fetched for had moved on to a newer one.
type orbSources struct {
	mutex sync.Mutex
	dir   string
}

// WriteOrbSource writes the source of the orb a reference names, replacing
// what was written for it before, and returns the path it wrote to.
func (c *Cache) WriteOrbSource(orbID, source string) (string, error) {
	s := &c.orbSources
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.dir == "" {
		dir, err := os.MkdirTemp("", "circleci-yaml-language-server-orbs-")
		if err != nil {
			return "", err
		}
		// The temporary directory may be reached through a symbolic link, as
		// it is on macOS, and the editor hands back the path it was given.
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			dir = resolved
		}
		s.dir = dir
	}

	path := filepath.Join(s.dir, filepath.FromSlash(orbID)+".yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	return path, os.WriteFile(path, []byte(source), 0o600)
}

// OrbIDOfSource returns the reference of the orb whose source was written to
// a path, and whether the path is one WriteOrbSource wrote to at all.
func (c *Cache) OrbIDOfSource(path string) (string, bool) {
	s := &c.orbSources
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.dir == "" {
		return "", false
	}

	rel, err := filepath.Rel(s.dir, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || !strings.HasSuffix(rel, ".yml") {
		return "", false
	}

	return strings.TrimSuffix(filepath.ToSlash(rel), ".yml"), true
}

// remove removes the directory and everything written to it.
func (s *orbSources) remove() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.dir == "" {
		return
	}
	_ = os.RemoveAll(s.dir)
	s.dir = ""
}
