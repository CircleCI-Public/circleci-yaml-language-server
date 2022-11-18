package complete

import (
	"fmt"
	"strings"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
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
			// Search the hub.
			searchString := img.Image.FullPath

			if ch.DocDiff != "" {
				searchString = searchString[0 : len(searchString)-len(ch.DocDiff)]
			}

			results := utils.SearchDockerHUB(searchString)
			i := 0

			for i < 5 && results.HasNext() {
				repo := results.Next()
				completion := fmt.Sprintf("%s/%s", repo.Namespace, repo.Name)

				if repo.Namespace == "library" {
					completion = repo.Name
				}

				node, _, _ := utils.NodeAtPos(ch.Doc.RootNode, ch.Params.Position)
				ch.addDockerImageCompletion(
					node,
					completion,
				)
				i++
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

func (ch *CompletionHandler) addDockerImageCompletion(node *sitter.Node, fullImageName string) {
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
		ogText = ogText[0:len(ch.DocDiff)]
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
