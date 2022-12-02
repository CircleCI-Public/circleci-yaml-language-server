# Contributing to the YAML Language Server

If you're looking to contribute to this project, there's a few things you should
know.

First, make sure you go through the [README](README.md).

Second, it's written in Go. If you are new to Go, we recommend the following
resources:

-   [A Tour of Go](https://tour.golang.org/welcome/1)
-   [The Go documentation](https://golang.org/doc/)

## Requirements

-   Go 1.19+
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

This repository embed a VSCode extension (located at `editors/vscode`) so you
can test your code within the editor.

1. In order to run the extension, you must first prepare installation. This
   command will install the necessary node packages and build the extension:

```
task prepare:vscode
```

2. You need to disable the CircleCI marketplace extension before testing in
   order to avoid conflicts between the two extensions (the local one and the
   marketplace one). To do so, please go to the `Extensions` tab, click on the
   CircleCI extension and click on `Disable`

3. Next, open a VSCode instance at the root of the project, open the
   `Run and Debug` tab and run it via the `Run Extension` on the dropdown menu
   at the top of the tab.

## Running End-2-End tests

End-to-end tests are tests performed on a running LSP server.
Tests are written in typescript (using Jest) and located in the `e2e` folder.

### Prepare your environment
You can run E2E tests, you will need NodeJS (16+) installed in your environment.

Prepare the tests with the command:

```
task prepare:test:e2e
```

This command will install all NodeJS dependencies needed for the tests (see `e2e/package.json`).

### Run tests


(Re)-build the server binary using the command:
```
task build
```

You can now run the test at any time using the command:

```
task test:e2e
```

This will:
* start a LSP server on port 10001 (update `PORT` env variable to change this)
* run all tests in `e2e/src` folders
* close the LSP server

If you want your tests to reach a already running server, use the following command:
```
task test:e2e:standalone
```

You may have to set the `PORT`.

### Update snapshots

To update snapshots, run:

```
task test:e2e:update
```

Snapshots are located at `e2e/src/snapshots`.

### Related environment variables
* `SPAWN_LSP_SERVER`: (default: `yes`) If truthy, then the LSP server will be spawn for tests and stopped at the end. Accepted value: `true`, `false`, `on`, `off`, `yes`, `no`, `1`, `0`.
* `PORT` (default: 10001) Port where to reach (and spawn if requested) the LSP server
* `LSP_SERVER_HOST`: Default: `localhost`. Host address of the LSP server
* `RPC_SERVER_BIN`: Default: `bin/start_server`. Name of the binary to use if the LSP server spawn has been requested.
