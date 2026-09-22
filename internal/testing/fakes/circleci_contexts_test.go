package fakes_test

// The context list is covered by the tests of its caller in pkg/utils. This
// route has no caller yet — the language server reads a context's variables
// inline from the list — so it is checked here rather than left unexercised.

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func TestContextEnvVarRoute(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.AddContext("org-acme", "ctx-deploy", "acme/deploy")
	fake.AddContextEnvVar("ctx-deploy", "DEPLOY_KEY")

	t.Run("reports a context's variables by name", func(t *testing.T) {
		var body struct {
			Items []struct {
				Variable  string `json:"variable"`
				ContextID string `json:"context_id"`
			} `json:"items"`
			NextPageToken *string `json:"next_page_token"`
		}

		status := getJSON(t, fake.URL()+"/api/v2/context/ctx-deploy/environment-variable", &body)
		assert.Check(t, cmp.Equal(status, http.StatusOK))
		assert.Assert(t, cmp.Len(body.Items, 1))
		assert.Check(t, cmp.Equal(body.Items[0].Variable, "DEPLOY_KEY"))
		assert.Check(t, cmp.Equal(body.Items[0].ContextID, "ctx-deploy"))
		assert.Check(t, cmp.Nil(body.NextPageToken))
	})

	t.Run("reports an unknown context as not found", func(t *testing.T) {
		var body struct {
			Message string `json:"message"`
		}

		status := getJSON(t, fake.URL()+"/api/v2/context/ctx-nope/environment-variable", &body)
		assert.Check(t, cmp.Equal(status, http.StatusNotFound))
		assert.Check(t, cmp.Equal(body.Message, "Context not found"))
	})
}
