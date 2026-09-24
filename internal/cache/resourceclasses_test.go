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
		runnerRoute   = "/api/v3/runner/resource"
		unknownConfig = uri.URI("file:///elsewhere/.circleci/config.yml")
	)

	runnerFake := func(t *testing.T) (*fakes.CircleCI, circleci.Config) {
		fake := fakes.NewCircleCI(t)
		fake.AddRunnerResourceClass("acme", "acme/linux-arm", "ARM builders")

		api := configFor(fake.URL())
		api.RunnerHost = fake.URL()
		return fake, api
	}

	// Every config of an organization names the same runners.
	t.Run("lists a namespace once for every file of it", func(t *testing.T) {
		fake, api := runnerFake(t)
		c := New()

		c.SetNamespaceOfFile(api, rocketConfig, "acme")
		c.SetNamespaceOfFile(api, gadgetConfig, "acme")

		assert.Check(t, cmp.DeepEqual(c.ResourceClassesOfFile(api, rocketConfig), []string{"acme/linux-arm"}))
		assert.Check(t, cmp.DeepEqual(c.ResourceClassesOfFile(api, gadgetConfig), []string{"acme/linux-arm"}))
		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, runnerRoute), 1))
	})

	t.Run("reports none for a file whose namespace is not known", func(t *testing.T) {
		_, api := runnerFake(t)
		c := New()

		c.SetNamespaceOfFile(api, rocketConfig, "")

		assert.Check(t, cmp.Len(c.ResourceClassesOfFile(api, rocketConfig), 0))
		assert.Check(t, cmp.Len(c.ResourceClassesOfFile(api, unknownConfig), 0))
	})

	t.Run("does not remember a failure", func(t *testing.T) {
		fake, api := runnerFake(t)
		c := New()

		t.Run("fail the listing", func(t *testing.T) {
			fake.SetStatus("GET "+runnerRoute, http.StatusInternalServerError)
			c.SetNamespaceOfFile(api, rocketConfig, "acme")
			assert.Check(t, cmp.Len(c.ResourceClassesOfFile(api, rocketConfig), 0))
		})

		t.Run("check the next call lists them", func(t *testing.T) {
			fake.SetStatus("GET "+runnerRoute, 0)
			assert.Check(t, cmp.DeepEqual(c.ResourceClassesOfFile(api, rocketConfig), []string{"acme/linux-arm"}))
		})
	})
}
