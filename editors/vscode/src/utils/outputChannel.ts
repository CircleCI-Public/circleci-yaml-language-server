import * as vscode from 'vscode';

let TRACE_OUTPUT_CHANNEL: vscode.OutputChannel | null = null;
export function traceOutputChannel() {
    if (!TRACE_OUTPUT_CHANNEL) {
        TRACE_OUTPUT_CHANNEL = vscode.window.createOutputChannel(
            'Circle CI Language Server Trace',
        );
    }
    return TRACE_OUTPUT_CHANNEL;
}
let OUTPUT_CHANNEL: vscode.OutputChannel | null = null;
export function outputChannel() {
    if (!OUTPUT_CHANNEL) {
        OUTPUT_CHANNEL = vscode.window.createOutputChannel(
            'Circle CI Language Server',
        );
    }
    return OUTPUT_CHANNEL;
}
