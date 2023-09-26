import { ResponsePayload } from '../RpcClient';
import MessageParser from './MessageParser';

describe('MessageParser', function () {
    type TestCase = {
        name: string;
        messages: string[];
        expected: unknown[];
    };

    const testCases: TestCase[] = [
        {
            name: 'should read a simple message',
            messages: [
                'Content-Length: 42\r\n\r\n{"some":"json-payload-of-length-42-bytes"}',
            ],
            expected: [{ some: 'json-payload-of-length-42-bytes' }],
        },
        {
            name: 'should read two messages',
            messages: [
                'Content-Length: 42\r\n\r\n{"some":"json-payload-of-length-42-bytes"}Content-Length: 42\r\n\r\n{"some":"json-payload-of-length-42-bytes"}',
            ],
            expected: [
                { some: 'json-payload-of-length-42-bytes' },
                { some: 'json-payload-of-length-42-bytes' },
            ],
        },
        {
            name: 'should read a message segmented after end of headers',
            messages: [
                'Content-Length: 42\r\n\r\n',
                '{"some":"json-payload-of-length-42-bytes"}',
            ],
            expected: [{ some: 'json-payload-of-length-42-bytes' }],
        },
        {
            name: 'should read a message segmented before the end of headers',
            messages: [
                'Content-Length: 42\r\n',
                '\r\n{"some":"json-payload-of-length-42-bytes"}',
            ],
            expected: [{ some: 'json-payload-of-length-42-bytes' }],
        },
        {
            name: 'should read a message segmented in the middle of the headers',
            messages: [
                'Content-',
                'Length: 42\r\n\r\n{"some":"json-payload-of-length-42-bytes"}',
            ],
            expected: [{ some: 'json-payload-of-length-42-bytes' }],
        },
        {
            name: 'should read a message segmented in the middle of the content',
            messages: [
                'Content-Length: 42\r\n\r\n{"some":',
                '"json-payload-of-length-42-bytes"}',
            ],
            expected: [{ some: 'json-payload-of-length-42-bytes' }],
        },
    ];

    test.each(testCases)('$name', ({ messages, expected }) => {
        const parser = new MessageParser();
        const responses: ResponsePayload[][] = [];

        for (const message of messages) {
            responses.push(parser.parseMessage(Buffer.from(message)));
        }
        expect(responses.flat()).toEqual(expected);
    });
});
