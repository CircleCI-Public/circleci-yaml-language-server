package cache

import (
	"net/http"
	"testing"

	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func TestResourceClassesOfFile(t *testing.T) {
	const (
		rocketConfig  = uri.URI("file:///rocket/.circleci/config.yml")
		gadgetConfig  = uri.URI("file:///gadget/.circleci/config.yml")
		unknownConfig = uri.URI("file:///elsewhere/.circleci/config.yml")
		orgsRoute     = "/api/v3/orgs"
		runnerRoute   = "/api/v3/runner/resource-classes"
		acmeOrgID     = "4b9e2c1a-0000-4000-8000-000000000001"
	)

	// Acme's classes are named for namespaces it has claimed, neither of which
	// is its own name.
	acmeClasses := []string{"acme-builders/linux-arm", "acme-labs/gpu"}

	runnerFake := func(t *testing.T) (*fakes.CircleCI, *circleci.V3Client) {
		fake := fakes.NewCircleCI(t)
		fake.AddOrg("gh/acme", acmeOrgID)
		fake.AddRunnerResourceClass(acmeOrgID, "acme-builders/linux-arm", "ARM builders")
		fake.AddRunnerResourceClass(acmeOrgID, "acme-labs/gpu", "")

		api := configFor(fake.URL())
		return fake, circleci.NewV3Client(api.HostUrl, api.Token, "", false)
	}

	t.Run("lists an organization's classes by its id, whatever their namespaces", func(t *testing.T) {
		fake, client := runnerFake(t)
		c := New()

		c.SetOrgOfFile(client, rocketConfig, "gh/acme")
		classes := c.ResourceClassesOfFile(client, rocketConfig)

		assert.Check(t, cmp.DeepEqual(classes, acmeClasses))

		var orgIDs []string
		for _, request := range fake.Requests() {
			if request.Path == runnerRoute {
				orgIDs = append(orgIDs, request.Query["filter[org_id]"])
			}
		}
		assert.Check(t, cmp.DeepEqual(orgIDs, []string{acmeOrgID}))
	})

	// Every config of an organization names the same runners.
	t.Run("lists an organization once for every file of it", func(t *testing.T) {
		fake, client := runnerFake(t)
		c := New()

		c.SetOrgOfFile(client, rocketConfig, "gh/acme")
		c.SetOrgOfFile(client, gadgetConfig, "gh/acme")
		rocketClasses := c.ResourceClassesOfFile(client, rocketConfig)
		gadgetClasses := c.ResourceClassesOfFile(client, gadgetConfig)

		assert.Check(t, cmp.DeepEqual(rocketClasses, acmeClasses))
		assert.Check(t, cmp.DeepEqual(gadgetClasses, acmeClasses))
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, orgsRoute), 1))
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, runnerRoute), 1))
	})

	t.Run("forgets the organization of a closed file", func(t *testing.T) {
		_, client := runnerFake(t)
		c := New()

		c.SetOrgOfFile(client, rocketConfig, "gh/acme")
		c.SetOrgOfFile(client, gadgetConfig, "gh/acme")
		c.ForgetFile(rocketConfig)

		assert.Check(t, cmp.Len(c.ResourceClassesOfFile(client, rocketConfig), 0))
		assert.Check(t, cmp.DeepEqual(c.ResourceClassesOfFile(client, gadgetConfig), acmeClasses),
			"another file of the organization keeps its classes")
	})

	t.Run("reports none for a file whose organization is not known", func(t *testing.T) {
		_, client := runnerFake(t)
		c := New()

		c.SetOrgOfFile(client, rocketConfig, "")
		rocketClasses := c.ResourceClassesOfFile(client, rocketConfig)
		unknownClasses := c.ResourceClassesOfFile(client, unknownConfig)

		assert.Check(t, cmp.Len(rocketClasses, 0))
		assert.Check(t, cmp.Len(unknownClasses, 0))
	})

	t.Run("asks nothing without a token", func(t *testing.T) {
		fake, client := runnerFake(t)
		anonymous := circleci.NewV3Client(client.Host, "", "", false)
		c := New()

		t.Run("open a file anonymously", func(t *testing.T) {
			c.SetOrgOfFile(anonymous, rocketConfig, "gh/acme")
			classes := c.ResourceClassesOfFile(anonymous, rocketConfig)
			requests := fake.Requests()

			assert.Check(t, cmp.Len(classes, 0))
			assert.Check(t, cmp.Len(requests, 0))
		})

		t.Run("check the classes are listed once a token is set", func(t *testing.T) {
			classes := c.ResourceClassesOfFile(client, rocketConfig)
			assert.Check(t, cmp.DeepEqual(classes, acmeClasses))
		})
	})

	// A repository whose organization CircleCI has never seen has no runners,
	// and asking again will not change that.
	t.Run("remembers an organization CircleCI does not know", func(t *testing.T) {
		fake, client := runnerFake(t)
		c := New()

		c.SetOrgOfFile(client, rocketConfig, "gh/nobody")
		classes := c.ResourceClassesOfFile(client, rocketConfig)

		assert.Check(t, cmp.Len(classes, 0))
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, orgsRoute), 1))
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, runnerRoute), 0))
	})

	t.Run("does not remember a failure", func(t *testing.T) {
		fake, client := runnerFake(t)
		c := New()

		t.Run("fail the listing", func(t *testing.T) {
			fake.SetStatus("GET "+runnerRoute, http.StatusInternalServerError)
			c.SetOrgOfFile(client, rocketConfig, "gh/acme")
			classes := c.ResourceClassesOfFile(client, rocketConfig)
			assert.Check(t, cmp.Len(classes, 0))
		})

		t.Run("check the next call lists them", func(t *testing.T) {
			fake.SetStatus("GET "+runnerRoute, 0)
			classes := c.ResourceClassesOfFile(client, rocketConfig)
			assert.Check(t, cmp.DeepEqual(classes, acmeClasses))
		})
	})
}
