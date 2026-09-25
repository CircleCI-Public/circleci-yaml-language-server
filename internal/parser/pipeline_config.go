package parser

import (
	"path"

	sitter "github.com/tree-sitter/go-tree-sitter"
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
// named config.yml, which is config however unfinished it is.
func (doc *YamlDocument) IsPipelineConfig() bool {
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
