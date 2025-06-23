package ast

import (
	"strconv"

	"go.lsp.dev/protocol"
)

type RetentionSettings struct {
	Caches TextAndRange
	Range  protocol.Range
}

func (r *RetentionSettings) ValidateCachesDuration() bool {
	if r.Caches.Text == "" {
		return true
	}

	if len(r.Caches.Text) < 2 || r.Caches.Text[len(r.Caches.Text)-1] != 'd' {
		return false
	}

	durationStr := r.Caches.Text[:len(r.Caches.Text)-1]

	duration, err := strconv.Atoi(durationStr)
	if err != nil {
		return false
	}

	return duration >= 1 && duration <= 15
}

func (r *RetentionSettings) ValidateCaches() []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	if r.Caches.Text != "" && !r.ValidateCachesDuration() {
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range:    r.Caches.Range,
			Message:  "Retention caches duration must be between 1d and 15d",
			Severity: protocol.DiagnosticSeverityError,
		})
	}

	return diagnostics
}
