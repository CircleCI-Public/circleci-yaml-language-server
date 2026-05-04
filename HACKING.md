# Contributing to the YAML Language Server

If you're looking to contribute to this project, there's a few things you should
know.

First, make sure you go through the [README](README.md).

Second, it's written in Go. If you are new to Go, we recommend the following
resources:

-   [A Tour of Go](https://tour.golang.org/welcome/1)
-   [The Go documentation](https://golang.org/doc/)

## Requirements

-   Go 1.23+
-   [Task](https://taskfile.dev/)
-   [detect-secrets](https://github.com/Yelp/detect-secrets)

## Getting setup

You should already have [installed Go](https://golang.org/doc/install).

You will need to install [Task](https://taskfile.dev/#/installation) in order to
run the commands in the `Taskfile.yml`. This file is used to run the commands
that are used to build, test, and lint. (Feel free to add commands to this file
if you find it useful!)

### 1. Get the repo

Clone the repo.

```
$ git clone git@github.com:CircleCI-Public/circleci-yaml-language-server.git
$ cd circleci-yaml-language-server
```

If you cloned the repo inside of your `$GOPATH`, you can use `GO111MODULE=on` in
order to use Go modules. We recommend cloning the repo outside of `$GOPATH` as
you would any other source code project, for example
`~/code/circleci-yaml-language-server`.

### 2. Install dependencies

```
$ task init
```

### 3. Build the binary

```
$ task build
```

Note: `bin/start_server` is the entry point for the language server.

### 4. Run tests

```
$ task test
```

## Managing Dependencies

We use Go 1.19 Modules for managing our dependencies.

You can read more about it on the wiki:
https://github.com/golang/go/wiki/Modules

## Linting your code

We use [`gofmt`](https://pkg.go.dev/cmd/gofmt) for linting.

In order to lint your code, you can run this command:

```
$ task lint
```

## Editor support

Go has great tooling such as [`gofmt`](https://golang.org/cmd/gofmt/) and
[`goimports`](https://godoc.org/golang.org/x/tools/cmd/goimports).

In particular, **please be sure to `gofmt` your code before committing**.

You can install `goimport` via:

```
$ go get golang.org/x/tools/cmd/goimports
```

The golang blog post
["go fmt your code"](https://blog.golang.org/go-fmt-your-code) has a lot more
info `gofmt`. To get it setup with [vim](https://github.com/fatih/vim-go) or
[emacs](https://github.com/dominikh/go-mode.el).

For example, I've the following in my `.emacs.d/init.el`:

```
(setq gofmt-command "goimports")
(require 'go-mode)
(add-hook 'before-save-hook 'gofmt-before-save)
(require 'go-rename)
```

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
  - Location: `pkg/services/diagnostics_test.go`

- **GitHub Releases**: `schema.json` is included in every release for reference
  and for tools that consume it directly.
