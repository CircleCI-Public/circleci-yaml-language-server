package complete

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCompleteDockerEntry(t *testing.T) {
	const config = `version: 2.1

executors:
  node:
    docker:
      - image: cimg/node:lts
        user: circleci
        

jobs:
  build:
    docker:
      - image: cimg/base:stable
        auth:
          username: me
          
      - image: postgres:16
        aws_auth:
          oidc_role_arn: arn:aws:iam::123:role/pull
          
    steps:
      - checkout
`
	below := func(text string, column uint32) []string {
		return completionLabels(t, config, positionBelow(t, config, text, column))
	}

	t.Run("an executor's image is offered the keys it doesn't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(below("user: circleci", 8), []string{
			"name", "entrypoint", "command", "environment", "auth", "aws_auth",
		}))
	})

	t.Run("an image's credentials are offered theirs", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(below("username: me", 10), []string{"password"}))
	})

	t.Run("so are its AWS credentials", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(below("oidc_role_arn: arn:aws:iam::123:role/pull", 10), []string{
			"aws_access_key_id", "aws_secret_access_key",
		}))
	})
}
