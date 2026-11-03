package parser

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
	sitter "github.com/odvcencio/gotreesitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// parseFunctions reads the top-level `functions` block. Anything but a
// mapping is left alone, as the compiler leaves it: the key has long been
// free for other uses, such as holding YAML anchors.
func (doc *YamlDocument) parseFunctions(functionsNode *sitter.Node) {
	blockMappingNode := GetChildMapping(functionsNode)
	if blockMappingNode == nil {
		return
	}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		if keyNode == nil {
			return
		}

		function := ast.Function{
			Alias:      doc.GetNodeText(keyNode),
			AliasRange: doc.NodeToRange(keyNode),
			Range:      doc.NodeToRange(child),
		}

		if valueNode != nil {
			function.ReferenceRange = doc.NodeToRange(valueNode)
			if scalar := GetFirstChild(valueNode); scalar != nil && isStringScalar(scalar.Type(yamltree.Language)) {
				function.IsString = true
				function.Reference = doc.GetNodeText(valueNode)
			}
		}

		doc.Functions[function.Alias] = function
	})
}

func isStringScalar(kind string) bool {
	switch kind {
	case "plain_scalar", "double_quote_scalar", "single_quote_scalar":
		return true
	}
	return false
}

// FunctionForStep returns the declared function a step runs, and the command
// it names after a `/`, if any. A step is only a function step when its alias
// is declared, so that orbs and commands named the same way resolve as before.
func (doc *YamlDocument) FunctionForStep(stepName string) (ast.Function, string, bool) {
	if function, ok := doc.Functions[stepName]; ok {
		return function, "", true
	}

	alias, command, found := strings.Cut(stepName, "/")
	if !found {
		return ast.Function{}, "", false
	}

	function, ok := doc.Functions[alias]
	return function, command, ok
}

// LookUpFunction finds a declaration in the functions catalog. The function
// is nil when none is published under the declaration's path, and the
// descriptor is nil when that function has no such version. An error means
// the catalog couldn't be read, as on a host that doesn't serve it, and says
// nothing about the declaration.
func (doc *YamlDocument) LookUpFunction(ctx context.Context, function ast.Function, c *cache.Cache) (*circleci.FunctionPackage, *circleci.FunctionDescriptor, error) {
	path, version, found := strings.Cut(function.Reference, "@")
	if !function.IsString || !found {
		return nil, nil, fmt.Errorf("function %s has no version", function.Alias)
	}

	client := doc.Context.Client()
	published, err := c.Functions.Function(ctx, client, path)
	if err != nil || published == nil {
		return nil, nil, err
	}

	for _, candidate := range published.Versions {
		if candidate.Version != version {
			continue
		}

		descriptor, err := c.Functions.Descriptor(ctx, client, candidate)
		return published, descriptor, err
	}

	return published, nil, nil
}

// OutputStep is a function step with an id, whose outputs the steps after it
// can read as `${{ steps.<id>.outputs.<name> }}`.
type OutputStep struct {
	ID       string
	Step     ast.NamedStep
	Function ast.Function
}

var (
	cacheKeyLine      = regexp.MustCompile(`^\s*key\s*:`)
	cacheKeysLine     = regexp.MustCompile(`^\s*keys\s*:`)
	cacheKeysHeadline = regexp.MustCompile(`^\s*keys\s*:\s*$`)
	listItemLine      = regexp.MustCompile(`^\s*-\s`)
	leadingWhitespace = regexp.MustCompile(`^\s*`)
)

// OutputStepsAt returns the function steps with an id that come before the
// step pos is in, in the job or command that holds it. It reports false when
// pos isn't where a step reads an output: the `with` of a function step, the
// command of a run step, the key of a save_cache step or the key or keys of a
// restore_cache step.
func (doc *YamlDocument) OutputStepsAt(pos protocol.Position) ([]OutputStep, bool) {
	var steps []ast.Step
	for _, job := range doc.Jobs {
		if position.InRange(job.Range, pos) {
			steps = job.Steps
		}
	}
	for _, command := range doc.Commands {
		if position.InRange(command.Range, pos) {
			steps = command.Steps
		}
	}

	current := -1
	for i, step := range steps {
		if step.GetRange().Start.Line <= pos.Line {
			current = i
		}
	}
	if current == -1 || !doc.readsOutputsAt(steps[current], pos) {
		return nil, false
	}

	var found []OutputStep
	for _, step := range steps[:current] {
		named, ok := step.(ast.NamedStep)
		if !ok {
			continue
		}
		function, _, ok := doc.FunctionForStep(named.Name)
		if !ok {
			continue
		}
		if id, ok := named.Parameters["id"].Value.(string); ok && id != "" {
			found = append(found, OutputStep{ID: id, Step: named, Function: function})
		}
	}
	return found, true
}

func (doc *YamlDocument) readsOutputsAt(step ast.Step, pos protocol.Position) bool {
	switch step := step.(type) {
	case ast.NamedStep:
		if _, _, ok := doc.FunctionForStep(step.Name); !ok {
			return false
		}
		with, ok := step.Parameters["with"]
		return ok && position.InRange(with.Range, pos)
	case ast.Run:
		return position.InRange(step.CommandRange, pos)
	case ast.SaveCache:
		return doc.atCacheKey(pos, false)
	case ast.RestoreCache:
		return doc.atCacheKey(pos, true)
	}
	return false
}

// atCacheKey reports whether pos is on the line of a cache step's `key`, or,
// when the step takes them, of its `keys` or an item of them. It reads the
// text, as a cache step's range is only its name.
func (doc *YamlDocument) atCacheKey(pos protocol.Position, takesKeys bool) bool {
	lines := strings.Split(string(doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}
	line := lines[pos.Line]
	if cacheKeyLine.MatchString(line) || takesKeys && cacheKeysLine.MatchString(line) {
		return true
	}
	if !takesKeys || !listItemLine.MatchString(line) {
		return false
	}

	indent := len(leadingWhitespace.FindString(line))
	for above := int(pos.Line) - 1; above >= 0; above-- {
		text := lines[above]
		if strings.TrimSpace(text) == "" || len(leadingWhitespace.FindString(text)) >= indent {
			continue
		}
		return cacheKeysHeadline.MatchString(text)
	}
	return false
}
