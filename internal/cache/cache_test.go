package cache

import (
	"os"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

// withContexts is a cache that has listed contexts of the given names for an
// organization.
func withContexts(orgID string, names ...string) *Cache {
	contexts := make([]*Context, 0, len(names))
	for _, name := range names {
		contexts = append(contexts, &Context{Name: name})
	}

	cache := New()
	cache.ContextCache.orgs.Put(orgID, indexContexts(contexts))
	return cache
}

// resolvedName is the name of the context a reference resolves to, or "" for
// none.
func resolvedName(cache *Cache, orgID, orgSlug, reference string) string {
	ctx := cache.ContextCache.ResolveWorkflowContext(orgID, orgSlug, reference)
	if ctx == nil {
		return ""
	}
	return ctx.Name
}

func TestContextCache_ResolveWorkflowContext(t *testing.T) {
	orgID := "org-uuid"
	cache := withContexts(orgID, "my-org/deploy", "my-org/staging")

	tests := []struct {
		name                string
		workflowContextName string
		organizationSlug    string
		wantName            string
	}{
		{
			name:                "exact match",
			workflowContextName: "my-org/deploy",
			wantName:            "my-org/deploy",
		},
		{
			name:                "suffix match",
			workflowContextName: "deploy",
			wantName:            "my-org/deploy",
		},
		{
			name:                "org-slug-prefix match",
			workflowContextName: "staging",
			organizationSlug:    "my-org",
			wantName:            "my-org/staging",
		},
		{
			name:                "no match",
			workflowContextName: "missing",
			wantName:            "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvedName(cache, orgID, tt.organizationSlug, tt.workflowContextName)
			assert.Check(t, cmp.Equal(got, tt.wantName))
		})
	}

	t.Run("resolves nothing for an organization not listed", func(t *testing.T) {
		assert.Check(t, cmp.Equal(resolvedName(cache, "another-org", "", "my-org/deploy"), ""))
	})
}

func TestContextCache_shortNames(t *testing.T) {
	orgID := "org-uuid"

	t.Run("a short name shared by several contexts resolves to none", func(t *testing.T) {
		cache := withContexts(orgID, "org-a/deploy", "org-b/deploy", "org-c/deploy")

		assert.Check(t, cmp.Equal(resolvedName(cache, orgID, "", "deploy"), ""))
		assert.Check(t, cmp.Equal(resolvedName(cache, orgID, "org-a", "deploy"), "org-a/deploy"),
			"the org slug still tells them apart")
	})

	// Which comes first in the listing must not matter.
	t.Run("a short name never hides a context of that full name", func(t *testing.T) {
		for _, names := range [][]string{
			{"deploy", "acme/deploy"},
			{"acme/deploy", "deploy"},
		} {
			cache := withContexts(orgID, names...)
			assert.Check(t, cmp.Equal(resolvedName(cache, orgID, "", "deploy"), "deploy"), "listed as %v", names)
		}
	})
}

// Everything read from the API host belongs to it, so a change of host has
// to forget it all. The project a file resolved to is among it, since its
// organization id is what contexts are looked up by.
func TestCache_ClearHostData(t *testing.T) {
	const uri = uri.URI("file:///workspace/.circleci/config.yml")

	cache := withContexts(acmeOrgID, "acme/deploy")
	t.Cleanup(cache.Close)
	var sourcePath string

	t.Run("fill the cache", func(t *testing.T) {
		cache.FileCache.SetFile(File{TextDocument: protocol.TextDocumentItem{URI: uri}})
		cache.FileCache.AddProjectSlugToFile(uri, circleci.Project{Slug: "gh/acme/rocket", OrganizationId: acmeOrgID})
		cache.FileCache.AddEnvVariableToProjectLinkedToFile(uri, "AWS_REGION")

		cache.OrbCache.SetOrb(&ast.OrbInfo{}, "circleci/go@1.7.1")
		cache.OrbPackages.packages.Put("circleci/go", &circleci.OrbPackage{Name: "circleci/go"})
		cache.OrbPackages.namespaces.Put("circleci", []circleci.OrbPackage{{Name: "circleci/go"}})
		cache.MachineOfferingsCache.Set(&circleci.Offerings{})
		cache.ProjectCache.projects.Put("gh/acme/rocket", circleci.Project{Slug: "gh/acme/rocket"})
		cache.ResourceClassCache.classes.Put("acme", []string{"acme/runner"})
		cache.NamespaceCache.namespaces.Put("acme", true)
		cache.DockerCache.Add("cimg", "go", true)

		var err error
		sourcePath, err = cache.WriteOrbSource("circleci/go@1.7.1", "version: 2.1\n")
		assert.NilError(t, err)
	})

	t.Run("clear it", func(t *testing.T) {
		cache.ClearHostData()
	})

	t.Run("check the files are still open, without their projects", func(t *testing.T) {
		file := cache.FileCache.GetFile(uri)
		assert.Assert(t, file != nil, "the file itself is still open")
		assert.Check(t, cmp.DeepEqual(file.Project, circleci.Project{}))
		assert.Check(t, cmp.Len(file.EnvVariables, 0))
	})

	t.Run("check what was read from the host is forgotten", func(t *testing.T) {
		assert.Check(t, !cache.OrbCache.HasOrb("circleci/go@1.7.1"), "orb")
		_, known := cache.OrbPackages.packages.Peek("circleci/go")
		assert.Check(t, !known, "orb package")
		_, known = cache.OrbPackages.namespaces.Peek("circleci")
		assert.Check(t, !known, "orbs of a namespace")
		assert.Check(t, !cache.ContextCache.IsOrganizationContextListLoaded(acmeOrgID), "contexts")
		_, known = cache.MachineOfferingsCache.catalog.Peek(catalogKey)
		assert.Check(t, !known, "machine catalog")
		_, known = cache.ProjectCache.projects.Peek("gh/acme/rocket")
		assert.Check(t, !known, "project")
		_, known = cache.ResourceClassCache.classes.Peek("acme")
		assert.Check(t, !known, "runner resource classes")
		_, known = cache.NamespaceCache.namespaces.Peek("acme")
		assert.Check(t, !known, "namespace")
	})

	t.Run("check the orb sources written are removed", func(t *testing.T) {
		_, err := os.Stat(sourcePath)
		assert.Check(t, cmp.ErrorIs(err, os.ErrNotExist))
		_, ok := cache.OrbIDOfSource(sourcePath)
		assert.Check(t, !ok, "a removed source is no longer the orb's")
	})

	t.Run("check what was read from Docker Hub is kept", func(t *testing.T) {
		_, known := cache.DockerCache.Get("cimg", "go")
		assert.Check(t, known, "Docker Hub is not the API host")
	})
}

func TestOrbSources(t *testing.T) {
	cache := New()
	t.Cleanup(cache.Close)

	path, err := cache.WriteOrbSource("circleci/go@1.7.1", "old")
	assert.NilError(t, err)

	t.Run("a later fetch replaces what was written", func(t *testing.T) {
		again, err := cache.WriteOrbSource("circleci/go@1.7.1", "new")
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(again, path))

		content, err := os.ReadFile(path)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(string(content), "new"))
	})

	t.Run("only a path written to is an orb's", func(t *testing.T) {
		orbID, ok := cache.OrbIDOfSource(path)
		assert.Check(t, ok)
		assert.Check(t, cmp.Equal(orbID, "circleci/go@1.7.1"))

		_, ok = cache.OrbIDOfSource("/workspace/.circleci/config.yml")
		assert.Check(t, !ok, "a config is not an orb's source")
	})

	t.Run("another cache writes somewhere of its own", func(t *testing.T) {
		other := New()
		t.Cleanup(other.Close)

		_, ok := other.OrbIDOfSource(path)
		assert.Check(t, !ok, "a source belongs to the cache that wrote it")
	})
}
