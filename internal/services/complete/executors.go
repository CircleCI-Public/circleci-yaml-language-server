package complete

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (ch *CompletionHandler) completeExecutors() {
	executor, err := findExecutor(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	if executor.IsUncomplete() {
		ch.addCompletionItemField("docker")
		ch.addCompletionItemField("macos")
		ch.addCompletionItemField("machine")
		return
	}

	switch executor := executor.(type) {
	case ast2.DockerExecutor:
		ch.completeDockerExecutor(executor)
	case ast2.MachineExecutor:
		ch.completeMachineExecutor(executor)
	case ast2.MacOSExecutor:
		ch.completeMacOSExecutor(executor)
	}
}

func findExecutor(pos protocol.Position, doc parser.YamlDocument) (ast2.Executor, error) {
	for _, executor := range doc.Executors {
		if position.InRange(executor.GetRange(), pos) {
			return executor, nil
		}
	}

	return nil, fmt.Errorf("no executor found")
}

func (ch *CompletionHandler) completeDockerExecutor(executor ast2.DockerExecutor) {
	if position.InRange(executor.ResourceClassRange, ch.Params.Position) {
		ch.addResourceClassCompletion(ch.Cache.Offerings(ch.Context.Api).DockerResourceClasses())
		return
	}

	// Check if we are in an image's range
	for _, img := range executor.Image {
		if position.InRange(img.ImageRange, ch.Params.Position) {
			// Suggest docker images w/ dockerhub package to perform search

			completionString, ok := typedImage(img, ch.Params.Position)
			if !ok {
				return
			}

			node, _, _ := position.NodeAt(ch.Doc.RootNode, ch.Params.Position)

			// Based on the reduced string, extract image info
			theImg := parser.ParseDockerImageValue(completionString)

			if theImg.Tag == "" && !strings.HasSuffix(completionString, ":") {
				// Search for repositories
				results := dockerhub.Search(ch.Context.DockerHub, completionString)
				i := 0

				for i < 5 && results.HasNext() {
					repo := results.Next()

					ch.addDockerImageCompletion(
						node,
						repo.Namespace,
						repo.Name,
						"latest",
						true,
					)
					i++
				}
			} else {
				// Search for tags instead
				results, err := dockerhub.SearchTags(ch.Context.DockerHub, img.Image.Namespace, img.Image.Name, theImg.Tag)
				if err != nil {
					return
				}

				i := 0

				for i < 10 && results.HasNext() {
					tag := results.Next()

					ch.addDockerImageCompletion(
						node,
						img.Image.Namespace,
						img.Image.Name,
						tag.Name,
						false,
					)

					i++
				}
			}

			break
		}
	}
}

// typedImage is the part of an image's value before pos, which is what the
// Docker Hub searches run on: "- image: cimg/node|" (| is the cursor) searches
// for cimg/node, and "- image: cimg/n|ode" for cimg/n.
//
// It is false when pos is not inside the value, or nothing has been typed
// there yet. Text inserted to make the document parse (see
// ModifyTextForAutocomplete) goes in at the cursor, after the prefix, so it is
// never part of it.
func typedImage(img ast2.DockerImage, pos protocol.Position) (string, bool) {
	value := img.ImageValueRange
	if pos.Line != value.Start.Line || value.Start.Line != value.End.Line {
		return "", false
	}

	if pos.Character <= value.Start.Character {
		return "", false
	}

	typed := int(pos.Character - value.Start.Character)
	if typed > len(img.Image.FullPath) {
		return "", false
	}

	return img.Image.FullPath[:typed], true
}

func (ch *CompletionHandler) completeMachineExecutor(executor ast2.MachineExecutor) {
	if position.InRange(executor.ResourceClassRange, ch.Params.Position) {
		for _, resourceClass := range ch.Cache.Offerings(ch.Context.Api).MachineResourceClasses() {
			ch.addCompletionItem(resourceClass)
		}
		if ch.Context.Api.IsLoggedIn() {
			customResourceClasses := ch.Cache.ResourceClassesOfFile(ch.Context.Api, ch.Doc.URI)
			for _, resourceClass := range customResourceClasses {
				ch.addCompletionItem(resourceClass)
			}
		}
		return
	}

	images := ch.Cache.Offerings(ch.Context.Api).MachineImages()

	if position.InRange(executor.ImageRange, ch.Params.Position) {
		for _, img := range images {
			ch.addCompletionItem(img)
		}
		return
	}

	if executor.Image == "" {
		extendedRange := executor.ImageRange
		extendedRange.End.Character += 999

		if position.InRange(extendedRange, ch.Params.Position) {
			for _, img := range images {
				ch.addCompletionItem(img)
			}

			return
		}
	}

	ch.checkAndAddResourceClassFieldCompletion(executor)
}

func (ch *CompletionHandler) completeMacOSExecutor(executor ast2.MacOSExecutor) {
	if position.InRange(executor.ResourceClassRange, ch.Params.Position) {
		ch.addResourceClassCompletion(ch.Cache.Offerings(ch.Context.Api).MacOSResourceClasses())
		return
	} else {
		ch.checkAndAddResourceClassFieldCompletion(executor)
	}
}

func (ch *CompletionHandler) addResourceClassCompletion(resourceClasses []string) {
	for _, resourceClass := range resourceClasses {
		ch.addCompletionItem(resourceClass)
	}
}

func (ch *CompletionHandler) checkAndAddResourceClassFieldCompletion(executor ast2.Executor) {
	if executor.GetResourceClass() == "" {
		ch.addCompletionItemField("resource_class")
		return
	}
}

func (ch *CompletionHandler) addDockerImageCompletion(node *sitter.Node, namespace, name, tag string, retrigger bool) {
	if node == nil {
		return
	}

	// Special case
	if node.Parent().Kind() == "double_quote_scalar" {
		node = node.Parent()
	}

	ogText := ch.Doc.GetNodeText(node)
	if ch.DocTag == "edit-value" && len(ch.DocDiff) > 0 {
		// Snip any diffs in value that could come from
		// altering the document (see ModifyTextForAutoComplete)
		ogText = ogText[0 : len(ogText)-len(ch.DocDiff)]
	}

	fullImageName := name

	if namespace != "library" {
		fullImageName = namespace + "/" + fullImageName
	}

	if tag != "" {
		fullImageName = fullImageName + ":" + tag
	}

	var command protocol.Command

	if retrigger {
		command = protocol.Command{
			Command: "circleci-language-server.selectTagAndComplete",
		}
	}

	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label: fullImageName,

		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      position.Start(node).Line,
					Character: position.Start(node).Character,
				},
				End: protocol.Position{
					Line: position.End(node).Line,

					// Important to use the text length
					// because the node could come from an altered document
					// which would extand it's total range (& Endpoint)
					Character: position.Start(node).Character + uint32(len(ogText)),
				},
			},
			NewText: fullImageName,
		},

		Command: command,
	})
}

// Orb executor

func (ch *CompletionHandler) getOrbExecutors(orb ast2.Orb) []ast2.Executor {
	orbInfo := ch.GetOrbInfo(orb)

	res := []ast2.Executor{}
	for _, executors := range orbInfo.Executors {
		res = append(res, executors)
	}

	return res
}
