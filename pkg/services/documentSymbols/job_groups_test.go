package documentSymbols

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func parseDoc(t *testing.T, yaml string) parser.YamlDocument {
	t.Helper()
	context := testHelpers.GetDefaultLsContext()
	doc, err := parser.ParseFromContent([]byte(yaml), context, uri.File("test.yml"), protocol.Position{})
	assert.Nil(t, err)
	return doc
}

func TestResolveJobGroupsSymbols_WithGroups(t *testing.T) {
	yaml := `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"
job-groups:
  deploy-group:
    jobs:
      - build
      - deploy:
          requires:
            - build
`
	doc := parseDoc(t, yaml)
	symbols := resolveJobGroupsSymbols(&doc)

	assert.Len(t, symbols, 1, "should return one top-level Job Groups symbol")
	assert.Equal(t, "Job Groups", symbols[0].Name)

	children := symbols[0].Children
	assert.Len(t, children, 1, "should have one job group")
	assert.Equal(t, "deploy-group", children[0].Name)

	jobsChildren := children[0].Children
	assert.Len(t, jobsChildren, 1, "group should have a Jobs child")
	assert.Equal(t, "Jobs", jobsChildren[0].Name)

	invocations := jobsChildren[0].Children
	assert.Len(t, invocations, 2, "Jobs should list both job invocations")

	names := []string{invocations[0].Name, invocations[1].Name}
	assert.Contains(t, names, "build")
	assert.Contains(t, names, "deploy")
}

func TestResolveJobGroupsSymbols_MultipleGroups(t *testing.T) {
	yaml := `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"
job-groups:
  group-a:
    jobs:
      - build
  group-b:
    jobs:
      - deploy
`
	doc := parseDoc(t, yaml)
	symbols := resolveJobGroupsSymbols(&doc)

	assert.Len(t, symbols, 1)
	assert.Len(t, symbols[0].Children, 2, "should have two job groups")

	groupNames := []string{symbols[0].Children[0].Name, symbols[0].Children[1].Name}
	assert.Contains(t, groupNames, "group-a")
	assert.Contains(t, groupNames, "group-b")
}

func TestResolveJobGroupsSymbols_NoJobGroups(t *testing.T) {
	yaml := `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
`
	doc := parseDoc(t, yaml)
	symbols := resolveJobGroupsSymbols(&doc)

	assert.Nil(t, symbols, "should return nil when no job-groups exist")
}
