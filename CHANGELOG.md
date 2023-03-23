## 0.1.14 (2023-01-17)

-   Support for YAML object merging (merge keys)
-   Support for YAML alias and anchors inside step definitions
-   Support for parameter values written as multi-lines
-   Replaced unintelligible diagnostic "Must validate one and only one schema ..." with more meaningful diagnostic
-   Autocompletion for orb names
-   Improved testing
-   Fix and refactor Orbs caching
-   Fixed parameter default values not recognizing orb executors

## [0.3.1](https://github.com/CircleCI-Public/circleci-yaml-language-server/compare/0.3.0...0.3.1) (2023-03-23)


### Bug Fixes

* Fix error on spawn ([96a303f](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/96a303f9c4509d85982ff3eff3e3cc4cce34b709))

## [0.3.0](https://github.com/CircleCI-Public/circleci-yaml-language-server/compare/0.2.0...0.3.0) (2023-03-23)


### Features

* Added getWorkflows command ([14957bd](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/14957bd130467b5c80c49b50cc6ba5d19331067d))
* code outline & breadcrumbs ([#112](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/112)) ([e8390c6](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/e8390c65f09e525249c18ae4aeb2f1af32cd6f12))


### Bug Fixes

* added machine bool field to ast for outline fix ([de1dbf2](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/de1dbf23591fea69459c64c2d5e015601bd5114d))
* updated valid xcode versions ([cf1bd6d](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/cf1bd6d55e9f32d209c06ddfb435af3f0f08a5d3))
* **definition:** Fix orb parameter definition ([faa58b1](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/faa58b14174468991aca7b7c35af20f7d5578a4e))

## [0.2.0](https://github.com/CircleCI-Public/circleci-yaml-language-server/compare/0.1.15...0.2.0) (2023-03-09)


### Features

* Added a detailed parsing of an orb string ([7a8168c](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/7a8168cbf8ee0dea632eda59d568912906e034f0))
* Added autocomplete of built-in CCI env variables ([#98](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/98)) ([6bd5479](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/6bd5479468c1352fde43e3d22d917f36e46e9782))
* Added autocomplete of env_var_name parameters and `environment` field ([5018ef5](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/5018ef554c289effa9ba9b81eeb1ab1ffbc30b40))
* Added autocompletion for context env variables ([d26b851](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/d26b8513f690e621b32e33b6f2f4ff1b95c549cc))
* Added autocompletion for project env variables ([b8bfffd](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/b8bfffd3e6ef6f8b726bdd10178968d18fcb716a))
* added new ubuntu machine ([#116](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/116)) ([ce25d17](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/ce25d17d4cb486a3fe7027a347a45ade97c56338))
* Autocomplete orb versions ([5ba2a60](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/5ba2a60311e45e460fdc003b8e2a69613ec5b24e))
* Check and validate namespace of executor ([#96](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/96)) ([f0915bd](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/f0915bdf8bc10510e9d1af5bacc5a6bae36541c3))
* telemetry for autocompletion ([#110](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/110)) ([8756e86](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/8756e86ef58b1f3f6bb2ffd5ff68883753952480))
* Validate `context` variable inside job reference ([b471bab](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/b471babe64dbdf5aa25929dd581b4680eb88cedb))


### Bug Fixes

* Default values not recognized in job parameters of type `steps` ([f319f8a](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/f319f8a50dffca1f1bfe7456aff5502355a124d1))
* Default values not recognized in job parameters of type `steps` ([1fdeaa8](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/1fdeaa8cad7590edd3e1517f0b9e3a1bad12730c))
* different error messages for non-existing orbs vs wrong version orbs ([#106](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/106)) ([db534e9](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/db534e95ba5ec89a4360da17c7bcba3f1db77738))
* Fix executor name not being correctly parsed with short hand syntax ([b8ca5b7](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/b8ca5b7a9ba1707eb1546a99e09d4ca44acff4a1))
* Fix invalid executor as being wrong orb's executor ([eca0aa9](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/eca0aa97ce9ce325c7cd2241789eab940742b26f))
* Fix slack message ([77a6f97](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/77a6f97c3c4b681bb8a38f139b9d64f7fff83d65))
* Fix slack message ([b336867](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/b336867ee272a637cbd26c784a6c6bdd8167a7da))
* fixed incorrect suggestion "A new major version exists" when using a non-semver orb version ([ba8d1ed](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/ba8d1eda8ce6b3125bcbf37d152cc5464e0a2579))
* fixed invalid syntax validation for parameter entered in-line ([bfe3443](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/bfe3443e93e391dfd18bb3e2eb4d4a69f5a42a8c))
* Fixed various problems involved with anchors/aliases ([17f439b](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/17f439b555605e66bb0da9f74f88c795ab7426e8))
* Improve error message when dealing with an unknown executor reference ([a2a6f20](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/a2a6f209007609a8e2981523d9c01e3013340765))
* min prop warning when using multiple merge keys ([#107](https://github.com/CircleCI-Public/circleci-yaml-language-server/issues/107)) ([225ddf4](https://github.com/CircleCI-Public/circleci-yaml-language-server/commit/225ddf4dd05572dfa9e4703eceedb7e0ab4b6565))
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
