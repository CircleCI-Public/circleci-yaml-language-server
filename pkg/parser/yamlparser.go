package parser

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func ParseFile(content []byte, context *utils.LsContext) YamlDocument {
	rootNode := GetRootNode(content)

	doc := YamlDocument{
		Content:            content,
		Context:            context,
		RootNode:           rootNode,
		Commands:           make(map[string]ast.Command),
		Orbs:               make(map[string]ast.Orb),
		Jobs:               make(map[string]ast.Job),
		Workflows:          make(map[string]ast.Workflow),
		Executors:          make(map[string]ast.Executor),
		PipelineParameters: make(map[string]ast.Parameter),
		Diagnostics:        &[]protocol.Diagnostic{},

		LocalOrbInfo: make(map[string]*ast.OrbInfo),
	}

	return doc
}

func (doc *YamlDocument) ParseYAML(context *utils.LsContext, offset protocol.Position) {
	if len(*doc.Diagnostics) > 0 {
		return
	}
	doc.Offset = offset
	blockMappingNode := GetBlockMappingNode(doc.RootNode)
	doc.YamlAnchors = ParseYamlAnchors(doc)

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)

		switch keyName {
		case "version":
			if valueNode != nil {
				doc.parseVersion(valueNode)
			}

			doc.VersionRange = doc.NodeToRange(child)

		case "orbs":
			if valueNode != nil {
				doc.OrbsRange = doc.NodeToRange(valueNode)
				doc.parseOrbs(valueNode)
			} else {
				doc.OrbsRange = doc.NodeToRange(child)
			}

		case "commands":
			if valueNode != nil {
				doc.CommandsRange = doc.NodeToRange(valueNode)
				doc.parseCommands(valueNode)
			} else {
				doc.CommandsRange = doc.NodeToRange(child)
			}

		case "jobs":
			if valueNode == nil {
				break
			}

			doc.JobsRange = doc.NodeToRange(valueNode)
			doc.parseJobs(valueNode)

		case "workflows":
			if valueNode == nil {
				break
			}

			doc.WorkflowRange = doc.NodeToRange(valueNode)
			doc.parseWorkflows(valueNode)

		case "executors":
			if valueNode != nil {
				doc.ExecutorsRange = doc.NodeToRange(valueNode)
				doc.parseExecutors(valueNode)
			} else {
				doc.ExecutorsRange = doc.NodeToRange(child)
			}

		case "description":
			if valueNode == nil {
				break
			}

			doc.Description = doc.GetNodeText(valueNode)

		case "parameters":
			if valueNode != nil {
				doc.PipelineParametersRange = doc.NodeToRange(valueNode)
				doc.PipelineParameters = doc.parseParameters(valueNode)
			} else {
				doc.PipelineParametersRange = doc.NodeToRange(child)
			}
		}
	})

	doc.assignContexts()
}

func (doc *YamlDocument) ValidateYAML() {
	rootNode := doc.RootNode

	ExecQuery(rootNode, "(ERROR) @flows", func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := capture.Node
			diagnostic := utils.CreateErrorDiagnosticFromNode(node, "Error! Please fix you yaml file")
			doc.addDiagnostic(diagnostic)
		}
	})

	// rootNode should be of type "stream"
	if document := GetChildOfType(rootNode, "document"); document == nil {
		diagnostic := utils.CreateErrorDiagnosticFromNode(rootNode, "Invalid yaml file")
		doc.addDiagnostic(diagnostic)
	}
}

func ParseFromURI(URI protocol.URI, context *utils.LsContext) (YamlDocument, error) {
	content, err := os.ReadFile(URI.Filename())
	if err != nil {
		return YamlDocument{}, err
	}
	doc, err := ParseFromContent([]byte(content), context, URI, protocol.Position{})

	return doc, err
}

func ParseFromUriWithCache(URI protocol.URI, cache *utils.Cache, context *utils.LsContext) (YamlDocument, error) {
	cachedFile := cache.FileCache.GetFile(URI)

	if cachedFile == nil {
		return YamlDocument{}, fmt.Errorf("file hasn't been opened: %s", URI.Filename())
	}

	content := []byte(cachedFile.TextDocument.Text)

	doc, err := ParseFromContent(content, context, URI, protocol.Position{})

	return doc, err
}

func ParseFromContent(content []byte, context *utils.LsContext, URI protocol.URI, offset protocol.Position) (YamlDocument, error) {
	doc := ParseFile([]byte(content), context)
	doc.URI = URI

	doc.ValidateYAML()
	doc.ParseYAML(context, offset)

	return doc, nil
}

type YamlAnchor struct {
	Name            string
	DefinitionRange protocol.Range
	References      *[]protocol.Range
	ValueNode       *sitter.Node
}

type YamlDocument struct {
	Content        []byte
	RootNode       *sitter.Node
	Version        float32
	Description    string
	URI            protocol.URI
	Diagnostics    *[]protocol.Diagnostic
	Context        *utils.LsContext
	SchemaLocation string

	Orbs               map[string]ast.Orb
	LocalOrbs          []LocalOrb
	Executors          map[string]ast.Executor
	Commands           map[string]ast.Command
	Jobs               map[string]ast.Job
	Workflows          map[string]ast.Workflow
	PipelineParameters map[string]ast.Parameter
	YamlAnchors        map[string]YamlAnchor

	OrbsRange               protocol.Range
	ExecutorsRange          protocol.Range
	CommandsRange           protocol.Range
	JobsRange               protocol.Range
	WorkflowRange           protocol.Range
	PipelineParametersRange protocol.Range
	VersionRange            protocol.Range

	LocalOrbInfo map[string]*ast.OrbInfo

	LocalOrbName string
	Offset       protocol.Position
}

func (doc *YamlDocument) IsBuiltIn(commandName string) bool {
	builtInCommands := []string{
		"run",
		"checkout",
		"setup_remote_docker",
		"save_cache",
		"restore_cache",
		"store_artifacts",
		"store_test_results",
		"persist_to_workspace",
		"attach_workspace",
		"add_ssh_keys",
		"steps",
		"when",   // Has nothing to do here, tech debt to resolve
		"unless", // Has nothing to do here, tech debt to resolve
	}

	return utils.FindInArray(builtInCommands, commandName) != -1
}

func (doc *YamlDocument) IsOrbReference(orbReference string) bool {
	splittedCommand := strings.Split(orbReference, "/")

	if len(splittedCommand) != 2 {
		return false
	}

	orbName := splittedCommand[0]
	_, ok := doc.Orbs[orbName]

	return ok
}

func (doc *YamlDocument) CouldBeOrbReference(orbReference string) (string, bool) {
	splittedCommand := strings.Split(orbReference, "/")

	if len(splittedCommand) != 2 {
		return "", false
	}

	return splittedCommand[0], true
}

func (doc *YamlDocument) IsOrbCommand(orbCommand string, cache *utils.Cache) bool {
	splittedCommand := strings.Split(orbCommand, "/")

	if len(splittedCommand) != 2 {
		return false
	}

	orbName := splittedCommand[0]
	commandName := splittedCommand[1]

	orbInfo, err := doc.GetOrbInfoFromName(orbName, cache)

	if err != nil || orbInfo == nil {
		return false
	}

	_, ok := orbInfo.Commands[commandName]

	return ok
}

func (doc *YamlDocument) IsOrbJob(orbCommand string, cache *utils.Cache) bool {
	splittedCommand := strings.Split(orbCommand, "/")

	if len(splittedCommand) != 2 {
		return false
	}

	orbName := splittedCommand[0]
	commandName := splittedCommand[1]

	orbInfo, err := doc.GetOrbInfoFromName(orbName, cache)

	if err != nil || orbInfo == nil {
		return false
	}

	_, ok := orbInfo.Jobs[commandName]

	return ok
}

func (doc *YamlDocument) IsGivenOrb(commandName string, orbName string) bool {
	if !doc.IsOrbReference(commandName) {
		return false
	}

	splittedCommand := strings.Split(commandName, "/")

	return splittedCommand[0] == orbName
}

func (doc *YamlDocument) IsAlias(commandName string) bool {
	return strings.HasPrefix(commandName, "*")
}

func (doc *YamlDocument) DoesJobExist(jobName string) bool {
	_, ok := doc.Jobs[jobName]
	return ok
}

func (doc *YamlDocument) DoesCommandExist(commandName string) bool {
	_, ok := doc.Commands[commandName]
	return ok
}

func (doc *YamlDocument) DoesExecutorExist(executorName string) bool {
	_, ok := doc.Executors[executorName]
	return ok
}

func (doc *YamlDocument) DoesWorkflowExist(workflowName string) bool {
	_, ok := doc.Workflows[workflowName]
	return ok
}

func (doc *YamlDocument) GetWorkflows() []ast.TextAndRange {
	workflows := doc.Workflows

	workflowRes := []ast.TextAndRange{}
	for _, workflow := range workflows {
		workflowRes = append(workflowRes, ast.TextAndRange{
			Text: workflow.Name,
			Range: protocol.Range{
				Start: workflow.NameRange.Start,
				End:   workflow.NameRange.End,
			},
		})
	}

	return workflowRes
}

func (doc *YamlDocument) parseVersion(versionNode *sitter.Node) {
	parsedVersion, err := strconv.ParseFloat(doc.GetNodeText(versionNode), 32)
	if err != nil {
		return
	}
	doc.Version = float32(parsedVersion)
}

func (doc *YamlDocument) addDiagnostic(diagnostic protocol.Diagnostic) {
	*doc.Diagnostics = append(*doc.Diagnostics, diagnostic)
}

func (doc *YamlDocument) InsertText(pos protocol.Position, text string) (YamlDocument, error) {
	content := doc.Content
	posIdx := utils.PosToIndex(pos, content)
	newContent := ""

	for i, r := range content {
		if i == posIdx {
			newContent += text
		}
		newContent += string(r)
	}

	return ParseFromContent([]byte(newContent), doc.Context, doc.URI, doc.Offset)
}

type ModifiedYamlDocument struct {
	// The modified YAML Document
	Document YamlDocument

	// A short slug-like description of the way the document was modifier
	Tag string

	// Content added to the document
	Diff string
}

func (doc *YamlDocument) ModifyTextForAutocomplete(pos protocol.Position) []ModifiedYamlDocument {
	node, _, err := utils.NodeAtPos(doc.RootNode, pos)
	if err != nil {
		return []ModifiedYamlDocument{
			{
				Document: *doc,
				Tag:      "original",
			},
		}
	}

	res := []ModifiedYamlDocument{}

	if node.Parent().Type() == "double_quote_scalar" {
		// Fixes a crash, investigate later
		// Autocompletion still works fine.
		return []ModifiedYamlDocument{
			{
				Document: *doc,
				Tag:      "original",
			},
		}
	}

	text := doc.GetNodeText(node)

	test1, err := doc.InsertText(pos, "- a: 1")
	if err == nil && len(*test1.Diagnostics) == 0 && strings.TrimSpace(text)[0] != '-' {
		res = append(res, ModifiedYamlDocument{
			Document: test1,
			Tag:      "edit-item",
			Diff:     "- a: 1",
		})
	}

	test2, err := doc.InsertText(pos, "a: 1")
	if err == nil && len(*test2.Diagnostics) == 0 {
		res = append(res, ModifiedYamlDocument{
			Document: test2,
			Tag:      "edit-key",
			Diff:     "a: 1",
		})
	}

	test3, err := doc.InsertText(pos, "a")
	if err == nil && len(*test3.Diagnostics) == 0 {
		res = append(res, ModifiedYamlDocument{
			Document: test3,
			Tag:      "edit-value",
			Diff:     "a",
		})
	}

	res = append(res, ModifiedYamlDocument{
		Document: *doc,
		Tag:      "original",
	})

	return res
}

func (doc *YamlDocument) DoesCommandOrJobOrExecutorExist(name string, includeCommands bool) bool {
	if _, ok := doc.Jobs[name]; ok {
		return true
	}

	if _, ok := doc.Commands[name]; ok && includeCommands {
		return true
	}

	if _, ok := doc.Executors[name]; ok {
		return true
	}

	if doc.IsOrbReference(name) {
		return true
	}

	return false
}

func (doc *YamlDocument) GetParamsWithPosition(position protocol.Position) map[string]ast.Parameter {
	if utils.PosInRange(doc.CommandsRange, position) {
		for _, command := range doc.Commands {
			if utils.PosInRange(command.Range, position) {
				return command.Parameters
			}
		}
	}

	if utils.PosInRange(doc.JobsRange, position) {
		for _, job := range doc.Jobs {
			if utils.PosInRange(job.Range, position) {
				return job.Parameters
			}
		}
	}

	if utils.PosInRange(doc.OrbsRange, position) {
		for _, orb := range doc.Orbs {
			if !orb.Url.IsLocal {
				continue
			}

			if !utils.PosInRange(orb.Range, position) {
				continue
			}

			orbInfo := doc.LocalOrbInfo[orb.Name]

			return GetOrbParameters(orbInfo, position)
		}

		return map[string]ast.Parameter{}
	}

	if utils.PosInRange(doc.ExecutorsRange, position) {
		for _, executor := range doc.Executors {
			if utils.PosInRange(executor.GetRange(), position) {
				return executor.GetParameters()
			}
		}
	}

	return map[string]ast.Parameter{}
}

func GetOrbParameters(orb *ast.OrbInfo, position protocol.Position) map[string]ast.Parameter {
	if utils.PosInRange(orb.CommandsRange, position) {
		for _, command := range orb.Commands {
			if utils.PosInRange(command.Range, position) {
				return command.Parameters
			}
		}
	}

	if utils.PosInRange(orb.JobsRange, position) {
		for _, job := range orb.Jobs {
			if utils.PosInRange(job.Range, position) {
				return job.Parameters
			}
		}
	}

	if utils.PosInRange(orb.ExecutorsRange, position) {
		for _, executor := range orb.Executors {
			if utils.PosInRange(executor.GetRange(), position) {
				return executor.GetParameters()
			}
		}
	}

	return map[string]ast.Parameter{}
}

func (doc *YamlDocument) GetExecutorDefinedAtPosition(position protocol.Position) ast.Executor {
	for _, executor := range doc.Executors {
		if utils.PosInRange(executor.GetRange(), position) {
			return executor
		}
	}

	return ast.BaseExecutor{}
}

func (doc *YamlDocument) GetDefinedParams(entityName string, cache *utils.Cache) map[string]ast.Parameter {
	var definedParams map[string]ast.Parameter

	if command, ok := doc.Commands[entityName]; ok {
		definedParams = command.Parameters
	}

	if job, ok := doc.Jobs[entityName]; ok {
		definedParams = job.Parameters
	}

	if doc.IsOrbCommand(entityName, cache) || doc.IsOrbJob(entityName, cache) {
		return doc.GetOrbDefinedParams(entityName, cache)
	}

	return definedParams
}

func (doc *YamlDocument) GetOrbDefinedParams(entityName string, cache *utils.Cache) map[string]ast.Parameter {
	var definedParams map[string]ast.Parameter

	splittedName := strings.Split(entityName, "/")
	orbName := splittedName[0]
	commandOrJob := splittedName[1]

	orbInfo, err := doc.GetOrFetchOrbInfo(doc.Orbs[orbName], cache)
	if err != nil {
		return definedParams
	}

	if command, ok := orbInfo.Commands[commandOrJob]; ok {
		definedParams = command.Parameters
	}

	if job, ok := orbInfo.Jobs[commandOrJob]; ok {
		definedParams = job.Parameters
	}

	return definedParams
}

func (doc *YamlDocument) ToOrbParsedAttributes() ast.OrbParsedAttributes {
	return ast.OrbParsedAttributes{
		Commands:           doc.Commands,
		Jobs:               doc.Jobs,
		Executors:          doc.Executors,
		PipelineParameters: doc.PipelineParameters,

		ExecutorsRange:          doc.ExecutorsRange,
		CommandsRange:           doc.CommandsRange,
		JobsRange:               doc.JobsRange,
		PipelineParametersRange: doc.PipelineParametersRange,
		WorkflowRange:           doc.WorkflowRange,
		OrbsRange:               doc.OrbsRange,
	}
}

func (doc *YamlDocument) FromOrbParsedAttributesToYamlDocument(orb ast.OrbParsedAttributes) YamlDocument {
	return YamlDocument{
		LocalOrbName: orb.Name,

		RootNode: doc.RootNode,

		Commands:           orb.Commands,
		Jobs:               orb.Jobs,
		Executors:          orb.Executors,
		PipelineParameters: orb.PipelineParameters,

		ExecutorsRange:          orb.ExecutorsRange,
		CommandsRange:           orb.CommandsRange,
		JobsRange:               orb.JobsRange,
		PipelineParametersRange: orb.PipelineParametersRange,
		WorkflowRange:           orb.WorkflowRange,
		OrbsRange:               orb.OrbsRange,
		Content:                 doc.Content,
	}
}
