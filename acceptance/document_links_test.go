package acceptance

import (
	"fmt"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	pos "github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// linksConfig declares orbs and Docker images in each of the ways that link
// somewhere, and in some that don't.
const linksConfig = `version: 2.1

orbs:
  node: circleci/node@5.0.0
  go: "circleci/go@volatile"
  remote: https://example.com/orbs/tools.yml
  inline:
    commands:
      hello:
        steps:
          - run: echo hello

executors:
  database:
    docker:
      - image: cimg/postgres:16.4
      - image: redis

jobs:
  build:
    parameters:
      tag:
        type: string
        default: current
    docker:
      - image: "cimg/base:current"
      - image: library/ubuntu:24.04
      - image: gcr.io/acme/tools:1
      - image: cimg/node:<< parameters.tag >>
    steps:
      - checkout
`

func TestDocumentLinks(t *testing.T) {
	session := start(t, linkedProjectFake(t), linksConfig, testToken)
	session.open(t, linksConfig)

	var links []protocol.DocumentLink
	err := session.client.Call(protocol.MethodTextDocumentDocumentLink, protocol.DocumentLinkParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: session.workspace.URI()},
	}, &links)
	assert.NilError(t, err)

	content := []byte(linksConfig)
	got := []string{}
	for _, link := range links {
		start := pos.ToIndex(link.Range.Start, content)
		end := pos.ToIndex(link.Range.End, content)
		target := ""
		if link.Target != nil {
			target = string(*link.Target)
		}
		got = append(got, fmt.Sprintf("%d:%s -> %s", link.Range.Start.Line+1, linksConfig[start:end], target))
	}

	// The session is on a server of its own, the fake, whose orbs have no
	// page in circleci.com's registry.
	assert.Check(t, cmp.DeepEqual(got, []string{
		"6:https://example.com/orbs/tools.yml -> https://example.com/orbs/tools.yml",
		"16:cimg/postgres:16.4 -> https://hub.docker.com/r/cimg/postgres",
		"17:redis -> https://hub.docker.com/_/redis",
		"26:cimg/base:current -> https://hub.docker.com/r/cimg/base",
		"27:library/ubuntu:24.04 -> https://hub.docker.com/_/ubuntu",
	}))
}
