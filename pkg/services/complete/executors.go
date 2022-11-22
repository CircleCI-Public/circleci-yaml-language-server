package complete

import (
	"fmt"
	"strings"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	"github.com/circleci/circleci-yaml-language-server/pkg/dockerhub"
	yamlparser "github.com/circleci/circleci-yaml-language-server/pkg/parser"
	"github.com/circleci/circleci-yaml-language-server/pkg/parser/validate"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (ch *CompletionHandler) completeExecutors() {
	executor, err := findExecutor(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	if executor.IsUncomplete() {
		ch.addCompletionItemField("docker")
		ch.addCompletionItemField("macos")
		ch.addCompletionItemField("windows")
		ch.addCompletionItemField("machine")
		return
	}

	switch executor := executor.(type) {
	case ast.DockerExecutor:
		ch.completeDockerExecutor(executor)
	case ast.MachineExecutor:
		ch.completeMachineExecutor(executor)
	case ast.MacOSExecutor:
		ch.completeMacOSExecutor(executor)
	case ast.WindowsExecutor:
		ch.completeWindowsExecutor(executor)
	}
}

func findExecutor(pos protocol.Position, doc yamlparser.YamlDocument) (ast.Executor, error) {
	for _, executor := range doc.Executors {
		if utils.PosInRange(executor.GetRange(), pos) {
			return executor, nil
		}
	}

	return nil, fmt.Errorf("no executor found")
}

func (ch *CompletionHandler) completeDockerExecutor(executor ast.DockerExecutor) {
	if utils.PosInRange(executor.ResourceClassRange, ch.Params.Position) {
		ch.addResourceClassCompletion(validate.ValidDockerResourceClasses)
		return
	}

	// Check if we are in an image's range
	for _, img := range executor.Image {
		if utils.PosInRange(img.ImageRange, ch.Params.Position) {
			// Suggest docker images w/ dockerhub package to perform search

			node, _, _ := utils.NodeAtPos(ch.Doc.RootNode, ch.Params.Position)

			// The dockerhub searches are based on strings
			// We need the string content of the current docker image but they are some things to consider
			// - The current completion could run on an "altered document" (cf ModifyTextForAutocomplete)
			//   in this case, we should strip any additional content
			// - The user cursor position
			//    "- image: cimg/node|" (| is caret) should search for cimg/node
			//    "- image: cimg/n|ode" should search for cim/n
			//	This help with suggesting tags after the current completion (cf: addDockerImageCompletion -> Command)
			completionString := img.Image.FullPath
			if ch.DocDiff != "" {
				completionString = completionString[0 : len(completionString)-len(ch.DocDiff)]
			}

			diff := img.ImageRange.End.Character - ch.Params.Position.Character
			completionString = completionString[0 : len(img.Image.FullPath)-int(diff)]

			// Based on the reduced string, extract image info
			theImg := yamlparser.ParseDockerImageValue(completionString)

			if theImg.Tag == "" && completionString[len(completionString)-1:] != ":" {
				// Search for repositories
				results := dockerhub.Search(completionString)
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
				results, err := dockerhub.SearchTags(img.Image.Namespace, img.Image.Name, theImg.Tag)

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

func (ch *CompletionHandler) completeMachineExecutor(executor ast.MachineExecutor) {
	if utils.PosInRange(executor.ResourceClassRange, ch.Params.Position) {
		if strings.HasPrefix(executor.ResourceClass, "arm.") {
			ch.addResourceClassCompletion(validate.ValidARMResourceClasses)
		} else if strings.HasPrefix(executor.ResourceClass, "gpu.nvidia") || strings.HasPrefix(executor.ResourceClass, "windows.gpu.nvidia") {
			ch.addResourceClassCompletion(validate.ValidNvidiaGPUResourceClasses)
		} else {
			ch.addResourceClassCompletion(validate.ValidLinuxResourceClasses)
		}
		return
	}

	if utils.PosInRange(executor.ImageRange, ch.Params.Position) {
		for _, img := range validate.ValidARMOrMachineImages {
			ch.addCompletionItem(img)
		}
		return
	} else if executor.Image == "" {
		extendedRange := executor.ImageRange
		extendedRange.End.Character += 999

		if utils.PosInRange(extendedRange, ch.Params.Position) {
			for _, img := range validate.ValidARMOrMachineImages {
				ch.addCompletionItem(img)
			}

			return
		}
	}

	ch.checkAndAddResourceClassFieldCompletion(executor)
}

func (ch *CompletionHandler) completeMacOSExecutor(executor ast.MacOSExecutor) {
	if utils.PosInRange(executor.ResourceClassRange, ch.Params.Position) {
		ch.addResourceClassCompletion(validate.ValidMacOSResourceClasses)
		return
	} else {
		ch.checkAndAddResourceClassFieldCompletion(executor)
	}
}

func (ch *CompletionHandler) completeWindowsExecutor(executor ast.WindowsExecutor) {
	if utils.PosInRange(executor.ResourceClassRange, ch.Params.Position) {
		ch.addResourceClassCompletion(validate.ValidLinuxResourceClasses)
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

func (ch *CompletionHandler) checkAndAddResourceClassFieldCompletion(executor ast.Executor) {
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
	if node.Parent().Type() == "double_quote_scalar" {
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

	var command *protocol.Command = nil

	if retrigger {
		command = &protocol.Command{
			Command: "circleci-language-server.selectTagAndComplete",
		}
	}

	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label: fullImageName,

		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      node.StartPoint().Row,
					Character: node.StartPoint().Column,
				},
				End: protocol.Position{
					Line: node.EndPoint().Row,

					// Important to use the text length
					// because the node could come from an altered document
					// which would extand it's total range (& Endpoint)
					Character: node.StartPoint().Column + uint32(len(ogText)),
				},
			},
			NewText: fullImageName,
		},

		Command: command,
	})
}

// Orb executor

func (ch *CompletionHandler) getOrbExecutors(orb ast.Orb) []ast.Executor {
	remoteOrb := ch.Cache.OrbCache.GetOrb(orb.Url.GetOrbID())

	res := []ast.Executor{}
	for _, executors := range remoteOrb.Executors {
		res = append(res, executors)
	}

	return res
}
