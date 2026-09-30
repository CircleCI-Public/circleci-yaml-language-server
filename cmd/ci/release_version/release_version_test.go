package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

func TestNextVersion(t *testing.T) {
	testCases := []struct {
		name     string
		current  string
		releases []release
		expected string
	}{
		{
			name:     "First prerelease after a stable release",
			current:  "0.41.1",
			releases: []release{{TagName: "0.41.1", Name: "v0.41.1"}},
			expected: "0.41.2-pre.1",
		},
		{
			name:    "One more than the highest prerelease of the next patch",
			current: "0.41.1",
			releases: []release{
				{TagName: "0.41.2-pre.3", Name: "v0.41.2-pre.3"},
				{TagName: "0.41.2-pre.10", Name: "v0.41.2-pre.10"},
				{TagName: "0.41.1-pre.12", Name: "v0.41.1-pre.12"},
				{TagName: "0.41.1", Name: "v0.41.1"},
			},
			expected: "0.41.2-pre.11",
		},
		{
			name:    "Unreadable tags are skipped",
			current: "0.41.1",
			releases: []release{
				{TagName: "not-a-version"},
				{TagName: "0.41.2-rc.4"},
			},
			expected: "0.41.2-pre.1",
		},
		{
			name:    "The manifest's version once release-please has drafted it",
			current: "0.42.0",
			releases: []release{
				{TagName: "0.42.0", Name: "v0.42.0", Draft: true},
				{TagName: "0.41.2-pre.2", Name: "v0.41.2-pre.2"},
			},
			expected: "0.42.0",
		},
		{
			name:     "A published release of the manifest's version is no draft",
			current:  "0.42.0",
			releases: []release{{TagName: "0.42.0", Name: "v0.42.0"}},
			expected: "0.42.1-pre.1",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nextVersion(tt.current, tt.releases)
			assert.NilError(t, err)
			assert.Check(t, cmp.Equal(got, tt.expected))
		})
	}

	t.Run("A manifest version that isn't semver", func(t *testing.T) {
		_, err := nextVersion("v1", nil)
		assert.Check(t, cmp.ErrorContains(err, "manifest version"))
	})
}

func TestListReleases(t *testing.T) {
	pages := [][]release{
		{{TagName: "0.41.2-pre.1"}, {TagName: "0.41.1"}},
		{{TagName: "0.42.0", Name: "v0.42.0", Draft: true}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases" || r.Header.Get("Authorization") != "Bearer token" {
			http.NotFound(w, r)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		releases := []release{}
		if page >= 1 && page <= len(pages) {
			releases = pages[page-1]
		}
		_ = json.NewEncoder(w).Encode(releases)
	}))
	t.Cleanup(server.Close)

	t.Run("Every page", func(t *testing.T) {
		cl := httpcl.New(httpcl.Config{BaseURL: server.URL, AuthToken: "token"})
		got, err := listReleases(context.Background(), cl, "owner/repo")
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, append(pages[0], pages[1]...)))
	})

	t.Run("An error response", func(t *testing.T) {
		cl := httpcl.New(httpcl.Config{BaseURL: server.URL, AuthToken: "token"})
		_, err := listReleases(context.Background(), cl, "owner/other")
		assert.Check(t, cmp.ErrorContains(err, "404"))
	})
}

func TestManifestVersion(t *testing.T) {
	dir := t.TempDir()

	t.Run("The root package's version", func(t *testing.T) {
		path := filepath.Join(dir, "manifest.json")
		assert.NilError(t, os.WriteFile(path, []byte(`{".": "0.41.1"}`), 0o600))
		got, err := manifestVersion(path)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(got, "0.41.1"))
	})

	t.Run("No root package", func(t *testing.T) {
		path := filepath.Join(dir, "other.json")
		assert.NilError(t, os.WriteFile(path, []byte(`{"editors/vscode": "1.0.0"}`), 0o600))
		_, err := manifestVersion(path)
		assert.Check(t, cmp.ErrorContains(err, "no version"))
	})
}
