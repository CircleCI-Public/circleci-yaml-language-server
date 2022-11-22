import * as vscode from 'vscode';
import { doHover } from './hover';

import { LSP } from './server';

type CommandName = string;
type CommandHandler = (...args: any[]) => any;

let lsp: LSP | undefined = undefined;

export async function activate(context: vscode.ExtensionContext) {
    try {
        lsp = new LSP(context);
        await lsp?.start();
        const commandHandlers: Record<CommandName, CommandHandler> = {
            'circleci-language-server.restartServer': () => {
                lsp?.restart();
            },
            'circleci-language-server.selectTagAndComplete': () => {
                // Change editor selection to have a tag selected
                const editor = vscode.window.activeTextEditor
                const document = vscode.window.activeTextEditor?.document

                if (!document || !editor) { return }

                const r = document.getWordRangeAtPosition(editor.selection.start, new RegExp("([A-Za-z0-9_]+[\.|\-]*)+"))

                if (r?.start) {
                    editor.selections = [
                        // Important to activate the cursor at the START of the selection. Has an importance when autocompleting in Language Server
                        new vscode.Selection(
                            r.end,
                            r.start,
                        )
                    ]
                }

                // Trigger completion again
                vscode.commands.executeCommand("editor.action.triggerSuggest")
            }
        };
        const wrap = (
            name: CommandName,
            handler: CommandHandler,
        ): CommandHandler => {
            return async (...args: any[]) => {
                try {
                    await handler(...args);
                } catch (e) {
                    console.error('container', 'command failed:', name, e);
                }
            };
        };
        Object.keys(commandHandlers).forEach((commandName) => {
            context.subscriptions.push(
                vscode.commands.registerCommand(
                    commandName,
                    wrap(commandName, commandHandlers[commandName]),
                ),
            );
        });

        const redHatYAMLExtension =
            vscode.extensions.getExtension('redhat.vscode-yaml');

        if (!redHatYAMLExtension?.isActive) {
            vscode.languages.registerHoverProvider(
                {
                    scheme: 'file',
                    language: 'yaml',
                    pattern: '**/.circleci/**/*',
                },
                {
                    provideHover: (
                        document: vscode.TextDocument,
                        position: vscode.Position,
                    ): vscode.ProviderResult<vscode.Hover> => {
                        return doHover(
                            context,
                            {
                                ...document,
                                uri: document.uri.toString(),
                            },
                            position,
                        );
                    },
                },
            );
        }
    } catch (e) {
        console.trace();
        console.error(e);
    }
}

export async function deactivate(context: vscode.ExtensionContext) {
    await lsp?.stop();
}
