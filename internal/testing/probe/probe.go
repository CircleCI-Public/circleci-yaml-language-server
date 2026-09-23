// Package probe reports whether a real API still looks the way this
// repository's fakes pretend it does.
//
// A probe is not a test. It talks to the live service, so it cannot run on a
// pull request, and a failure is as likely to mean "the service is having a bad
// day" as "our fake is wrong". Those two are reported separately — drift exits
// 1, an unreachable service exits 2 — so that whatever runs a probe on a
// schedule can raise the first and shrug at the second.
//
// A probe also prints what it found, because the tools these replaced were
// debugging aids and that is still the quickest way to see what the language
// server sees.
package probe

import (
	"errors"
	"fmt"
	"io"
	"time"
)

// Exit codes a probe reports. They are its whole interface to whatever runs
// it, so they are part of its contract.
const (
	// ExitOK means every check passed: the API looks as expected.
	ExitOK = 0
	// ExitDrift means the API no longer looks the way this repository relies
	// on it looking. Someone has to look: either the service changed, or our
	// idea of it was wrong.
	ExitDrift = 1
	// ExitUnavailable means the probe could not find out — the service was
	// unreachable, the credentials it needed were missing, or it took so long
	// that waiting was pointless. Nothing was learned, and nobody needs
	// waking up.
	ExitUnavailable = 2
)

// ErrUnavailable marks an error as "could not probe" rather than "the contract
// drifted". Wrap an error with Unavailable to report it that way.
var ErrUnavailable = errors.New("could not reach the API")

// Unavailable marks err as a failure to reach the API.
func Unavailable(err error) error {
	return fmt.Errorf("%w: %w", ErrUnavailable, err)
}

// Probe collects the results of one service's checks.
type Probe struct {
	name string
	out  io.Writer

	passed      int
	drifted     []string
	unavailable error
}

// New returns a probe of one service, writing its report to out.
func New(name string, out io.Writer) *Probe {
	probe := &Probe{name: name, out: out}
	probe.printf("probe: %s\n", name)

	return probe
}

// Note records something the probe found, for whoever reads the output. It is
// not a check and cannot fail.
func (p *Probe) Note(format string, args ...any) {
	p.printf("  - "+format+"\n", args...)
}

// Check runs one expectation of the API.
//
// A check that returns an error wrapping ErrUnavailable ends the probe: the
// rest would fail for the same reason, and reporting them as drift would be a
// lie.
func (p *Probe) Check(what string, check func() error) {
	if p.unavailable != nil {
		p.printf("  skip  %s\n", what)

		return
	}

	err := check()

	switch {
	case err == nil:
		p.passed++
		p.printf("  ok    %s\n", what)
	case errors.Is(err, ErrUnavailable):
		p.unavailable = err
		p.printf("  ????  %s: %v\n", what, err)
	default:
		p.drifted = append(p.drifted, what)
		p.printf("  DRIFT %s: %v\n", what, err)
	}
}

// Report prints the summary and returns the exit code the caller should use.
func (p *Probe) Report() int {
	switch {
	case p.unavailable != nil:
		p.printf("%s: could not be probed: %v\n", p.name, p.unavailable)

		return ExitUnavailable

	case len(p.drifted) > 0:
		p.printf("%s: %d drifted, %d passed\n", p.name, len(p.drifted), p.passed)
		for _, what := range p.drifted {
			p.printf("  drifted: %s\n", what)
		}

		return ExitDrift

	default:
		p.printf("%s: %d checks passed\n", p.name, p.passed)

		return ExitOK
	}
}

// Within bounds a whole probe by the clock, reporting it unavailable if it
// overruns.
//
// It exists because not every client this probes can be given a deadline: the
// Docker Hub cursor loops until it is satisfied, and a response it cannot
// decode makes it loop forever. A probe that hangs tells a schedule nothing,
// so it is better to give up and say so.
func Within(timeout time.Duration, checks func() int) int {
	code := make(chan int, 1)

	go func() {
		code <- checks()
	}()

	select {
	case reported := <-code:
		return reported
	case <-time.After(timeout):
		fmt.Printf("probe gave up after %s\n", timeout)

		return ExitUnavailable
	}
}

func (p *Probe) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(p.out, format, args...)
}
