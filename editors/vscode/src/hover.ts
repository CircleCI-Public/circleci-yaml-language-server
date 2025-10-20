import * as path from 'path';
import * as vscode from 'vscode';
import { readFileSync } from 'fs';
import { Position, TextDocument } from 'vscode-languageserver-textdocument';
import {
    getLanguageService as getLanguageServiceVscode,
    Hover,
    JSONSchema,
    LanguageService,
} from 'vscode-json-languageservice';
import { parse as parseYAML } from 'yaml-language-server/out/server/src/languageservice/parser/yamlParser07';
import { matchOffsetToDocument } from 'yaml-language-server/out/server/src/languageservice/utils/arrUtils';

import { isInDevMode } from './utils';

let lsHover: LanguageService | undefined = undefined;

export async function doHover(
    context: vscode.ExtensionContext,
    document: TextDocument,
    position: Position,
): Promise<vscode.Hover | null> {
    if (!lsHover) {
        lsHover = getYamlLanguageService(context);
    }

    // @ts-ignore
    let hover = await lsHover.doHover(document, position);

    if (hover && Array.isArray(hover?.contents)) {
        const markdownString = new vscode.MarkdownString(
            hover.contents.join('\n').replace(/\\/g, ''),
        );

        let range: vscode.Range | undefined = undefined;
        if (hover.range) {
            range = new vscode.Range(
                new vscode.Position(
                    hover.range.start.line,
                    hover.range.start.character,
                ),
                new vscode.Position(
                    hover.range.end.line,
                    hover.range.end.character,
                ),
            );
        }

        return new vscode.Hover(markdownString, range);
    }

    return hover as vscode.Hover;
}

export const getYamlLanguageService = function (
    context: vscode.ExtensionContext,
): LanguageService {
    const schemaLocation = isInDevMode()
        ? context.asAbsolutePath(path.join('..', '..', 'schema.json'))
        : context.asAbsolutePath(path.join('schema.json'));

    const schema = readFileSync(schemaLocation, 'utf8');
    const parsedSchema = JSON.parse(schema) as JSONSchema;

    const circleciHoverLanguageService: LanguageService =
        getCircleHoverLanguageService(parsedSchema);

    return setHoverInLanguageServer(circleciHoverLanguageService);
};

export const getCircleHoverLanguageService = function (
    schema: JSONSchema,
): LanguageService {
    const builtInParams = {};
    const languageService = getLanguageServiceVscode({
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

export const setHoverInLanguageServer = function (
    circleciHoverLanguageService: LanguageService,
): LanguageService {
    const builtInParams = {};

    const languageService = getLanguageServiceVscode({
        ...builtInParams,
    });

    languageService.doHover = function (
        document: TextDocument,
        position: Position,
    ): Thenable<Hover | null> {
        const doc = parseYAML(document.getText());
        const offset = document.offsetAt(position);
        const currentDoc = matchOffsetToDocument(offset, doc);
        if (!currentDoc) {
            return Promise.resolve(null);
        }

        const currentDocIndex = doc.documents.indexOf(currentDoc);
        currentDoc.currentDocIndex = currentDocIndex;

        return circleciHoverLanguageService.doHover(
            document,
            position,
            currentDoc,
        );
    };

    return languageService;
};
