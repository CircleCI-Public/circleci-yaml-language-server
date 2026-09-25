package parser

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

// ParseFile parses a config. The document owns the tree its nodes belong to:
// close it once nothing reads RootNode, or anything found under it, any more.
func ParseFile(content []byte, context *session.Settings) YamlDocument {
	tree := yamltree.Parse(content)

	doc := YamlDocument{
		Content:            content,
		Context:            context,
		tree:               tree,
		RootNode:           tree.Root(),
		Commands:           make(map[string]ast2.Command),
		Orbs:               make(map[string]ast2.Orb),
		Jobs:               make(map[string]ast2.Job),
		JobGroups:          make(map[string]ast2.JobGroup),
		Workflows:          make(map[string]ast2.Workflow),
		Executors:          make(map[string]ast2.Executor),
		PipelineParameters: make(map[string]ast2.Parameter),
		Diagnostics:        &[]protocol.Diagnostic{},

		LocalOrbInfo: make(map[string]*ast2.OrbInfo),
	}

	return doc
}

func (doc *YamlDocument) ParseYAML(context *session.Settings, offset protocol.Position) {
	if len(*doc.Diagnostics) > 0 {
		return
	}
	doc.Offset = offset
	blockMappingNode := GetBlockMappingNode(doc.RootNode)
	doc.YamlAnchors = ParseYamlAnchors(doc)

	doc.SuppressionInfo = ParseSuppressionComments(doc)

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)

		switch keyName {
		case "version":
			if valueNode != nil {
				doc.parseVersion(valueNode)
			}

			doc.VersionRange = doc.NodeToRange(child)

		case "setup":
			text := strings.TrimSpace(doc.GetNodeText(valueNode))
			if len(text) != 0 && text != "false" {
				doc.Setup = true
			}
			doc.SetupRange = doc.NodeToRange(child)

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

		case "job-groups":
			if valueNode == nil {
				break
			}
			doc.JobGroupsRange = doc.NodeToRange(valueNode)
			doc.parseJobGroups(valueNode)

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

var errorsQuery = yamltree.MustCompileQuery("(ERROR) @flows")

func (doc *YamlDocument) ValidateYAML() {
	rootNode := doc.RootNode

	errorsQuery.Run(rootNode, func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := &capture.Node
			diag := diagnostic.ErrorFromNode(node, "Error! Please fix your yaml file")
			doc.addDiagnostic(diag)
		}
	})

	// rootNode should be of type "stream"
	if document := GetChildOfType(rootNode, "document"); document == nil {
		diag := diagnostic.ErrorFromNode(rootNode, "Invalid yaml file")
		doc.addDiagnostic(diag)
	}
}

func ParseFromURI(URI uri.URI, context *session.Settings) (YamlDocument, error) {
	content, err := os.ReadFile(URI.FsPath())
	if err != nil {
		return YamlDocument{}, err
	}
	doc, err := ParseFromContent([]byte(content), context, URI, protocol.Position{})

	return doc, err
}

var ErrCacheMissing = errors.New("file not found in cache")

func ParseFromUriWithCache(URI uri.URI, cache *cache.Cache, context *session.Settings) (YamlDocument, error) {
	cachedFile := cache.FileCache.GetFile(URI)

	if cachedFile == nil {
		return YamlDocument{}, fmt.Errorf("%w: %s", ErrCacheMissing, URI.FsPath())
	}

	content := []byte(cachedFile.TextDocument.Text)

	doc, err := ParseFromContent(content, context, URI, protocol.Position{})

	return doc, err
}

func ParseFromContent(content []byte, context *session.Settings, URI uri.URI, offset protocol.Position) (YamlDocument, error) {
	doc := ParseFile([]byte(content), context)
	doc.URI = URI

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
	Content []byte
	// tree owns RootNode and every node under it. Copies of a document share
	// it, and it is freed by Close.
	tree           *yamltree.Tree
	RootNode       *sitter.Node
	Version        float32
	Description    string
	URI            uri.URI
	Diagnostics    *[]protocol.Diagnostic
	Context        *session.Settings
	SchemaLocation string

	Setup              bool
	Orbs               map[string]ast2.Orb
	LocalOrbs          []LocalOrb
	Executors          map[string]ast2.Executor
	Commands           map[string]ast2.Command
	Jobs               map[string]ast2.Job
	JobGroups          map[string]ast2.JobGroup
	Workflows          map[string]ast2.Workflow
	PipelineParameters map[string]ast2.Parameter
	YamlAnchors        map[string]YamlAnchor

	SetupRange              protocol.Range
	OrbsRange               protocol.Range
	ExecutorsRange          protocol.Range
	CommandsRange           protocol.Range
	JobsRange               protocol.Range
	JobGroupsRange          protocol.Range
	WorkflowRange           protocol.Range
	PipelineParametersRange protocol.Range
	VersionRange            protocol.Range

	LocalOrbInfo map[string]*ast2.OrbInfo

	LocalOrbName string
	Offset       protocol.Position

	SuppressionInfo *SuppressionInfo
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
		"deploy",
		"when",   // Has nothing to do here, tech debt to resolve
		"unless", // Has nothing to do here, tech debt to resolve
	}

	return slices.Contains(builtInCommands, commandName)
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

// Takes the name of anything that may be in an orb and returns if it is inside an orb that we can not use
func (doc *YamlDocument) IsFromUnfetchableOrb(name string) bool {
	components := strings.Split(name, "/")
	if len(components) != 2 {
		return false
	}

	orb, ok := doc.Orbs[components[0]]
	if !ok {
		return false
	}

	if orb.IsPlaceholder || orb.Url.IsURL {
		return true
	}

	hasParamInTag, _ := paramref.IsPartiallyReferenced(orb.Url.Version)
	return hasParamInTag
}

func (doc *YamlDocument) IsOrbCommand(orbCommand string, cache *cache.Cache) bool {
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

func (doc *YamlDocument) IsOrbJob(orbCommand string, cache *cache.Cache) bool {
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

func (doc *YamlDocument) DoesJobGroupExist(jobGroupName string) bool {
	_, ok := doc.JobGroups[jobGroupName]
	return ok
}

// FindJobGroupContainingJob returns the name of the job-group that contains
// a job invocation with the given jobName. Returns ("", false) if no group
// contains it.
func (doc *YamlDocument) FindJobGroupContainingJob(jobName string) (string, bool) {
	for _, group := range doc.JobGroups {
		for _, inv := range group.JobInvocations {
			if inv.JobName == jobName {
				return group.Name, true
			}
		}
	}
	return "", false
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

func (doc *YamlDocument) GetWorkflows() []ast2.TextAndRange {
	workflows := doc.Workflows

	workflowRes := []ast2.TextAndRange{}
	for _, workflow := range workflows {
		workflowRes = append(workflowRes, ast2.TextAndRange{
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

func (doc *YamlDocument) addDiagnostic(diag protocol.Diagnostic) {
	*doc.Diagnostics = append(*doc.Diagnostics, diag)
}

// Close frees the document's syntax tree. RootNode, and every node found under
// it, must not be read afterwards. Closing a copy closes the tree they share,
// and closing again does nothing.
func (doc *YamlDocument) Close() {
	doc.tree.Close()
}

func (doc *YamlDocument) InsertText(pos protocol.Position, text string) (YamlDocument, error) {
	content := doc.Content
	posIdx := position.ToIndex(pos, content)

	// The text goes in before the character at the position, so at the very
	// end of the content, where there is none, it does not go in at all.
	newContent := slices.Clone(content)
	if posIdx < len(content) && utf8.RuneStart(content[posIdx]) {
		newContent = slices.Concat(content[:posIdx], []byte(text), content[posIdx:])
	}

	return ParseFromContent(newContent, doc.Context, doc.URI, doc.Offset)
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
	node, _, err := position.NodeAt(doc.RootNode, pos)
	if err != nil {
		return []ModifiedYamlDocument{
			{
				Document: *doc,
				Tag:      "original",
			},
		}
	}

	res := []ModifiedYamlDocument{}

	// The node at the position is the root when nothing narrower holds it, as
	// between the documents of a stream.
	if parent := node.Parent(); parent != nil && parent.Kind() == "double_quote_scalar" {
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

	// Each candidate is a document of its own. The ones kept belong to the
	// caller; the ones that do not parse cleanly are closed here, as nothing
	// else will ever see them.
	candidates := []struct {
		diff string
		tag  string
		// keep decides whether a candidate that parsed cleanly is offered.
		keep bool
	}{
		{"- a: 1", "edit-item", !strings.HasPrefix(strings.TrimSpace(text), "-")},
		{"a: 1", "edit-key", true},
		{"a", "edit-value", true},
	}

	for _, candidate := range candidates {
		edited, err := doc.InsertText(pos, candidate.diff)
		if err != nil {
			continue
		}
		if !candidate.keep || len(*edited.Diagnostics) != 0 {
			edited.Close()
			continue
		}
		res = append(res, ModifiedYamlDocument{
			Document: edited,
			Tag:      candidate.tag,
			Diff:     candidate.diff,
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

func (doc *YamlDocument) GetParamsWithPosition(pos protocol.Position) map[string]ast2.Parameter {
	if position.InRange(doc.CommandsRange, pos) {
		for _, command := range doc.Commands {
			if position.InRange(command.Range, pos) {
				return command.Parameters
			}
		}
	}

	if position.InRange(doc.JobsRange, pos) {
		for _, job := range doc.Jobs {
			if position.InRange(job.Range, pos) {
				return job.Parameters
			}
		}
	}

	if position.InRange(doc.OrbsRange, pos) {
		for _, orb := range doc.Orbs {
			if !orb.Url.IsLocal {
				continue
			}

			if !position.InRange(orb.Range, pos) {
				continue
			}

			orbInfo := doc.LocalOrbInfo[orb.Name]

			return GetOrbParameters(orbInfo, pos)
		}
	}

	if position.InRange(doc.ExecutorsRange, pos) {
		for _, executor := range doc.Executors {
			if position.InRange(executor.GetRange(), pos) {
				return executor.GetParameters()
			}
		}
	}

	return map[string]ast2.Parameter{}
}

func GetOrbParameters(orb *ast2.OrbInfo, pos protocol.Position) map[string]ast2.Parameter {
	if position.InRange(orb.CommandsRange, pos) {
		for _, command := range orb.Commands {
			if position.InRange(command.Range, pos) {
				return command.Parameters
			}
		}
	}

	if position.InRange(orb.JobsRange, pos) {
		for _, job := range orb.Jobs {
			if position.InRange(job.Range, pos) {
				return job.Parameters
			}
		}
	}

	if position.InRange(orb.ExecutorsRange, pos) {
		for _, executor := range orb.Executors {
			if position.InRange(executor.GetRange(), pos) {
				return executor.GetParameters()
			}
		}
	}

	return map[string]ast2.Parameter{}
}

func (doc *YamlDocument) GetExecutorDefinedAtPosition(pos protocol.Position) ast2.Executor {
	for _, executor := range doc.Executors {
		if position.InRange(executor.GetRange(), pos) {
			return executor
		}
	}

	return ast2.BaseExecutor{}
}

// EntityKind is what a name is used as: a step names a command, and a
// workflow's job entry names a job. A config may define a command and a job
// with the same name, so a lookup by name has to be told which it wants.
type EntityKind int

const (
	CommandEntity EntityKind = iota
	JobEntity
)

// GetDefinedParams returns the parameters of the command or job that
// entityName refers to, local or from an orb, preferring the kind it is used
// as. The other kind is only a fallback: a name used as the wrong kind is
// reported where its existence is checked.
func (doc *YamlDocument) GetDefinedParams(entityName string, kind EntityKind, cache *cache.Cache) map[string]ast2.Parameter {
	attributes := doc.ToOrbParsedAttributes()
	name := entityName

	if orbName, orbEntity, ok := strings.Cut(entityName, "/"); ok && !strings.Contains(orbEntity, "/") {
		orbInfo, err := doc.GetOrbInfoFromName(orbName, cache)
		if err == nil && orbInfo != nil {
			attributes = orbInfo.OrbParsedAttributes
			name = orbEntity
		}
	}

	command, isCommand := attributes.Commands[name]
	job, isJob := attributes.Jobs[name]

	switch {
	case isCommand && (kind == CommandEntity || !isJob):
		return command.Parameters
	case isJob:
		return job.Parameters
	}

	return nil
}

func (doc *YamlDocument) ToOrbParsedAttributes() ast2.OrbParsedAttributes {
	return ast2.OrbParsedAttributes{
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

func (doc *YamlDocument) FromOrbParsedAttributesToYamlDocument(orb ast2.OrbParsedAttributes) YamlDocument {
	return YamlDocument{
		URI:          orb.URI,
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
