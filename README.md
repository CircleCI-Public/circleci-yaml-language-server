![CircleCI YAML Language Server](./assets/header.png)

# CircleCI YAML Language Server

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/CircleCI-Public/circleci-yaml-language-server/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/CircleCI-Public/circleci-yaml-language-server/tree/main)
[![GitHub release](https://img.shields.io/github/v/release/circleci-public/circleci-yaml-language-server)](https://github.com/circleci-public/circleci-yaml-language-server/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](./LICENSE.md)

A [Language Server Protocol](https://microsoft.github.io/language-server-protocol/)
server for CircleCI configuration files. It brings validation, completion,
hover documentation, navigation and refactoring for `.circleci/config.yml` to
any editor with an LSP client.

It powers the config helper in CircleCI's
[VS Code extension](https://marketplace.visualstudio.com/items?itemName=circleci.circleci),
and it's open source so that it can be used from any other editor, and so that
the community can see how it works and help improve it.

## <a name="features"></a>Features

<!-- Copied from circleci-vscode-extension/README.md, please keep sync! -->

This project provides in-file assistance to writing, editing and navigating
CircleCI Configuration files. It offers:

- **Rich code navigation through “go-to-definition” and “go-to-reference”
  commands**. This is especially convenient when working on large configuration
  files, to verify the definition of custom jobs, executors parameters, or in
  turn view where any of them are referenced in the file. Assisted code
  navigation also works for Orbs, allowing to explore their definition directly
  in the IDE when using the go-to-definition feature on an orb-defined command
  or parameter.

<p align="center">
    <img src="./assets/go-to-def.gif" alt="circleci-vscode-go-to-definition" width="50%"/>
</p>

- **Contextual documentation and usage hints when hovering on specific keys**,
  so to avoid you having to continuously switch to your browser to check the
  docs whenever you are editing your configuration. That said, links to the
  official CircleCI documentation are also provided on hover - for easier
  navigation.

<p align="center">
    <img src="./assets/hover-hints.png" alt="circleci-vscode-documentation-on-hover" width="50%"/>
</p>

- **Syntax validation** - which makes it much easier to identify typos,
  incorrect use of parameters, incomplete definitions, wrong types, invalid or
  deprecated machine versions, etc.

<p align="center">
    <img src="./assets/syntax-validation.gif" alt="circleci-vscode-syntax-validation" width="50%"/>
</p>

- **Usage warnings** - which can help identify deprecated parameters, unused
  jobs or executors, or missing keys that prevent you from taking advantage of
  CircleCI’s full capabilities

<p align="center">
    <img src="./assets/warnings.png" alt="circleci-vscode-usage-warnings" width="50%"/>
</p>

- **Auto completion**, available both on built-in keys and parameters and on
  user-defined variables

<p align="center">
    <img src="./assets/auto-complete.png" alt="circleci-vscode-autocomplete" width="50%"/>
</p>

- **Diagnostic suppression** - allows you to suppress specific warnings and errors using special comments:
  - `# cci-ignore` - suppress diagnostic on the same line
  - `# cci-ignore-next-line` - suppress diagnostic on the next line
  - `# cci-ignore-start` / `# cci-ignore-end` - suppress diagnostics in a range
  - `# cci-ignore-file` - suppress all diagnostics in the file

  Quick-fix code actions are available to automatically insert suppression comments.

<p align="center">
    <img src="./assets/ignore.gif" alt="circleci-ignore-comments" width="50%"/>
</p>

## <a name="contributing"></a>Contributing

Please feel free to contribute to this project, we update our Hall of fame -
[CONTRIBUTORS.md](./CONTRIBUTORS.md) with all contributors who helped us fix a
bug. You can find all the contribution guidelines and useful information in
[CONTRIBUTING.md](./CONTRIBUTING.md). Development instructions for the CircleCI
YAML Language Server can be found in [HACKING.md](HACKING.md).

## <a name="architecture"></a>Architecture diagram

![alt text](./assets/diagram.jpg)

## Language Server clients

- VSCode:
  [CircleCI VSCode extension](https://marketplace.visualstudio.com/items?itemName=circleci.circleci)
- Neovim: [tomoakley Neovim plugin](https://github.com/tomoakley/circleci.nvim)
- Emacs: [circleci-yaml-mode](./editors/emacs/)

If you wish to implement a client for your editor, please take a look at
[ADD_A_CLIENT.md](./ADD_A_CLIENT.md)

## <a name="quicklinks"></a>Quick links

- [Install the VS Code extension using our language server](https://marketplace.visualstudio.com/items?itemName=circleci.circleci)

- [Discover how the language server was leveraged for the VS Code extension](https://youtu.be/Sdi7ctAXe2A)

- [Learn more about CircleCI](https://circleci.com/)

- [Do you have questions? Ask the circleCI community Discuss](https://discuss.circleci.com/)
