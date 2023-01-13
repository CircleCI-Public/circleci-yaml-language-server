import { HEADER_LINE_SEPARATOR } from './common';

type Response = { isComplete: boolean; content: string };

/**
 * This class aims at simplifying the manipulation of the message data
 * You can ask for the part of the message you want and it will return it and advance the buffer
 */
class DataReader {
    #data: string;

    constructor(data: string) {
        this.#data = data;
    }

    get hasMoreData() {
        return this.#data.length > 0;
    }

    parseHeaders(): Response {
        let content = '';

        while (this.#data.length !== 0) {
            const header = this.#getNextHeader();
            if (header === HEADER_LINE_SEPARATOR) {
                return { isComplete: true, content };
            }
            content += header;
        }
        return { isComplete: false, content };
    }

    parseContent(contentLength: number): Response {
        const content = this.#data.slice(0, contentLength);
        const isComplete = content.length === contentLength;

        this.#data = this.#data.slice(contentLength);
        return { isComplete, content };
    }

    #getNextHeader(): string {
        let eofIndex = this.#data.indexOf(HEADER_LINE_SEPARATOR);

        if (eofIndex === -1) {
            const line = this.#data.slice();
            this.#data = '';
            return line;
        }
        const line = this.#data.slice(
            0,
            eofIndex + HEADER_LINE_SEPARATOR.length,
        );
        this.#data = this.#data.slice(eofIndex + HEADER_LINE_SEPARATOR.length);
        return line;
    }
}

export default DataReader;
export type { Response };
