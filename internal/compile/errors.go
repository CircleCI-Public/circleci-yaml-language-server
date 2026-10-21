package compile

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/ir"
)

// Error is a compile error, where it happened, and the calls that led there,
// innermost first.
type Error struct {
	Message string
	Loc     ir.Loc
	Frames  []ir.Frame
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)
	if e.Loc != (ir.Loc{}) {
		fmt.Fprintf(&b, "\n  at %s", formatLoc(e.Loc))
	}
	for _, f := range e.Frames {
		fmt.Fprintf(&b, "\n  in %s '%s', called from %s", f.Kind, f.Name, formatLoc(f.Loc))
	}
	return b.String()
}

func formatLoc(l ir.Loc) string {
	return fmt.Sprintf("%s:%d:%d", l.Source, l.Line, l.Col)
}

// At says where evaluation is, for the errors it gives.
type At struct {
	Loc    ir.Loc
	Frames []ir.Frame
}

func (a At) errorf(format string, args ...any) *Error {
	return &Error{Message: fmt.Sprintf(format, args...), Loc: a.Loc, Frames: a.Frames}
}
