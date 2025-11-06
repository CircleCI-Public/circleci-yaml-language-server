package parser

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func TestParseSuppressionComments(t *testing.T) {
	tests := []struct {
		name                    string
		yaml                    string
		wantFileWideSuppression bool
		wantSuppressedLines     map[uint32]bool
		wantSuppressedRanges    []SuppressionRange
		wantDiagnosticsCount    int
	}{
		{
			name: "File-wide suppression",
			yaml: `# cci-ignore-file
version: 2.1
jobs:
  test:
    docker:
      - image: invalid`,
			wantFileWideSuppression: true,
			wantSuppressedLines:     map[uint32]bool{},
			wantSuppressedRanges:    []SuppressionRange{},
			wantDiagnosticsCount:    0,
		},
		{
			name: "Inline suppression",
			yaml: `version: 2.1
jobs:
  test: # cci-ignore
    docker:
      - image: foo`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{2: true},
			wantSuppressedRanges:    []SuppressionRange{},
			wantDiagnosticsCount:    0,
		},
		{
			name: "Next-line suppression",
			yaml: `version: 2.1
jobs:
  test:
    docker:
      # cci-ignore-next-line
      - image: foo`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{5: true},
			wantSuppressedRanges:    []SuppressionRange{},
			wantDiagnosticsCount:    0,
		},
		{
			name: "Range suppression",
			yaml: `version: 2.1
jobs:
  # cci-ignore-start
  test:
    docker:
      - image: foo
  # cci-ignore-end
  another:
    docker:
      - image: bar`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{},
			wantSuppressedRanges: []SuppressionRange{
				{StartLine: 2, EndLine: 6},
			},
			wantDiagnosticsCount: 0,
		},
		{
			name: "Multiple range suppressions",
			yaml: `version: 2.1
# cci-ignore-start
jobs:
  test:
# cci-ignore-end
    docker:
      - image: foo
# cci-ignore-start
  another:
    docker:
# cci-ignore-end
      - image: bar`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{},
			wantSuppressedRanges: []SuppressionRange{
				{StartLine: 1, EndLine: 4},
				{StartLine: 7, EndLine: 10},
			},
			wantDiagnosticsCount: 0,
		},
		{
			name: "Unclosed range suppression",
			yaml: `version: 2.1
# cci-ignore-start
jobs:
  test:
    docker:
      - image: foo`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{},
			wantSuppressedRanges:    []SuppressionRange{},
			wantDiagnosticsCount:    1,
		},
		{
			name: "Range end without start",
			yaml: `version: 2.1
jobs:
  test:
    # cci-ignore-end
    docker:
      - image: foo`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{},
			wantSuppressedRanges:    []SuppressionRange{},
			wantDiagnosticsCount:    1,
		},
		{
			name: "Nested range start (error)",
			yaml: `version: 2.1
# cci-ignore-start
jobs:
  # cci-ignore-start
  test:
    docker:
# cci-ignore-end
      - image: foo`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{},
			wantSuppressedRanges: []SuppressionRange{
				{StartLine: 1, EndLine: 6},
			},
			wantDiagnosticsCount: 1,
		},
		{
			name: "Multiple suppression types",
			yaml: `version: 2.1
# cci-ignore-start
jobs:
  test: # cci-ignore
# cci-ignore-end
    docker:
      # cci-ignore-next-line
      - image: foo`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{3: true, 7: true},
			wantSuppressedRanges: []SuppressionRange{
				{StartLine: 1, EndLine: 4},
			},
			wantDiagnosticsCount: 0,
		},
		{
			name: "No suppressions",
			yaml: `version: 2.1
jobs:
  test:
    docker:
      - image: foo`,
			wantFileWideSuppression: false,
			wantSuppressedLines:     map[uint32]bool{},
			wantSuppressedRanges:    []SuppressionRange{},
			wantDiagnosticsCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := ParseFile([]byte(tt.yaml), &utils.LsContext{})
			suppressionInfo := ParseSuppressionComments(&doc)

			if suppressionInfo.FileWideSuppression != tt.wantFileWideSuppression {
				t.Errorf("FileWideSuppression = %v, want %v", suppressionInfo.FileWideSuppression, tt.wantFileWideSuppression)
			}

			if len(suppressionInfo.SuppressedLines) != len(tt.wantSuppressedLines) {
				t.Errorf("SuppressedLines count = %v, want %v", len(suppressionInfo.SuppressedLines), len(tt.wantSuppressedLines))
			}

			for line := range tt.wantSuppressedLines {
				if !suppressionInfo.SuppressedLines[line] {
					t.Errorf("Expected line %d to be suppressed", line)
				}
			}

			if len(suppressionInfo.SuppressedRanges) != len(tt.wantSuppressedRanges) {
				t.Errorf("SuppressedRanges count = %v, want %v", len(suppressionInfo.SuppressedRanges), len(tt.wantSuppressedRanges))
			}

			for i, wantRange := range tt.wantSuppressedRanges {
				if i >= len(suppressionInfo.SuppressedRanges) {
					break
				}
				gotRange := suppressionInfo.SuppressedRanges[i]
				if gotRange.StartLine != wantRange.StartLine || gotRange.EndLine != wantRange.EndLine {
					t.Errorf("SuppressedRange[%d] = {%d, %d}, want {%d, %d}",
						i, gotRange.StartLine, gotRange.EndLine, wantRange.StartLine, wantRange.EndLine)
				}
			}

			if len(*doc.Diagnostics) != tt.wantDiagnosticsCount {
				t.Errorf("Diagnostics count = %v, want %v", len(*doc.Diagnostics), tt.wantDiagnosticsCount)
			}
		})
	}
}

func TestIsDiagnosticSuppressed(t *testing.T) {
	tests := []struct {
		name            string
		suppressionInfo *SuppressionInfo
		diagnostic      protocol.Diagnostic
		want            bool
	}{
		{
			name:            "Nil suppression info",
			suppressionInfo: nil,
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 5, Character: 10},
				},
			},
			want: false,
		},
		{
			name: "File-wide suppression",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: true,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges:    []SuppressionRange{},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 5, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Line suppressed",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{5: true},
				SuppressedRanges:    []SuppressionRange{},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 5, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Line not suppressed",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{3: true},
				SuppressedRanges:    []SuppressionRange{},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 5, Character: 10},
				},
			},
			want: false,
		},
		{
			name: "Range suppression - diagnostic at start",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges: []SuppressionRange{
					{StartLine: 5, EndLine: 10},
				},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 5, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Range suppression - diagnostic in middle",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges: []SuppressionRange{
					{StartLine: 5, EndLine: 10},
				},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 0},
					End:   protocol.Position{Line: 7, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Range suppression - diagnostic at end",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges: []SuppressionRange{
					{StartLine: 5, EndLine: 10},
				},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 10, Character: 0},
					End:   protocol.Position{Line: 10, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Range suppression - diagnostic before range",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges: []SuppressionRange{
					{StartLine: 5, EndLine: 10},
				},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 10},
				},
			},
			want: false,
		},
		{
			name: "Range suppression - diagnostic after range",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges: []SuppressionRange{
					{StartLine: 5, EndLine: 10},
				},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 12, Character: 0},
					End:   protocol.Position{Line: 12, Character: 10},
				},
			},
			want: false,
		},
		{
			name: "Multi-line diagnostic suppressed by next-line",
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{5: true},
				SuppressedRanges:    []SuppressionRange{},
			},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 8, Character: 10},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDiagnosticSuppressed(tt.suppressionInfo, tt.diagnostic)
			if got != tt.want {
				t.Errorf("isDiagnosticSuppressed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiagnosticOverlapsRange(t *testing.T) {
	tests := []struct {
		name       string
		r          SuppressionRange
		diagnostic protocol.Diagnostic
		want       bool
	}{
		{
			name: "Diagnostic at start of range",
			r:    SuppressionRange{StartLine: 5, EndLine: 10},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 5, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Diagnostic at end of range",
			r:    SuppressionRange{StartLine: 5, EndLine: 10},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 10, Character: 0},
					End:   protocol.Position{Line: 10, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Diagnostic in middle of range",
			r:    SuppressionRange{StartLine: 5, EndLine: 10},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 0},
					End:   protocol.Position{Line: 7, Character: 10},
				},
			},
			want: true,
		},
		{
			name: "Diagnostic before range",
			r:    SuppressionRange{StartLine: 5, EndLine: 10},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 0},
					End:   protocol.Position{Line: 3, Character: 10},
				},
			},
			want: false,
		},
		{
			name: "Diagnostic after range",
			r:    SuppressionRange{StartLine: 5, EndLine: 10},
			diagnostic: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 12, Character: 0},
					End:   protocol.Position{Line: 12, Character: 10},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diagnosticOverlapsRange(tt.r, tt.diagnostic)
			if got != tt.want {
				t.Errorf("diagnosticOverlapsRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterSuppressedDiagnostics(t *testing.T) {
	tests := []struct {
		name            string
		diagnostics     []protocol.Diagnostic
		suppressionInfo *SuppressionInfo
		wantCount       int
	}{
		{
			name: "Filter out file-wide suppressed diagnostics",
			diagnostics: []protocol.Diagnostic{
				{Range: protocol.Range{Start: protocol.Position{Line: 1}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 2}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 3}}},
			},
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: true,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges:    []SuppressionRange{},
			},
			wantCount: 0,
		},
		{
			name: "Filter out line-specific suppressions",
			diagnostics: []protocol.Diagnostic{
				{Range: protocol.Range{Start: protocol.Position{Line: 1}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 2}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 3}}},
			},
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{1: true, 3: true},
				SuppressedRanges:    []SuppressionRange{},
			},
			wantCount: 1,
		},
		{
			name: "Filter out range suppressions",
			diagnostics: []protocol.Diagnostic{
				{Range: protocol.Range{Start: protocol.Position{Line: 1}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 5}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 7}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 15}}},
			},
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges: []SuppressionRange{
					{StartLine: 5, EndLine: 10},
				},
			},
			wantCount: 2,
		},
		{
			name: "No suppressions",
			diagnostics: []protocol.Diagnostic{
				{Range: protocol.Range{Start: protocol.Position{Line: 1}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 2}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 3}}},
			},
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{},
				SuppressedRanges:    []SuppressionRange{},
			},
			wantCount: 3,
		},
		{
			name: "Mixed suppressions",
			diagnostics: []protocol.Diagnostic{
				{Range: protocol.Range{Start: protocol.Position{Line: 1}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 3}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 5}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 7}}},
				{Range: protocol.Range{Start: protocol.Position{Line: 15}}},
			},
			suppressionInfo: &SuppressionInfo{
				FileWideSuppression: false,
				SuppressedLines:     map[uint32]bool{3: true},
				SuppressedRanges: []SuppressionRange{
					{StartLine: 5, EndLine: 10},
				},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterSuppressedDiagnostics(tt.diagnostics, tt.suppressionInfo)
			if len(got) != tt.wantCount {
				t.Errorf("FilterSuppressedDiagnostics() returned %v diagnostics, want %v", len(got), tt.wantCount)
			}
		})
	}
}
