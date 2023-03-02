package complete

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

var orbCache = OrbCache{
	registryOrbs: make(map[string]*NamespaceOrbResponse),
	orbData:      make(map[string]*OrbGQLData),
}

func (ch *CompletionHandler) completeOrbs() {
	if ch.DocTag != "original" {
		return
	}

	for _, orb := range ch.Doc.Orbs {
		if orb.ValueNode == nil || !utils.PosInRange(orb.ValueRange, ch.Params.Position) || orb.ValueNode.Type() != "flow_node" {
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
	fmt.Printf("ch.wantOrbVersionCompletion(node) = %+v\n", ch.wantOrbVersionCompletion(node))
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

	orbHasVersion := utils.IsDefaultRange(def.Version.Range)
	if !orbHasVersion {
		return false
	}
	cursorIsOnVersion := utils.PosInRange(def.Version.Range, ch.Params.Position)

	return cursorIsOnVersion
}

func (ch *CompletionHandler) completeOrbVersion(node *sitter.Node) {
	def := ch.Doc.GetOrbURLDefinition(node)
	orbName := fmt.Sprintf("%s/%s", def.Namespace.Text, def.Name.Text)
	completions, err := ch.getOrbVersionCompletions(
		orbName,
		ch.Doc.Context.Api.HostUrl,
		ch.Doc.Context.Api.Token,
		ch.Doc.Context.UserIdForTelemetry,
	)
	if err != nil {
		return
	}

	for i, completion := range completions {
		ch.Items = append(ch.Items, protocol.CompletionItem{
			Label: completion,
			// TODO: this sorting implementation may encounter problems for orbs having more than 256
			// versions
			SortText: fmt.Sprintf("%c", i),
			TextEdit: &protocol.TextEdit{
				Range:   def.Version.Range,
				NewText: completion,
			},
		})
	}
}

func (ch *CompletionHandler) getOrbVersionCompletions(name, hostUrl, token, userId string) ([]string, error) {
	orbName := strings.TrimSuffix(name, "@")

	orbData, err := orbCache.GetVersionsOfOrb(orbName, hostUrl, token, userId)
	if err != nil {
		return nil, err
	}

	versions := make([]string, len(orbData.Versions))
	for i, version := range orbData.Versions {
		versions[i] = version.Version
	}
	return versions, nil
}

func (ch *CompletionHandler) completeOrbName(node *sitter.Node) {
	completions, err := getOrbNameCompletions(
		ch.Doc.GetNodeText(node),
		ch.Doc.Context.Api.HostUrl,
		ch.Doc.Context.Api.Token,
		ch.Doc.Context.UserIdForTelemetry,
	)
	if err != nil {
		return
	}

	for _, completion := range completions {
		ch.addReplaceTextCompletionItem(node, completion)
	}
}

func getOrbNameCompletions(name, hostUrl, token, userId string) ([]string, error) {
	parts := strings.Split(name, "/")
	registry := parts[0]

	response, err := orbCache.GetOrbsOfRegistry(registry, hostUrl, token, userId)

	if err != nil {
		return nil, err
	}

	completions := make([]string, len(response.RegistryNamespace.Orbs.Edges))

	for i, v := range response.RegistryNamespace.Orbs.Edges {
		if len(v.Node.Versions) > 0 {
			completions[i] = fmt.Sprintf("%s@%s", v.Node.Name, v.Node.Versions[0].Version)
		}
	}

	return completions, nil
}
