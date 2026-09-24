package cache

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func testContext(name string) *Context {
	return &Context{Name: name}
}

func TestContextCache_ResolveWorkflowContext(t *testing.T) {
	orgID := "org-uuid"
	orgSlug := "my-org"

	cache := New()
	cache.ContextCache.SetOrganizationContext(orgID, testContext("my-org/deploy"))
	cache.ContextCache.SetOrganizationContext(orgID, testContext("my-org/staging"))

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
			organizationSlug:    orgSlug,
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
			got := cache.ContextCache.ResolveWorkflowContext(orgID, tt.organizationSlug, tt.workflowContextName)
			if tt.wantName == "" {
				if got != nil {
					t.Fatalf("ResolveWorkflowContext() = %q, want nil", got.Name)
				}
				return
			}
			if got == nil {
				t.Fatalf("ResolveWorkflowContext() = nil, want %q", tt.wantName)
			}
			if got.Name != tt.wantName {
				t.Fatalf("ResolveWorkflowContext() = %q, want %q", got.Name, tt.wantName)
			}
		})
	}
}

func TestContextCache_SetOrganizationContext_shortNameCollision(t *testing.T) {
	orgID := "org-uuid"
	cache := New()

	cache.ContextCache.SetOrganizationContext(orgID, testContext("org-a/deploy"))
	cache.ContextCache.SetOrganizationContext(orgID, testContext("org-b/deploy"))

	if got := cache.ContextCache.GetOrganizationContext(orgID, "deploy"); got != nil {
		t.Fatalf("short key should be dropped after collision, got %q", got.Name)
	}

	if got := cache.ContextCache.ResolveWorkflowContext(orgID, "org-a", "deploy"); got == nil || got.Name != "org-a/deploy" {
		t.Fatalf("org-slug lookup should still resolve org-a/deploy, got %#v", got)
	}

	if got := cache.ContextCache.ResolveWorkflowContext(orgID, "", "deploy"); got != nil {
		t.Fatalf("ambiguous short name should not resolve without org slug, got %q", got.Name)
	}

	cache.ContextCache.SetOrganizationContext(orgID, testContext("org-c/deploy"))
	if got := cache.ContextCache.GetOrganizationContext(orgID, "deploy"); got != nil {
		t.Fatalf("short key should stay dropped after later collision, got %q", got.Name)
	}
}

func TestContextCache_listLoadedTracking(t *testing.T) {
	orgID := "org-uuid"
	cache := New()
	if cache.ContextCache.IsOrganizationContextListLoaded(orgID) {
		t.Fatal("expected list not loaded initially")
	}
	cache.ContextCache.MarkOrganizationContextListLoaded(orgID)
	if !cache.ContextCache.IsOrganizationContextListLoaded(orgID) {
		t.Fatal("expected list loaded after mark")
	}
}

// The project a file resolved to belongs to the host it was resolved on, so a
// change of host has to forget it along with the rest of that host's data.
func TestCache_ClearHostData_forgetsResolvedProjects(t *testing.T) {
	const uri = uri.URI("file:///workspace/.circleci/config.yml")

	cache := New()
	cache.FileCache.SetFile(File{TextDocument: protocol.TextDocumentItem{URI: uri}})
	cache.FileCache.AddProjectSlugToFile(uri, circleci.Project{Slug: "gh/acme/rocket", OrganizationId: "org-acme"})
	cache.FileCache.AddEnvVariableToProjectLinkedToFile(uri, "AWS_REGION")

	cache.ClearHostData()

	file := cache.FileCache.GetFile(uri)
	assert.Assert(t, file != nil, "the file itself is still open")
	assert.Check(t, cmp.DeepEqual(file.Project, circleci.Project{}))
	assert.Check(t, cmp.Len(file.EnvVariables, 0))
}
