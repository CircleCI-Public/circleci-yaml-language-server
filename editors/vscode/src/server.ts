import * as os from 'os';
import * as net from 'net';
import * as path from 'path';
import getPort from 'get-port';
import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as lc from 'vscode-languageclient/node';

import {
    createDeferredPromise,
    outputChannel,
    isAppleSilicon,
    isInDevMode,
} from './utils';
import { readFileSync } from 'fs';

export class LSP {
    private _server: cp.ChildProcess | undefined;
    private readonly serverPath: string;
    private readonly context: vscode.ExtensionContext;
    private _client: lc.LanguageClient | undefined;

    constructor(context: vscode.ExtensionContext) {
        this.context = context;
        const isDev = isInDevMode();
        const serverBinary = this.getServerBinaryFileName();

        if (!serverBinary) {
            throw new Error('Unsupported platform');
        }

        this.serverPath = isDev
            ? context.asAbsolutePath(
                  path.join('..', '..', 'bin', 'start_server'),
              )
            : context.asAbsolutePath(path.join('bin', serverBinary));
        this.serverPath =
            os.platform() == 'win32'
                ? `${this.serverPath}.exe`
                : this.serverPath;
    }

    getServerBinaryFileName(): string | undefined {
        switch (os.platform()) {
            case 'darwin':
                const onAppleSilicon = isAppleSilicon();
                return `${os.platform()}-${
                    onAppleSilicon ? 'arm64' : 'amd64'
                }-lsp`;

            case 'linux':
                const arch = os.arch();
                return `${os.platform()}-${
                    ['arm64', 'arm'].includes(arch) ? 'arm64' : 'amd64'
                }-lsp`;

            case 'win32':
                if (os.arch() == 'x64') {
                    return 'windows-amd64-lsp';
                }
        }
    }

    get server(): cp.ChildProcess {
        if (!this._server) {
            throw new Error('Server not initialized');
        }
        return this._server;
    }

    get client(): lc.LanguageClient {
        if (!this._client) {
            throw new Error('Client not initialized');
        }
        return this._client;
    }

    async start() {
        this._client = await this.initLSPClient();
    }

    async stop() {
        await this.client.stop();
        this.server.kill();
    }

    async restart() {
        await this.stop();
        await this.start();
    }

    private async spawnLSPServer(port: number): Promise<cp.ChildProcess> {
        const inDevMode = isInDevMode();

        const schemaLocation = inDevMode
            ? this.context.asAbsolutePath(path.join('..', '..', 'schema.json'))
            : this.context.asAbsolutePath(path.join('schema.json'));

        const servProcess = cp.spawn(this.serverPath, [], {
            env: {
                SCHEMA_LOCATION: schemaLocation,
                HOME: os.homedir(),
                PORT: port.toString(),
            },
        });
        const promise = createDeferredPromise<cp.ChildProcess>();

        servProcess.on('message', outputChannel().appendLine);
        servProcess.on('error', outputChannel().appendLine);
        servProcess.on('exit', outputChannel().appendLine);
        servProcess.on('close', outputChannel().appendLine);
        servProcess.on('disconnect', outputChannel().appendLine);
        servProcess.stderr.on('data', outputChannel().appendLine);
        servProcess.stdout.on('data', outputChannel().appendLine);

        const timeout = setTimeout(() => {
            promise.reject('LSP server did not start in time');
        }, 10000);

        const serverStarted = (data: string) => {
            const value = Buffer.isBuffer(data) ? data.toString() : data;
            if (value.trim().startsWith('Server started')) {
                clearTimeout(timeout);
                promise.resolve(servProcess);
                servProcess.stdout.removeListener('data', serverStarted);
            }
        };

        servProcess.stdout.on('data', serverStarted);

        return promise.promise;
    }

    private spawnLSPClient(): lc.LanguageClient {
        /**
         * Spawn and connect to the LSP server
         */
        const serverOptions = async () => {
            const port = await getPort();
            this._server = await this.spawnLSPServer(port);
            const connectionInfo = {
                port,
                host: 'localhost',
            };
            const socket = net.connect(connectionInfo);
            const result: lc.StreamInfo = {
                writer: socket,
                reader: socket,
            };
            return await Promise.resolve(result);
        };

        const clientOptions: lc.LanguageClientOptions = {
            documentSelector: [
                {
                    scheme: 'file',
                    language: 'yaml',
                    pattern: '**/.circleci/**/*',
                },
            ],
            synchronize: {
                configurationSection: ['yaml'],
                fileEvents:
                    vscode.workspace.createFileSystemWatcher('**/*.(yml|yaml)'),
            },
            diagnosticPullOptions: {
                onChange: true,
                onSave: true,
                onTabs: true,
            },
            diagnosticCollectionName: 'cci-diag',
            initializationOptions: {
                isCciExtension: true,
                isCrashTelemetryEnabled: true,
                telemetryID: 'id',
            },

            outputChannel: outputChannel(),
        };

        const client = new lc.LanguageClient(
            'cci-language-server',
            'CircleCI Language Server',
            serverOptions,
            clientOptions,
        );

        client.onTelemetry((event) => console.log('Telemetry event', event));

        /*
         * Example of request to activate rollbar
         *
         * client.sendRequest('workspace/executeCommand', {
         *     command: 'setRollbarInformation',
         *     arguments: [
         *         {
         *             enabled: true,
         *             environment: 'development',
         *             sessionId: vscode.env.sessionId,
         *             machineId: vscode.env.machineId,
         *             machine: `${os.platform}/${os.arch}`,
         *             personId: 'id',
         *             requestIp: '1.2.4.8',
         *         },
         *     ],
         * });
         */

        return client;
    }

    private async initLSPClient(): Promise<lc.LanguageClient> {
        const client = this.spawnLSPClient();

        await client.start();

        const token = process.env.TOKEN;
        const setTokenCommand = {
            command: 'setToken',
            arguments: [token],
        };
        await client.sendRequest('workspace/executeCommand', setTokenCommand);

        const selfHostedUrl = process.env.SELF_HOSTED_URL;
        const setHostUrlCommand = {
            command: 'setSelfHostedUrl',
            arguments: [selfHostedUrl],
        };
        await client.sendRequest('workspace/executeCommand', setHostUrlCommand);

        const projectSlug = 'gh/CircleCI-Public/circleci-yaml-language-server';
        const setProjectSlugCommand = {
            command: 'setProjectSlug',
            arguments: [projectSlug],
        };
        await client.sendRequest(
            'workspace/executeCommand',
            setProjectSlugCommand,
        );

        const filePath = path.join(
            __dirname,
            '..',
            '..',
            '..',
            '.circleci',
            'config.yml',
        );
        const content = readFileSync(filePath, {
            encoding: 'utf-8',
        });
        const getWorkflowsCommand = {
            command: 'getWorkflows',
            arguments: [content, filePath],
        };
        const res = await client.sendRequest(
            'workspace/executeCommand',
            getWorkflowsCommand,
        );
        console.log(res);

        return client;
    }
}
