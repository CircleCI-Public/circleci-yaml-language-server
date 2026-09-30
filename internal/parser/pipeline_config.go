package parser

import (
	"path"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// pipelineConfigKeys are the top-level keys that only pipeline config has.
var pipelineConfigKeys = map[string]bool{
	"version":    true,
	"setup":      true,
	"orbs":       true,
	"executors":  true,
	"commands":   true,
	"jobs":       true,
	"job-groups": true,
	"workflows":  true,
	"parameters": true,
	"functions":  true,
}

// IsPipelineConfig reports whether the document is pipeline config. The
// client sends every YAML file under .circleci/, which also holds files for
// other tools, such as test-suites.yml for Smarter Testing. A mapping with
// none of the pipeline's top-level keys is one of those, unless the file is
// named config.yml, which is config however unfinished it is. A ytt template
// isn't config until ytt renders it, whatever its name.
func (doc *YamlDocument) IsPipelineConfig() bool {
	if doc.isYttTemplate() {
		return false
	}

	switch path.Base(string(doc.URI)) {
	case "config.yml", "config.yaml":
		return true
	}

	// Anything but a mapping, an empty file included, is left to the schema
	// to report.
	blockMappingNode := GetBlockMappingNode(doc.RootNode)
	if blockMappingNode == nil {
		return true
	}

	found := false
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, _ := doc.GetKeyValueNodes(child)
		if keyNode != nil && pipelineConfigKeys[doc.GetNodeText(keyNode)] {
			found = true
		}
	})

	return found
}

// isYttTemplate reports whether the document has a ytt annotation, a comment
// starting `#@`, such as `#@ load("@ytt:data", "data")` or
// `key: #@ data.values.key`.
func (doc *YamlDocument) isYttTemplate() bool {
	found := false
	commentsQuery.Run(doc.RootNode, func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			if strings.HasPrefix(doc.GetNodeText(&capture.Node), "#@") {
				found = true
			}
		}
	})
	return found
}

// IsUnderUnreadTopLevelKey reports whether node sits under a top-level key
// the compiler doesn't read, such as one that only holds anchors for use
// elsewhere. The compiler only reads that content where an alias brings it
// into a job or a command.
func (doc *YamlDocument) IsUnderUnreadTopLevelKey(node *sitter.Node) bool {
	start, end := node.StartByte(), node.EndByte()
	for _, rng := range doc.unreadRanges {
		if start >= rng[0] && end <= rng[1] {
			return true
		}
	}
	return false
}

// unreadTopLevelRanges finds the byte ranges of the top-level pairs the
// compiler doesn't read.
func (doc *YamlDocument) unreadTopLevelRanges() [][2]uint {
	rootMapping := GetBlockMappingNode(doc.RootNode)
	if rootMapping == nil {
		return nil
	}

	var ranges [][2]uint
	for i := uint(0); i < rootMapping.NamedChildCount(); i++ {
		pair := rootMapping.NamedChild(i)
		if pair.Kind() != "block_mapping_pair" {
			continue
		}
		if keyNode, _ := doc.GetKeyValueNodes(pair); keyNode != nil && !pipelineConfigKeys[doc.GetNodeText(keyNode)] {
			ranges = append(ranges, [2]uint{pair.StartByte(), pair.EndByte()})
		}
	}
	return ranges
}

// HasNoWorkflows reports whether the config leaves out `workflows`, or gives
// it no value. The compiler then makes a workflow that runs the job named
// build.
func (doc *YamlDocument) HasNoWorkflows() bool {
	return position.IsDefaultRange(doc.WorkflowRange)
}
