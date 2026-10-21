package ir

import (
	"fmt"
	"slices"
)

// Statuses normalises the statuses written on a requires entry, matching the
// reference compiler: none means success, terminal means all of them, and
// not_run means skipped. The result is sorted and deduplicated.
func Statuses(written []string) ([]string, error) {
	all := []string{"canceled", "failed", "skipped", "success", "unauthorized"}
	if len(written) == 0 {
		return []string{"success"}, nil
	}
	var out []string
	for _, s := range written {
		switch {
		case s == "terminal":
			out = append(out, all...)
		case s == "not_run":
			out = append(out, "skipped")
		case slices.Contains(all, s):
			out = append(out, s)
		default:
			return nil, fmt.Errorf("invalid requires status %q", s)
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}
