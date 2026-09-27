package fakes_test

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func TestOrgAndRunnerRoutes(t *testing.T) {
	const acmeOrgID = "4b9e2c1a-0000-4000-8000-000000000001"

	fake := fakes.NewCircleCI(t)
	fake.AddOrg("gh/acme", acmeOrgID)
	fake.AddRunnerResourceClass(acmeOrgID, "acme-builders/linux-arm", "ARM builders")

	type orgs struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	type classes struct {
		Data []struct {
			Attributes struct {
				ResourceClass string `json:"resource_class"`
				Description   string `json:"description"`
			} `json:"attributes"`
		} `json:"data"`
	}

	t.Run("looks an organization up by slug", func(t *testing.T) {
		var body orgs

		status := getJSON(t, fake.URL()+"/api/v3/orgs?filter[slug]=gh/acme", &body)
		assert.Check(t, cmp.Equal(status, http.StatusOK))
		assert.Assert(t, cmp.Len(body.Data, 1))
		assert.Check(t, cmp.Equal(body.Data[0].ID, acmeOrgID))
	})

	t.Run("reports an unknown slug as an empty list", func(t *testing.T) {
		var body orgs

		status := getJSON(t, fake.URL()+"/api/v3/orgs?filter[slug]=gh/nobody", &body)
		assert.Check(t, cmp.Equal(status, http.StatusOK))
		assert.Check(t, cmp.Len(body.Data, 0))
	})

	t.Run("reports the resource classes of an organization", func(t *testing.T) {
		var body classes

		status := getJSON(t, fake.URL()+"/api/v3/runner/resource-classes?filter[org_id]="+acmeOrgID, &body)
		assert.Check(t, cmp.Equal(status, http.StatusOK))
		assert.Assert(t, cmp.Len(body.Data, 1))
		assert.Check(t, cmp.Equal(body.Data[0].Attributes.ResourceClass, "acme-builders/linux-arm"))
		assert.Check(t, cmp.Equal(body.Data[0].Attributes.Description, "ARM builders"))
	})

	t.Run("refuses an unknown organization", func(t *testing.T) {
		var body map[string]any

		status := getJSON(t, fake.URL()+"/api/v3/runner/resource-classes?filter[org_id]=nobody", &body)
		assert.Check(t, cmp.Equal(status, http.StatusForbidden))
	})

	t.Run("requires an organization", func(t *testing.T) {
		var body map[string]any

		status := getJSON(t, fake.URL()+"/api/v3/runner/resource-classes", &body)
		assert.Check(t, cmp.Equal(status, http.StatusBadRequest))
	})
}
