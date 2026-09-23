// Package workspace builds the project an acceptance test opens a config from.
//
// A config on its own is not enough: the server reads the project slug from
// the git remote of the directory holding .circleci (utils.GetProjectSlug), and
// the project, its organization, its contexts and its environment variables all
// key off that slug. So a workspace is a directory, a config, and a repository
// with a remote.
package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// DefaultRemote is the origin remote a workspace is created with. It resolves
// to the project slug "gh/acme/rocket", as it would with a ".git" suffix.
const DefaultRemote = "https://github.com/acme/rocket"

// DefaultSlug is the project slug DefaultRemote resolves to, which is what a
// fake has to know about for the server to resolve the project.
const DefaultSlug = "gh/acme/rocket"

// Workspace is a project directory holding a CircleCI config.
type Workspace struct {
	Root       string
	ConfigPath string
}

// New writes config to .circleci/config.yml in a new directory, and makes that
// directory a repository whose origin remote resolves to DefaultSlug.
func New(t *testing.T, config string) *Workspace {
	return NewWithRemote(t, config, DefaultRemote)
}

// NewWithRemote is New with a remote of its own, for a case about how a slug is
// resolved — or not resolved.
func NewWithRemote(t *testing.T, config, remote string) *Workspace {
	t.Helper()

	root := t.TempDir()

	workspace := &Workspace{
		Root:       root,
		ConfigPath: filepath.Join(root, ".circleci", "config.yml"),
	}

	if err := os.MkdirAll(filepath.Dir(workspace.ConfigPath), 0o755); err != nil {
		t.Fatalf("creating .circleci: %v", err)
	}

	workspace.Write(t, config)

	repository, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatalf("initialising a repository in %s: %v", root, err)
	}

	if remote != "" {
		_, err = repository.CreateRemote(&gitconfig.RemoteConfig{
			Name: "origin",
			URLs: []string{remote},
		})
		if err != nil {
			t.Fatalf("adding the origin remote: %v", err)
		}
	}

	return workspace
}

// Write replaces the config on disk. The server reads a document's content
// from the protocol rather than from the file, so this is for the cases that
// care what is on disk — an orb resolved from a local path, for instance.
func (w *Workspace) Write(t *testing.T, config string) {
	t.Helper()

	if err := os.WriteFile(w.ConfigPath, []byte(config), 0o600); err != nil {
		t.Fatalf("writing %s: %v", w.ConfigPath, err)
	}
}

// URI is the config file, as a client would name it.
func (w *Workspace) URI() protocol.URI {
	return uri.File(w.ConfigPath)
}

// RootURI is the project directory, for the initialize handshake.
func (w *Workspace) RootURI() protocol.URI {
	return uri.File(w.Root)
}
