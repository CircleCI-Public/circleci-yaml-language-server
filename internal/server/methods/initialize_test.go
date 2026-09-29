package methods

import (
	"context"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func TestInitializeSchemaHovers(t *testing.T) {
	testCases := []struct {
		name    string
		options map[string]any
		want    bool
	}{
		{name: "on for a client that sends no options", options: nil, want: true},
		{name: "on for a client that sends other options", options: map[string]any{"editor": "an-editor"}, want: true},
		{name: "off for the VS Code extension", options: map[string]any{"isCciExtension": true}, want: false},
		{name: "on for the VS Code extension when it asks", options: map[string]any{"isCciExtension": true, "schemaHovers": true}, want: true},
		{name: "off for any client that asks", options: map[string]any{"schemaHovers": false}, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			methods := New(context.Background(), nil, cache.New(), session.Settings{}, "")
			params := &protocol.InitializeParams{}
			if tc.options != nil {
				options, err := protocol.Marshal(tc.options)
				assert.NilError(t, err)
				params.InitializationOptions = options
			}

			_, err := methods.Initialize(context.Background(), params)
			assert.NilError(t, err)

			got := methods.Settings().SchemaHovers
			assert.Check(t, cmp.Equal(got, tc.want))
		})
	}
}
