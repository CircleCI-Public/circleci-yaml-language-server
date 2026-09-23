package utils

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func Test_fromUrlToProjectSlug(t *testing.T) {
	type args struct {
		projectUrl string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "github",
			args: args{
				projectUrl: "https://github.com/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "gh/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "ssh github",
			args: args{
				projectUrl: "git@github.com:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "gh/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "github with .git",
			args: args{
				projectUrl: "https://github.com/CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "gh/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "ssh url github",
			args: args{
				projectUrl: "ssh://git@github.com/CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "gh/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "ssh url github with a port",
			args: args{
				projectUrl: "ssh://git@github.com:22/CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "gh/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "bitbucket with .git",
			args: args{
				projectUrl: "https://bitbucket.org/CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "bb/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "bitbucket",
			args: args{
				projectUrl: "https://bitbucket.org/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "bb/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "ssh bitbucket",
			args: args{
				projectUrl: "git@bitbucket.org:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "bb/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "gitlab",
			args: args{
				projectUrl: "https://gitlab.com/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "",
		},
		{
			name: "ssh gitlab",
			args: args{
				projectUrl: "git@gitlab.com:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "",
		},
		{
			name: "invalid",
			args: args{
				projectUrl: "https://invalid.com/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "",
		},
		{
			name: "ssh invalid",
			args: args{
				projectUrl: "git@invalid.com:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fromUrlToProjectSlug(tt.args.projectUrl); got != tt.want {
				t.Errorf("fromUrlToProjectSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProjectId(t *testing.T) {
	const slug = "gh/acme/rocket"

	projectFake := func(t *testing.T) *fakes.CircleCI {
		t.Helper()

		fake := fakes.NewCircleCI(t)
		fake.AddProject(slug, "proj-rocket", "org-acme", "gh/acme")

		return fake
	}

	t.Run("reports the project and its organization", func(t *testing.T) {
		fake := projectFake(t)

		project, err := GetProjectId(slug, lsContextFor(fake.URL()))
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

		project, err := GetProjectId("gh/acme/unknown", lsContextFor(fake.URL()))
		assert.Check(t, httpcl.HasStatusCode(err, 404), "got %v", err)
		assert.Check(t, cmp.DeepEqual(project, Project{}))
	})

	t.Run("reports a failing host", func(t *testing.T) {
		fake := projectFake(t)
		fake.SetStatus("GET /api/v2/project/"+slug, http.StatusInternalServerError)

		project, err := GetProjectId(slug, lsContextFor(fake.URL()))
		assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)
		assert.Check(t, cmp.DeepEqual(project, Project{}))
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		fake := projectFake(t)
		fake.SetBody("GET /api/v2/project/"+slug, "{")

		project, err := GetProjectId(slug, lsContextFor(fake.URL()))
		assert.Check(t, cmp.ErrorContains(err, "decode response"))
		assert.Check(t, cmp.DeepEqual(project, Project{}))
	})

	t.Run("reports an unreachable host", func(t *testing.T) {
		fake := projectFake(t)
		lsContext := lsContextFor(fake.URL())
		fake.Close()

		_, err := GetProjectId(slug, lsContext)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})

	// A self-hosted URL comes from user settings, so it is not necessarily a
	// URL at all.
	t.Run("reports an unusable host URL", func(t *testing.T) {
		_, err := GetProjectId(slug, lsContextFor("not a url"))
		assert.Check(t, err != nil, "an unparseable host must be reported, not panic")
	})
}
