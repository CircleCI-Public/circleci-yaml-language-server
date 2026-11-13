# Add a client

Language Server clients are always welcome. If you write one, please open an
issue so that we can [list it](/README.md#editor-setup)! Please read this
document before you start, as there are some specifics to be aware of.

## Running the server

Start the binary with `-stdio` and speak LSP over its stdin and stdout. Logs go
to stderr. See the [README](/README.md#installation) for where to get the
binary and the other flags.

## Which files the server serves

The server only answers for files in a `.circleci` directory, such as
`.circleci/config.yml` or a config under `.circleci/continue/`, and for the orb
sources it writes out itself for go-to-definition. Any other document is left
alone: it isn't checked, nothing is published for it, and every request about
it is answered with `null`.

So a client that can only pick a server by file extension, as Claude Code and
other coding agents do, can send it every `.yml` and `.yaml` file. GitHub
workflows, Compose files and Kubernetes manifests won't get false errors.

## `schema.json`

The [`schema.json`](/schema.json) used for YAML validation is **embedded in the
binary** at compile time. No external schema file is needed to run the language
server.

If you need to override the built-in schema (e.g., for development), you can
pass `-schema /path/to/schema.json` to the LS executable. The schema file is
also included in every GitHub release for reference.

## Hover

The server answers
[`textDocument/hover`](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover)
with Markdown:

- On a name, what it refers to: a step's command, a job, an executor, a
  function, an orb, a parameter or a pipeline value.
- On a key, such as `resource_class` or `run`, the built-in schema's
  description of it. A name the config chooses, such as a job's, isn't
  described this way.

A client that already shows the schema's descriptions itself, from
`schema.json`, can turn the key hovers off with the `schemaHovers`
initialization option, so that each isn't shown twice.

## Initialization options

The server reads these from the `initialize` request's
`initializationOptions`:

| Option                | Type    | Meaning                                                                                                                                                                                                                                                                                                                                       |
| --------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `schemaHovers`        | boolean | Whether hovering a key shows the schema's description of it. On by default, except with `isCciExtension`.                                                                                                                                                                                                                                     |
| `isCciExtension`      | boolean | Set by CircleCI's VS Code extension: an orb that can't be found tells the user to sign in there. Without `schemaHovers`, it also turns the key hovers off, as versions of the extension that don't send it show them itself.                                                                                                                  |
| `userAgent`           | string  | Appended to the server's user agent in its requests to CircleCI.                                                                                                                                                                                                                                                                              |
| `gitHubSignInCommand` | string  | A command of the client's that signs the user in to GitHub, and then sends the server a token with `setGitHubToken` or, with `fetchesGitHubFiles`, runs `gitHubAccountChanged`. An orb on GitHub that can't be fetched without a token then tells the user to sign in, with a quick fix that runs the command, rather than to set `GH_TOKEN`. |
| `fetchesGitHubFiles`  | boolean | Set by a client that can fetch files from GitHub as the user, but can't give the server a token: the server asks it for a private orb on GitHub with [`circleci/fetchGitHubFile`](#circlecifetchgithubfile). Off by default.                                                                                                                  |
| `editDebounceMs`      | number  | How many milliseconds after a change the document is checked, 1000 by default, so that a burst of keystrokes is checked once. A client whose changes each carry a whole edit, as an agent's do, can set it to 0 to have each checked at once.                                                                                                 |

Example Typescript usage:

```typescript
const clientOptions: LanguageClientOptions = {
  initializationOptions: { schemaHovers: false },
};
```

## Configuration

To better handle the usage of private orbs, self-hosted runners or even contexts
you can authenticate to the Language Server with custom LS commands.

### `setToken`

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

### `setSelfHostedUrl`

For users using [CircleCI Server](https://circleci.com/pricing/server/), you can
set the URL of the self-hosted Server with this command.

Example Typescript usage:

```typescript
await lsClient.sendRequest(`workspace/executeCommand`, {
  command: 'setSelfHostedUrl',
  arguments: ['<self-hosted-url>'],
});
```

### `setGitHubToken`

An orb can be referenced by the URL of its source, such as
`https://raw.githubusercontent.com/acme/orbs/main/go.yml`. The language server
fetches it, and when the file is on GitHub and can't be fetched without
credentials, as in a private repository, it tries again with this GitHub token.
The token is only ever sent to `raw.githubusercontent.com` and `github.com`.

Without this command, the language server uses `GH_TOKEN` or `GITHUB_TOKEN`
from its environment, if either is set.

Example Typescript usage:

```typescript
await lsClient.sendRequest(`workspace/executeCommand`, {
  command: 'setGitHubToken',
  arguments: ['<github-token>'],
});
```

### `circleci/fetchGitHubFile`

A client that can fetch files from GitHub as the user, but can't hand the
server a token, as the JetBrains plugin can't since IntelliJ 2026.3, can fetch
private orbs for the server instead. It sets the `fetchesGitHubFiles`
initialization option, and answers this request, which the server sends it for
an orb URL that:

1. is `https`, on `raw.githubusercontent.com` or `github.com`; no other URL is
   ever sent to the client,
2. couldn't be fetched without credentials, and
3. there is no GitHub token for, from `setGitHubToken`, `GH_TOKEN` or
   `GITHUB_TOKEN`. A token, when there is one, is used rather than the client.

```typescript
// server -> client
interface FetchGitHubFileParams {
  url: string;
}

// client -> server
interface FetchGitHubFileResult {
  found: boolean;
  source?: string; // the file's text; required when found is true
}
```

Answer `{ found: false }` when no GitHub account is signed in, or the account
can't see the file. The orb is then reported as one that couldn't be fetched,
with the `gitHubSignInCommand` quick fix if there is one, and asked for again
after a while, or at once after `gitHubAccountChanged`. Answer with an error when
the fetch failed for another reason; the server will ask again next time.
An answer that is neither, or none within 30 seconds, counts as an error.

```typescript
lsClient.onRequest('circleci/fetchGitHubFile', async ({ url }) => {
  const source = await fetchAsSignedInAccount(url); // undefined when not found
  return source === undefined ? { found: false } : { found: true, source };
});
```

### `gitHubAccountChanged`

Tells a server the client fetches GitHub files for that the GitHub account has
changed: signed in, signed out or switched. The server forgets the orbs it has
fetched, and found missing, and checks the open documents again. It takes no
arguments.

```typescript
await lsClient.sendRequest(`workspace/executeCommand`, {
  command: 'gitHubAccountChanged',
  arguments: [],
});
```
