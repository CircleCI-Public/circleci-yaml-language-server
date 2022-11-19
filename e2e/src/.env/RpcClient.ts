// ============================================================
// Import packages
import net from 'net';
import { ProtocolParams } from '../types';

// ============================================================
// Import modules
import CustomEvent from './CustomEvent';

type RequestID = number;

enum Status {
  initiated = 'initiated',
  connecting = 'connecting',
  connected = 'connected',
  ending = 'ending',
  error = 'error',
}

enum EventType {
  dataReceived = 'data-received',
  requestSent = 'request-sent',
  requestWillBeSent = 'request-will-be-send',
  responseReceived = 'response-received',
  error = 'on-error',
}

type RequestDefinitions = {
  [key in string]: Record<string, unknown> | undefined;
};

type ResolvePromise = (value: unknown) => void;

type RequestResult = Record<string, unknown> | undefined;

type RequestPayload<T = unknown> = {
  id: RequestID,
  jsonrpc: '2.0',
  method: string,
  params: T,
};

type ResponsePayload<T = unknown> = {
  id: RequestID,
  jsonrpc: '2.0',
  result: T,
  error?: unknown,
};

// ============================================================
// Class
class RpcClient<O extends RequestDefinitions> extends EventTarget {
  // List of typed requests
  #requests : Record<
  RequestID,
  'done' | undefined | { resolve: ResolvePromise, reject: ResolvePromise }
  > = {};

  readonly host: string;

  readonly port: number;

  readonly testPath: string;

  #connectingPromise: Promise<boolean> | undefined;

  #endingPromise: Promise<boolean> | undefined;

  #socket: net.Socket | undefined;

  #status: Status;

  constructor(port: number, host = '127.0.0.1', testPath = '') {
    super();

    this.testPath = testPath;
    this.port = port;
    this.host = host;
    this.#status = Status.initiated;
  }

  get connected() {
    return this.#status === Status.connected;
  }

  get status() {
    return this.#status;
  }

  // eslint-disable-next-line class-methods-use-this
  get #nextId() : number {
    globalThis.latestRequestId += 1;
    return globalThis.latestRequestId;
  }

  async request<T extends keyof O>(
    method: T,
    params: O[T],
  ) : Promise<undefined | ProtocolParams> {
    return this.requestRaw(method as string, params) as unknown as ProtocolParams | undefined;
  }

  async requestRaw(
    method: string,
    params: RequestResult,
  ) {
    if (this.status !== Status.connected) {
      throw new Error('Client not connected');
    }

    const socket = this.#socket;

    if (!socket) {
      throw new Error('Unexpected error: Socket not defined');
    }

    const id = this.#nextId;

    // Request to send
    const request : RequestPayload = {
      jsonrpc: '2.0',
      id,
      method,
      params,
    };

    // Promise that will be resolve in #onData(),
    // when a response with the matching ID will be received
    const responsePromise = new Promise((resolve, reject) => {
      this.#requests[id] = { resolve, reject };
    });

    // Firing event: request-will-be-send
    this.dispatchEvent(new CustomEvent(
      EventType.requestWillBeSent,
      { detail: { request, str: JSON.stringify(request) } },
    ));

    const message = buildMessage(request);

    await new Promise((resolve) => {
      socket.write(
        message,
        (err) => {
          if (err) {
            this.#dispatchErrorEvent('Error while sending data', err);
            return;
          }

          // Firing event: request-sent
          this.dispatchEvent(new CustomEvent(
            EventType.requestSent,
            { detail: { request } },
          ));
          resolve(undefined);
        },
      );
    });

    return responsePromise;
  }

  /**
   * Start the client.
   * End's when the client as finish to start.
   * If the client is already started, then do nothing
   *
   * Return true if the client was able to connect, false otherwise
   */
  async connect(timeout: number) : Promise<boolean> {
    if (this.#connectingPromise) {
      return this.#connectingPromise;
    }

    this.#status = Status.connecting;

    this.#connectingPromise = connectToServer(this.port, this.host, timeout)
      .then((socket) => {
        if (!socket) {
          this.#status = Status.error;
          return false;
        }

        socket.addListener('data', this.#onData.bind(this));
        this.#socket = socket;
        this.#connectingPromise = undefined;
        this.#status = Status.connected;

        return true;
      });

    return this.#connectingPromise;
  }

  /**
   * Close the socket server.
   */
  async disconnect(graceTime = 10) : Promise<boolean> {
    if (this.#status === Status.initiated) {
      return true;
    }

    if (this.#status === Status.ending) {
      return this.#endingPromise || true;
    }

    this.#status = Status.ending;

    if (!this.#socket) {
      throw new Error('Unexpected error: Socket not defined');
    }

    const socket = this.#socket;

    const closePromise = new Promise<boolean>((resolve) => {
      socket.end(() => resolve(true));
    });

    const timeoutPromise = new Promise<boolean>((resolve) => {
      setTimeout(() => resolve(false), graceTime);
    });

    this.#endingPromise = Promise.race([
      closePromise,
      timeoutPromise,
    ]);

    return this.#endingPromise;
  }

  #dispatchErrorEvent(context: string, error: unknown) {
    const event = new CustomEvent(
      EventType.error,
      {
        detail: {
          context,
          error,
        },
      },
    );

    this.dispatchEvent(event);
  }

  /**
   * Listener of the event "data" of the socket object.
   */
  #onData(data: Buffer) {
    const payload = parseMessage(data);

    if (!payload) {
      this.#dispatchErrorEvent('Received message with no body', data.toString());
      return;
    }

    const event = new CustomEvent(
      EventType.dataReceived,
      {
        detail: { data: payload },
      },
    );

    this.dispatchEvent(event);

    const { id, result, error } = payload;

    // Some data may came with not ID
    if (id === undefined || id === null) {
      return;
    }

    const promiseHandler = this.#requests[id];
    this.#requests[id] = 'done';

    if (promiseHandler === undefined) {
      this.#dispatchErrorEvent(
        `Unknown request id: ${id}`,
        { payload },
      );
      return;
    }

    if (promiseHandler === 'done') {
      this.#dispatchErrorEvent(
        `Received a response that has already been processed: ${id}`,
        { payload },
      );
      return;
    }

    if (error) {
      promiseHandler.reject(error);
      return;
    }

    promiseHandler.resolve(result);
  }
}

// ============================================================
// Helpers

/**
 * Build JSON RPC message according to LSP protocol.
 * See "Base Protocol" in LSP documentation
 *
 * @link https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#baseProtocol
 * @param payload Payload that will be send to LSP server
 * @returns
 */
function buildMessage(payload: RequestPayload): string {
  const content = JSON.stringify(payload);

  const contentType = 'Content-Type: application/vscode-jsonrpc; charset=utf-8';
  const contentLength = `Content-Length: ${content.length.toString()}`;

  const message = [
    contentType,
    contentLength,
    '',
    `${content}`,
  ].join('\r\n');

  return message;
}

/**
 * Parse a JSON RPC message accordingly to LSP protocol.
 * See "Base protocol" in LSP documentation
 *
 * @link https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#baseProtocol
 * @param message Message received
 * @returns
 */
function parseMessage(message: Buffer): ResponsePayload | undefined {
  const lines = message
    .toString()
    .split('\r\n')
    .filter((line) => line && !line.startsWith('Content-'));

  if (!message.length) {
    return undefined;
  }

  const content = lines[0];

  if (!content) {
    return undefined;
  }

  try {
    return JSON.parse(content);
  } catch (err) {
    throw new Error(`Error while parsing message: ${content}`);
  }
}

async function connectToServer(
  port: number,
  host: string,
  timeout: number,
): Promise<net.Socket | undefined> {
  const startDate = Date.now();

  while (Date.now() - startDate < timeout) {
    // eslint-disable-next-line no-await-in-loop
    const socket = await trySocketConnection(port, host);

    if (socket) {
      return socket;
    }
    // eslint-disable-next-line no-await-in-loop
    await wait(100); // Waiting 100ms before trying again to connect
  }

  return undefined;
}

function trySocketConnection(port: number, host: string) : Promise<net.Socket | undefined> {
  return new Promise((resolve) => {
    const socket = new net.Socket();

    socket.once('connect', () => resolve(socket));
    socket.once('error', () => resolve(undefined));

    socket.connect(
      port,
      host,
    );
  });
}

/**
 * Return a promise that would be resolved
 * once the duration has finish
 */
async function wait(duration: number) {
  return new Promise((resolve) => {
    setTimeout(resolve, duration);
  });
}

// ============================================================
// Types

export default RpcClient;
export {
  EventType,
  RequestDefinitions,
  RequestPayload,
  ResponsePayload,
  Status,
};
