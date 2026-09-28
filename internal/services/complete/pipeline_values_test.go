package complete

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestCompletePipelineValues(t *testing.T) {
	const config = `version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo << pipeline.
      - run: echo << pipeline.git. >>
      - run: echo << pipeline.parameters.

workflows:
  main:
    when: << pipeline.git.branch == "main" and pipeline.trigger_parameters.github_app.
    jobs:
      - build
`
	itemsAfter := func(t *testing.T, text string) []protocol.CompletionItem {
		t.Helper()
		return completionItemsAfter(t, config, text)
	}

	t.Run("after pipeline., the first segments are offered", func(t *testing.T) {
		items := itemsAfter(t, "echo << pipeline.")
		labels := labelsOf(items)
		assert.Check(t, cmp.Contains(labels, "git"))
		assert.Check(t, cmp.Contains(labels, "number"))
		assert.Check(t, cmp.Contains(labels, "parameters"))
		assert.Check(t, !slices.Contains(labels, "config_source"), "private values are left out")
		sorted := slices.Sorted(slices.Values(labels))
		unique := slices.Compact(slices.Clone(sorted))
		assert.Check(t, cmp.DeepEqual(sorted, unique), "each segment is offered once")

		git := findItem(t, items, "git")
		assert.Check(t, cmp.Equal(git.InsertText, protocol.NewOptional("git.")))

		number := findItem(t, items, "number")
		assert.Check(t, cmp.Equal(number.InsertText, protocol.NewOptional("number >>")))
		assert.Check(t, cmp.Equal(number.Detail, protocol.NewOptional("uint")))
	})

	t.Run("a value that's already closed isn't closed again", func(t *testing.T) {
		items := itemsAfter(t, "echo << pipeline.git.")
		branch := findItem(t, items, "branch")
		assert.Check(t, cmp.Equal(branch.InsertText, protocol.NewOptional("branch")))
		assert.Check(t, !slices.Contains(labelsOf(items), "deploy"), "pipeline parameters aren't offered")
	})

	t.Run("the pipeline parameters come from the config", func(t *testing.T) {
		got := labelsOf(itemsAfter(t, "echo << pipeline.parameters."))
		assert.Check(t, cmp.DeepEqual(got, []string{"deploy"}))
	})

	t.Run("in an expression, a replaced value is marked deprecated", func(t *testing.T) {
		items := itemsAfter(t, "pipeline.trigger_parameters.github_app.")
		repoName := findItem(t, items, "repo_name")
		assert.Check(t, cmp.DeepEqual(repoName.Tags, []protocol.CompletionItemTag{protocol.CompletionItemTagDeprecated}))
		tag := findItem(t, items, "tag")
		assert.Check(t, cmp.Len(tag.Tags, 0))
	})
}

func TestCompleteBareExpressions(t *testing.T) {
	const config = `version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

jobs:
  build:
    parameters:
      os:
        type: string
        default: linux
    docker:
      - image: cimg/base:stable
    steps:
      - when:
          condition: pipeline.
          steps:
            - checkout

workflows:
  main:
    when: pipeline.git.
    jobs:
      - build:
          filters: pipeline.git.branch == "main" and pipeline.parameters.
  other:
    unless: pip
    jobs:
      - build:
          filters: parameters.
`
	t.Run("in a workflow's condition, a value isn't closed with >>", func(t *testing.T) {
		items := completionItemsAfter(t, config, "when: pipeline.git.")
		branch := findItem(t, items, "branch")
		assert.Check(t, cmp.Equal(branch.InsertText, protocol.NewOptional("branch")))
	})

	t.Run("in a filter, the pipeline parameters are offered", func(t *testing.T) {
		items := completionItemsAfter(t, config, "and pipeline.parameters.")
		assert.Check(t, cmp.DeepEqual(labelsOf(items), []string{"deploy"}))
		deploy := findItem(t, items, "deploy")
		assert.Check(t, cmp.Equal(deploy.InsertText, protocol.NewOptional("deploy")))
	})

	t.Run("a word being started is offered pipeline", func(t *testing.T) {
		items := completionItemsAfter(t, config, "unless: pip")
		pipeline := findItem(t, items, "pipeline")
		assert.Check(t, cmp.Equal(pipeline.InsertText, protocol.NewOptional("pipeline.")))
	})

	t.Run("a job's parameters aren't offered, since a workflow has none", func(t *testing.T) {
		got := labelsOf(completionItemsAfter(t, config, "filters: parameters."))
		assert.Check(t, !slices.Contains(got, "os"))
	})

	t.Run("a step's condition isn't an expression, so nothing is offered", func(t *testing.T) {
		got := labelsOf(completionItemsAfter(t, config, "condition: pipeline."))
		assert.Check(t, !slices.Contains(got, "git"))
	})
}

// completionItemsAfter are the items offered with the cursor straight after
// text, on the first line of config that contains it.
func completionItemsAfter(t *testing.T, config, text string) []protocol.CompletionItem {
	t.Helper()
	lines := strings.Split(config, "\n")
	line := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, text) })
	assert.Assert(t, line != -1, "no line with %q", text)
	column := strings.Index(lines[line], text) + len(text)
	pos := protocol.Position{Line: uint32(line), Character: uint32(column)}
	return completionItemsWith(t, testHelpers.DefaultSettings(), cache.New(), config, pos)
}

func labelsOf(items []protocol.CompletionItem) []string {
	labels := []string{}
	for _, item := range items {
		labels = append(labels, item.Label)
	}
	return labels
}

func findItem(t *testing.T, items []protocol.CompletionItem, label string) protocol.CompletionItem {
	t.Helper()
	i := slices.IndexFunc(items, func(item protocol.CompletionItem) bool { return item.Label == label })
	assert.Assert(t, i != -1, "no item %q", label)
	return items[i]
}
