import { normalizeURI } from '../.env/index';
import { ProtocolReturns } from '../types';
import CustomEvent from '../.env/CustomEvent';
import { EventType, RequestPayload } from '../.env/RpcClient';
import type {
  CommandParameters,
  Diagnostic,
  Position,
  ProtocolParams,
  PublishDiagnosticsParams,
} from '../types';

/**
 * Send a command to the LSP server and return the response.
 */
async function command<T extends keyof CommandParameters>(
  commandName: T,
  params: CommandParameters[T],
) {
  const response = await globalThis.rpcClient.request(commandName, params);

  if (!response) {
    return response;
  }

  return normalize(response);
}

/**
 * Log all data received by the RPC client.
 * Use this function to debug easily tests.
 *
 * Example:
 *
 * installRpcLogger();
 *
 * describe('Some suite test', () => {
 * });
 *
 */
function installRpcLogger() {
  beforeAll(() => {
    globalThis.rpcClient.addEventListener(
      EventType.requestSent,
      (data: Event) => {
        const payload = (data as CustomEvent<{ request: RequestPayload }>)?.detail?.request;
        // eslint-disable-next-line no-console
        console.log('Data sent:', payload);
      },
    );

    globalThis.rpcClient.addEventListener(
      EventType.dataReceived,
      (data: Event) => {
        const payload = (data as CustomEvent<NotificationPayload>)?.detail;
        // eslint-disable-next-line no-console
        console.log('Data received:', payload);
      },
    );
  });
}

/**
 * Normalize protocol params object.
 * The main transformation is the normalization of the URI
 */
function normalize<T extends ProtocolParams | ProtocolReturns>(data: T) : T {
  if ('textDocument' in data) {
    return {
      ...data,
      textDocument: {
        ...data.textDocument,
        uri: normalizeURI(data.textDocument.uri),
      },
    };
  }

  if ('uri' in data) {
    return {
      ...data,
      uri: normalizeURI(data.uri),
    };
  }

  return data;
}

function position(lineIndex: number, characterIndex: number): Position {
  return {
    character: characterIndex,
    line: lineIndex,
  };
}

/**
 * Send a command to the LSP without any type checking and return
 * the response.
 */
function rawCommand(
  commandName: string,
  params: Record<string, unknown>,
) {
  return globalThis.rpcClient.requestRaw(commandName, params);
}

/**
 * Expect that the next data that are received by the RPC client is a diagnostic list.
 * Throw an error if not.
 * @params sortDiagnostics If true, diagnostics will be sorted so the order is predictable.
 *                         Useful for snapshot testing.
 * @returns Return the diagnostic list.
 */
async function immediateDiagnostics(sortDiagnostics = true): Promise<PublishDiagnosticsParams> {
  const diagnostics = await new Promise<PublishDiagnosticsParams>((resolve, reject) => {
    globalThis.rpcClient.addEventListener(
      EventType.dataReceived,
      (data: Event) => {
        const payload = (data as CustomEvent<NotificationPayload>)?.detail?.data;

        if (payload?.method !== 'textDocument/publishDiagnostics') {
          reject(new Error(`Expecting notification, received: ${payload?.method}`));
        }

        if (!payload?.params) {
          throw new Error(`Invalid payload: ${payload}`);
        }

        resolve(normalize(payload.params));
      },
      {
        once: true,
      },
    );
  });

  if (sortDiagnostics) {
    diagnostics.diagnostics.sort(sortDiagnosticList);
  }

  return diagnostics;
}

function sortDiagnosticList(a: Diagnostic, b: Diagnostic) {
  return sortPosition(a.range.start, b.range.start)
  || sortPosition(a.range.end, b.range.end)
  || sortSeverity(a, b)
  || a.message.localeCompare(b.message);
}

function sortSeverity(diagA: Diagnostic, diagB: Diagnostic) {
  if (diagA.severity === diagB.severity) {
    return 0;
  }

  const a = diagA.severity ?? 99999;
  const b = diagB.severity ?? 99999;

  return a > b ? -1 : 1;
}

function sortPosition(a: Position, b: Position): number {
  if (a.line !== b.line) {
    return a.line > b.line ? -1 : 1;
  }

  if (a.character === b.character) {
    return 0;
  }

  return a.character > b.character ? -1 : 1;
}

type NotificationPayload = {
  data: RequestPayload<PublishDiagnosticsParams>
};

export {
  configFilePath,
  configFileContent,
  configFileUri,
} from '../.env/index';

export {
  command,
  immediateDiagnostics,
  installRpcLogger,
  normalize,
  position,
  rawCommand,
};
