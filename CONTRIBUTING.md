# Contributing Guidelines

Contributions are always welcome. Please read this document before reporting a
bug or opening a pull request.

- [Setting up the project](#setting-up-the-project)
- [Reporting a bug](#reporting-a-bug)
- [Opening a pull request](#opening-a-pull-request)
- [Hall of Fame](#hall-of-fame)
- [Code of Conduct](#code-of-conduct)
- [License](#license)

## Setting up the project

See [HACKING.md](./HACKING.md).

## Reporting a bug

Think you've found a bug? Let us know!

Before opening an issue:

- Check you're on the latest version. We may already have fixed it.
- [Search the issue tracker](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues?q=is%3Aissue)
  in case someone has already reported it.

A good report includes the server version (`circleci-yaml-lsp -version`, or the
CircleCI extension's version), your editor, and the smallest config that shows
the problem.

### Security disclosure

Please don't report security issues in public. Follow the
[responsible disclosure policy](https://circleci.com/security/) instead; see
[SECURITY.md](./SECURITY.md).

## Opening a pull request

To contribute,
[fork](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/fork-a-repo)
`circleci-yaml-language-server`, commit your changes, and
[open a pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request).

- Run `task test` and `task check` before you push.
- Add tests for what you change: acceptance tests for new behavior, and the
  package's own tests for edge cases and errors.
- Use [Conventional Commits](https://www.conventionalcommits.org/) for commit
  messages and the PR title, such as `fix: ...` or `feat: ...`. Only `feat`
  and `fix` changes trigger a release.

Your pull request will be reviewed as soon as possible, and you may be asked to
make changes to it.

## Hall of Fame

Have you reported a bug that was fixed, or sent a patch that fixed one? Thank
you! Send us a pull request adding yourself to
[CONTRIBUTORS.md](./CONTRIBUTORS.md).

## Code of Conduct

All community members are expected to follow our
[code of conduct](./CODE_OF_CONDUCT.md).

## License

CircleCI's `circleci-yaml-language-server` is released under the
[Apache License 2.0](./LICENSE.md).
