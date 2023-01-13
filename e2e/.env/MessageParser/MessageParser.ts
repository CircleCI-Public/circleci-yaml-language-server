import assert from 'assert';
import DataReader from './DataReader';
import { ResponsePayload } from '../RpcClient';
import {
    CONTENT_LENGTH_HEADER_LOWERCASE_PREFIX,
    HEADER_LINE_SEPARATOR,
} from './common';

type ParsingState =
    | { kind: 'parsing-body'; contentLength: number }
    | { kind: 'parsing-headers' };

class MessageParser {
    #state: ParsingState = { kind: 'parsing-headers' };
    #remaining: string = '';

    /**
     * Takes a buffer and returns the LSP messages it has parsed from it
     * This method may buffer data between consecutives calls. This is why a class is needed
     * For more information, see the documentation in ./index.ts
     */
    parseMessage(data: Buffer): ResponsePayload[] {
        const responses: ResponsePayload[] = [];
        const content = this.#remaining + data.toString();
        const reader = new DataReader(content);

        while (reader.hasMoreData) {
            if (this.#state.kind === 'parsing-headers') {
                this.#parseHeaders(reader);
                continue;
            }
            if (this.#state.kind === 'parsing-body') {
                const newResponse = this.#parseContent(reader);
                if (newResponse !== undefined) {
                    responses.push(newResponse);
                }
                continue;
            }
        }
        return responses;
    }

    #parseHeaders(reader: DataReader) {
        assert(this.#state.kind === 'parsing-headers');
        const headers = reader.parseHeaders();

        if (!headers.isComplete) {
            this.#remaining = headers.content;
            return;
        }

        this.#remaining = '';
        const contentLengthHeader = headers.content
            .split(HEADER_LINE_SEPARATOR)
            .find((header) =>
                header
                    .toLowerCase()
                    .startsWith(CONTENT_LENGTH_HEADER_LOWERCASE_PREFIX),
            );

        assert(
            contentLengthHeader !== undefined,
            'Content-Length header not found',
        );
        const contentLength = Number(
            contentLengthHeader.slice(
                CONTENT_LENGTH_HEADER_LOWERCASE_PREFIX.length,
            ),
        );
        this.#state = {
            kind: 'parsing-body',
            contentLength,
        };
    }

    #parseContent(reader: DataReader): ResponsePayload | undefined {
        assert(this.#state.kind === 'parsing-body');
        const content = reader.parseContent(this.#state.contentLength);

        if (!content.isComplete) {
            this.#remaining = content.content;
            return;
        }
        this.#remaining = '';
        this.#state = { kind: 'parsing-headers' };
        return JSON.parse(content.content);
    }
}

export default MessageParser;
