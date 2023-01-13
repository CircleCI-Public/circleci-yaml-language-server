/**
 * This module parses socket messages into JSON RPC ResponsePayloads accordingly to the LSP
 * specification.
 * See "Base protocol" in LSP documentation:
 * https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#baseProtocol
 *
 * A first implementation of this parsing was made which was using the fact that every socket
 * message was an LSP message. Which is not enforced by the protocol as it handled only by the
 * socket transport.
 *
 * More concretly, the format of an LSP message is like this:
 * ```
 * Content-Length: 42
 *
 * {"some":"json-payload-of-length-42-bytes"}
 * ```
 *
 * And even if it's possible that you receive a socket with the string shown above
 *
 * You can also receive something like this:
 * Message1: 'Content-Length: 42
 * '
 * Message2: '{"some":"json-payload-of-length-42-bytes"}'
 *
 * Or even like this:
 * Message1: 'Content'
 * Message2: '-Length: 42
 *
 * {"some":"json-payload-of-length-42-bytes"}'
 *
 * As there is no rule on how the socket should send you the message
 *
 *
 * This module aims at enabling this by buffering the potential old data that was received from the
 * previous message to give you the final LSP message
 */
export { default } from './MessageParser';
