/* eslint-disable no-console */
/**
 * This file define the custom jest testing environment.
 */
// ============================================================
// Import packages
import { spawn } from 'child_process';
import { EnvironmentContext, JestEnvironmentConfig } from '@jest/environment';
import { TestEnvironment } from 'jest-environment-node';

import type {
  ChildProcess,
} from 'child_process';

// ============================================================
// Import modules
import { CommandDefinitions } from '../types';
import {
  getJsonSchemaLocation,
  getLspClientHost,
  getLspClientPort,
  getServerBinaryPath,
  shouldSpawnServer,
} from './config';
import RpcClient, { Status } from './RpcClient';
import { projectRoot } from './utils';

import type {
  ProjectConfig,
} from './config';

// ============================================================
// Class

/**
 * Define the LSP Testing environment variables.
 */
class LSPTestingEnvironment<T extends CommandDefinitions> extends TestEnvironment {
  readonly port: number;

  readonly host: string;

  readonly serverBinPath: string | undefined;

  readonly testPath: string;

  #client: RpcClient<T>;

  #serverProcess: ChildProcess | undefined;

  #projectConfig: ProjectConfig;

  constructor(
    config: JestEnvironmentConfig,
    context: EnvironmentContext,
  ) {
    super(config, context);

    this.#projectConfig = config.projectConfig;

    this.port = determinePort(this.#projectConfig);
    this.host = getLspClientHost(config.projectConfig);
    this.testPath = context.testPath;

    this.#client = new RpcClient<T>(this.port, this.host, this.testPath);
  }

  /**
   * Setup the environment.
   * The setup will:
   *  - Setup the NodeJS environment
   *  - Spawn the LSP server if requested (see configuration)
   *  - Connect the client
   *
   * If the client didn't connect after 10s, the setup fail.
   */
  async setup() {
    globalThis.latestRequestId += 1;

    await super.setup();
    await this.#spawnServer();
    await this.#connectClient();
  }

  async teardown(): Promise<void> {
    await super.teardown();
    await this.#disconnectClient();
    await this.#stopServer();
  }

  async #disconnectClient() {
    console.info('Disconnecting client...');
    const promises = [
      super.teardown(),
      this.#client.disconnect(),
    ];

    await Promise.all(promises);
    console.info('Client disconnected');
  }

  async #connectClient() {
    const address = `${this.host}:${this.port}`;
    console.info(`Trying to connect to LSP server on ${address}...`);
    await this.#client.connect(10000);

    if (this.#client.status !== Status.connected) {
      throw new Error(`Unable to connect to the server ${address}`);
    }

    console.info('Connected to LSP server');

    this.global.rpcClient = this.#client as unknown as typeof this.global.rpcClient;
  }

  async #spawnServer() {
    if (!shouldSpawnServer(this.#projectConfig)) {
      console.info('Not starting LSP server');
    }

    const serverBinPath = getServerBinaryPath(this.#projectConfig);
    if (!serverBinPath) {
      throw new Error('Server bin path not defined but expected to be spawn');
    }

    const jsonSchema = getJsonSchemaLocation(this.#projectConfig);
    if (!jsonSchema) {
      throw new Error('JSON schema must be defined to spawn the server');
    }

    const address = `${this.host}:${this.port}`;

    console.info(`Starting the LSP server on ${address}...`);
    console.info(`  . Server binary: ${serverBinPath}`);
    console.info(`  . JSON Schema: ${jsonSchema}`);

    this.#serverProcess = await spawnServer(this.port, this.host, serverBinPath, jsonSchema);

    console.info(`[${address}] LSP server started`);
  }

  #stopServer() {
    if (!this.#serverProcess) {
      return;
    }
    const address = `${this.host}:${this.port}`;
    console.info(`[${address}] Stopping LSP server...`);
    try {
      this.#serverProcess.kill();
    } catch (err) {
      console.error(`[${address}] Error while stopping LSP server`, err);
    }

    console.info(`[${address}] LSP server stopped`);
  }
}

async function spawnServer(
  port: number,
  host: string,
  serverBinPath: string,
  jsonSchema: string,
) {
  globalThis.latestRequestId = 0;

  const serverProcess = spawn(
    serverBinPath,
    [
      '--port', port.toString(),
      '--host', host,
      '--schema', jsonSchema,
    ],
    {
      cwd: projectRoot,
    },
  );

  return serverProcess;
}

/**
 * Determine the LSP port.
 *
 * Will return the port defined in the configuration if the
 * LSP server is not expected to spawn.
 *
 * If the LSP server must be spawned, then each test suite will
 * have it's own server: the port will be a number incremented
 * from the port defined in the configuration.
 */
function determinePort(projectConfig: ProjectConfig): number {
  if (!shouldSpawnServer(projectConfig)) {
    return getLspClientPort(projectConfig);
  }

  const workerId = parseInt(process.env.JEST_WORKER_ID || '', 10);

  if (Number.isNaN(workerId)) {
    throw new Error('Worker ID not found');
  }

  const firstPort = getLspClientPort(projectConfig);

  return firstPort + workerId - 1;
}

// ============================================================
// Exports
export default LSPTestingEnvironment;
