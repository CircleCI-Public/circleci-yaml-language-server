package cache

import (
	"net/http"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

// catalogRoute is the route the machine catalog is served on.
const catalogRoute = "GET /api/v3/catalog/offerings"

// offeringsFake builds a fake serving a small but representative catalog: a
// couple of Linux classes, one Windows class, one macOS class, and a deprecated
// entry for each executor.
func offeringsFake(t *testing.T) *fakes.CircleCI {
	t.Helper()

	fake := fakes.NewCircleCI(t)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux: map[string][]string{
			"medium": {"ubuntu-2404:current"},
			"large":  {"ubuntu-2404:current"},
		},
		Windows: map[string][]string{
			"windows.medium": {"windows-server-2022-gui:current"},
		},
		MacOS: map[string][]string{
			"m4pro.medium": {"xcode:16.4.0"},
		},
		Deprecated: map[string][]string{
			"linux":   {"ubuntu-2004:current"},
			"windows": {"windows-server-2019:current"},
			"macos":   {"xcode:14.0.0"},
		},
	})

	return fake
}

func TestMachineOfferings(t *testing.T) {
	t.Run("fetches the catalog once and caches it", func(t *testing.T) {
		fake := offeringsFake(t)
		cache := New()

		offerings := cache.Offerings(configFor(fake.URL()))
		assert.Assert(t, offerings != nil)
		assert.Check(t, cmp.DeepEqual(offerings.Linux["medium"], []string{"ubuntu-2404:current"}))

		// Every completion and validation pass asks for the catalog, so it has
		// to be fetched once for the life of the cache.
		cache.Offerings(configFor(fake.URL()))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v3/catalog/offerings")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("authenticates with Circle-Token", func(t *testing.T) {
		fake := offeringsFake(t)

		New().Offerings(configFor(fake.URL()))

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.Equal(requests[0].CircleToken, testToken))
	})

	// A failed fetch has to leave the accessors reporting nothing, so that
	// validation skips the check rather than flagging valid config — and it
	// must not be retried for every executor either. It is remembered for
	// memo.NotFoundLifetime.
	t.Run("reports nothing when the host fails, and does not retry", func(t *testing.T) {
		fake := offeringsFake(t)
		fake.SetStatus(catalogRoute, http.StatusInternalServerError)
		cache := New()

		offerings := cache.Offerings(configFor(fake.URL()))
		assert.Check(t, cmp.Nil(offerings))

		cache.Offerings(configFor(fake.URL()))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v3/catalog/offerings")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("concurrent callers share one fetch", func(t *testing.T) {
		fake := offeringsFake(t)
		cache := New()

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				assert.Check(t, cache.Offerings(configFor(fake.URL())) != nil)
			})
		}
		wg.Wait()

		requestCount := fake.RequestCount(http.MethodGet, "/api/v3/catalog/offerings")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("reports nothing when the host is unreachable", func(t *testing.T) {
		fake := offeringsFake(t)
		hostUrl := fake.URL()
		fake.Close()

		offerings := New().Offerings(configFor(hostUrl))
		assert.Check(t, cmp.Nil(offerings))
	})

	t.Run("reports nothing for a malformed body", func(t *testing.T) {
		fake := offeringsFake(t)
		fake.SetBody(catalogRoute, "{")

		offerings := New().Offerings(configFor(fake.URL()))
		assert.Check(t, cmp.Nil(offerings))
	})

	// A well-formed response with no classes in it is as useless as a failure,
	// and has to be treated the same way.
	t.Run("reports nothing for an empty catalog", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)

		offerings := New().Offerings(configFor(fake.URL()))
		assert.Check(t, cmp.Nil(offerings))
	})
}

// TestDeprecatedOfferings goes through the API rather than through a
// pre-populated cache, because the deprecated groups are keyed by executor
// rather than by resource class and that only shows up in the decoded body.
func TestDeprecatedOfferings(t *testing.T) {
	fake := offeringsFake(t)
	api := configFor(fake.URL())
	cache := New()

	anyOrder := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	deprecatedImages := cache.Offerings(api).DeprecatedMachineImages()
	assert.Check(t, cmp.DeepEqual(deprecatedImages, []string{
		"ubuntu-2004:current", "windows-server-2019:current",
	}, anyOrder))

	// The config field is the bare version, so the "xcode:" prefix the API
	// reports has to come off.
	deprecatedXcode := cache.Offerings(api).DeprecatedXcodeVersions()
	assert.Check(t, cmp.DeepEqual(deprecatedXcode, []string{"14.0.0"}, anyOrder))
}

func TestOfferingAccessors(t *testing.T) {
	cache := New()
	cache.MachineOfferingsCache.Set(&circleci.Offerings{
		Linux: map[string][]string{
			"medium":            {"ubuntu-2404:current"},
			"large":             {"ubuntu-2404:current"},
			"medium.gen3":       {"ubuntu-2404:current"},
			"gpu.nvidia.medium": {"linux-cuda-12:current"},
		},
		Windows: map[string][]string{
			"windows.medium": {"windows-server-2022-gui:current"},
		},
		MacOS: map[string][]string{
			"m4pro.medium": {"xcode:16.4.0"},
		},
	})
	ctx := configFor("")

	// These accessors gather from maps, so the order they come back in is not
	// meaningful and the comparison has to be order-insensitive.
	anyOrder := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	images := cache.Offerings(ctx).MachineImages()
	assert.Check(t, cmp.DeepEqual(images, []string{
		"ubuntu-2404:current", "windows-server-2022-gui:current", "linux-cuda-12:current",
	}, anyOrder))

	machineClasses := cache.Offerings(ctx).MachineResourceClasses()
	assert.Check(t, cmp.DeepEqual(machineClasses, []string{
		"large", "medium", "medium.gen3", "gpu.nvidia.medium", "windows.medium",
	}, anyOrder))

	xcodeVersions := cache.Offerings(ctx).XcodeVersions()
	assert.Check(t, cmp.DeepEqual(xcodeVersions, []string{"16.4.0"}, anyOrder))

	macOSClasses := cache.Offerings(ctx).MacOSResourceClasses()
	assert.Check(t, cmp.DeepEqual(macOSClasses, []string{"m4pro.medium"}, anyOrder))

	// Docker excludes machine-only .gen/.multi/gpu variants from offerings, and
	// includes Docker-only sizes (small, medium+, and the .gen2 family).
	dockerClasses := cache.Offerings(ctx).DockerResourceClasses()
	assert.Check(t, cmp.DeepEqual(dockerClasses, []string{
		"large", "medium", "medium+", "small",
		"small.gen2", "medium.gen2", "medium+.gen2", "large.gen2",
		"xlarge.gen2", "2xlarge.gen2", "2xlarge+.gen2",
	}, anyOrder))
}

func TestMachinePairs_NilWhenUnavailable(t *testing.T) {
	cache := New()
	cache.MachineOfferingsCache.Set(nil) // as a failed fetch leaves it

	pairs := cache.Offerings(configFor("")).MachinePairs()
	assert.Check(t, cmp.Nil(pairs))
}
