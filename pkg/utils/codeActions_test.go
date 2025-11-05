package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestAppendSuppressionCodeActions(t *testing.T) {
	docURI := protocol.URI("file:///test.yml")

	tests := []struct {
		name                 string
		diagnostics          []protocol.Diagnostic
		docContent           []byte
		wantCodeActionTitles []string
		validateEdit         func(t *testing.T, actions []protocol.CodeAction)
	}{
		{
			name: "Indentation preservation",
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 4},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message: "Test error",
				},
			},
			docContent: []byte("version: 2.1\n    indented: line\n"),
			validateEdit: func(t *testing.T, actions []protocol.CodeAction) {
				// Find "Ignore this line" action
				for _, action := range actions {
					if action.Title == "Ignore this line" {
						edits := action.Edit.Changes[docURI]
						assert.Equal(t, "    # cci-ignore-next-line\n", edits[0].NewText)
						return
					}
				}
				t.Error("'Ignore this line' action not found")
			},
		},
		{
			name: "Inline suppression",
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 7},
					},
					Message: "Single line error",
				},
			},
			docContent: []byte("version: 2.1"),
			wantCodeActionTitles: []string{
				"Ignore this line",
				"Ignore this line (inline)",
				"Ignore all errors in this file",
			},
			validateEdit: func(t *testing.T, actions []protocol.CodeAction) {
				// Find inline action
				for _, action := range actions {
					if action.Title == "Ignore this line (inline)" {
						edits := action.Edit.Changes[docURI]
						assert.Equal(t, " # cci-ignore", edits[0].NewText)
						return
					}
				}
				t.Error("'Ignore this line (inline)' action not found")
			},
		},
		{
			name: "Next line suppression",
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 4},
					},
					Message: "Test error",
				},
			},
			docContent: []byte("version: 2.1\njobs:\n  test: value"),
			validateEdit: func(t *testing.T, actions []protocol.CodeAction) {
				// Find next line action
				for _, action := range actions {
					if action.Title == "Ignore this line" {
						edits := action.Edit.Changes[docURI]
						assert.Equal(t, "# cci-ignore-next-line\n", edits[0].NewText)
						assert.Equal(t, uint32(1), edits[0].Range.Start.Line)
						return
					}
				}
				t.Error("'Ignore this line' action not found")
			},
		},
		{
			name: "Range suppression",
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 3, Character: 5},
					},
					Message: "Multi-line error",
				},
			},
			docContent: []byte("version: 2.1\njobs:\n  test:\n    docker:\n"),
			wantCodeActionTitles: []string{
				"Ignore this line",
				"Ignore this range",
				"Ignore all errors in this file",
			},
			validateEdit: func(t *testing.T, actions []protocol.CodeAction) {
				// Find range action
				for _, action := range actions {
					if action.Title == "Ignore this range" {
						edits := action.Edit.Changes[docURI]
						assert.Len(t, edits, 2)
						assert.Equal(t, "# cci-ignore-start\n", edits[0].NewText)
						assert.Equal(t, uint32(1), edits[0].Range.Start.Line)
						assert.Equal(t, "# cci-ignore-end\n", edits[1].NewText)
						assert.Equal(t, uint32(4), edits[1].Range.Start.Line)
						return
					}
				}
				t.Error("'Ignore this range' action not found")
			},
		},
		{
			name: "Preserve existing code actions",
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 7},
					},
					Message: "Test error",
					Data: []protocol.CodeAction{
						{
							Title: "Existing action",
							Kind:  "quickfix",
						},
					},
				},
			},
			docContent: []byte("version: 2.1"),
			validateEdit: func(t *testing.T, actions []protocol.CodeAction) {
				// Check that existing action is preserved
				found := false
				for _, action := range actions {
					if action.Title == "Existing action" {
						found = true
						break
					}
				}
				assert.True(t, found, "Existing code action should be preserved")
				assert.GreaterOrEqual(t, len(actions), 4, "Should have existing + suppression actions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AppendSuppressionCodeActions(docURI, tt.diagnostics, tt.docContent)
			assert.NoError(t, err)
			assert.Len(t, result, len(tt.diagnostics))

			actions, ok := result[0].Data.([]protocol.CodeAction)
			assert.True(t, ok)

			if len(tt.wantCodeActionTitles) > 0 {
				titles := make([]string, len(actions))
				for j, action := range actions {
					titles[j] = action.Title
				}
				for _, expectedTitle := range tt.wantCodeActionTitles {
					assert.Contains(t, titles, expectedTitle)
				}
			}

			if tt.validateEdit != nil {
				tt.validateEdit(t, actions)
			}
		})
	}
}
