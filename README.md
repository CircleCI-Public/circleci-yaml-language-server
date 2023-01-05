# CircleCI YAML Language Server

This is CircleCI's YAML Language Server.

[Code of Conduct](./CODE_OF_CONDUCT.md) |
[Contribution Guidelines](./CONTRIBUTING.md) | [Hacking](./HACKING.md)

[![CircleCI](https://circleci.com/gh/CircleCI-Public/circleci-yaml-language-server/tree/master.svg?style=svg)](https://app.circleci.com/pipelines/github/CircleCI-Public/circleci-yaml-language-server)
[![GitHub release](https://img.shields.io/github/v/release/circleci-public/circleci-yaml-language-server)](https://github.com/circleci-public/circleci-yaml-language-server/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](./LICENSE.md)

## Features

<!-- Copied from circleci-vscode-extension/README.md, please keep sync! -->

This project provides in-file assistance to writing, editing and navigating
CircleCI Configuration files. It offers:

-   **Rich code navigation through “go-to-definition” and “go-to-reference”
    commands**. This is especially convenient when working on large
    configuration files, to verify the definition of custom jobs, executors
    parameters, or in turn view where any of them are referenced in the file.
    Assisted code navigation also works for Orbs, allowing to explore their
    definition directly in the IDE when using the go-to-definition feature on an
    orb-defined command or parameter.

<div style="text-align:center">
    <img src="https://images.ctfassets.net/il1yandlcjgk/3JVSu8rTDQRJcIMGF866oJ/444091930e6c64dc9d52f17755e16af9/config_helper_go-to-definition-optimised.gif" alt="circleci-vscode-go-to-definition" width="380"/>
</div>

-   **Contextual documentation and usage hints when hovering on specific keys**,
    so to avoid you having to continuously switch to your browser to check the
    docs whenever you are editing your configuration. That said, links to the
    official CircleCI documentation are also provided on hover - for easier
    navigation.

<div style="text-align:center">
    <img src="https://images.ctfassets.net/il1yandlcjgk/6bloAnI35jXou9Q91aGFgU/356b1554d42c77fb2708fac980a8d592/config_helper_on-hover-documentation.png" alt="circleci-vscode-documentation-on-hover" width="380"/>
</div>

-   **Syntax validation** - which makes it much easier to identify typos,
    incorrect use of parameters, incomplete definitions, wrong types, invalid or
    deprecated machine versions, etc.

<div style="text-align:center">
    <img src="https://images.ctfassets.net/il1yandlcjgk/1dF1ic2cUczaMYdxnSUZuF/fbc1eb5d5894a803caf297c57e808738/config_helper_syntax-validation.gif" alt="circleci-vscode-syntax-validation" width="380"/>
</div>

-   **Usage warnings** - which can help identify deprecated parameters, unused
    jobs or executors, or missing keys that prevent you from taking advantage of
    CircleCI’s full capabilities

<div style="text-align:center">
    <img src="https://images.ctfassets.net/il1yandlcjgk/404bGppmtCgvE0WOmPk0E/0a92e373929d6d19c8f5a742f1097511/config_helper_usage-warning.png" alt="circleci-vscode-usage-warnings" width="380"/>
</div>

-   **Auto completion**, available both on built-in keys and parameters and on
    user-defined variables

<div style="text-align:center">
    <img src="https://images.ctfassets.net/il1yandlcjgk/3jXaQvhOQgayhV9O4nfAhZ/b6d55e689ddfcf7673ab7e9b76ba0a53/config_helper_autocomplete.png" alt="circleci-vscode-autocomplete" width="380"/>
</div>

## Platforms, Deployment and Package Managers

The tool is deployed through
[GitHub Releases](https://github.com/CircleCI-Public/circleci-yaml-language-server/releases).
Green builds on the `master` branch will publish a new GitHub release. These
releases contain binaries for macOS, Linux and Windows.

This is a project in active development, and we target a release frequency of one release per week on average. However, we reserve the right of releasing more or less frequently when necessary.

## Contributing

Development instructions for the CircleCI YAML Language Server can be found in
[HACKING.md](HACKING.md).

## Architecture diagram

![alt text](./assets/diagram.jpg)
