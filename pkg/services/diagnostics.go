package languageservice

import (
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser/validate"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type DiagnosticType struct {
	diagnostics  *[]protocol.Diagnostic
	yamlDocument yamlparser.YamlDocument
}

func Diagnostic(params protocol.PublishDiagnosticsParams, cache *utils.Cache, context *utils.LsContext, schemaLocation string) protocol.PublishDiagnosticsParams {
	diagnostics, _ := DiagnosticFile(params.URI, cache, context, schemaLocation)

	diagnosticParams := protocol.PublishDiagnosticsParams{
		URI:         params.URI,
		Diagnostics: diagnostics,
	}

	return diagnosticParams
}

func DiagnosticFile(uri protocol.URI, cache *utils.Cache, context *utils.LsContext, schemaLocation string) ([]protocol.Diagnostic, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(uri, cache, context)
	yamlDocument.SchemaLocation = schemaLocation

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	return DiagnosticYAML(yamlDocument, cache, context)
}

func DiagnosticString(content string, cache *utils.Cache, context *utils.LsContext, schemaLocation string) ([]protocol.Diagnostic, error) {
	yamlDocument, err := yamlparser.ParseFromContent([]byte(content), context, uri.File(""))
	yamlDocument.SchemaLocation = schemaLocation

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	return DiagnosticYAML(yamlDocument, cache, context)
}

func DiagnosticYAML(yamlDocument yamlparser.YamlDocument, cache *utils.Cache, context *utils.LsContext) ([]protocol.Diagnostic, error) {
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
		Context:     context,
	}
	validateStruct.Validate()
	diag.addDiagnostics(*validateStruct.Diagnostics)

	return *diag.diagnostics, nil
}

func (diag *DiagnosticType) addDiagnostics(diagnostic []protocol.Diagnostic) {
	*diag.diagnostics = append(*diag.diagnostics, diagnostic...)
}
