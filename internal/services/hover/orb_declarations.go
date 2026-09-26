package hover

import (
	"fmt"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// OrbDeclaration is the hover for an orb's declaration under `orbs:`: the
// orb's description, and the commands, jobs and executors it gives. For an
// inline orb, only its name has one, rather than all of its body.
func OrbDeclaration(doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	for _, declared := range doc.Orbs {
		onName := position.InRange(declared.NameRange, pos)
		onReference := !declared.Url.IsLocal && position.InRange(declared.ValueRange, pos)
		if declared.IsPlaceholder || (!onName && !onReference) {
			continue
		}

		orb, err := doc.GetOrFetchOrbInfo(declared, c)
		if err != nil || orb == nil {
			return "", false
		}

		var b strings.Builder
		fmt.Fprintf(&b, "**%s** orb", declared.Name)
		if !declared.Url.IsLocal {
			reference := declared.Url.Name
			if declared.Url.Version != "" {
				reference += "@" + declared.Url.Version
			}
			fmt.Fprintf(&b, " `%s`", reference)
		}
		if orb.Description != "" {
			fmt.Fprintf(&b, "\n\n%s", orb.Description)
		}
		writeNames(&b, "Commands", keys(orb.Commands))
		writeNames(&b, "Jobs", keys(orb.Jobs))
		writeNames(&b, "Executors", keys(orb.Executors))
		return b.String(), true
	}
	return "", false
}

func keys[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func writeNames(b *strings.Builder, heading string, names []string) {
	if len(names) == 0 {
		return
	}
	fmt.Fprintf(b, "\n\n%s: `%s`", heading, strings.Join(names, "`, `"))
}
