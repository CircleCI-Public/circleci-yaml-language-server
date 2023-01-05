# Contributing Guidelines

Contributions are always welcome; however, please read this document in its
entirety before submitting a Pull Request or Reporting a bug.

### Table of Contents

-   [Reporting a bug](#reporting-a-bug)
    -   [Security disclosure](#security-disclosure)
-   [Creating an issue](#creating-an-issue)
-   [Opening a pull request](#opening-a-pull-request)
-   [Hall of Fame](#hall-of-fame)
-   [Code of Conduct](#code-of-conduct)
-   [License](#license)

---

# Setting up the project

Please read [HACKING.md](./HACKING.md)

# Reporting a Bug

Think you've found a bug? Let us know!

### Security disclosure

Security is a top priority for us. If you have encountered a security issue
please responsibly disclose it by following our
[security disclosure](https://circleci.com/docs/2.0/security/) document.

# Creating an Issue

Your issue must follow these guidelines for it to be considered:

#### Before submitting

-   Check you’re on the latest version, we may have already fixed your bug!
-   [Search our issue tracker](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/search&type=issues)
    for your problem, someone may have already reported it

# Opening a Pull Request

To contribute, [fork](https://docs.github.com/en/get-started/quickstart/fork-a-repo)
`circleci-yaml-language-server`, commit your changes, and
[open a pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests).

Your request will be reviewed as soon as possible. You may be asked to make
changes to your submission during the review process.

#### Before submitting

Test your change thoroughly, to do this you can use the VSCode extension in
`editors/vscode` in order to only test the LSP in an extension that would only
run the LSP and nothing else.

To do so, open a VSCode instance at the root of the project, open the
`Run and Debug` tab and run it via the `Run Extension` on the dropdown menu at
the top of the tab.

# Hall of Fame

Have you reported a bug that was fixed or even sent a patch that fixed one?

First of all, you rock! Thank you so much for your help!

Please send us a pull request and add yourself to the
[CONTRIBUTORS.md](./CONTRIBUTORS.md) hall of fame.

# Code of Conduct

All community members are expected to adhere to our
[code of conduct](./CODE_OF_CONDUCT.md).

# License

CircleCI's `circleci-yaml-language-server` is released under the
[Apache License 2.0](./LICENSE.md).
