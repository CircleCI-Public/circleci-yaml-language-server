package parser

import (
	"fmt"
	"os"
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
}

func (validator *JSONSchemaValidator) ParseJsonSchema() (*gojsonschema.Schema, error) {
	schemaLocation := os.Getenv("SCHEMA_LOCATION")

	if schemaLocation == "" {
		return nil, fmt.Errorf("could not load JSON Schema: SCHEMA_LOCATION not set")
	}

	URI := uri.New(schemaLocation)
	loader := gojsonschema.NewReferenceLoader(string(URI))

	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return nil, err
	}

	validator.schema = schema

	return schema, err
}

func handleYAMLErrors(err string, content []byte, node *sitter.Node) ([]protocol.Diagnostic, error) {
	diagnostics := []protocol.Diagnostic{}

	reWithLine, _ := regexp.Compile(`(?s)^yaml:\s(?P<Error>[\w\d\s]+):\n(?P<Lines>\s+line \d+:.+\n?)+`)

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

			diagnostics = append(diagnostics, utils.CreateErrorDiagnosticFromRange(rng, err))
		}
		return diagnostics, nil
	}

	if !reWithLine.MatchString(err) {
		return []protocol.Diagnostic{utils.CreateErrorDiagnosticFromNode(node, err)}, nil
	}

	// For errors providing line numbers, we add a diagnostic on the
	// specified lines
	mes := reWithLine.FindAllStringSubmatch(err, -1)[0]
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
			return []protocol.Diagnostic{utils.CreateErrorDiagnosticFromNode(node, err)}, nil
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

		return diagnostics
	}

	// This is needed so that the yaml library resolves anchor and aliases for us
	tmpYML, err := yaml.Marshal(file)
	if err != nil {
		// Should never happen
		return []protocol.Diagnostic{utils.CreateErrorDiagnosticFromNode(rootNode, err.Error())}
	}
	yaml.Unmarshal(tmpYML, &file)

	yamlloader := gojsonschema.NewGoLoader(file)
	result, err := validator.schema.Validate(yamlloader)
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

				diagnostic := utils.CreateErrorDiagnosticFromNode(node, resErr.Description())
				diagnostics = append(diagnostics, diagnostic)
			}
		}
	}

	return diagnostics
}
