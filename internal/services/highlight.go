package languageservice

import (
	"context"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/services/definition"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// DocumentHighlight is every mention in the document of what is named at a
// position, such as a job's definition and each workflow that runs it.
func DocumentHighlight(
	ctx context.Context,
	params protocol.DocumentHighlightParams, cache *cache.Cache, context *session.Settings,
) ([]protocol.DocumentHighlight, error) {
	doc, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	// What can be renamed has its mentions found to the name, where
	// references can find a larger range around it.
	if target, ok := renameTargetAt(doc, params.Position); ok {
		if sites, err := renameSites(doc, target); err == nil {
			return highlights(sites), nil
		}
	}

	if sites, ok := orbMentionsAt(doc, params.Position); ok {
		return highlights(sites), nil
	}

	// Anything else is found as references are, which is from where it is
	// defined.
	at := params.Position
	var definedAt protocol.Range
	def := definition.DefinitionStruct{
		Cache:  cache,
		Params: protocol.DefinitionParams{TextDocumentPositionParams: params.TextDocumentPositionParams},
		Doc:    doc,
	}
	links, _ := def.Definition(ctx)
	for _, link := range links {
		if link.URI == params.TextDocument.URI && link.NameRange != (protocol.Range{}) {
			definedAt = link.NameRange
			at = definedAt.Start
			break
		}
	}

	ref := ReferenceHandler{
		Doc: doc,
		Params: protocol.ReferenceParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: params.TextDocument,
			Position:     at,
		}},
		Cache:      cache,
		FoundSteps: &[]StepRangeAndName{},
	}
	locations, err := ref.GetReferences(ctx)
	if err != nil {
		return nil, err
	}
	var ranges []protocol.Range
	for _, location := range locations {
		if location.URI == params.TextDocument.URI {
			ranges = append(ranges, location.Range)
		}
	}
	if len(ranges) == 0 {
		return nil, nil
	}

	if definedAt != (protocol.Range{}) {
		start, end := position.ToIndex(definedAt.Start, doc.Content), position.ToIndex(definedAt.End, doc.Content)
		name := string(doc.Content[start:end])
		for i, rng := range ranges {
			ranges[i] = parameterNameIn(doc.Content, rng, name)
		}
		ranges = append(ranges, definedAt)
	}

	slices.SortFunc(ranges, func(a, b protocol.Range) int { return position.Compare(a.Start, b.Start) })
	return highlights(slices.CompactFunc(ranges, position.AreRangeEqual)), nil
}

// parameterNameIn is where name is in rng, when rng is a reference to a
// parameter such as `<< parameters.name >>`, and rng otherwise.
func parameterNameIn(content []byte, rng protocol.Range, name string) protocol.Range {
	start := position.ToIndex(rng.Start, content)
	text := string(content[start:position.ToIndex(rng.End, content)])
	if !strings.HasPrefix(text, "<<") {
		return rng
	}
	i := strings.LastIndex(text, "."+name)
	if i < 0 {
		return rng
	}
	from := position.FromIndex(start+i+1, content)
	return protocol.Range{Start: from, End: position.Advance(from, []byte(name))}
}

func highlights(ranges []protocol.Range) []protocol.DocumentHighlight {
	found := make([]protocol.DocumentHighlight, 0, len(ranges))
	for _, rng := range ranges {
		found = append(found, protocol.DocumentHighlight{Range: rng, Kind: protocol.DocumentHighlightKindText})
	}
	return found
}

// orbMentionsAt are the mentions of the orb named at pos: its name where it
// is declared, and where each step, job or executor taken from it names it.
func orbMentionsAt(doc yamlparser.YamlDocument, pos protocol.Position) ([]protocol.Range, bool) {
	type use struct {
		name string
		rng  protocol.Range
	}
	var uses []use
	for _, step := range allNamedSteps(doc) {
		uses = append(uses, use{step.Name, step.Range})
	}
	for _, invocations := range invocationLists(doc) {
		for _, invocation := range invocations {
			uses = append(uses, use{invocation.JobName, invocation.JobNameRange})
		}
	}
	for _, job := range doc.Jobs {
		uses = append(uses, use{job.Executor, job.ExecutorRange})
	}

	// prefixes are where each use names its orb, by orb.
	prefixes := map[string][]protocol.Range{}
	for _, use := range uses {
		orb, _, ok := strings.Cut(use.name, "/")
		if !ok {
			continue
		}
		if _, declared := doc.Orbs[orb]; !declared {
			continue
		}
		whole, ok := nameWithin(doc.Content, use.rng, use.name)
		if !ok {
			continue
		}
		prefix := protocol.Range{Start: whole.Start, End: position.Advance(whole.Start, []byte(orb))}
		prefixes[orb] = append(prefixes[orb], prefix)
	}

	for name, orb := range doc.Orbs {
		mentions := append([]protocol.Range{orb.NameRange}, prefixes[name]...)
		if slices.ContainsFunc(mentions, func(rng protocol.Range) bool { return position.InRange(rng, pos) }) {
			slices.SortFunc(mentions, func(a, b protocol.Range) int { return position.Compare(a.Start, b.Start) })
			return slices.CompactFunc(mentions, position.AreRangeEqual), true
		}
	}
	return nil, false
}
