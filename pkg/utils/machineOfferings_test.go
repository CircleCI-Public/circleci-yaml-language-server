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

func TestFetchMachineOfferings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/machine/offerings" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"attributes":{"linux":{"medium":["ubuntu-2404:current"]},"windows":{},"macos":{},"deprecated":[]}}}`))
	}))
	defer server.Close()

	cache := CreateCache()
	FetchMachineOfferings(lsContextFor(server.URL), cache)

	offerings := cache.MachineOfferingsCache.Get()
	if offerings == nil || len(offerings.Linux["medium"]) != 1 {
		t.Fatalf("expected offerings to be cached, got %#v", offerings)
	}
}

func TestFetchMachineOfferings_FailureLeavesCacheEmpty(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cache := CreateCache()
	FetchMachineOfferings(lsContextFor(server.URL), cache)
	FetchMachineOfferings(lsContextFor(server.URL), cache) // second call should not retry

	if cache.MachineOfferingsCache.Get() != nil {
		t.Fatal("expected cache to stay empty on failure")
	}
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
		Deprecated: []string{"16.3.0"},
	})
	ctx := lsContextFor("")

	// macOS is excluded from machine pairs; deprecated images stay valid everywhere.
	assert.ElementsMatch(t, MachineImages(ctx, cache), []string{
		"ubuntu-2404:current", "windows-server-2022-gui:current", "16.3.0",
	})
	assert.ElementsMatch(t, MachineResourceClasses(ctx, cache), []string{
		"large", "medium", "windows.medium",
	})

	assert.ElementsMatch(t, XcodeVersions(ctx, cache), []string{"16.3.0", "16.4.0"})
	assert.ElementsMatch(t, MacOSResourceClasses(ctx, cache), []string{"m4pro.medium"})

	// Docker reuses the Linux classes, plus small and medium+.
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
