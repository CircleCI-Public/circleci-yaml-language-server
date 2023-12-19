"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.outputChannel = exports.traceOutputChannel = void 0;
const vscode = require("vscode");
let TRACE_OUTPUT_CHANNEL = null;
function traceOutputChannel() {
    if (!TRACE_OUTPUT_CHANNEL) {
        TRACE_OUTPUT_CHANNEL = vscode.window.createOutputChannel('Circle CI Language Server Trace');
    }
    return TRACE_OUTPUT_CHANNEL;
}
exports.traceOutputChannel = traceOutputChannel;
let OUTPUT_CHANNEL = null;
function outputChannel() {
    if (!OUTPUT_CHANNEL) {
        OUTPUT_CHANNEL = vscode.window.createOutputChannel('Circle CI Language Server');
    }
    return OUTPUT_CHANNEL;
}
exports.outputChannel = outputChannel;
//# sourceMappingURL=outputChannel.js.map