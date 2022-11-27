package languageservice

import (
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser/validate"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

type DiagnosticType struct {
	diagnostics  *[]protocol.Diagnostic
	yamlDocument yamlparser.YamlDocument
}

func Diagnostic(params protocol.PublishDiagnosticsParams, cache *utils.Cache, schemaLocation string) protocol.PublishDiagnosticsParams {
	diagnostics, _ := DiagnosticFile(params.URI, cache, schemaLocation)

	diagnosticParams := protocol.PublishDiagnosticsParams{
		URI:         params.URI,
		Diagnostics: diagnostics,
	}

	return diagnosticParams
}

func DiagnosticFile(uri protocol.URI, cache *utils.Cache, schemaLocation string) ([]protocol.Diagnostic, error) {
	yamlDocument, err := yamlparser.ParseFileWithCache(uri, cache)
	yamlDocument.SchemaLocation = schemaLocation

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	if yamlDocument.Version < 2.1 {
		// TODO: Handle error
		return []protocol.Diagnostic{}, nil
	}

	return DiagnosticYAML(yamlDocument, cache)
}

func DiagnosticString(content string, cache *utils.Cache, schemaLocation string) ([]protocol.Diagnostic, error) {
	yamlDocument, err := yamlparser.ParseContent([]byte(content))
	yamlDocument.SchemaLocation = schemaLocation

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	return DiagnosticYAML(yamlDocument, cache)
}

func DiagnosticYAML(yamlDocument yamlparser.YamlDocument, cache *utils.Cache) ([]protocol.Diagnostic, error) {
	if yamlDocument.Version < 2.1 {
		// TODO: Handle error
		return []protocol.Diagnostic{}, nil
	}

	diag := DiagnosticType{
		diagnostics:  &[]protocol.Diagnostic{},
		yamlDocument: yamlDocument,
	}

	yamlDocument.ValidateYAML()
	diag.addDiagnostics(*yamlDocument.Diagnostics)

	validator := yamlparser.JSONSchemaValidator{}
	err := validator.LoadJsonSchema(yamlDocument.SchemaLocation)

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	diag.addDiagnostics(
		validator.ValidateWithJSONSchema(diag.yamlDocument.RootNode, diag.yamlDocument.Content),
	)

	validateStruct := validate.Validate{
		Doc:         diag.yamlDocument,
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       cache,
	}
	validateStruct.Validate()
	diag.addDiagnostics(*validateStruct.Diagnostics)

	return *diag.diagnostics, nil
}

func (diag *DiagnosticType) addDiagnostics(diagnostic []protocol.Diagnostic) {
	*diag.diagnostics = append(*diag.diagnostics, diagnostic...)
}
