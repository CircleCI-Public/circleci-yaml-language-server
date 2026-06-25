package utils

import "testing"

func testContext(name string) *Context {
	return &Context{Name: name}
}

func TestContextCache_ResolveWorkflowContext(t *testing.T) {
	orgID := "org-uuid"
	orgSlug := "my-org"

	cache := CreateCache()
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
	cache := CreateCache()

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
	cache := CreateCache()
	if cache.ContextCache.IsOrganizationContextListLoaded(orgID) {
		t.Fatal("expected list not loaded initially")
	}
	cache.ContextCache.MarkOrganizationContextListLoaded(orgID)
	if !cache.ContextCache.IsOrganizationContextListLoaded(orgID) {
		t.Fatal("expected list loaded after mark")
	}
}
