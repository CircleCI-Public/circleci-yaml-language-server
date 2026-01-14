package languageservice

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
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
	yamlDocument, err := yamlparser.ParseFromContent([]byte(content), context, uri.File(""), protocol.Position{})
	yamlDocument.SchemaLocation = schemaLocation

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	return DiagnosticYAML(yamlDocument, cache, context)
}

func DiagnosticYAML(yamlDocument yamlparser.YamlDocument, cache *utils.Cache, context *utils.LsContext) ([]protocol.Diagnostic, error) {
	if yamlDocument.Version != 0 && yamlDocument.Version < 2.1 {
		// TODO: Handle error
		return []protocol.Diagnostic{}, nil
	}

	diag := DiagnosticType{
		diagnostics:  &[]protocol.Diagnostic{},
		yamlDocument: yamlDocument,
	}

	yamlDocument.ValidateYAML()
	diag.addDiagnostics(*yamlDocument.Diagnostics)

	validator := yamlparser.JSONSchemaValidator{
		Doc: yamlDocument,
	}
	err := validator.LoadJsonSchema(yamlDocument.SchemaLocation)

	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	diag.addDiagnostics(
		validator.ValidateWithJSONSchema(diag.yamlDocument.RootNode, diag.yamlDocument.Content),
	)

	validateStruct := validate.Validate{
		APIs: validate.ValidateAPIs{
			DockerHub: dockerhub.NewAPI(),
		},
		Doc:         diag.yamlDocument,
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       cache,
		Context:     context,
	}
	validateStruct.Validate()
	diag.addDiagnostics(*validateStruct.Diagnostics)

	*diag.diagnostics = deduplicateDiagnosticsByRange(*diag.diagnostics)

	// after ALL diagnostics are added, filter out the ones that the user wishes to suppress via cci-ignore comments
	*diag.diagnostics = yamlparser.FilterSuppressedDiagnostics(*diag.diagnostics, diag.yamlDocument.SuppressionInfo)

	// append some extra add code actions to every diagnostic to suppress said diagnostic
	*diag.diagnostics, err = utils.AppendSuppressionCodeActions(yamlDocument.URI, *diag.diagnostics, yamlDocument.Content)
	if err != nil {
		return []protocol.Diagnostic{}, err
	}

	return *diag.diagnostics, nil
}

func (diag *DiagnosticType) addDiagnostics(diagnostic []protocol.Diagnostic) {
	*diag.diagnostics = append(*diag.diagnostics, diagnostic...)
}

// deduplicateDiagnosticsByRange removes duplicate diagnostics at same range
// The primary use case of this is to handle YAML anchors where same content
// is referenced multiple times, causing duplicate diagnostics on the original
// anchored text.
func deduplicateDiagnosticsByRange(diagnostics []protocol.Diagnostic) []protocol.Diagnostic {
	seen := make(map[string]bool)
	result := []protocol.Diagnostic{}

	for _, diag := range diagnostics {
		// Create unique key from range + message + severity
		key := fmt.Sprintf("%v-%s-%v", diag.Range, diag.Message, diag.Severity)
		if !seen[key] {
			seen[key] = true
			result = append(result, diag)
		}
	}

	return result
}
