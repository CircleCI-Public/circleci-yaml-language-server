package methods

import (
	"fmt"

	"github.com/rollbar/rollbar-go"
	"github.com/segmentio/encoding/json"
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

type InitializationOptions struct {
	IsCCIExtension          bool   `json:"isCciExtension"`
	IsCrashTelemetryEnabled bool   `json:"isCrashTelemetryEnabled"`
	TelemetryID             string `json:"telemetryID"`
}

func (methods *Methods) Initialize(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.InitializeParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	content, _ := json.Marshal(params.InitializationOptions)
	opts := InitializationOptions{}
	_ = json.Unmarshal(content, &opts)
	enableRollbar := opts.IsCrashTelemetryEnabled && isRollbarEnabled()
	rollbar.SetEnabled(enableRollbar)
	if enableRollbar {
		rollbar.SetCodeVersion(ServerVersion)
		rollbar.SetPerson(opts.TelemetryID, "", "")
	}
	methods.LsContext.IsCciExtension = opts.IsCCIExtension

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
			CodeActionProvider: &protocol.CodeActionRegistrationOptions{
				CodeActionOptions: protocol.CodeActionOptions{
					CodeActionKinds: []protocol.CodeActionKind{
						"quickfix",
					},
					ResolveProvider: true,
				},
			},
			DocumentSymbolProvider: true,
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "circleci-language-server",
			Version: ServerVersion,
		},
	}
	return reply(methods.Ctx, v, nil)
}
