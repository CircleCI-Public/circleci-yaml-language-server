enum Commands {
  DocumentDidOpen = 'textDocument/didOpen',
  DocumentDidClose = 'textDocument/didClose',
  DocumentDidChange = 'textDocument/didChange',
  DocumentHover = 'textDocument/hover',
  SemanticToken = 'textDocument/semanticTokens/full',
  Definition = 'textDocument/definition',
  Reference = 'textDocument/references',
  Completion = 'textDocument/completion',
  Shutdown = 'shutdown',
  Exit = 'exit',
  PublishDiagnostics = 'textDocument/publishDiagnostics',
}

enum DiagnosticSeverity {
  Error = 1,
  Warning = 2,
  Information = 3,
  Hint = 4,
}

enum DiagnosticTag {
  Unnecessary = 1,
  Deprecated = 2,
}

type DocumentURI = string;
type DocumentVersion = number;
type LanguageIdentifier = string;
type TriggerKind = number;

type CodeDescription = {
  href: string,
};

type CompletionContext = {
  triggerCharacter?: string,
  triggerKind?: TriggerKind,
};

type DiagnosticRelatedInformation = {
  location: Location,
  message: string,
};

type Diagnostic = {
  range: Range,
  severity?: DiagnosticSeverity
  code?: string | number,
  codeDescription?: CodeDescription,
  source?: string,
  message: string,
  tags?: DiagnosticTag[],
  relatedInformation?: DiagnosticRelatedInformation[],
  data?: unknown,
};

type Location = {
  uri: DocumentURI,
  range: Range,
};

type PartialResultParams = {
  partialResultToken?: ProgressToken,
};

type Position = {
  line: number,
  character: number,
};

type ProgressToken = {
  name: string,
  number: number,
};

type Range = {
  start: Position,
  end: Position,
};

type TextDocumentContentChangeEvent = {
  // Range is the range of the document that changed.
  range: Range,

  // RangeLength is the length of the range that got replaced.
  rangeLength?: number,

  // Text is the new text of the document.
  text: string,
};

type TextDocumentItem = {
  textDocument: {
    languageId: LanguageIdentifier

    text: string,
  } & VersionedTextDocumentIdentifier
};

type VersionedTextDocumentIdentifier = {
  uri: DocumentURI,
  version: DocumentVersion,
};

type WorkDoneProgressParams = {
  workDoneToken?: ProgressToken,
};

type DocumentDidOpenCommand = TextDocumentItem;

type DocumentDidCloseCommand = TextDocumentItem;

type DocumentDidChange = {
  // TextDocument is the document that did change. The version number points
  // to the version after all provided content changes have
  // been applied.
  textDocument: VersionedTextDocumentIdentifier,

  // ContentChanges is the actual content changes. The content changes describe single state changes
  // to the document. So if there are two content changes c1 and c2 for a document
  // in state S then c1 move the document to S' and c2 to S''.
  contentChanges: TextDocumentContentChangeEvent[],
};

type ReferenceContext = {
  includeDeclaration: boolean,
};

type TextDocumentPositionParams = {
  textDocument: TextDocumentIdentifier,
  position: Position,
};

type TextDocumentIdentifier = {
  uri: DocumentURI,
};

// ============================================================
// RPC Parameters
type DocumentHover =
  & TextDocumentPositionParams
  & WorkDoneProgressParams;

type SemanticTokensParams = {
  textDocument: TextDocumentIdentifier,
}
& PartialResultParams
& WorkDoneProgressParams;

type DefinitionParams =
  & TextDocumentPositionParams
  & WorkDoneProgressParams
  & PartialResultParams;

type ReferenceParams = {
  context: ReferenceContext,
}
& TextDocumentPositionParams
& WorkDoneProgressParams
& PartialResultParams;

type CompletionParams = {
  context?: CompletionContext,
}
& TextDocumentPositionParams
& WorkDoneProgressParams
& PartialResultParams;

type PublishDiagnosticsParams = {
  diagnostics: Diagnostic[],
  uri: DocumentURI,
  version?: DocumentVersion,
};

type CommandParameters = {
  [Commands.Completion]: CompletionParams,
  [Commands.DocumentDidOpen]: DocumentDidOpenCommand,
  [Commands.DocumentDidClose]: DocumentDidCloseCommand,
  [Commands.DocumentDidChange]: DocumentDidChange,
  [Commands.DocumentHover]: DocumentHover,
  [Commands.Exit]: undefined,
  [Commands.Definition] : DefinitionParams,
  [Commands.PublishDiagnostics]: PublishDiagnosticsParams,
  [Commands.Reference]: ReferenceParams,
  [Commands.SemanticToken]: SemanticTokensParams,
  [Commands.Shutdown]: undefined,
};

type CommandDefinitions = {
  [key in string]: Record<string, unknown> | undefined;
};

type CommandResult = Record<string, unknown> | undefined;

type ProtocolParams = DocumentDidOpenCommand
| DocumentDidCloseCommand
| DocumentDidChange
| DocumentHover
| SemanticTokensParams
| DefinitionParams
| ReferenceParams
| CompletionParams
| PublishDiagnosticsParams;

type EnvOptions = {
  lspServer?: {
    // Port of the LSP server
    // Env variable: PORT
    // Default: <none>
    port?: number,

    // Host of the LSP Server
    // Env variable: LSP_SERVER_HOST
    // Default: localhost
    host?: string,

    // Should the server be spawn ?
    // Accepted values are: 1, 0, on, off, true, false, yes, no
    // Env variable: SPAWN_LSP_SERVER
    // Default: false
    spawn?: boolean,

    // Path to the server binary
    // If relative, it will be relative to the current folder
    // Env variable: RPC_SERVER_BIN
    // Default: <none>
    binPath?: string,

    // Path to the schema.
    // If relative, it will be relative to the current folder
    // Env variable: SCHEMA_LOCATION
    // Default: <none>
    jsonSchemaLocation?: string,
  },
};

export {
  Commands,
};

export type {
  CommandDefinitions,
  CommandParameters,
  CommandResult,
  EnvOptions,

  Diagnostic,
  Position,
  Range,

  CompletionParams,
  DocumentDidOpenCommand,
  DocumentDidCloseCommand,
  DocumentDidChange,
  DocumentHover,
  DefinitionParams,
  ReferenceParams,
  ProtocolParams,
  PublishDiagnosticsParams,
  SemanticTokensParams,
};
