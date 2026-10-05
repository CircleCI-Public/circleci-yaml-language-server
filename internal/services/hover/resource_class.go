package hover

import (
	"context"
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// ResourceClass is the hover for the resource class a job or an executor
// gives: what the catalog calls it, and its CPUs and memory. A class the
// catalog doesn't list, such as a self-hosted runner's, gets none.
func ResourceClass(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	name, line, ok := resourceClassAt(doc, pos)
	if !ok {
		return "", false
	}
	executor, ok := executorGivingResourceClassOn(ctx, doc, c, line)
	if !ok {
		return "", false
	}

	var executors []string
	switch executor.(type) {
	case ast.DockerExecutor:
		executors = []string{circleci.ExecutorDocker}
	case ast.MachineExecutor:
		executors = circleci.MachineExecutors
	case ast.MacOSExecutor:
		executors = []string{circleci.ExecutorMacOS}
	}
	class, ok := c.Offerings(ctx, doc.Context.Api).Class(name, executors...)
	if !ok || class.Summary() == "" {
		return "", false
	}
	return fmt.Sprintf("**%s** resource class\n\n%s", name, class.Summary()), true
}

// resourceClassAt is the value of the resource_class key the cursor is on the
// value of, and the line the key is on.
func resourceClassAt(doc yamlparser.YamlDocument, pos protocol.Position) (string, uint32, bool) {
	if doc.RootNode == nil {
		return "", 0, false
	}
	offset := uint(position.ToIndex(pos, doc.Content))
	pair := enclosingPair(doc.RootNode.NamedDescendantForByteRange(offset, offset))
	if pair == nil {
		return "", 0, false
	}
	key, value := pair.ChildByFieldName("key"), pair.ChildByFieldName("value")
	if key == nil || value == nil || keyName(doc.GetNodeText(key)) != "resource_class" ||
		offset < value.StartByte() || offset > value.EndByte() {
		return "", 0, false
	}
	return keyName(doc.GetNodeText(value)), uint32(key.StartPosition().Row), true
}

// executorGivingResourceClassOn is the executor whose resource class is given
// on line: a named executor, or the one a job runs on, whether the job gives
// the class or its executor in place does.
func executorGivingResourceClassOn(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, line uint32) (ast.Executor, bool) {
	givesOn := func(rng protocol.Range) bool {
		return !position.IsDefaultRange(rng) && rng.Start.Line == line
	}

	for _, executor := range doc.Executors {
		if givesOn(resourceClassRange(executor)) {
			return executor, true
		}
	}
	for _, job := range doc.Jobs {
		executor, ok := doc.JobExecutor(ctx, job, c)
		if !ok {
			continue
		}
		if givesOn(job.ResourceClassRange) || (position.InRange(job.Range, protocol.Position{Line: line}) && givesOn(resourceClassRange(executor))) {
			return executor, true
		}
	}
	return nil, false
}

// resourceClassRange is where an executor gives its resource class.
func resourceClassRange(executor ast.Executor) protocol.Range {
	switch executor := executor.(type) {
	case ast.DockerExecutor:
		return executor.ResourceClassRange
	case ast.MachineExecutor:
		return executor.ResourceClassRange
	case ast.MacOSExecutor:
		return executor.ResourceClassRange
	}
	return protocol.Range{}
}
