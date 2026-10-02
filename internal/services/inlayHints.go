package languageservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// InlayHints show, in the range asked for, the version each orb that isn't
// pinned to a whole one resolves to, and how many jobs each matrix expands
// to.
//
// An orb's version is shown only once the orb has been fetched, so that
// asking for hints never waits on the network.
func InlayHints(
	ctx context.Context, params protocol.InlayHintParams, cache *cache.Cache, context *session.Settings,
) ([]protocol.InlayHint, error) {
	doc, err := yamlparser.ParseFromUriWithCache(ctx, params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	yes := true
	hints := []protocol.InlayHint{}
	add := func(at protocol.Position, label, tooltip string) {
		if !inRange(params.Range, at) {
			return
		}
		hints = append(hints, protocol.InlayHint{
			Position:    at,
			Label:       protocol.String(label),
			Tooltip:     protocol.String(tooltip),
			PaddingLeft: &yes,
		})
	}

	for _, orb := range doc.Orbs {
		if orb.Url.IsLocal || orb.Url.IsURL || orb.Url.HasReference() || fullVersion.MatchString(orb.Url.Version) {
			continue
		}
		info := cache.OrbCache.GetOrb(orb.Url.GetOrbID())
		if info == nil || info.RemoteInfo.Version == "" {
			continue
		}
		add(orb.ValueRange.End, "→ "+info.RemoteInfo.Version,
			fmt.Sprintf("%s resolves to %s", orbReference(orb.Url), info.RemoteInfo.Version))
	}

	for _, invocations := range invocationLists(doc) {
		for _, invocation := range invocations {
			if !invocation.HasMatrix {
				continue
			}
			count := len(invocation.MatrixNames)
			label := fmt.Sprintf("%d jobs", count)
			if count == 1 {
				label = "1 job"
			}
			at := position.Advance(invocation.MatrixRange.Start, []byte("matrix:"))
			add(at, label, "Runs "+strings.Join(invocation.MatrixNames, ", "))
		}
	}

	slices.SortFunc(hints, func(a, b protocol.InlayHint) int { return position.Compare(a.Position, b.Position) })
	return hints, nil
}

// inRange reports whether pos is in rng, its end included.
func inRange(rng protocol.Range, pos protocol.Position) bool {
	return position.Compare(rng.Start, pos) <= 0 && position.Compare(pos, rng.End) <= 0
}
