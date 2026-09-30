package parser

import (
	"cmp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// parseAlias reads an entry of `commands`, `jobs` or `executors` whose value
// is a string, such as `build: orb/build`, as an alias. A value of any other
// type, such as a number or a mapping, is not one.
func (doc *YamlDocument) parseAlias(entryNode *sitter.Node) (ast2.Alias, bool) {
	keyNode, valueNode := doc.GetKeyValueNodes(entryNode)
	if keyNode == nil || valueNode == nil {
		return ast2.Alias{}, false
	}

	scalar := doc.scalarOf(valueNode, 0)
	switch scalar.Kind() {
	case "double_quote_scalar", "single_quote_scalar":
	case "plain_scalar":
		if child := GetFirstChild(scalar); child == nil || child.Kind() != "string_scalar" {
			return ast2.Alias{}, false
		}
	default:
		return ast2.Alias{}, false
	}

	return ast2.Alias{
		Name:        doc.getAttributeName(doc.GetNodeText(keyNode)),
		NameRange:   doc.NodeToRange(keyNode),
		Target:      doc.ScalarText(valueNode),
		TargetRange: doc.NodeToRange(valueNode),
		Range:       doc.NodeToRange(entryNode),
	}, true
}

// ResolveCommand returns the command a step names: the config's own, an
// orb's as `orb-alias/command-name`, or either through an alias.
func (doc *YamlDocument) ResolveCommand(name string, cache *cache.Cache) (ast2.Command, bool) {
	if alias, ok := doc.CommandAlias(name); ok {
		name = alias.Target
	}
	return resolveElement(doc, name, cache, func(attributes ast2.OrbParsedAttributes) map[string]ast2.Command {
		return attributes.Commands
	})
}

// ResolveJob returns the job a workflow names: the config's own, an orb's as
// `orb-alias/job-name`, or either through an alias.
func (doc *YamlDocument) ResolveJob(name string, cache *cache.Cache) (ast2.Job, bool) {
	if alias, ok := doc.JobAlias(name); ok {
		name = alias.Target
	}
	return resolveElement(doc, name, cache, func(attributes ast2.OrbParsedAttributes) map[string]ast2.Job {
		return attributes.Jobs
	})
}

// ResolveExecutor returns the executor a job names: the config's own, an
// orb's as `orb-alias/executor-name`, or either through an alias.
func (doc *YamlDocument) ResolveExecutor(name string, cache *cache.Cache) (ast2.Executor, bool) {
	if alias, ok := doc.ExecutorAlias(name); ok {
		name = alias.Target
	}
	return resolveElement(doc, name, cache, func(attributes ast2.OrbParsedAttributes) map[string]ast2.Executor {
		return attributes.Executors
	})
}

// resolveElement looks a name up among the config's elements of one kind,
// or, for `orb-alias/element-name`, among the orb's.
func resolveElement[T any](doc *YamlDocument, name string, cache *cache.Cache, elements func(ast2.OrbParsedAttributes) map[string]T) (T, bool) {
	if element, ok := elements(doc.ToOrbParsedAttributes())[name]; ok {
		return element, true
	}

	var none T
	orbName, elementName, ok := (ast2.Alias{Target: name}).OrbTarget()
	if !ok {
		return none, false
	}
	orbInfo, err := doc.GetOrbInfoFromName(orbName, cache)
	if err != nil || orbInfo == nil {
		return none, false
	}
	element, ok := elements(orbInfo.OrbParsedAttributes)[elementName]
	return element, ok
}

// DockerResourceClass is the resource class of a job on the Docker executor,
// given in place or through the executor it names. It is false for a job on
// another executor, and for a class that can't be read: a parameter, or a
// self-hosted runner's.
func (doc *YamlDocument) DockerResourceClass(job ast2.Job, cache *cache.Cache) (string, bool) {
	class := ""
	if !position.IsDefaultRange(job.DockerRange) {
		class = job.Docker.ResourceClass
	} else {
		if job.Executor == "" || paramref.ContainsReference(job.Executor) {
			return "", false
		}
		executor, ok := doc.ResolveExecutor(job.Executor, cache)
		docker, isDocker := executor.(ast2.DockerExecutor)
		if !ok || !isDocker {
			return "", false
		}
		class = cmp.Or(job.ResourceClass, docker.ResourceClass)
	}

	if paramref.ContainsReference(class) || strings.Contains(class, "/") {
		return "", false
	}
	// The class a Docker job runs on when it gives none.
	return cmp.Or(class, "medium"), true
}
