import DataReader, { Response } from './DataReader';

describe('DataReader', function () {
    describe('parseHeaders', function () {
        type TestCase = {
            name: string;
            input: string;
            expected: {
                result: Response;
                hasMoreData: boolean;
            };
        };
        const testCases: TestCase[] = [
            {
                name: 'should read standard headers',
                input: `Content-Length: 42\r\n\r\n`,
                expected: {
                    result: {
                        isComplete: true,
                        content: `Content-Length: 42\r\n`,
                    },
                    hasMoreData: false,
                },
            },
            {
                name: 'should read complete headers',
                input: `Content-Length: 42\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n`,
                expected: {
                    result: {
                        isComplete: true,
                        content: `Content-Length: 42\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n`,
                    },
                    hasMoreData: false,
                },
            },
            {
                name: 'should read standard message',
                input: `Content-Length: 42\r\n\r\n{"some":"json-payload-of-length-42-bytes"}`,
                expected: {
                    result: {
                        isComplete: true,
                        content: 'Content-Length: 42\r\n',
                    },
                    hasMoreData: true,
                },
            },
            {
                name: 'should read incomplete headers',
                input: `Content-Length: 42\r\n`,
                expected: {
                    result: {
                        isComplete: false,
                        content: `Content-Length: 42\r\n`,
                    },
                    hasMoreData: false,
                },
            },
            {
                name: 'should read broken headers',
                input: `Content-Length:`,
                expected: {
                    result: {
                        isComplete: false,
                        content: 'Content-Length:',
                    },
                    hasMoreData: false,
                },
            },
        ];

        test.each(testCases)('$name', ({ input, expected }) => {
            const dataReader = new DataReader(input);

            expect(dataReader.parseHeaders()).toEqual(expected.result);
            expect(dataReader.hasMoreData).toEqual(expected.hasMoreData);
        });
    });

    describe('parseContent', function () {
        type TestCase = {
            name: string;
            input: {
                contructorParameter: string;
                contentLength: number;
            };
            expected: {
                result: Response;
                hasMoreData: boolean;
            };
        };
        const testCases: TestCase[] = [
            {
                name: 'should parse standard content',
                input: {
                    contructorParameter:
                        '{"some":"json-payload-of-length-42-bytes"}',
                    contentLength: 42,
                },
                expected: {
                    result: {
                        isComplete: true,
                        content: '{"some":"json-payload-of-length-42-bytes"}',
                    },
                    hasMoreData: false,
                },
            },
            {
                name: 'should parse content with more data',
                input: {
                    contructorParameter:
                        '{"some":"json-payload-of-length-42-bytes"}Content-Length: 42\r\n',
                    contentLength: 42,
                },
                expected: {
                    result: {
                        isComplete: true,
                        content: '{"some":"json-payload-of-length-42-bytes"}',
                    },
                    hasMoreData: true,
                },
            },
            {
                name: 'should parse incomplete content',
                input: {
                    contructorParameter: '{"some":"longer message',
                    contentLength: 42,
                },
                expected: {
                    result: {
                        isComplete: false,
                        content: '{"some":"longer message',
                    },
                    hasMoreData: false,
                },
            },
        ];

        test.each(testCases)(
            '$name',
            ({ input: { contentLength, contructorParameter }, expected }) => {
                const dataReader = new DataReader(contructorParameter);

                expect(dataReader.parseContent(contentLength)).toEqual(
                    expected.result,
                );
                expect(dataReader.hasMoreData).toEqual(expected.hasMoreData);
            },
        );
    });
});
