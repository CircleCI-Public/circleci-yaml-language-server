package fakes_test

// Runner resource classes are fetched from runner.<host> in production, which
// nothing can reach in a test until that host is overridable, so this route has
// no caller yet and is checked here rather than left unexercised.

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func TestRunnerResourceRoute(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.AddRunnerResourceClass("acme", "acme/linux-arm", "ARM builders")

	type response struct {
		Items []struct {
			ResourceClass string `json:"resource_class"`
			Description   string `json:"description"`
		} `json:"items"`
	}

	t.Run("reports the resource classes of a namespace", func(t *testing.T) {
		var body response

		status := getJSON(t, fake.URL()+"/api/v3/runner/resource?namespace=acme", &body)
		assert.Check(t, cmp.Equal(status, http.StatusOK))
		assert.Assert(t, cmp.Len(body.Items, 1))
		assert.Check(t, cmp.Equal(body.Items[0].ResourceClass, "acme/linux-arm"))
		assert.Check(t, cmp.Equal(body.Items[0].Description, "ARM builders"))
	})

	// An organization with no runners is an empty list, not an error, so a
	// caller cannot tell the two apart and must not try.
	t.Run("reports an unknown namespace as an empty list", func(t *testing.T) {
		var body response

		status := getJSON(t, fake.URL()+"/api/v3/runner/resource?namespace=nobody", &body)
		assert.Check(t, cmp.Equal(status, http.StatusOK))
		assert.Check(t, cmp.Len(body.Items, 0))
	})
}
