package methods

import (
	"context"
	"time"

	"github.com/bep/debounce"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/version"
)

var TokenTypes = []string{
	string(protocol.SemanticTokenTypesKeyword),
	string(protocol.SemanticTokenTypesNamespace),
	string(protocol.SemanticTokenTypesClass),
	string(protocol.SemanticTokenTypesComment),
	string(protocol.SemanticTokenTypesFunction),
}

var TokenModifiers = []string{
	string(protocol.SemanticTokenModifiersDeclaration),
	string(protocol.SemanticTokenModifiersAbstract),
}

// maxEditDebounceMs bounds the editDebounceMs a client can choose. A longer
// wait than this reads as a mistake, so the default is kept instead.
const maxEditDebounceMs = 10_000

func (methods *Methods) Initialize(_ context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	options := map[string]interface{}{}
	if len(params.InitializationOptions) > 0 {
		// Options that are not an object carry nothing we read.
		_ = protocol.Unmarshal(params.InitializationOptions, &options)
	}
	isCciExtension := options["isCciExtension"] == true
	// Off for the VS Code extension unless it asks for them.
	schemaHovers, ok := options["schemaHovers"].(bool)
	if !ok {
		schemaHovers = !isCciExtension
	}
	gitHubSignInCommand, _ := options["gitHubSignInCommand"].(string)
	methods.updateSettings(func(settings *session.Settings) {
		settings.IsCciExtension = settings.IsCciExtension || isCciExtension
		settings.SchemaHovers = schemaHovers
		settings.GitHubSignInCommand = gitHubSignInCommand
		settings.DefinitionLinks = definitionLinks(params.Capabilities)
	})
	if userAgent, ok := options["userAgent"].(string); ok {
		version.UserAgent += " " + userAgent
	}
	if ms, ok := options["editDebounceMs"].(float64); ok && ms >= 0 && ms <= maxEditDebounceMs {
		methods.editDebounce = time.Duration(ms) * time.Millisecond
		methods.debounceEdit = debounce.New(methods.editDebounce)
	}

	folders, _ := params.WorkspaceFolders.Get()
	for _, folder := range folders {
		methods.workspaceFolders = append(methods.workspaceFolders, folder.URI)
	}

	yes := true
	incremental := protocol.TextDocumentSyncKindIncremental
	workDoneProgress := protocol.WorkDoneProgressOptions{WorkDoneProgress: &yes}

	v := &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			RenameProvider: &protocol.RenameOptions{PrepareProvider: &yes},
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: &yes,
				Change:    &incremental,
			},
			SemanticTokensProvider: &protocol.SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{
					TokenTypes:     TokenTypes,
					TokenModifiers: TokenModifiers,
				},
				Full: protocol.Boolean(true),
			},
			DefinitionProvider: &protocol.DefinitionOptions{
				WorkDoneProgressOptions: workDoneProgress,
			},
			ReferencesProvider: &protocol.ReferenceOptions{
				WorkDoneProgressOptions: workDoneProgress,
			},
			CompletionProvider: &protocol.CompletionOptions{
				// After `pipeline.` or `parameters.`, and between the parts
				// of a pipeline value.
				TriggerCharacters: []string{"."},
			},
			HoverProvider: &protocol.HoverOptions{
				WorkDoneProgressOptions: workDoneProgress,
			},
			ExecuteCommandProvider: protocol.ExecuteCommandOptions{
				Commands: []string{"setToken"},
			},
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{
					protocol.CodeActionKindQuickFix,
				},
				ResolveProvider: &yes,
			},
			DocumentSymbolProvider:  protocol.Boolean(true),
			WorkspaceSymbolProvider: protocol.Boolean(true),
		},
		ServerInfo: protocol.ServerInfo{
			Name:    "circleci-language-server",
			Version: protocol.NewOptional(version.Server),
		},
	}
	return v, nil
}

// definitionLinks is whether a client with capabilities reads a definition as
// a link.
func definitionLinks(capabilities protocol.ClientCapabilities) bool {
	textDocument := capabilities.TextDocument
	if textDocument == nil || textDocument.Definition == nil || textDocument.Definition.LinkSupport == nil {
		return false
	}
	return *textDocument.Definition.LinkSupport
}
