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

func Diagnostic(params protocol.PublishDiagnosticsParams, cache *utils.Cache, context *utils.LsContext) protocol.PublishDiagnosticsParams {
	diagnostics, _ := DiagnosticFile(params.URI, cache, context)

	diagnosticParams := protocol.PublishDiagnosticsParams{
		URI:         params.URI,
		Diagnostics: diagnostics,
	}

	return diagnosticParams
}

func DiagnosticFile(uri protocol.URI, cache *utils.Cache, context *utils.LsContext) ([]protocol.Diagnostic, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(uri, cache, context)

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	return DiagnosticYAML(yamlDocument, cache, context), nil
}

func DiagnosticString(content string, cache *utils.Cache, context *utils.LsContext) ([]protocol.Diagnostic, error) {
	yamlDocument, err := yamlparser.ParseFromContent([]byte(content), context)

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	return DiagnosticYAML(yamlDocument, cache, context), nil
}

func DiagnosticYAML(yamlDocument yamlparser.YamlDocument, cache *utils.Cache, context *utils.LsContext) []protocol.Diagnostic {
	if yamlDocument.Version < 2.1 {
		// TODO: Handle error
		return []protocol.Diagnostic{}
	}

	diag := DiagnosticType{
		diagnostics:  &[]protocol.Diagnostic{},
		yamlDocument: yamlDocument,
	}

	yamlDocument.ValidateYAML()
	diag.addDiagnostics(*yamlDocument.Diagnostics)

	validator := yamlparser.JSONSchemaValidator{}
	validator.LoadJsonSchema()

	diag.addDiagnostics(validator.ValidateWithJSONSchema(diag.yamlDocument.RootNode, diag.yamlDocument.Content))

	validateStruct := validate.Validate{
		Doc:         diag.yamlDocument,
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       cache,
		Context:     context,
	}
	validateStruct.Validate()
	diag.addDiagnostics(*validateStruct.Diagnostics)

	return *diag.diagnostics
}

func (diag *DiagnosticType) addDiagnostics(diagnostic []protocol.Diagnostic) {
	*diag.diagnostics = append(*diag.diagnostics, diagnostic...)
}
