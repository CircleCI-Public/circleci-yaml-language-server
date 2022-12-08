package utils

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func CreateErrorDiagnosticFromRange(rng protocol.Range, msg string) protocol.Diagnostic {
	return CreateDiagnosticFromRange(
		rng,
		protocol.DiagnosticSeverityError,
		msg,
		[]protocol.CodeAction{},
	)
}

func CreateWarningDiagnosticFromRange(rng protocol.Range, msg string) protocol.Diagnostic {
	return CreateDiagnosticFromRange(
		rng,
		protocol.DiagnosticSeverityWarning,
		msg,
		[]protocol.CodeAction{},
	)
}

func CreateEmptyAssignationWarning(rng protocol.Range) protocol.Diagnostic {
	return CreateWarningDiagnosticFromRange(rng, "Empty assignation")
}

func CreateInformationDiagnosticFromRange(rng protocol.Range, msg string) protocol.Diagnostic {
	return CreateDiagnosticFromRange(
		rng,
		protocol.DiagnosticSeverityInformation,
		msg,
		[]protocol.CodeAction{},
	)
}

func CreateHintDiagnosticFromRange(rng protocol.Range, msg string) protocol.Diagnostic {
	return CreateDiagnosticFromRange(
		rng,
		protocol.DiagnosticSeverityHint,
		msg,
		[]protocol.CodeAction{},
	)
}

func CreateDiagnosticFromRange(
	rng protocol.Range,
	severity protocol.DiagnosticSeverity,
	msg string,
	codeAction []protocol.CodeAction,
) protocol.Diagnostic {
	return protocol.Diagnostic{
		Range:    rng,
		Severity: severity,
		Source:   "cci-language-server",
		Message:  msg,
		Data:     codeAction,
	}
}

func CreateWarningDiagnosticFromNode(node *sitter.Node, msg string) protocol.Diagnostic {
	start, end := node.StartPoint(), node.EndPoint()
	rng := protocol.Range{
		Start: protocol.Position{Line: start.Row, Character: start.Column},
		End:   protocol.Position{Line: end.Row, Character: end.Column},
	}

	return CreateWarningDiagnosticFromRange(rng, msg)
}

func CreateErrorDiagnosticFromNode(node *sitter.Node, msg string) protocol.Diagnostic {
	start, end := node.StartPoint(), node.EndPoint()
	rng := protocol.Range{
		Start: protocol.Position{Line: start.Row, Character: start.Column},
		End:   protocol.Position{Line: end.Row, Character: end.Column},
	}

	return CreateErrorDiagnosticFromRange(rng, msg)
}
