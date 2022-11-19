import type { Config as JestConfig } from 'jest';

import type {
  EnvOptions,
} from './src/types';

/*
 * For a detailed explanation regarding each configuration property and type check, visit:
 * https://jestjs.io/docs/configuration
 */

type Config = JestConfig & {
  testEnvironmentOptions: EnvOptions,
};

const config : Config = {
  // Automatically clear mock calls, instances, contexts and results before every test
  clearMocks: true,

  // A preset that is used as a base for Jest's configuration
  preset: 'ts-jest',

  reporters: [
    'default',
    [
      'jest-junit',
      {
        outputDirectory: 'reports',
      },
    ],
  ],

  // The test environment that will be used for testing
  testEnvironment: './src/.env/environment.ts',

  // Options that will be passed to the testEnvironment
  testEnvironmentOptions: {
    lspServer: {
      // Port of the LSP server
      // Env variable: PORT
      // Default: <none>
      port: 10001,

      // Host of the LSP Server
      // Env variable: LSP_SERVER_HOST
      // Default: localhost
      host: 'localhost',

      // Should the server be spawn ?
      // Accepted values are: 1, 0, on, off, true, false, yes, no
      // Env variable: SPAWN_LSP_SERVER
      // Default: false
      spawn: true,

      // Path to the server binary
      // Env variable: RPC_SERVER_BIN
      // Default: <none>
      binPath: '../bin/start_server', // Name of the build after the "task build" command

      // Path to the schema.
      // If relative, it will be relative to the current folder
      // Env variable: SCHEMA_LOCATION
      // Default: <none>
      jsonSchemaLocation: '../publicschema.json',
    },
  },
};

export default config;
