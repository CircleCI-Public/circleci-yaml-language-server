package circleci

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func TestGetProject(t *testing.T) {
	const slug = "gh/acme/rocket"

	projectFake := func(t *testing.T) *fakes.CircleCI {
		t.Helper()

		fake := fakes.NewCircleCI(t)
		fake.AddProject(slug, "proj-rocket", "org-acme", "gh/acme")

		return fake
	}

	t.Run("reports the project and its organization", func(t *testing.T) {
		fake := projectFake(t)

		project, err := GetProject(configFor(fake.URL()), slug)
		assert.NilError(t, err)

		// The organization id is what the context and env var lookups key on,
		// so it matters more here than the project's own id.
		assert.Check(t, cmp.Equal(project.Id, "proj-rocket"))
		assert.Check(t, cmp.Equal(project.Slug, slug))
		assert.Check(t, cmp.Equal(project.Name, "rocket"))
		assert.Check(t, cmp.Equal(project.OrganizationId, "org-acme"))
		assert.Check(t, cmp.Equal(project.OrganizationSlug, "gh/acme"))
		assert.Check(t, cmp.Equal(project.OrganizationName, "acme"))
		assert.Check(t, cmp.Equal(project.VcsInfo.Provider, "GitHub"))
		assert.Check(t, cmp.Equal(project.VcsInfo.Default_branch, "main"))

		t.Run("authenticating with Circle-Token", func(t *testing.T) {
			requests := fake.Requests()
			assert.Assert(t, cmp.Len(requests, 1))
			assert.Check(t, cmp.Equal(requests[0].Path, "/api/v2/project/"+slug))
			assert.Check(t, cmp.Equal(requests[0].CircleToken, testToken))
		})
	})

	// A project the token cannot see is indistinguishable from one that does
	// not exist, and either way there is no organization to look contexts up
	// against.
	t.Run("reports an unknown project", func(t *testing.T) {
		fake := projectFake(t)

		project, err := GetProject(configFor(fake.URL()), "gh/acme/unknown")
		assert.Check(t, httpcl.HasStatusCode(err, 404), "got %v", err)
		assert.Check(t, cmp.DeepEqual(project, Project{}))
	})

	t.Run("reports a failing host", func(t *testing.T) {
		fake := projectFake(t)
		fake.SetStatus("GET /api/v2/project/"+slug, http.StatusInternalServerError)

		project, err := GetProject(configFor(fake.URL()), slug)
		assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)
		assert.Check(t, cmp.DeepEqual(project, Project{}))
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		fake := projectFake(t)
		fake.SetBody("GET /api/v2/project/"+slug, "{")

		project, err := GetProject(configFor(fake.URL()), slug)
		assert.Check(t, cmp.ErrorContains(err, "decode response"))
		assert.Check(t, cmp.DeepEqual(project, Project{}))
	})

	t.Run("reports an unreachable host", func(t *testing.T) {
		fake := projectFake(t)
		api := configFor(fake.URL())
		fake.Close()

		_, err := GetProject(api, slug)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})

	// A self-hosted URL comes from user settings, so it is not necessarily a
	// URL at all.
	t.Run("reports an unusable host URL", func(t *testing.T) {
		_, err := GetProject(configFor("not a url"), slug)
		assert.Check(t, err != nil, "an unparseable host must be reported, not panic")
	})
}
