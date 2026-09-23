package diagnostic

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

// Diagnostic messages should start with a uppercase letter
func Message(message string) string {
	if len(message) == 0 {
		return message
	}

	return strings.ToUpper(message[0:1]) + message[1:]
}

func Error(rng protocol.Range, msg string) protocol.Diagnostic {
	return New(
		rng,
		protocol.DiagnosticSeverityError,
		msg,
		[]protocol.CodeAction{},
	)
}

func Warning(rng protocol.Range, msg string) protocol.Diagnostic {
	return New(
		rng,
		protocol.DiagnosticSeverityWarning,
		msg,
		[]protocol.CodeAction{},
	)
}

func EmptyAssignationWarning(rng protocol.Range) protocol.Diagnostic {
	return Warning(rng, "Empty assignation")
}

func Deprecated(rng protocol.Range, msg string) protocol.Diagnostic {
	diagnostic := Warning(rng, msg)
	diagnostic.Tags = []protocol.DiagnosticTag{
		protocol.DiagnosticTagDeprecated,
	}
	return diagnostic
}

func Information(rng protocol.Range, msg string) protocol.Diagnostic {
	return New(
		rng,
		protocol.DiagnosticSeverityInformation,
		msg,
		[]protocol.CodeAction{},
	)
}

func Hint(rng protocol.Range, msg string) protocol.Diagnostic {
	return New(
		rng,
		protocol.DiagnosticSeverityHint,
		msg,
		[]protocol.CodeAction{},
	)
}

func New(
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

func WarningFromNode(node *sitter.Node, msg string) protocol.Diagnostic {
	start, end := node.StartPoint(), node.EndPoint()
	rng := protocol.Range{
		Start: protocol.Position{Line: start.Row, Character: start.Column},
		End:   protocol.Position{Line: end.Row, Character: end.Column},
	}

	return Warning(rng, msg)
}

func ErrorFromNode(node *sitter.Node, msg string) protocol.Diagnostic {
	start, end := node.StartPoint(), node.EndPoint()
	rng := protocol.Range{
		Start: protocol.Position{Line: start.Row, Character: start.Column},
		End:   protocol.Position{Line: end.Row, Character: end.Column},
	}

	return Error(rng, msg)
}
