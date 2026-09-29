package parser

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/xeipuuv/gojsonschema"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gopkg.in/yaml.v3"

	schema "github.com/CircleCI-Public/circleci-yaml-language-server"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamlbool"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

type JSONSchemaValidator struct {
	schema *gojsonschema.Schema
	keys   *schemaKeys
	Doc    YamlDocument
}

func (validator *JSONSchemaValidator) LoadJsonSchema(schemaLocation string) error {
	location := schemaURI(schemaLocation)
	loader := gojsonschema.NewReferenceLoader(location)

	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return err
	}

	validator.schema = schema
	if data, err := os.ReadFile(uri.URI(location).FsPath()); err == nil {
		validator.keys = newSchemaKeys(data)
	}

	return nil
}

// schemaURI is the location of a schema as a URI: a file URI as it is, and a
// path turned into one.
func schemaURI(location string) string {
	if unescaped, err := url.PathUnescape(location); err == nil {
		location = unescaped
	}
	if strings.HasPrefix(location, "file://") {
		return location
	}
	return string(uri.File(location))
}

// embeddedSchema is the schema built into the server, compiled the first time
// it is needed and shared after. Compiling it was a third of what diagnostics
// cost, and it cannot change while the server runs; validating against a
// schema only reads it, so every request can share one.
var embeddedSchema = sync.OnceValues(func() (*gojsonschema.Schema, error) {
	return gojsonschema.NewSchema(gojsonschema.NewBytesLoader(schema.EmbeddedSchemaJSON))
})

var embeddedSchemaKeys = sync.OnceValue(func() *schemaKeys {
	return newSchemaKeys(schema.EmbeddedSchemaJSON)
})

// LoadEmbeddedJsonSchema validates against the schema built into the server.
func (validator *JSONSchemaValidator) LoadEmbeddedJsonSchema() error {
	compiled, err := embeddedSchema()
	if err != nil {
		return err
	}

	validator.schema = compiled
	validator.keys = embeddedSchemaKeys()

	return nil
}

func (validator *JSONSchemaValidator) LoadJsonSchemaFromBytes(data []byte) error {
	loader := gojsonschema.NewBytesLoader(data)

	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return err
	}

	validator.schema = schema
	validator.keys = newSchemaKeys(data)

	return nil
}

// collectionKeyErrors reports each key that is a map or a list, which YAML
// allows but no config can use. Unquoted template syntax, such as
// `{{ .Name }}`, is the usual way to write one by accident.
func collectionKeyErrors(rootNode *sitter.Node) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}
	for node := range yamltree.Walk(rootNode) {
		var key *sitter.Node
		switch node.Kind() {
		case "block_mapping_pair", "flow_pair":
			key = node.ChildByFieldName("key")
		case "flow_node":
			// A key without a value, as in `{ a }`, is a flow_node of its
			// flow_mapping rather than a flow_pair.
			if parent := node.Parent(); parent != nil && parent.Kind() == "flow_mapping" {
				key = node
			}
		}
		if key == nil || key.NamedChildCount() == 0 {
			continue
		}

		switch key.NamedChild(0).Kind() {
		case "flow_mapping", "block_mapping":
			diagnostics = append(diagnostics, diagnostic.ErrorFromNode(key,
				"A map can't be used as a key; quote it if it's meant as text"))
		case "flow_sequence", "block_sequence":
			diagnostics = append(diagnostics, diagnostic.ErrorFromNode(key,
				"A list can't be used as a key; quote it if it's meant as text"))
		}
	}
	return diagnostics
}

func handleYAMLErrors(err string, content []byte, rootNode *sitter.Node) ([]protocol.Diagnostic, error) {
	diagnostics := []protocol.Diagnostic{}

	if strings.Contains(err, "yaml: unknown anchor") {
		anchorName := strings.Split(err, "'")[1]
		regex, _ := regexp.Compile(anchorName)
		res := regex.FindAllStringIndex(string(content), -1)

		if len(res) == 0 {
			return []protocol.Diagnostic{}, nil
		}

		for _, match := range res {
			rng := protocol.Range{
				Start: position.FromIndex(match[0], content),
				End:   position.FromIndex(match[1], content),
			}
			// The name is found wherever it is in the text, which can be
			// somewhere no node is, as where tree-sitter recovered from an
			// error.
			node, _, nodeErr := position.NodeAt(rootNode, rng.Start)
			if nodeErr == nil && node.Kind() == "alias_name" {
				diagnostics = append(diagnostics, diagnostic.Error(rng, err))
			}
		}
		return diagnostics, nil
	}

	if strings.Contains(err, "yaml: map merge requires map or sequence of maps as the value") {
		return []protocol.Diagnostic{}, nil
	}

	// The error has no position, and names the key as a Go value.
	if strings.HasPrefix(err, "yaml: invalid map key") {
		if diagnostics := collectionKeyErrors(rootNode); len(diagnostics) > 0 {
			return diagnostics, nil
		}
	}

	reError, _ := regexp.Compile(`(?s)^yaml: line (?P<Lines>\d+):\s(?P<Error>.+)`)

	if reError.MatchString(err) {
		info := reError.FindAllStringSubmatch(err, -1)[0]
		lineError := info[2]
		lineNumber, error := strconv.Atoi(info[1])

		// If, for some reason, the Atoi fail, we return the original error
		if error != nil {
			return []protocol.Diagnostic{diagnostic.ErrorFromNode(rootNode, err)}, nil
		}

		// The error counts lines from one.
		lineRange := position.LineContentRange(lineNumber-1, content)

		diag := diagnostic.Error(
			lineRange,
			diagnostic.Message(lineError),
		)

		return []protocol.Diagnostic{diag}, nil
	}

	reMultilineError, _ := regexp.Compile(`(?s)^yaml:\s(?P<Error>[\w\d\s]+):\n(?P<Lines>\s+line \d+:.+\n?)+`)

	if !reMultilineError.MatchString(err) {
		return []protocol.Diagnostic{diagnostic.ErrorFromNode(rootNode, err)}, nil
	}

	// For errors providing line numbers, we add a diagnostic on the
	// specified lines
	mes := reMultilineError.FindAllStringSubmatch(err, -1)[0]
	lines := strings.Split(mes[2], "\n")

	re := regexp.MustCompile(`^\s+line\s+(\d+):\s(.+)$`)

	lineIndexes := []int{}
	lineErrors := []string{}

	for _, line := range lines {
		info := re.FindAllStringSubmatch(line, -1)[0]

		lineError := info[2]
		lineNumber, error := strconv.Atoi(info[1])

		// If, for some reason, the Atoi fail, we return the original error
		if error != nil {
			return []protocol.Diagnostic{diagnostic.ErrorFromNode(rootNode, err)}, nil
		}

		lineIndexes = append(lineIndexes, lineNumber-1)
		lineErrors = append(lineErrors, lineError)
	}

	lineContentRanges := position.AllLineContentRange(lineIndexes, content)

	for i, lineContentRange := range lineContentRanges {
		lineError := lineErrors[i]

		if duplicateMergeKeyErrorRegex.MatchString(lineError) {
			continue
		}

		diag := diagnostic.Error(
			lineContentRange,
			lineError,
		)

		diagnostics = append(diagnostics, diag)
	}

	return diagnostics, nil
}

// ValidateWithJSONSchema checks a config against the JSON schema. It returns
// the schema's errors and, separately, go-yaml's errors decoding the config.
// When nothing could be decoded, it returns only go-yaml's.
func (validator *JSONSchemaValidator) ValidateWithJSONSchema(rootNode *sitter.Node, content []byte) (schemaDiagnostics, yamlDiagnostics []protocol.Diagnostic) {
	var file interface{}

	if err := decodeAsCompiled(content, &file); err != nil {
		yamlDiagnostics, _ = handleYAMLErrors(err.Error(), content, rootNode)
		if file == nil {
			return nil, yamlDiagnostics
		}
	}

	yamlLoader := gojsonschema.NewGoLoader(file)

	result, err := validator.schema.Validate(yamlLoader)
	if err != nil {
		// Should never happen
		return []protocol.Diagnostic{diagnostic.ErrorFromNode(rootNode, err.Error())}, yamlDiagnostics
	}

	jsonSchemaDiags := []protocol.Diagnostic{}

	if !result.Valid() {
		// Some errors are dropped, along with the combinator errors (oneOf,
		// if/then/else) they caused:
		//   - a value that is only a reference, such as << parameters.flag >>,
		//     is a string until the config is compiled, so the schema rejects
		//     it wherever it wants anything else;
		//   - the compiler drops the null-valued key of a flattened step
		//     before checking it, so it passes, and the validator warns
		//     about it instead.
		var combinators []protocol.Diagnostic
		var dropped []protocol.Range
		namePatterns := invalidNamePatterns(result.Errors())

		for _, resErr := range result.Errors() {
			if _, isNamePattern := namePatterns[resErr.Field()]; isNamePattern && resErr.Type() == "pattern" {
				continue
			}

			fields := strings.Split(resErr.Field(), ".")
			if len(fields) == 1 && fields[0] == "(root)" {
				diag := diagnostic.ErrorFromNode(rootNode, resErr.Description())
				jsonSchemaDiags = append(jsonSchemaDiags, diag)
				continue
			}

			node, err := FindDeepestNode(rootNode, content, fields)
			if err != nil {
				continue
			}

			if validator.doesNodeUseParameter(node) || validator.isInFlattenedStep(node) {
				dropped = append(dropped, validator.Doc.NodeToRange(node))
				continue
			}

			if name, ok := resErr.Details()["property"].(string); ok && resErr.Type() == "invalid_property_name" {
				if nameNode, err := FindDeepestNode(node, content, []string{name}); err == nil {
					node = nameNode.ChildByFieldName("key")
				}
				jsonSchemaDiags = append(jsonSchemaDiags, diagnostic.ErrorFromNode(node,
					invalidNameMessage(fields, name, namePatterns[resErr.Field()])))
				continue
			}

			if validator.isJobWithoutExecutor(resErr, fields, node, content) {
				jsonSchemaDiags = append(jsonSchemaDiags, diagnostic.ErrorFromNode(node.ChildByFieldName("key"),
					"A job needs an executor: give it one of `docker`, `machine`, `macos` or `executor`."))
				continue
			}

			if step, property, ok := ignoredStepProperty(resErr, fields); ok {
				if propertyNode, err := FindDeepestNode(node, content, []string{property}); err == nil {
					node = propertyNode.ChildByFieldName("key")
				}
				jsonSchemaDiags = append(jsonSchemaDiags, diagnostic.WarningFromNode(node,
					fmt.Sprintf("%s has no %s option, so this is ignored.", step, property)+
						validator.didYouMean(fields, property)))
				continue
			}

			if property, ok := resErr.Details()["property"].(string); ok && resErr.Type() == "additional_property_not_allowed" {
				node = keyNode(node, content, property)
				jsonSchemaDiags = append(jsonSchemaDiags, diagnostic.ErrorFromNode(node,
					fmt.Sprintf("`%s` isn't allowed here.", property)+validator.didYouMean(fields, property)))
				continue
			}

			if text, ok := validator.yaml11Boolean(node); ok && resErr.Type() == "invalid_type" {
				jsonSchemaDiags = append(jsonSchemaDiags, diagnostic.ErrorFromNode(node, fmt.Sprintf(
					"`%s` is read as a boolean; quote it if it's meant as text.", text)))
				continue
			}

			diag := diagnostic.ErrorFromNode(node, resErr.Description())
			if isCombinatorError(resErr) {
				combinators = append(combinators, diag)
				continue
			}

			jsonSchemaDiags = append(jsonSchemaDiags, diag)
		}

		// A combinator error only says that some schema inside it failed. When
		// that failure is reported within its range, it says nothing more.
		leaves := slices.Clone(jsonSchemaDiags)
		for _, combinator := range combinators {
			if onlyContainsDropped(combinator, dropped, leaves) ||
				hasAnotherDiagInsideRange(leaves, combinator.Range) {
				continue
			}

			jsonSchemaDiags = append(jsonSchemaDiags, combinator)
		}
	}

	return removeUselessMustValidateError(jsonSchemaDiags), yamlDiagnostics
}

// decodeAsCompiled decodes a config as the compiler reads it. The compiler's
// parser is YAML 1.1's, which reads a plain yes, no, on or off as a boolean.
func decodeAsCompiled(content []byte, out *interface{}) error {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return err
	}
	if root.Kind == 0 {
		return nil
	}
	resolveYAML11Booleans(&root)
	return root.Decode(out)
}

// resolveYAML11Booleans tags each plain value that is a boolean in YAML 1.1 as
// one. Keys are left as they are.
func resolveYAML11Booleans(node *yaml.Node) {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Style == 0 && node.ShortTag() == "!!str" && yamlbool.IsYAML11(node.Value) {
			node.Tag = "!!bool"
			node.Value = strconv.FormatBool(yamlbool.Value(strings.ToLower(node.Value)))
		}
	case yaml.MappingNode:
		for i := 1; i < len(node.Content); i += 2 {
			resolveYAML11Booleans(node.Content[i])
		}
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, child := range node.Content {
			resolveYAML11Booleans(child)
		}
	}
}

// yaml11Boolean is the text of a value that is a boolean only in YAML 1.1,
// given it or the pair it is the value of.
func (validator *JSONSchemaValidator) yaml11Boolean(node *sitter.Node) (string, bool) {
	if node.Kind() == "block_mapping_pair" || node.Kind() == "flow_pair" {
		_, node = validator.Doc.GetKeyValueNodes(node)
		if node == nil {
			return "", false
		}
	}
	text := validator.Doc.GetRawNodeText(node)
	return text, yamlbool.IsYAML11(text)
}

// openSteps are the built-in steps whose options the compiler doesn't close,
// so it passes any it doesn't know on to be ignored.
var openSteps = map[string]bool{
	"run":                  true,
	"checkout":             true,
	"attach_workspace":     true,
	"persist_to_workspace": true,
	"save_cache":           true,
	"restore_cache":        true,
	"store_artifacts":      true,
	"store_test_results":   true,
	"add_ssh_keys":         true,
	"setup_remote_docker":  true,
	"deploy":               true,
}

// ignoredStepProperty reports whether err is an option the schema doesn't
// know on one of the openSteps, returning the step and the option.
func ignoredStepProperty(err gojsonschema.ResultError, fields []string) (step, property string, ok bool) {
	if err.Type() != "additional_property_not_allowed" || len(fields) < 2 {
		return "", "", false
	}

	step = fields[len(fields)-1]
	if _, isIndex := strconv.Atoi(fields[len(fields)-2]); isIndex != nil || !openSteps[step] {
		return "", "", false
	}

	property, ok = err.Details()["property"].(string)
	return step, property, ok
}

// keyNode returns the key of property in the mapping node, or node itself when
// it can't be found.
func keyNode(node *sitter.Node, content []byte, property string) *sitter.Node {
	pair, err := FindDeepestNode(node, content, []string{property})
	if err != nil {
		return node
	}
	if key := pair.ChildByFieldName("key"); key != nil {
		return key
	}
	return node
}

// didYouMean suggests the key allowed at fields closest to property, if one
// is close enough to be what was meant.
func (validator *JSONSchemaValidator) didYouMean(fields []string, property string) string {
	if key := closestKey(property, validator.keys.at(fields)); key != "" {
		return fmt.Sprintf(" Did you mean `%s`?", key)
	}
	return ""
}

var executorKeys = []string{"docker", "machine", "macos", "executor"}

// isJobWithoutExecutor reports whether err is the schema saying a job needs
// one of the executorKeys, and the job has none of them. The schema only
// names one of them, which reads as if that one were the fix.
func (validator *JSONSchemaValidator) isJobWithoutExecutor(err gojsonschema.ResultError, fields []string, job *sitter.Node, content []byte) bool {
	property, _ := err.Details()["property"].(string)
	if err.Type() != "required" || !slices.Contains(executorKeys, property) ||
		len(fields) < 2 || fields[len(fields)-2] != "jobs" || job.ChildByFieldName("key") == nil {
		return false
	}

	_, value := validator.Doc.GetKeyValueNodes(job)
	if value == nil {
		return false
	}
	mapping := GetChildMapping(value)
	if mapping == nil {
		return false
	}

	return !slices.ContainsFunc(executorKeys, func(key string) bool {
		child, err := findChildNode(mapping, content, key)
		return err == nil && child != nil
	})
}

// invalidNamePatterns maps each field holding a name the schema rejects to
// the pattern names there must match. The schema reports a bad name twice,
// once naming it and once giving the pattern, both at the field.
func invalidNamePatterns(errs []gojsonschema.ResultError) map[string]string {
	patterns := map[string]string{}
	for _, err := range errs {
		if err.Type() == "invalid_property_name" {
			patterns[err.Field()] = ""
		}
	}
	for _, err := range errs {
		if _, ok := patterns[err.Field()]; ok && err.Type() == "pattern" {
			patterns[err.Field()] = fmt.Sprint(err.Details()["pattern"])
		}
	}
	return patterns
}

var nameRules = map[string]string{
	`^[a-z][a-z\d_-]*$`:         `start with a lowercase letter, and have only lowercase letters, digits, "_" and "-"`,
	`^[A-Za-z][A-Za-z\s\d_-]*$`: `start with a letter, and have only letters, digits, spaces, "_" and "-"`,
}

func invalidNameMessage(fields []string, name, pattern string) string {
	kind := strings.TrimSuffix(fields[len(fields)-1], "s")
	rule, ok := nameRules[pattern]
	if !ok {
		rule = "match " + pattern
	}
	return fmt.Sprintf("%q isn't a valid %s name: it must %s.", name, kind, rule)
}

// isCombinatorError reports whether err only says that a value failed one
// of several schemas, the actual failure being reported beside it.
func isCombinatorError(err gojsonschema.ResultError) bool {
	switch err.Type() {
	case "number_one_of", "number_any_of", "number_all_of", "condition_then", "condition_else":
		return true
	}

	return false
}

// onlyContainsDropped reports whether every failure inside the combinator
// error was dropped: there is one of those in its range, and no other error
// is.
func onlyContainsDropped(combinator protocol.Diagnostic, dropped []protocol.Range, others []protocol.Diagnostic) bool {
	containsDropped := slices.ContainsFunc(dropped, func(rng protocol.Range) bool {
		return position.InRange(combinator.Range, rng.Start)
	})
	if !containsDropped {
		return false
	}

	return !slices.ContainsFunc(others, func(other protocol.Diagnostic) bool {
		return position.InRange(combinator.Range, other.Range.Start)
	})
}

// doesNodeUseParameter reports whether the node's value is only a reference,
// such as << parameters.my_param >> or << pipeline.number >>, whose value the
// schema cannot check before the config is compiled.
//
// Example:
//
//	`when: << parameters.my_param >>`
//
//	But in the JSON Schema, the `when` key is defined as an object, so the validation
//	will fail if we don't ignore it.
func (validator *JSONSchemaValidator) doesNodeUseParameter(node *sitter.Node) bool {
	if node.Kind() == "block_mapping_pair" || node.Kind() == "flow_pair" {
		_, valueNode := validator.Doc.GetKeyValueNodes(node)
		if valueNode == nil {
			return false
		}

		return paramref.IsOnlyReference(validator.Doc.GetNodeText(valueNode))
	}

	return paramref.IsOnlyReference(validator.Doc.GetNodeText(node))
}

func removeUselessMustValidateError(diags []protocol.Diagnostic) []protocol.Diagnostic {
	resDiags := []protocol.Diagnostic{}
	for i, diag := range diags {
		if diag.Message == protocol.String("Must validate one and only one schema (oneOf)") {
			if hasAnotherDiagInsideRange(diags, diag.Range) {
				continue
			}
			resDiags = append(resDiags, diagnostic.New(
				diag.Range,
				diag.Severity,
				"Invalid structure",
				[]protocol.CodeAction{},
			))
		} else {
			resDiags = append(resDiags, diags[i])
		}
	}

	return resDiags
}

func (validator *JSONSchemaValidator) isInFlattenedStep(node *sitter.Node) bool {
	start := validator.Doc.NodeToRange(node).Start
	for _, rng := range validator.Doc.FlattenedSteps {
		if position.InRange(rng, start) {
			return true
		}
	}
	return false
}

func hasAnotherDiagInsideRange(diags []protocol.Diagnostic, rangeToCheck protocol.Range) bool {
	for _, diag := range diags {
		if position.InRange(rangeToCheck, diag.Range.Start) {
			return true
		}
	}

	return false
}

var duplicateMergeKeyErrorRegex = regexp.MustCompile(`mapping key "<<" already defined at line \d+\n?`)
