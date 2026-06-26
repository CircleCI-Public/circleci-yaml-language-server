package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func lsContextFor(url string) *LsContext {
	return &LsContext{Api: ApiContext{HostUrl: url}}
}

func TestMachineOfferings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/offerings" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"attributes":{"linux":{"medium":["ubuntu-2404:current"]},"windows":{},"macos":{}}}}`))
	}))
	defer server.Close()

	o := machineOfferings(lsContextFor(server.URL), CreateCache())
	if o == nil || len(o.Linux["medium"]) != 1 {
		t.Fatalf("expected offerings to be cached, got %#v", o)
	}
}

func TestMachineOfferings_FailureLeavesCacheEmpty(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cache := CreateCache()
	if machineOfferings(lsContextFor(server.URL), cache) != nil {
		t.Fatal("expected nil offerings on failure")
	}
	machineOfferings(lsContextFor(server.URL), cache) // second call should not retry

	if calls != 1 {
		t.Fatalf("expected the API to be hit once, got %d", calls)
	}
}

func TestOfferingAccessors(t *testing.T) {
	cache := CreateCache()
	cache.MachineOfferingsCache.Set(&Offerings{
		Linux: map[string][]string{
			"medium": {"ubuntu-2404:current"},
			"large":  {"ubuntu-2404:current"},
		},
		Windows: map[string][]string{
			"windows.medium": {"windows-server-2022-gui:current"},
		},
		MacOS: map[string][]string{
			"m4pro.medium": {"16.4.0"},
		},
	})
	ctx := lsContextFor("")

	assert.ElementsMatch(t, MachineImages(ctx, cache), []string{
		"ubuntu-2404:current", "windows-server-2022-gui:current",
	})
	assert.ElementsMatch(t, MachineResourceClasses(ctx, cache), []string{
		"large", "medium", "windows.medium",
	})
	assert.ElementsMatch(t, XcodeVersions(ctx, cache), []string{"16.4.0"})
	assert.ElementsMatch(t, MacOSResourceClasses(ctx, cache), []string{"m4pro.medium"})
	assert.ElementsMatch(t, DockerResourceClasses(ctx, cache), []string{
		"large", "medium", "medium+", "small",
	})
}

func TestMachinePairs_NilWhenUnavailable(t *testing.T) {
	cache := CreateCache()
	cache.MachineOfferingsCache.attempted = true // simulate a failed fetch

	if pairs := MachinePairs(lsContextFor(""), cache); pairs != nil {
		t.Fatalf("expected nil pairs when offerings unavailable, got %#v", pairs)
	}
}
