# Add a client

Implementations of Language Server clients are always welcome, if you happen to
create one, please open an issue so that we can
[reference your work](/README.md#language-server-clients)! However please read
this document before starting implementation as there are some specifities to be
aware of.

### `schema.json`

The [`schema.json`](/schema.json) used for YAML validation is **embedded in the
binary** at compile time. No external schema file is needed to run the language
server.

If you need to override the built-in schema (e.g., for development), you can
pass `-schema /path/to/schema.json` to the LS executable. The schema file is
also included in every GitHub release for reference.

### Hover

The [`textDocument/hover`](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover) functionality is not yet implemented by the LS. The VS Code extension currently works around this by implementing hover hints client-side using the
[vscode-json-languageservice](https://github.com/microsoft/vscode-json-languageservice).

### Configuration

To better handle the usage of private orbs, self-hosted runners or even contexts
you can authenticate to the Language Server with custom LS commands.

##### `setToken`

The `setToken` command takes a CircleCI API token as only argument. Users can
get API token from
[User settings](https://app.circleci.com/settings/user/tokens) in the CircleCI
app.

Example Typescript usage:

```typescript
await lsClient.sendRequest(`workspace/executeCommand`, {
  command: 'setToken',
  arguments: ['<user-token>'],
});
```

##### `setSelfHostedUrl`

For users using [CircleCI Server](https://circleci.com/pricing/server/), you can
set the URL of the self-hosted Server with this command.

Example Typescript usage:

```typescript
await lsClient.sendRequest(`workspace/executeCommand`, {
  command: 'setSelfHostedUrl',
  arguments: ['<self-hosted-url>'],
});
```
