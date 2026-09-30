package testHelpers

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

// MachineOfferings is a minimal offerings set: separate linux, windows, and macOS
// classes, so a linux class paired with a windows image is an invalid pair.
func MachineOfferings() *circleci.Offerings {
	return &circleci.Offerings{
		Linux:   map[string][]string{"medium": {circleci.CurrentLinuxImage}},
		Windows: map[string][]string{"windows.medium": {"windows-server-2022-gui:current"}},
		MacOS: map[string][]string{
			"m4pro.medium": {"xcode:26.5.0"},
			"m4pro.large":  {"xcode:26.5.0"},
		},
		Deprecated: map[string][]string{
			"linux":   {"ubuntu-2004:2024.04.4"},
			"windows": {},
			"macos":   {"xcode:26.0.1"},
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
