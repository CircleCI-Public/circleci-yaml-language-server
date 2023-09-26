// ============================================================
// Import packages
import path from 'path';
import { JestEnvironmentConfig } from '@jest/environment';

// ============================================================
// Import modules
import type { EnvOptions } from '../utils';

// ============================================================
// Module's constants and variables
const DEFAULT_RPC_SERVER_PORT = 10001;
const DEFAULT_RPC_SERVER_HOST = '127.0.0.1';

type GlobalConfig = JestEnvironmentConfig['globalConfig'];
type ProjectConfig = JestEnvironmentConfig['projectConfig'];

// ============================================================
// Functions
function getEnvOptions(config: ProjectConfig): EnvOptions {
  return config.testEnvironmentOptions;
}

function getJsonSchemaLocation(config: ProjectConfig): string | undefined {
  const schemaLocation = process.env.SCHEMA_LOCATION
    || getEnvOptions(config).lspServer?.jsonSchemaLocation;

  if (!schemaLocation) {
    return undefined;
  }

  if (path.isAbsolute(schemaLocation)) {
    return schemaLocation;
  }

  return path.resolve(schemaLocation);
}

/**
 * Return the host to use for the RPC client.
 * The function will:
 *  - look for the PORT environment variable
 *  - look for the "rpcServer.port" value of the testEnvironmentOptions configuration object
 *  - use the default value (10001)
 */
function getLspClientHost(config: ProjectConfig) : string {
  if (process.env.RPC_SERVER_HOST) {
    return process.env.RPC_SERVER_HOST;
  }

  // Looking within configuration file
  const envOptions : EnvOptions = getEnvOptions(config);

  return envOptions.lspServer?.host || DEFAULT_RPC_SERVER_HOST;
}

/**
 * Return the port to use for the RPC client.
 * The function will:
 *  - look for the PORT environment variable
 *  - look for the "rpcServer.port" value of the testEnvironmentOptions configuration object
 *  - use the default value (10001)
 */
function getLspClientPort(config: ProjectConfig) : number {
  const envOptions : EnvOptions = getEnvOptions(config);

  if (process.env.PORT) {
    const portStr = process.env.PORT;

    const port = parseInt(portStr, 10);

    if (!Number.isNaN(port)) {
      return port;
    }
  }

  return envOptions.lspServer?.port || DEFAULT_RPC_SERVER_PORT;
}

/**
 * Return the LSP binary path.
 */
function getServerBinaryPath(config: ProjectConfig): string | undefined {
  const envOptions : EnvOptions = getEnvOptions(config);

  const binPath = process.env.RPC_SERVER_BIN || envOptions.lspServer?.binPath;

  if (!binPath) {
    return undefined;
  }

  if (binPath[0] === '/') {
    return binPath;
  }

  if (binPath[0] === '~') {
    return path.join(
      process.env.HOME as string,
      binPath.slice(1),
    );
  }

  return path.resolve(binPath);
}

function shouldSpawnServer(config: ProjectConfig): boolean {
  const envValue = process.env.SPAWN_LSP_SERVER || '';

  const truthyValue = ['1', 'on', 'true', 'yes'];
  const falsyValue = ['0', 'off', 'false', 'no'];

  if (truthyValue.includes(envValue)) {
    return true;
  }

  if (falsyValue.includes(envValue)) {
    return false;
  }

  const envOptions : EnvOptions = getEnvOptions(config);

  return envOptions.lspServer?.spawn || false;
}

export {
  getJsonSchemaLocation,
  getLspClientHost,
  getLspClientPort,
  getServerBinaryPath,
  shouldSpawnServer,
};

export type {
  GlobalConfig,
  ProjectConfig,
};
