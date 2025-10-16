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
   > installed, it would display hover hints from the JSON schema from schemastore, rather than the local schema.json
   > that you will often be making changes to locally and want to test.

## Understanding the `schema.json` file

The CircleCI YAML Language Server uses the standardized [JSON schema](https://json-schema.org/) to help perform basic structural validations and documentation hover hints the CircleCI YAML files. Our schema lives in `schema.json` and is utilized in multiple different places both inside and outside the codebase.

### Use Cases for `schema.json`

- **External Tools**: This `schema.json` is notably used by the Red Hat YAML extension. Most VSCode users who open YAML files will have this extension installed. This extension by default will pull in schemas from [schemastore.org](https://www.schemastore.org/api/json/catalog.json). This extension on its own can detect if a user is reading a CircleCI config, then it will automatically pull in our `schema.json` by looking at schemastore, which in turn pulls the latest version of `schema.json` from the main branch of this repository.
  - The Red Hat YAML language gets the schema.json from this URL: `https://raw.githubusercontent.com/CircleCI-Public/circleci-yaml-language-server/refs/heads/main/schema.json`

    > [!NOTE]
    > A user without the CircleCI VS Code extension installed, and just the Red Hat YAML language server
    > installed, still benefits from this `schema.json`. The Red Hat YAML Language server on its own will provide:
    >
    > 1. schema validation
    > 2. a hover provider in VS Code for documentation hints
    >
    > The benefit of installing the CircleCI Extension in VSCode is that it also pulls in the Go binary, providing more
    > complex validations against a user's CircleCI config that aren't possible with JSON Schema alone.

- **CircleCI Go Language Server Binary**: The main language server reads this schema via the `SCHEMA_LOCATION` environment variable. The Go language server binary uses a JSON schema validation library and validates the config against the schema.
  - As mentioned above, JSON Schema validation is also handled the Red Hat YAML language server. Our language server is intended to work standalone (but still be compatible with other language servers), so we also perform JSON schema validation in case the user only has our language server installed.
  - Location: `pkg/services/diagnostics.go`, `pkg/services/validate.go`, etc.

- **CircleCI VSCode Extension**: CircleCI's closed-source VS Code extension will automatically pull in the latest version of the CircleCI language server from the GitHub releases page. The VS Code extension has some logic such that:
  1. if it detects that the Red Hat YAML Language Server extension is not installed, it will register a hover provider
     so that when the user hovers over the YAML code, it will provide hover hints from the `schema.json` it downloaded
     from the CircleCI language server's releases page
  2. if it detects the Red Hat YAML Language Server extension, it will defer the hover hints to the Red Hat
     YAML Language Server, otherwise the user would see two instances of the hover hints.

- **Go Tests**: Used for validation testing
  - Location: `pkg/services/diagnostics_test.go`
