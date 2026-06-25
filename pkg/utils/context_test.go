package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func Test_getContext(t *testing.T) {
	orgID := "11111111-2222-3333-4444-555555555555"
	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
		wantItems  int
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body: `{
				"items": [{
					"name": "my-org/deploy",
					"id": "ctx-id",
					"created_at": "2024-01-02T03:04:05Z",
					"environment_variables": []
				}]
			}`,
			wantItems: 1,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"message":"Unauthorized"}`,
			wantErr:    "HTTP 401",
		},
		{
			name:       "invalid json on success status",
			statusCode: http.StatusOK,
			body:       `{`,
			wantErr:    "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Circle-Token"); got != "test-token" {
					t.Errorf("Circle-Token = %q, want %q", got, "test-token")
				}
				if !strings.Contains(r.URL.Path, "/api/v2/context") {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if got := r.URL.Query().Get("owner-id"); got != orgID {
					t.Fatalf("owner-id = %q, want %q", got, orgID)
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			lsContext := &LsContext{
				Api: ApiContext{
					Token:   "test-token",
					HostUrl: server.URL,
				},
			}

			got, err := getContext(lsContext, orgID, "", false)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("getContext() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("getContext() error = %q, want containing %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("getContext() unexpected error: %v", err)
			}
			if len(got.Items) != tt.wantItems {
				t.Fatalf("len(Items) = %d, want %d", len(got.Items), tt.wantItems)
			}
			if got.Items[0].Name != "my-org/deploy" {
				t.Fatalf("Items[0].Name = %q, want %q", got.Items[0].Name, "my-org/deploy")
			}
			if !got.Items[0].CreatedAt.Equal(createdAt) {
				t.Fatalf("Items[0].CreatedAt = %v, want %v", got.Items[0].CreatedAt, createdAt)
			}
		})
	}
}


func Test_getContext_withoutEnvVarsQueryParam(t *testing.T) {
	orgID := "11111111-2222-3333-4444-555555555555"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("include-env-vars") {
			t.Fatalf("unexpected include-env-vars query param: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	lsContext := &LsContext{Api: ApiContext{Token: "test-token", HostUrl: server.URL}}
	if _, err := getContext(lsContext, orgID, "", false); err != nil {
		t.Fatalf("getContext() error = %v", err)
	}
}

func Test_getContext_withEnvVarsQueryParam(t *testing.T) {
	orgID := "11111111-2222-3333-4444-555555555555"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("include-env-vars"); got != "true" {
			t.Fatalf("include-env-vars = %q, want true", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	lsContext := &LsContext{Api: ApiContext{Token: "test-token", HostUrl: server.URL}}
	if _, err := getContext(lsContext, orgID, "", true); err != nil {
		t.Fatalf("getContext() error = %v", err)
	}
}

func Test_GetAllContextWithEnvVars_mergesEnvVarNames(t *testing.T) {
	orgID := "11111111-2222-3333-4444-555555555555"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
		if r.URL.Query().Has("include-env-vars") {
			_, _ = w.Write([]byte(`{"items":[{"name":"my-org/deploy","id":"ctx-id","created_at":"2024-01-02T03:04:05Z","environment_variables":[{"variable":"SECRET","truncated_value":"x","created_at":"2024-01-02T03:04:05Z","updated_at":"2024-01-02T03:04:05Z"}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[{"name":"my-org/deploy","id":"ctx-id","created_at":"2024-01-02T03:04:05Z"}]}`))
	}))
	defer server.Close()

	cache := CreateCache()
	lsContext := &LsContext{Api: ApiContext{Token: "test-token", HostUrl: server.URL}}
	if err := GetAllContext(lsContext, orgID, cache); err != nil {
		t.Fatalf("GetAllContext() error = %v", err)
	}
	if err := GetAllContextWithEnvVars(lsContext, orgID, cache); err != nil {
		t.Fatalf("GetAllContextWithEnvVars() error = %v", err)
	}
	ctx := cache.ContextCache.GetOrganizationContext(orgID, "my-org/deploy")
	if ctx == nil {
		t.Fatal("expected context in cache")
	}
	if len(ctx.envVariables) != 1 || ctx.envVariables[0] != "SECRET" {
		t.Fatalf("envVariables = %#v, want [SECRET]", ctx.envVariables)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}
