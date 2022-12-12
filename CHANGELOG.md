## 0.1.10 (2022-12-08)

-   Support for anchors and aliases in syntax validation
-   Fix `machine: true` being marked as deprecated when using a self-hosted
    instance of CircleCI (only available when using the official CircleCI VSCode
    extension)
-   Fix a bug where an orb's version marked as invalid, but was not
-   Added code action (quick fix) when an orb is outdated
-   Added code action (quick fix) to delete `version: 2` inside the `workflows`
    attribute, which is deprecated, when using a version 2.1 configuration
-   Added end to end tests

## 0.1.9 (2022-12-05)

-   Support of private orbs when using self hosted instance of CircleCI (only
    available when using the official CircleCI VSCode extension)
-   Update the supported Ubuntu versions

## 0.1.8 (2022-11-28)

-   Add autocomplete for docker images
-   Introduce support of private orbs
-   Support parameters inside docker images
-   Update supported XCode versions
-   Fix a bug where an orb would appear unused when appearing only in post/pre
    steps
-   Fix `docker` keyword not being highlighted in the right color
-   Improve documentation

## 0.1.7 (2022-11-22)

-   Syntax validation
-   Syntax highlighting
-   Go-to-definition
-   Go-to-reference
-   On-hover documentation and usage hints
-   Autocompletion
