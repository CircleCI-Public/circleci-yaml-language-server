# corpus

Checks the parser, and later the compiler, against a corpus of configs:

- the YAML under this repo's `testdata` directories
- configs `survey fetch` wrote to `bin/survey/`, if any
- every `build.yml` in the reference corpus, the reference compiler's test
  cases, when `REFERENCE_CORPUS_DIR` (or `-corpus`) names a local copy

```sh
REFERENCE_CORPUS_DIR=<path-to-reference-corpus> go run ./cmd/dev/corpus parse > run.jsonl
go run ./cmd/dev/survey compare before.jsonl run.jsonl
```

Each JSON line is one problem, in the shape `survey diagnose` writes (`file`,
`line`, `severity`, `message`, `text`) plus `check`, so `survey compare` diffs
two runs. A summary of counts goes to stderr.

## Fixtures stay out of this repo

This repo is public. The reference corpus is private and unlicensed, so no
file from it is ever copied here, not even as a test fixture. The harness
reads it in place, and `TestParse` skips the reference corpus when
`REFERENCE_CORPUS_DIR` is unset, so CI never needs it. This repo's own
`testdata` is always checked.

## parse

For each config, `ParseFromContent` and `ParseForCompile` must:

- not panic (`check: panic`)
- give the same `Jobs`, `Commands`, `Workflows`, `Conditions`,
  `StepConditions`, `FilterExpressions`, `FlattenedSteps` and `Diagnostics`
  (`check: equivalence`, with the diff in `message`)
- leave every `when`, `unless` and `with_tool_cache` step written in the
  config in `CompileInfo` (`check: coverage`). The source is read with a
  second YAML parser, counting each sequence item that maps one of those keys
  to a mapping, by line. That count must match the conditions reachable from
  `CompileInfo.Steps` and its inline orbs, each counted once by range.

## templates

```sh
REFERENCE_CORPUS_DIR=<path-to-reference-corpus> go run ./cmd/dev/corpus templates > run.jsonl
```

Checks `internal/compile` against the reference compiler's own expectations:

- each case of each `replacer/*.txt` fixture must render to its expected
  output (`check: template`). A fixture is the template, a line of `=`, then
  cases separated by lines of `-`: an EDN map of variables, a blank line, and
  the expected output.
- each workflow in a `logic/*/build.yml` that compiles must be kept in its
  `config.yml` if, and only if, its `when` and `unless` allow it
  (`check: condition`). The variables are the pipeline parameters'
  defaults. Step conditions wait for the expander, as they sit inside jobs
  and commands.

`TestTemplates` runs the same checks, and skips when `REFERENCE_CORPUS_DIR`
is unset.

## Planned

- `compile`: for each case with a `config.yml`, compile `build.yml` and
  compare the graph with `workflows.*.jobs[*].upstream` (node names and
  `if_status`, including synthesised no-op, lock and unlock nodes), and each
  job IR with `jobs.*` (expanded steps). Orbs come from a fake resolver over
  `orb-registry/<ns>/<orb>/<ver>/orb.yml`, mapping `@X` to `X.0.0`, `@X.Y` to
  `X.Y.0` and `@volatile` to `1.0.0`. Maps compare unordered, lists ordered.
  Cases that need pipeline values the corpus doesn't hold go on a skip list.
- `errors`: for each case with `errors.txt` or `parse_errors.txt`, compile
  must fail. Matching messages, by normalising `at <config>:L:C:` against
  `Loc`, comes after.
- Real-world parity: `survey fetch` configs compiled here and compared with
  `circleci config process`.
