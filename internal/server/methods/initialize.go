package methods

import (
	"context"

	"go.lsp.dev/protocol"

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

func (methods *Methods) Initialize(_ context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	options := map[string]interface{}{}
	if len(params.InitializationOptions) > 0 {
		// Options that are not an object carry nothing we read.
		_ = protocol.Unmarshal(params.InitializationOptions, &options)
	}
	if isCciExtension, ok := options["isCciExtension"]; ok && isCciExtension == true {
		methods.Settings.IsCciExtension = true
	}
	if userAgent, ok := options["userAgent"].(string); ok {
		version.UserAgent += " " + userAgent
	}

	yes := true
	incremental := protocol.TextDocumentSyncKindIncremental
	workDoneProgress := protocol.WorkDoneProgressOptions{WorkDoneProgress: &yes}

	v := &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			RenameProvider: protocol.Boolean(false),
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
				// TriggerCharacters: []string{":"},
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
			DocumentSymbolProvider: protocol.Boolean(true),
		},
		ServerInfo: protocol.ServerInfo{
			Name:    "circleci-language-server",
			Version: protocol.NewOptional(version.Server),
		},
	}
	return v, nil
}
