// Package logging sets up the language server's logs: log/slog to write them,
// and charm's log to format them, the way the CircleCI CLI does.
//
// Code logs through slog's package functions — slog.Info, slog.DebugContext
// and so on — and Setup decides where that goes, once, at start-up.
package logging

import (
	"io"
	"log/slog"
	"os"

	"charm.land/log/v2"
)

// New returns a logger writing human-readable lines to w. Debug messages are
// written only when debug is set.
func New(w io.Writer, debug bool) *slog.Logger {
	level := log.InfoLevel
	if debug {
		level = log.DebugLevel
	}

	return slog.New(log.NewWithOptions(w, log.Options{
		Level: level,
		// Unlike a CLI command, a language server runs for as long as the
		// editor does, so each line says when it happened.
		ReportTimestamp: true,
	}))
}

// Setup makes a logger writing to stderr the default, for slog's package
// functions and for the standard library's log package alike. Stderr, because
// over stdio the protocol has stdout to itself.
func Setup(debug bool) {
	slog.SetDefault(New(os.Stderr, debug))
}
