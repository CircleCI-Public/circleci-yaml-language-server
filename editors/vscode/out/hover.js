"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.setHoverInLanguageServer = exports.getCircleHoverLanguageService = exports.getYamlLanguageService = exports.doHover = void 0;
const path = require("path");
const vscode = require("vscode");
const fs_1 = require("fs");
const vscode_json_languageservice_1 = require("vscode-json-languageservice");
const yamlParser07_1 = require("yaml-language-server/out/server/src/languageservice/parser/yamlParser07");
const arrUtils_1 = require("yaml-language-server/out/server/src/languageservice/utils/arrUtils");
const utils_1 = require("./utils");
let lsHover = undefined;
async function doHover(context, document, position) {
    if (!lsHover) {
        lsHover = (0, exports.getYamlLanguageService)(context);
    }
    // @ts-ignore
    let hover = await lsHover.doHover(document, position);
    if (hover && Array.isArray(hover?.contents)) {
        const markdownString = new vscode.MarkdownString(hover.contents.join('\n').replace(/\\/g, ''));
        let range = undefined;
        if (hover.range) {
            range = new vscode.Range(new vscode.Position(hover.range.start.line, hover.range.start.character), new vscode.Position(hover.range.end.line, hover.range.end.character));
        }
        return new vscode.Hover(markdownString, range);
    }
    return hover;
}
exports.doHover = doHover;
const getYamlLanguageService = function (context) {
    const publicSchemaLocation = (0, utils_1.isInDevMode)()
        ? context.asAbsolutePath(path.join('..', '..', 'publicschema.json'))
        : context.asAbsolutePath(path.join('publicschema.json'));
    const publicSchema = (0, fs_1.readFileSync)(publicSchemaLocation, 'utf8');
    const parsedPublicSchema = JSON.parse(publicSchema);
    const circleciHoverLanguageService = (0, exports.getCircleHoverLanguageService)(parsedPublicSchema);
    return (0, exports.setHoverInLanguageServer)(circleciHoverLanguageService);
};
exports.getYamlLanguageService = getYamlLanguageService;
const getCircleHoverLanguageService = function (schema) {
    const builtInParams = {};
    const languageService = (0, vscode_json_languageservice_1.getLanguageService)({
        ...builtInParams,
    });
    languageService.configure({
        validate: true,
        allowComments: false,
        schemas: [
            {
                uri: 'json',
                fileMatch: ['*'],
                schema: schema,
            },
        ],
    });
    return languageService;
};
exports.getCircleHoverLanguageService = getCircleHoverLanguageService;
const setHoverInLanguageServer = function (circleciHoverLanguageService) {
    const builtInParams = {};
    const languageService = (0, vscode_json_languageservice_1.getLanguageService)({
        ...builtInParams,
    });
    languageService.doHover = function (document, position) {
        const doc = (0, yamlParser07_1.parse)(document.getText());
        const offset = document.offsetAt(position);
        const currentDoc = (0, arrUtils_1.matchOffsetToDocument)(offset, doc);
        if (!currentDoc) {
            return Promise.resolve(null);
        }
        const currentDocIndex = doc.documents.indexOf(currentDoc);
        currentDoc.currentDocIndex = currentDocIndex;
        return circleciHoverLanguageService.doHover(document, position, currentDoc);
    };
    return languageService;
};
exports.setHoverInLanguageServer = setHoverInLanguageServer;
//# sourceMappingURL=hover.js.map