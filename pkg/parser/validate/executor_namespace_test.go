package validate

import (
	"net/http"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

// validateNamespaceAgainst runs the namespace check for a self-hosted runner
// resource class against the given fake, and returns whatever diagnostics it
// produced.
func validateNamespaceAgainst(fake *fakes.CircleCI, namespace string) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}
	val := Validate{
		Diagnostics: &diagnostics,
		Cache:       utils.CreateCache(),
		Context:     testHelpers.GetLsContextForHost(fake.URL()),
	}

	val.validateExecutorNamespace(namespace, protocol.Range{})

	return *val.Diagnostics
}

func TestValidateExecutorNamespace(t *testing.T) {
	t.Run("accepts a namespace that exists", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")

		diagnostics := validateNamespaceAgainst(fake, "acme")
		assert.Check(t, cmp.Len(diagnostics, 0))
	})

	t.Run("flags a namespace that does not exist", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")

		diagnostics := validateNamespaceAgainst(fake, "nope")
		assert.Assert(t, cmp.Len(diagnostics, 1))

		assert.Check(t, cmp.Equal(diagnostics[0].Message, `Namespace "nope" does not exist`))
		assert.Check(t, cmp.Equal(diagnostics[0].Severity, protocol.DiagnosticSeverityError))
	})

	// A failed request is not evidence a namespace is missing. Reporting one
	// would flag valid configs whenever the API is unreachable, so everything
	// short of a definitive 404 has to stay silent.
	t.Run("stays silent when the namespace cannot be checked", func(t *testing.T) {
		for _, testCase := range []struct {
			name   string
			status int
		}{
			{"a server error", http.StatusInternalServerError},
			{"an unauthorized token", http.StatusUnauthorized},
			{"a rate limit", http.StatusTooManyRequests},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				fake := fakes.NewCircleCI(t)
				fake.AddNamespace("ns-acme", "acme")
				fake.SetStatus("GET /api/v3/namespaces", testCase.status)

				diagnostics := validateNamespaceAgainst(fake, "acme")
				assert.Check(t, cmp.Len(diagnostics, 0))
			})
		}
	})

	t.Run("stays silent when no host is configured", func(t *testing.T) {
		diagnostics := []protocol.Diagnostic{}
		val := Validate{
			Diagnostics: &diagnostics,
			Cache:       utils.CreateCache(),
			Context:     testHelpers.GetLsContextForHost(""),
		}

		val.validateExecutorNamespace("acme", protocol.Range{})

		assert.Check(t, cmp.Len(*val.Diagnostics, 0))
	})
}
