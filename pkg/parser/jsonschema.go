package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/xeipuuv/gojsonschema"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gopkg.in/yaml.v3"
)

type JSONSchemaValidator struct {
	schema *gojsonschema.Schema
	Doc    YamlDocument
}

func (validator *JSONSchemaValidator) LoadJsonSchema(schemaLocation string) error {
	URI := uri.New(schemaLocation)
	loader := gojsonschema.NewReferenceLoader(string(URI))

	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		fmt.Printf("Error while loading JSON Schema \"%s\"\n", schemaLocation)
		fmt.Println(err.Error())
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
				Start: utils.IndexToPos(match[0], content),
				End:   utils.IndexToPos(match[1], content),
			}
			node, _, _ := utils.NodeAtPos(rootNode, rng.Start)
			if node.Type() == "alias_name" {
				diagnostics = append(diagnostics, utils.CreateErrorDiagnosticFromRange(rng, err))
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
			return []protocol.Diagnostic{utils.CreateErrorDiagnosticFromNode(rootNode, err)}, nil
		}

		lineRange := utils.AllLineContentRange([]int{lineNumber}, content)[0]

		diagnostic := utils.CreateErrorDiagnosticFromRange(
			lineRange,
			utils.ToDiagnosticMessage(lineError),
		)

		return []protocol.Diagnostic{diagnostic}, nil
	}

	reMultilineError, _ := regexp.Compile(`(?s)^yaml:\s(?P<Error>[\w\d\s]+):\n(?P<Lines>\s+line \d+:.+\n?)+`)

	if !reMultilineError.MatchString(err) {
		return []protocol.Diagnostic{utils.CreateErrorDiagnosticFromNode(rootNode, err)}, nil
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
			return []protocol.Diagnostic{utils.CreateErrorDiagnosticFromNode(rootNode, err)}, nil
		}

		lineIndexes = append(lineIndexes, lineNumber-1)
		lineErrors = append(lineErrors, lineError)
	}

	lineContentRanges := utils.AllLineContentRange(lineIndexes, content)

	for i, lineContentRange := range lineContentRanges {
		lineError := lineErrors[i]

		diagnostic := utils.CreateErrorDiagnosticFromRange(
			lineContentRange,
			lineError,
		)

		diagnostics = append(diagnostics, diagnostic)
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
		return []protocol.Diagnostic{utils.CreateErrorDiagnosticFromNode(rootNode, err.Error())}
	}

	if !result.Valid() {
		for _, resErr := range result.Errors() {
			fields := strings.Split(resErr.Field(), ".")
			if len(fields) == 1 && fields[0] == "(root)" {
				diagnostic := utils.CreateErrorDiagnosticFromNode(rootNode, resErr.Description())
				diagnostics = append(diagnostics, diagnostic)
			} else {
				node, err := FindDeepestNode(rootNode, content, fields)
				if err != nil {
					continue
				}

				if validator.doesNodeUseParameter(node) {
					continue
				}

				diagnostic := utils.CreateErrorDiagnosticFromNode(node, resErr.Description())
				diagnostics = append(diagnostics, diagnostic)
			}
		}
	}

	return diagnostics
}

// Keys that can have only a parameter inside it,
// and therefore the JSON Schema validation is not necessary for these keys.
//
// Example:
//
//	`when: << parameters.my_param >>`
//
//	But in the JSON Schema, the `when` key is defined as an object, so the validation
//	will fail if we don't ignore it.
var PARAMS_KEYS = []string{
	"when",
}

func (validator *JSONSchemaValidator) doesNodeUseParameter(node *sitter.Node) bool {
	if node.Type() == "block_mapping_pair" {
		keyNode, valueNode := validator.Doc.GetKeyValueNodes(node)
		if keyNode == nil || valueNode == nil {
			return false
		}
		key := validator.Doc.GetNodeText(keyNode)
		value := validator.Doc.GetNodeText(valueNode)

		if isInArray := utils.FindInArray(PARAMS_KEYS, key); utils.CheckIfOnlyParamUsed(value) && isInArray > 0 {
			return true
		}
	}

	return false
}
