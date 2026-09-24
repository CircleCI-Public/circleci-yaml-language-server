package complete

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (ch *CompletionHandler) completeOrbs() {
	if ch.DocTag != "original" {
		return
	}

	for _, orb := range ch.Doc.Orbs {
		if orb.ValueNode == nil || !position.InRange(orb.ValueRange, ch.Params.Position) || orb.ValueNode.Kind() != "flow_node" {
			continue
		}

		child := parser.GetFirstChild(orb.ValueNode)

		if child == nil {
			continue
		}

		ch.completeOrb(child)

		return
	}
}

func (ch *CompletionHandler) completeOrb(node *sitter.Node) {
	if ch.wantOrbVersionCompletion(node) {
		ch.completeOrbVersion(node)
	} else {
		ch.completeOrbName(node)
	}
}

// To know if we want to complete only the orb version or the complete orb, we detect if the cursor
// is placed on or after the '@' character
func (ch *CompletionHandler) wantOrbVersionCompletion(node *sitter.Node) bool {
	def := ch.Doc.GetOrbURLDefinition(node)

	orbHasVersion := !position.IsDefaultRange(def.Version.Range)
	if !orbHasVersion {
		return false
	}
	cursorIsOnVersion := position.InRange(def.Version.Range, ch.Params.Position)

	return cursorIsOnVersion
}

func (ch *CompletionHandler) completeOrbVersion(node *sitter.Node) {
	def := ch.Doc.GetOrbURLDefinition(node)
	orbName := fmt.Sprintf("%s/%s", def.Namespace.Text, def.Name.Text)
	completions, err := ch.getOrbVersionCompletions(orbName)
	if err != nil {
		return
	}

	for i, completion := range completions {
		ch.Items = append(ch.Items, protocol.CompletionItem{
			Label: completion,
			// TODO: this sorting implementation may encounter problems for orbs having more than 256
			// versions
			SortText: protocol.NewOptional(fmt.Sprintf("%c", i)),
			TextEdit: &protocol.TextEdit{
				Range:   def.Version.Range,
				NewText: completion,
			},
		})
	}
}

func (ch *CompletionHandler) getOrbVersionCompletions(name string) ([]string, error) {
	orbName := strings.TrimSuffix(name, "@")

	orb, err := ch.Cache.OrbPackages.Orb(ch.Doc.Context.OrbRegistry(), orbName)
	if err != nil {
		return nil, err
	}
	if orb == nil {
		return nil, fmt.Errorf("no orb named %s", orbName)
	}

	versions := make([]string, len(orb.Versions))
	for i, version := range orb.Versions {
		versions[i] = version.Version
	}
	return versions, nil
}

func (ch *CompletionHandler) completeOrbName(node *sitter.Node) {
	completions, err := ch.getOrbNameCompletions(ch.Doc.GetNodeText(node))
	if err != nil {
		return
	}

	for _, completion := range completions {
		ch.addReplaceTextCompletionItem(node, completion)
	}
}

func (ch *CompletionHandler) getOrbNameCompletions(name string) ([]string, error) {
	parts := strings.Split(name, "/")
	namespace := parts[0]

	orbs, err := ch.Cache.OrbPackages.InNamespace(ch.Doc.Context.OrbRegistry(), namespace)
	if err != nil {
		return nil, err
	}

	completions := make([]string, 0, len(orbs))

	// Versions are newest first, so the first is the one to suggest. An orb
	// with nothing published has no reference worth completing to.
	for _, orb := range orbs {
		if len(orb.Versions) == 0 {
			continue
		}

		completions = append(completions, fmt.Sprintf("%s@%s", orb.Name, orb.Versions[0].Version))
	}

	return completions, nil
}
