package parser

import (
	"net/url"
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
)

type JSONSchemaValidator struct {
	schema *gojsonschema.Schema
	Doc    YamlDocument
}

func (validator *JSONSchemaValidator) LoadJsonSchema(schemaLocation string) error {
	loader := gojsonschema.NewReferenceLoader(schemaURI(schemaLocation))

	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return err
	}

	validator.schema = schema

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

// LoadEmbeddedJsonSchema validates against the schema built into the server.
func (validator *JSONSchemaValidator) LoadEmbeddedJsonSchema() error {
	compiled, err := embeddedSchema()
	if err != nil {
		return err
	}

	validator.schema = compiled

	return nil
}

func (validator *JSONSchemaValidator) LoadJsonSchemaFromBytes(data []byte) error {
	loader := gojsonschema.NewBytesLoader(data)

	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return err
	}

	validator.schema = schema

	return nil
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

// Validates a config YML against a JSON Shema
// Returns a list of diagnostics and a boolean that suggest
// whether to continue diagnostic or not
func (validator *JSONSchemaValidator) ValidateWithJSONSchema(rootNode *sitter.Node, content []byte) []protocol.Diagnostic {
	var file interface{}
	diagnostics := make([]protocol.Diagnostic, 0)

	if err := yaml.Unmarshal(content, &file); err != nil {
		// Can only happen if anchor or alias are not properly defined and/or referenced
		yamlError, _ := handleYAMLErrors(err.Error(), content, rootNode)
		diagnostics = append(diagnostics, yamlError...)
	}

	yamlLoader := gojsonschema.NewGoLoader(file)

	result, err := validator.schema.Validate(yamlLoader)
	if err != nil {
		// Should never happen
		return []protocol.Diagnostic{diagnostic.ErrorFromNode(rootNode, err.Error())}
	}

	jsonSchemaDiags := []protocol.Diagnostic{}

	if !result.Valid() {
		// A value that is only a reference, such as << parameters.flag >>, is
		// a string until the config is compiled, so the schema rejects it
		// wherever it wants anything else. Those errors are dropped, along
		// with the combinator errors (oneOf, if/then/else) they caused.
		var combinators []protocol.Diagnostic
		var referenced []protocol.Range

		for _, resErr := range result.Errors() {
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

			if validator.doesNodeUseParameter(node) {
				referenced = append(referenced, validator.Doc.NodeToRange(node))
				continue
			}

			diag := diagnostic.ErrorFromNode(node, resErr.Description())
			if isCombinatorError(resErr) {
				combinators = append(combinators, diag)
				continue
			}

			jsonSchemaDiags = append(jsonSchemaDiags, diag)
		}

		for _, combinator := range combinators {
			if onlyContainsReferences(combinator, referenced, jsonSchemaDiags) {
				continue
			}

			jsonSchemaDiags = append(jsonSchemaDiags, combinator)
		}
	}

	jsonSchemaDiags = removeUselessMustValidateError(jsonSchemaDiags)
	diagnostics = append(diagnostics, jsonSchemaDiags...)

	return diagnostics
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

// onlyContainsReferences reports whether every failure inside the combinator
// error is one of the referenced values: there is one of those in its range,
// and no other error is.
func onlyContainsReferences(combinator protocol.Diagnostic, referenced []protocol.Range, others []protocol.Diagnostic) bool {
	containsReference := slices.ContainsFunc(referenced, func(rng protocol.Range) bool {
		return position.InRange(combinator.Range, rng.Start)
	})
	if !containsReference {
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

func hasAnotherDiagInsideRange(diags []protocol.Diagnostic, rangeToCheck protocol.Range) bool {
	for _, diag := range diags {
		if position.InRange(rangeToCheck, diag.Range.Start) {
			return true
		}
	}

	return false
}

var duplicateMergeKeyErrorRegex = regexp.MustCompile(`mapping key "<<" already defined at line \d+\n?`)
