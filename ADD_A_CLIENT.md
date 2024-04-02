# Add a client

Implementations of Language Server clients are always welcome, if you happen to
create one, please open an issue so that we can
[reference your work](/README.md#language-server-clients)! However please read
this document before starting implementation as there are some specifities to be
aware of.

### `schema.json`

The Language Server needs the [`schema.json`](/schema.json) to validate the
YAMLs. To run the LS, you must have the file available locally and provide its
path to the `-schema` parameter of the LS.

As the `schema.json` is versioned like the rest of the code, it is provided with
every release, this means that when updating the LS binary should always come
with an update of the `schema.json`.

### Hover

The `hover` functionality is not actually implemented by the LS. Nevertheless,
you will see reference to the functionality as it is implemented directly in the
Typescript code of the VSCode extension. The functionality is provided thanks to
the
[SchemaStore JSON schema for CircleCI configs](https://github.com/SchemaStore/schemastore/blob/master/src/schemas/json/circleciconfig.json)
that CircleCI help maintain and the
[vscode-json-languageservice](https://github.com/microsoft/vscode-json-languageservice).

If you plan on implementing the feature you should look for a way to take
advantage of this JSON and find an equivalent of `vscode-json-languageservice`
that work for your editor.

### Configuration

To be able to better handle the usage of private orbs, self-hosted runners or
even contexts you can authenticate to the Language Server with custom LS
commands.

##### `setToken`

The `setToken` command takes an API token as only argument. Users can get API
token from [here](https://app.circleci.com/settings/user/tokens).

Example Typescript usage:

```typescript
await lsClient.sendRequest(`workspace/executeCommand`, {
  command: 'setToken',
  arguments: ['<user-token>'],
});
```

##### `setSelfHostedUrl`

For users using the [Server](https://circleci.com/pricing/server/) version of
CircleCI, you can set the URL of the self-hosted Server with this command.

Example Typescript usage:

```typescript
await lsClient.sendRequest(`workspace/executeCommand`, {
  command: 'setSelfHostedUrl',
  arguments: ['<self-hosted-url>'],
});
```
