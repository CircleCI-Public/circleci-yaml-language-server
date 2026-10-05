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
const catalogRoute = "GET /api/v3/catalog/resource-classes"

// offeringsFake builds a fake serving a small but representative catalog,
// with a Docker class named as a Linux one is.
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
		Docker: map[string][]string{"medium": {}},
		Deprecated: map[string][]string{
			"linux":   {"ubuntu-2004:current"},
			"windows": {"windows-server-2019:current"},
			"macos":   {"xcode:14.0.0"},
		},
		ResourceClasses: map[string]map[string]fakes.ResourceClass{
			"linux":  {"medium": {Name: "Linux Medium", CPU: 2, RAMMB: 8192}},
			"docker": {"medium": {Name: "Medium", CPU: 2, RAMMB: 4096}},
		},
	})

	return fake
}

func TestMachineOfferings(t *testing.T) {
	t.Run("fetches the catalog once and caches it", func(t *testing.T) {
		fake := offeringsFake(t)
		cache := New()

		offerings := cache.Offerings(t.Context(), configFor(fake.URL()))
		assert.Assert(t, offerings != nil)
		assert.Check(t, cmp.DeepEqual(offerings.Linux["medium"], []string{"ubuntu-2404:current"}))

		// Every completion and validation pass asks for the catalog, so it has
		// to be fetched once for the life of the cache.
		cache.Offerings(t.Context(), configFor(fake.URL()))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v3/catalog/resource-classes")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("authenticates with Circle-Token", func(t *testing.T) {
		fake := offeringsFake(t)

		New().Offerings(t.Context(), configFor(fake.URL()))

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

		offerings := cache.Offerings(t.Context(), configFor(fake.URL()))
		assert.Check(t, cmp.Nil(offerings))

		cache.Offerings(t.Context(), configFor(fake.URL()))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v3/catalog/resource-classes")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("concurrent callers share one fetch", func(t *testing.T) {
		fake := offeringsFake(t)
		cache := New()

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				assert.Check(t, cache.Offerings(t.Context(), configFor(fake.URL())) != nil)
			})
		}
		wg.Wait()

		requestCount := fake.RequestCount(http.MethodGet, "/api/v3/catalog/resource-classes")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	t.Run("reports nothing when the host is unreachable", func(t *testing.T) {
		fake := offeringsFake(t)
		hostUrl := fake.URL()
		fake.Close()

		offerings := New().Offerings(t.Context(), configFor(hostUrl))
		assert.Check(t, cmp.Nil(offerings))
	})

	t.Run("reports nothing for a malformed body", func(t *testing.T) {
		fake := offeringsFake(t)
		fake.SetBody(catalogRoute, "{")

		offerings := New().Offerings(t.Context(), configFor(fake.URL()))
		assert.Check(t, cmp.Nil(offerings))
	})

	// A well-formed response with no classes in it is as useless as a failure,
	// and has to be treated the same way.
	t.Run("reports nothing for an empty catalog", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)

		offerings := New().Offerings(t.Context(), configFor(fake.URL()))
		assert.Check(t, cmp.Nil(offerings))
	})
}

// TestDeprecatedOfferings goes through the API rather than through a
// pre-populated cache, because the API lists deprecated images on each
// resource class, and they are gathered by executor as the body is decoded.
func TestDeprecatedOfferings(t *testing.T) {
	fake := offeringsFake(t)
	api := configFor(fake.URL())
	cache := New()

	anyOrder := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	deprecatedImages := cache.Offerings(t.Context(), api).DeprecatedMachineImages()
	assert.Check(t, cmp.DeepEqual(deprecatedImages, []string{
		"ubuntu-2004:current", "windows-server-2019:current",
	}, anyOrder))

	// The config field is the bare version, so the "xcode:" prefix the API
	// reports has to come off.
	deprecatedXcode := cache.Offerings(t.Context(), api).DeprecatedXcodeVersions()
	assert.Check(t, cmp.DeepEqual(deprecatedXcode, []string{"14.0.0"}, anyOrder))
}

func TestResourceClasses(t *testing.T) {
	fake := offeringsFake(t)
	offerings := New().Offerings(t.Context(), configFor(fake.URL()))
	assert.Assert(t, offerings != nil)

	t.Run("names and sizes a class", func(t *testing.T) {
		class, ok := offerings.Class("medium", circleci.MachineExecutors...)
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(class, circleci.ResourceClass{Name: "Linux Medium", CPU: 2, RAMMB: 8192}))
	})

	t.Run("a class is the executor's it is asked of", func(t *testing.T) {
		class, ok := offerings.Class("medium", circleci.ExecutorDocker)
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(class, circleci.ResourceClass{Name: "Medium", CPU: 2, RAMMB: 4096}))
	})

	t.Run("a class the executors don't offer isn't found", func(t *testing.T) {
		_, ok := offerings.Class("m4pro.medium", circleci.MachineExecutors...)
		assert.Check(t, !ok, "a macOS class was found on a machine executor")
		_, ok = offerings.Class("acme/runner", circleci.MachineExecutors...)
		assert.Check(t, !ok, "a self-hosted runner's class was found in the catalog")
	})

	t.Run("a class the catalog doesn't describe is still offered", func(t *testing.T) {
		class, ok := offerings.Class("large", circleci.MachineExecutors...)
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(class, circleci.ResourceClass{}))
		assert.Check(t, cmp.DeepEqual(offerings.Linux["large"], []string{"ubuntu-2404:current"}))
	})

	t.Run("a Docker class takes no images", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(offerings.Docker, map[string][]string{"medium": {}}))
	})
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
		Docker: map[string][]string{"small": {}, "medium": {}, "medium+.gen2": {}},
	})
	ctx := configFor("")

	// These accessors gather from maps, so the order they come back in is not
	// meaningful and the comparison has to be order-insensitive.
	anyOrder := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	images := cache.Offerings(t.Context(), ctx).MachineImages()
	assert.Check(t, cmp.DeepEqual(images, []string{
		"ubuntu-2404:current", "windows-server-2022-gui:current", "linux-cuda-12:current",
	}, anyOrder))

	machineClasses := cache.Offerings(t.Context(), ctx).MachineResourceClasses()
	assert.Check(t, cmp.DeepEqual(machineClasses, []string{
		"large", "medium", "medium.gen3", "gpu.nvidia.medium", "windows.medium",
	}, anyOrder))

	xcodeVersions := cache.Offerings(t.Context(), ctx).XcodeVersions()
	assert.Check(t, cmp.DeepEqual(xcodeVersions, []string{"16.4.0"}, anyOrder))

	macOSClasses := cache.Offerings(t.Context(), ctx).MacOSResourceClasses()
	assert.Check(t, cmp.DeepEqual(macOSClasses, []string{"m4pro.medium"}, anyOrder))

	dockerClasses := cache.Offerings(t.Context(), ctx).DockerResourceClasses()
	assert.Check(t, cmp.DeepEqual(dockerClasses, []string{"small", "medium", "medium+.gen2"}, anyOrder))
}

func TestDockerResourceClasses_NilWithoutDocker(t *testing.T) {
	// As a catalog from before the Docker executor was added to it: its
	// classes are then not checked, rather than all flagged.
	offerings := &circleci.Offerings{Linux: map[string][]string{"medium": {"ubuntu-2404:current"}}}
	classes := offerings.DockerResourceClasses()
	assert.Check(t, cmp.Nil(classes))
}

func TestRemoteDockerVersions(t *testing.T) {
	offerings := &circleci.Offerings{
		Docker: map[string][]string{
			"small": {}, "medium": {}, "medium+": {}, "large": {}, "small.gen2": {},
		},
		RemoteDocker: map[string][]string{
			"medium": {"default", "docker28"},
			"large":  {"default", "docker29"},
		},
		Deprecated: map[string][]string{"remote_docker": {"docker24"}},
	}

	t.Run("a class's own versions", func(t *testing.T) {
		got := offerings.RemoteDockerVersions("large")
		assert.Check(t, cmp.DeepEqual(got, []string{"default", "docker29"}))
	})

	t.Run("small and medium+ run on medium and large machines", func(t *testing.T) {
		small := offerings.RemoteDockerVersions("small")
		assert.Check(t, cmp.DeepEqual(small, []string{"default", "docker28"}))
		mediumPlus := offerings.RemoteDockerVersions("medium+")
		assert.Check(t, cmp.DeepEqual(mediumPlus, []string{"default", "docker29"}))
	})

	t.Run("a Docker class without remote Docker has none", func(t *testing.T) {
		got := offerings.RemoteDockerVersions("small.gen2")
		assert.Check(t, cmp.DeepEqual(got, []string{}))
	})

	t.Run("a class the catalog doesn't know isn't said", func(t *testing.T) {
		got := offerings.RemoteDockerVersions("xlarge")
		assert.Check(t, cmp.Nil(got))
	})

	t.Run("a catalog without remote Docker says nothing", func(t *testing.T) {
		without := &circleci.Offerings{Docker: offerings.Docker}
		got := without.RemoteDockerVersions("medium")
		assert.Check(t, cmp.Nil(got))
	})

	t.Run("deprecated versions are listed apart", func(t *testing.T) {
		got := offerings.DeprecatedRemoteDockerVersions()
		assert.Check(t, cmp.DeepEqual(got, []string{"docker24"}))
	})
}

func TestMachinePairs_NilWhenUnavailable(t *testing.T) {
	cache := New()
	cache.MachineOfferingsCache.Set(nil) // as a failed fetch leaves it

	pairs := cache.Offerings(t.Context(), configFor("")).MachinePairs()
	assert.Check(t, cmp.Nil(pairs))
}
