## 0.1.14 (2023-01-17)

-   Support for YAML object merging (merge keys)
-   Support for YAML alias and anchors inside step definitions
-   Support for parameter values written as multi-lines
-   Replaced unintelligible diagnostic "Must validate one and only one schema ..." with more meaningful diagnostic
-   Autocompletion for orb names
-   Improved testing
-   Fix and refactor Orbs caching
-   Fixed parameter default values not recognizing orb executors

## [0.2.0](https://github.com/CircleCI-Public/circleci-yaml-language-server/compare/0.1.15...0.2.0) (2023-02-20)


### Features

* Added autocomplete of built-in CCI env variables ([#98](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/98)) ([6bd5479](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/6bd5479468c1352fde43e3d22d917f36e46e9782))
* Added autocomplete of env_var_name parameters and `environment` field ([5018ef5](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/5018ef554c289effa9ba9b81eeb1ab1ffbc30b40))
* Added autocompletion for context env variables ([d26b851](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/d26b8513f690e621b32e33b6f2f4ff1b95c549cc))
* Added autocompletion for project env variables ([b8bfffd](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/b8bfffd3e6ef6f8b726bdd10178968d18fcb716a))
* Check and validate namespace of executor ([#96](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/96)) ([f0915bd](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/f0915bdf8bc10510e9d1af5bacc5a6bae36541c3))


### Bug Fixes

* Default values not recognized in job parameters of type `steps` ([f319f8a](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/f319f8a50dffca1f1bfe7456aff5502355a124d1))
* Default values not recognized in job parameters of type `steps` ([1fdeaa8](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/1fdeaa8cad7590edd3e1517f0b9e3a1bad12730c))
* Fix invalid executor as being wrong orb's executor ([eca0aa9](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/eca0aa97ce9ce325c7cd2241789eab940742b26f))
* Fix slack message ([77a6f97](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/77a6f97c3c4b681bb8a38f139b9d64f7fff83d65))
* Fix slack message ([b336867](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/b336867ee272a637cbd26c784a6c6bdd8167a7da))
* fixed incorrect suggestion "A new major version exists" when using a non-semver orb version ([ba8d1ed](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/ba8d1eda8ce6b3125bcbf37d152cc5464e0a2579))
* fixed invalid syntax validation for parameter entered in-line ([bfe3443](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/bfe3443e93e391dfd18bb3e2eb4d4a69f5a42a8c))
* Fixed various problems involved with anchors/aliases ([17f439b](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/17f439b555605e66bb0da9f74f88c795ab7426e8))
* No more false errors on self hosted runners ([4cb1b2f](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/4cb1b2f01dc7a73a09a8c4d1e6e243b8696de8fd))
* raise an error if an undeclared parameter is passed to a job/command ([16029fb](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/16029fb48e1a8c0dd5d965d8809fc524da2ca156))
* string values for step type parameters must be existing command name ([#81](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/81)) ([4b27ad5](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/4b27ad5cbe16bf44c103e407808ec7174ea51db1))
* When hovering on orb's method it doesn't shows issue in the problems tab ([#97](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/97)) ([0b21d32](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/0b21d323affccbef9c9f43460a9df1e53929f61a))

## 0.1.13 (2022-12-08)

-   Added newly supported xcode versions to validation
-   Fix: Job executors can contain anchors
-   Fix: Empty executors, pipeline parameters, orbs and commands are no longer
    marked as invalid
-   Fix: Crash when parameter value is not of the expected type
-   Fix: Unknown parameters 

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
