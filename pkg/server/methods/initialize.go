package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// Defined at build time by ldflags.sh
var ServerVersion string = "<dev build>"
var BuildTime string

type SemanticTokensOptions struct {
	WorkDoneProgress bool                          `json:"workDoneProgress,omitempty"`
	Legend           protocol.SemanticTokensLegend `json:"legend,omitempty"`
	Range            bool                          `json:"range,omitempty"`
	Full             bool                          `json:"full,omitempty"`
}

var TokenTypes = []protocol.SemanticTokenTypes{
	protocol.SemanticTokenKeyword,
	protocol.SemanticTokenNamespace,
	protocol.SemanticTokenClass,
	protocol.SemanticTokenComment,
	protocol.SemanticTokenFunction,
}

var TokenModifiers = []protocol.SemanticTokenModifiers{
	protocol.SemanticTokenModifierDeclaration,
	protocol.SemanticTokenModifierAbstract,
}

func (methods *Methods) Initialize(reply jsonrpc2.Replier) error {
	v := protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			RenameProvider: false,
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindIncremental,
			},
			SemanticTokensProvider: SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{
					TokenTypes:     TokenTypes,
					TokenModifiers: TokenModifiers,
				},
				Full:  true,
				Range: false,
			},
			DefinitionProvider: protocol.DefinitionOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
					WorkDoneProgress: true,
				},
			},
			ReferencesProvider: protocol.ReferenceOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
					WorkDoneProgress: true,
				},
			},
			CompletionProvider: &protocol.CompletionOptions{
				ResolveProvider: false,
				// TriggerCharacters: []string{":"},
			},
			HoverProvider: &protocol.HoverOptions{
				WorkDoneProgressOptions: protocol.WorkDoneProgressOptions{
					WorkDoneProgress: true,
				},
			},
			ExecuteCommandProvider: &protocol.ExecuteCommandOptions{
				Commands: []string{"setToken"},
			},
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "circleci-language-server",
			Version: ServerVersion,
		},
	}
	return reply(methods.Ctx, v, nil)
}
