package diagnostic

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
)

// Diagnostic messages should start with a uppercase letter
func Message(message string) string {
	if len(message) == 0 {
		return message
	}

	return strings.ToUpper(message[0:1]) + message[1:]
}

// MessageText is a diagnostic's message as text, whether it was sent as a
// string or as markup.
func MessageText(d protocol.Diagnostic) string {
	switch message := d.Message.(type) {
	case protocol.String:
		return string(message)
	case *protocol.MarkupContent:
		return message.Value
	default:
		return ""
	}
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
	diagnostic.Tags = protocol.NewDiagnosticTags(protocol.DiagnosticTagDeprecated)
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
		Source:   protocol.NewOptional("cci-language-server"),
		Message:  protocol.String(msg),
		Data:     codeaction.Data(codeAction),
	}
}

func WarningFromNode(node *sitter.Node, msg string) protocol.Diagnostic {
	rng := protocol.Range{Start: position.Start(node), End: position.End(node)}

	return Warning(rng, msg)
}

func ErrorFromNode(node *sitter.Node, msg string) protocol.Diagnostic {
	rng := protocol.Range{Start: position.Start(node), End: position.End(node)}

	return Error(rng, msg)
}
