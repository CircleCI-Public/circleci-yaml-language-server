// Package acceptance tests the language server from the outside: the real
// binary, run as a subprocess, driven over the protocol a client speaks, with
// the CircleCI API it calls served by a fake in the test's own process.
//
// What it is for is the tier AGENTS.md asks for — "higher level happy path
// tests" — and the things only the shipped binary has: the flags, the two
// transports, the startup line the extension waits for, and the order the
// server does its work in once a document is opened.
package acceptance

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/compiler"
)

// serverBinary is the compiled language server every test in this package
// runs. It is built once, by TestMain.
var serverBinary string

func TestMain(m *testing.M) {
	status, err := runTests(m)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	os.Exit(status)
}

func runTests(m *testing.M) (int, error) {
	ctx := context.Background()

	binaries := compiler.NewParallel(1)
	defer binaries.Cleanup()

	binaries.Add(compiler.Work{
		Result: &serverBinary,
		Name:   "start_server",
		Target: "..",
		Source: "./cmd/start_server",
	})

	if err := binaries.Run(ctx); err != nil {
		return 0, err
	}

	return m.Run(), nil
}
