package parser

import (
	"regexp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type SuppressionInfo struct {
	FileWideSuppression bool
	SuppressedLines     map[uint32]bool
	SuppressedRanges    []SuppressionRange
}

type SuppressionRange struct {
	StartLine uint32
	EndLine   uint32
}

var (
	ignoreFileRegex       = regexp.MustCompile(`^\s*#\s*cci-ignore-file\s*$`)
	ignoreInlineRegex     = regexp.MustCompile(`#\s*cci-ignore\s*$`)
	ignoreNextLineRegex   = regexp.MustCompile(`^\s*#\s*cci-ignore-next-line\s*$`)
	ignoreRangeStartRegex = regexp.MustCompile(`^\s*#\s*cci-ignore-start\s*$`)
	ignoreRangeEndRegex   = regexp.MustCompile(`^\s*#\s*cci-ignore-end\s*$`)
)

func ParseSuppressionComments(doc *YamlDocument) *SuppressionInfo {
	rootNode := doc.RootNode
	suppressionInfo := &SuppressionInfo{
		FileWideSuppression: false,
		SuppressedLines:     make(map[uint32]bool),
		SuppressedRanges:    []SuppressionRange{},
	}

	isRangeOpen := false
	suppressionRange := SuppressionRange{}

	// fetch all comments via tree-sitter and build up the suppression info
	ExecQuery(rootNode, "(comment) @comment", func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := capture.Node
			commentText := doc.GetNodeText(node)

			// cci-ignore-file
			if ignoreFileRegex.MatchString(commentText) {
				suppressionInfo.FileWideSuppression = true
				return
			}

			// cci-ignore
			if ignoreInlineRegex.MatchString(commentText) {
				line := doc.NodeToRange(node).Start.Line
				suppressionInfo.SuppressedLines[line] = true
			}

			// cci-ignore-next-line
			if ignoreNextLineRegex.MatchString(commentText) {
				line := doc.NodeToRange(node).Start.Line
				suppressionInfo.SuppressedLines[line+1] = true
			}

			// cci-ignore-start
			if ignoreRangeStartRegex.MatchString(commentText) {
				if isRangeOpen {
					diagnostic := utils.CreateErrorDiagnosticFromNode(node, "cci-ignore-start must have a closing cci-ignore-end before trying to open a new ignore-range")
					doc.addDiagnostic(diagnostic)
					return
				}
				isRangeOpen = true
				suppressionRange.StartLine = doc.NodeToRange(node).Start.Line
			}

			// cci-ignore-end
			if ignoreRangeEndRegex.MatchString(commentText) {
				if !isRangeOpen {
					diagnostic := utils.CreateErrorDiagnosticFromNode(node, "cci-ignore-end must have an opening cci-ignore-start")
					doc.addDiagnostic(diagnostic)
					return
				}
				isRangeOpen = false
				suppressionRange.EndLine = doc.NodeToRange(node).Start.Line
				suppressionInfo.SuppressedRanges = append(suppressionInfo.SuppressedRanges, suppressionRange)
			}
		}
	})

	// Check if a range was left open without closing
	if isRangeOpen {
		// Create diagnostic at the cci-ignore-start line
		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: suppressionRange.StartLine, Character: 0},
				End:   protocol.Position{Line: suppressionRange.StartLine, Character: 100},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "circleci",
			Message:  "cci-ignore-start is missing a closing cci-ignore-end",
		}
		doc.addDiagnostic(diagnostic)
	}

	return suppressionInfo
}

// isDiagnosticSuppressed returns true if the diagnostic is suppressed by any one of the cci-ignore comments in the file
func isDiagnosticSuppressed(suppressionInfo *SuppressionInfo, diagnostic protocol.Diagnostic) bool {
	if suppressionInfo == nil {
		return false
	}

	if suppressionInfo.FileWideSuppression {
		return true
	}

	// NOTE: for a multi-line diagnostic, having `# cci-ignore-next-line` before it would ignore the whole diagnostic
	if suppressionInfo.SuppressedLines[diagnostic.Range.Start.Line] {
		return true
	}

	for _, suppressionRange := range suppressionInfo.SuppressedRanges {
		if diagnosticOverlapsRange(suppressionRange, diagnostic) {
			return true
		}
	}

	return false
}

// diagnosticOverlapsRange returns true if the diagnostic's line range overlaps with the cci-ignore suppression range
func diagnosticOverlapsRange(r SuppressionRange, diagnostic protocol.Diagnostic) bool {
	return r.StartLine <= diagnostic.Range.Start.Line && r.EndLine >= diagnostic.Range.Start.Line
}

// FilterSuppressedDiagnostics returns a new slice of diagnostics that are not suppressed by any of the cci-ignore comments in the file
func FilterSuppressedDiagnostics(diagnostics []protocol.Diagnostic, suppression *SuppressionInfo) []protocol.Diagnostic {
	remainingDiagnostics := []protocol.Diagnostic{}

	for _, diagnostic := range diagnostics {
		if !isDiagnosticSuppressed(suppression, diagnostic) {
			remainingDiagnostics = append(remainingDiagnostics, diagnostic)
		}
	}

	return remainingDiagnostics
}
