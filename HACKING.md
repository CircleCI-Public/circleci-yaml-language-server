# Hacking on the CircleCI YAML Language Server

How to build, test and debug the language server. For how to report issues and
open pull requests, see [CONTRIBUTING.md](./CONTRIBUTING.md).

The server is written in Go. If you're new to Go, [A Tour of
Go](https://go.dev/tour/) and [the Go documentation](https://go.dev/doc/) are
good places to start.

## Requirements

-   [Go](https://go.dev/doc/install), at the version in `go.mod`
-   [Task](https://taskfile.dev/installation/), which runs the commands in
    `Taskfile.yml`
-   [Node.js](https://nodejs.org/), with npm, only to
    [test within VS Code](#testing-within-vscode)

## Getting set up

### 1. Get the repo

```
$ git clone git@github.com:CircleCI-Public/circleci-yaml-language-server.git
$ cd circleci-yaml-language-server
```

### 2. Build the binary

```
$ task build
```

Note: `bin/start_server` is the entry point for the language server.

To build every release target the way CI does, run `task snapshot`. It
cross-compiles with goreleaser, and leaves the release under its published names in
`dist/release`.

### 3. Run tests

```
$ task test
```

## Benchmarks

`bench/` times the compiled server the way an editor experiences it, in the
style of gopls's benchmarks: the real binary, started as a subprocess and
driven over LSP, against generated configs of two sizes — `small` (5 jobs) and
`large` (200 jobs, 5,000 steps) — and a fake CircleCI.

| Benchmark | Times |
| --- | --- |
| `Startup` | starting a server and the initialize handshake |
| `Open` | opening a config in a fresh server, to its first diagnostics |
| `Change` | an edit, to the diagnostics for it, with the edit debounce off |
| `Hover` | hovering over a step that runs one of the config's commands |
| `Completion` | completing a job name in a workflow |

```bash
$ task bench                       # every benchmark, 6 runs each
$ task bench -- -bench Hover       # just one
$ task bench COUNT=10              # more runs, for a tighter comparison
```

`ns/op` is the number to read. Allocation figures would be the client's, not
the server's, so they are not reported.

To see what a change does, run the benchmarks on both sides of it and compare
them with `benchstat`, which reports the difference and whether it is
significant:

```bash
$ git switch main && task bench && cp test-reports/bench.txt /tmp/old.txt
$ git switch my-branch && task bench
$ go tool -modfile tools/go.mod benchstat /tmp/old.txt test-reports/bench.txt
```

Timings depend on the machine and on what else it is doing, so only compare
runs made on the same machine, one after the other. Nothing in CI runs the
benchmarks yet.

## Contract probes

The tests run against fakes of the CircleCI and Docker Hub APIs
(`internal/testing/fakes`). Fakes drift: the service changes, or our reading of
it was wrong to begin with, and the tests keep passing either way. The probes
in `cmd/dev/orb` and `cmd/dev/dockerhub` are how that drift is caught — they make the
same calls against the real APIs and check that what comes back still has the
shape the fakes reproduce.

```bash
$ task probe                        # both
$ task probe:orb                    # the CircleCI orb registry
$ task probe:orb -- circleci/go@1.7 # ... resolving a reference of your choice
$ task probe:dockerhub              # Docker Hub
```

They print what they found, which also makes them the quickest way to see what
the language server sees for a given orb or image.

Each probe exits with one of three codes:

| Code | Meaning |
| --- | --- |
| 0 | the API looks as expected |
| 1 | drift: something the fakes rely on is no longer true, and someone has to look |
| 2 | the probe could not find out — unreachable, unauthorized, or too slow |

The distinction between 1 and 2 is the point: on a schedule, drift is worth
raising and an outage is not.

Neither probe needs a token; both read public data. `cmd/dev/orb` uses
`CIRCLE_TOKEN` when it is set, and takes `CIRCLECI_HOST` — see the comment at
the top of `cmd/dev/orb`.

Probes talk to the internet, so they should never run on a pull request: a
service having a bad day is not a reason to block a merge. Nothing in CI runs
them today — run them by hand when a fake is being written or doubted. A
schedule is where they belong eventually, which is what the exit codes above
are for, and Docker Hub's rate limit on anonymous callers is a reason to keep
any such schedule infrequent.

## Config survey

`cmd/dev/survey` runs the diagnostics over the CircleCI configs of whole GitHub
organizations, `circleci` and `CircleCI-Public` by default. It's how to
measure what a change does to real configs: which errors it removes, and
whether it adds any.

```bash
$ GH_TOKEN=$(gh auth token) task survey -- fetch -since 2024-09-01   # into bin/survey
$ GH_TOKEN=$(gh auth token) task survey -- diagnose > before.jsonl
$ # ... make the change ...
$ GH_TOKEN=$(gh auth token) task survey -- diagnose > after.jsonl
$ task survey -- compare before.jsonl after.jsonl
```

The configs come from private repositories too, so they're kept under `bin/`,
which git ignores, and must not be committed. When a false error is fixed, a
minimal config that reproduces it belongs in the tests instead.

## Managing dependencies

Dependencies are Go modules. Development tools, such as goreleaser and
benchstat, are pinned separately in `tools/go.mod` and run with
`go tool -modfile tools/go.mod`.

When a dependency is added, removed or changes license, regenerate
[CREDITS.md](./CREDITS.md) as described at the bottom of that file.

## Linting your code

We use [`golangci-lint`](https://golangci-lint.run) for formatting and linting.

```
$ task check   # every static check CI runs: formatting, lint, and go mod tidy
$ task fix     # apply what can be fixed automatically
$ task lint    # just the linters
$ task fmt     # just the formatters
```

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/),
which `task commitlint` checks. The type decides whether a change is released;
see [Releasing](#releasing).

## Testing within VSCode

This repository embeds a mini VSCode extension (located at `editors/vscode`) so you
can test your changes to the language server using VS Code locally.

1. In order to run the extension, you must first prepare installation. This
   command will install the necessary node packages and build the extension:

```bash
task prepare:vscode
```

2. You need to disable the CircleCI marketplace extension before testing in
   order to avoid conflicts between the two extensions (the local one and the
   marketplace one). To do so, please go to the `Extensions` tab, click on the
   CircleCI extension and click on `Disable`

3. Next, open a VSCode instance at the root of the project, open the
   `Run and Debug` tab and run it via the `Run Extension` on the dropdown menu
   at the top of the tab.

   > [!NOTE]
   > Do not do `Run Extension (user extensions enabled)`. Running with no other extensions enabled
   > could cause confusion. Specifically, if you had the Red Hat YAML Language Server extension
   > installed, it would display hover hints from the JSON schema from schemastore, rather than the local `schema.json`
   > that you will often be making changes to locally and want to test.

### Authentication / API token

Some language server features require a CircleCI API token. For example,
resolving private orbs, fetching contexts, and validating project-specific
settings. The embedded VS Code extension reads the token from the environment
and sends it to the running language server via the `setToken` LSP command
immediately after the server initialises.

Set `CIRCLE_TOKEN` before launching VS Code (the `TOKEN` environment variable
is also accepted as a fallback):

```bash
export CIRCLE_TOKEN=<your-token>
code .
```

If neither variable is set the extension starts without a token. The server
still works, but features that call the CircleCI API will be unavailable.

> [!NOTE]
> This is how the **development** extension passes the token. Production
> clients (e.g. the CircleCI VS Code marketplace extension) handle
> authentication through their own UI flows and send the same `setToken`
> command once the user has logged in. See `ADD_A_CLIENT.md` for details on
> implementing this in other editors.

## Understanding the `schema.json` file

The CircleCI YAML Language Server uses the standardized [JSON schema](https://json-schema.org/) to help perform basic structural validations on CircleCI YAML files. Our schema lives in `schema.json` at the repository root.

### Embedded schema

The schema is **embedded into the binary** at compile time using Go's `go:embed`
directive (see `schema_embed.go`). This means the binary works standalone — no
external files needed. You can override the built-in schema with the `-schema`
flag or the `SCHEMA_LOCATION` environment variable for development purposes.

### Use Cases for `schema.json`

- **Language Server Binary**: The Go binary embeds `schema.json` and uses it for
  JSON schema validation of CircleCI configs. An override via `-schema` or
  `SCHEMA_LOCATION` is supported but optional.

- **External Tools**: The `schema.json` is used by the Red Hat YAML extension. Most VS Code users who open YAML files will have this extension installed. This extension by default will pull in schemas from [schemastore.org](https://www.schemastore.org/api/json/catalog.json). It can detect if a user is reading a CircleCI config and automatically pull in our `schema.json` from schemastore, which in turn pulls the latest version from the main branch of this repository.
  - The Red Hat YAML language server gets the schema from: `https://raw.githubusercontent.com/CircleCI-Public/circleci-yaml-language-server/refs/heads/main/schema.json`

    > [!NOTE]
    > A user without the CircleCI VS Code extension installed, and just the Red Hat YAML language server
    > installed, still benefits from this `schema.json`. The Red Hat YAML Language server on its own will provide:
    >
    > 1. schema validation
    > 2. a hover provider in VS Code for documentation hints
    >
    > The benefit of installing the CircleCI Extension in VS Code is that it also pulls in the Go binary, providing more
    > complex validations against a user's CircleCI config that aren't possible with JSON Schema alone.

- **CircleCI VS Code Extension**: The closed-source VS Code extension implements
  hover hints client-side by reading `schema.json` from disk

- **Go Tests**: Tests use the embedded schema by default. The file-based schema
  can still be loaded explicitly for comparison tests.
  - Location: `internal/services`

- **GitHub Releases**: `schema.json` is included in every release for reference
  and for tools that consume it directly.

## Releasing

Releases are cut by [release-please](https://github.com/googleapis/release-please)
and published to
[GitHub Releases](https://github.com/CircleCI-Public/circleci-yaml-language-server/releases)
with binaries for macOS, Linux and Windows.

- Green builds on `main` open or update a release PR. It has to be approved
  and merged by hand.
- Only `feat` and `fix` commits trigger a release. A PR made only of `chore`,
  `docs`, `refactor` and so on is not released until something else is.
- The release is published by the pipeline on the release PR's merge commit,
  and only that one. If that pipeline doesn't run, for example because whoever
  merged the PR can't use the release workflow's restricted contexts, every
  later release job on `main` fails and names the commit. Someone who can use
  those contexts then needs to rerun that commit's `Build and release`
  workflow from failed.
