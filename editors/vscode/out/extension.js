"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deactivate = exports.activate = void 0;
const vscode = require("vscode");
const hover_1 = require("./hover");
const server_1 = require("./server");
let lsp = undefined;
async function activate(context) {
    try {
        lsp = new server_1.LSP(context);
        await lsp?.start();
        const commandHandlers = {
            'circleci-language-server.restartServer': () => {
                lsp?.restart();
            },
            'circleci-language-server.selectTagAndComplete': () => {
                // Change editor selection to have a tag selected
                const editor = vscode.window.activeTextEditor;
                const document = vscode.window.activeTextEditor?.document;
                if (!document || !editor) {
                    return;
                }
                const r = document.getWordRangeAtPosition(editor.selection.start, new RegExp('([A-Za-z0-9_]+[.|-]*)+'));
                if (r?.start) {
                    editor.selections = [
                        // Important to activate the cursor at the START of the selection. Has an importance when autocompleting in Language Server
                        new vscode.Selection(r.end, r.start),
                    ];
                }
                // Trigger completion again
                vscode.commands.executeCommand('editor.action.triggerSuggest');
            },
        };
        const wrap = (name, handler) => {
            return async (...args) => {
                try {
                    await handler(...args);
                }
                catch (e) {
                    console.error('container', 'command failed:', name, e);
                }
            };
        };
        Object.keys(commandHandlers).forEach((commandName) => {
            context.subscriptions.push(vscode.commands.registerCommand(commandName, wrap(commandName, commandHandlers[commandName])));
        });
        const redHatYAMLExtension = vscode.extensions.getExtension('redhat.vscode-yaml');
        if (!redHatYAMLExtension?.isActive) {
            vscode.languages.registerHoverProvider({
                scheme: 'file',
                language: 'yaml',
                pattern: '**/.circleci/**/*',
            }, {
                provideHover: (document, position) => {
                    return (0, hover_1.doHover)(context, {
                        ...document,
                        uri: document.uri.toString(),
                    }, position);
                },
            });
        }
    }
    catch (e) {
        console.trace();
        console.error(e);
    }
}
exports.activate = activate;
async function deactivate(context) {
    await lsp?.stop();
}
exports.deactivate = deactivate;
//# sourceMappingURL=extension.js.map