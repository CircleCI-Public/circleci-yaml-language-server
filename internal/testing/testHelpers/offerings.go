package testHelpers

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

// MachineOfferings is a minimal offerings set: separate linux, windows, and macOS
// classes, so a linux class paired with a windows image is an invalid pair.
// small.gen2 is a Docker class with no remote Docker versions.
func MachineOfferings() *circleci.Offerings {
	docker := map[string][]string{}
	for _, class := range []string{
		"small", "medium", "medium+", "large", "xlarge", "2xlarge", "2xlarge+",
		"small.gen2", "medium.gen2", "medium+.gen2", "large.gen2", "xlarge.gen2", "2xlarge.gen2", "2xlarge+.gen2",
		"arm.medium", "arm.large", "arm.xlarge", "arm.2xlarge",
		"arm.medium.gen2", "arm.large.gen2", "arm.xlarge.gen2", "arm.2xlarge.gen2",
	} {
		docker[class] = []string{}
	}
	remoteDocker := []string{"default", "docker28", "docker29", "edge", "previous"}

	return &circleci.Offerings{
		RemoteDocker: map[string][]string{
			"medium":      remoteDocker,
			"large":       remoteDocker,
			"medium.gen2": {"default", "docker29", "edge"},
		},
		Docker:  docker,
		Linux:   map[string][]string{"medium": {circleci.CurrentLinuxImage}},
		Windows: map[string][]string{"windows.medium": {"windows-server-2022-gui:current"}},
		MacOS: map[string][]string{
			"m4pro.medium": {"xcode:26.5.0"},
			"m4pro.large":  {"xcode:26.5.0"},
		},
		Deprecated: map[string][]string{
			"linux":         {"ubuntu-2004:2024.04.4"},
			"windows":       {},
			"macos":         {"xcode:26.0.1"},
			"remote_docker": {"docker24"},
		},
	}
}

// DefaultCache returns a cache that already holds MachineOfferings. Validating
// against DefaultSettings needs one: a cache without a catalog fetches it from
// the settings' host, which is production.
func DefaultCache() *cache.Cache {
	c := cache.New()
	c.MachineOfferingsCache.Set(MachineOfferings())
	return c
}
