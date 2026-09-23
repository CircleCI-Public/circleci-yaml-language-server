// Package compiler compiles the binaries an acceptance test runs.
//
// It is a copy of backplane-go's testing/compiler and the releases/compiler it
// wraps, reduced to what this repository needs. Copied rather than imported:
// backplane-go is not a dependency here, and taking it on for a test helper
// would bring its observability and closer packages along with it. The
// coverage instrumentation is left out, because nothing here reports coverage
// from an acceptance run.
//
// Compiling the real binary is the point: an acceptance test that runs what
// ships covers the flag parsing, the transport and the startup handshake a
// client depends on, none of which an in-process harness would touch.
package compiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"
)

// Work is one binary to compile.
type Work struct {
	// Name is the name of the binary produced.
	Name string
	// Target is the directory to compile from, relative to the test, e.g. "..".
	Target string
	// Source is the Go package to compile, e.g. "./cmd/start_server".
	Source string
	// Tags are extra build tags.
	Tags string
	// Environment is extra environment for the compiler, e.g. "GOOS=linux".
	Environment []string

	// Result is where the path of the compiled binary is written.
	Result *string
}

// Parallel compiles work concurrently into one temporary directory.
type Parallel struct {
	dir         string
	parallelism int
	work        chan Work
}

// NewParallel returns a compiler that builds into a temporary directory, at
// most parallelism binaries at a time.
func NewParallel(parallelism int) *Parallel {
	dir, err := os.MkdirTemp("", "acceptance-binaries")
	if err != nil {
		panic(err)
	}

	if parallelism <= 0 {
		parallelism = 2
	}

	return &Parallel{
		dir:         dir,
		parallelism: parallelism,
		work:        make(chan Work, 100),
	}
}

// Dir is the directory the binaries are compiled into.
func (p *Parallel) Dir() string {
	return p.dir
}

// Add queues work. It panics on work that cannot be compiled, because a test
// binary that is misconfigured has nothing useful to report later.
func (p *Parallel) Add(work Work) {
	switch {
	case work.Name == "":
		panic("compiler: work.Name not set")
	case work.Target == "":
		panic("compiler: work.Target not set")
	case work.Source == "":
		panic("compiler: work.Source not set")
	}

	p.work <- work
}

// Run compiles everything queued, reporting the first failure.
func (p *Parallel) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	for range p.parallelism {
		group.Go(func() error {
			for {
				select {
				case work := <-p.work:
					if err := p.compile(ctx, work); err != nil {
						return err
					}
				default:
					return nil
				}
			}
		})
	}

	return group.Wait()
}

// Cleanup removes the compiled binaries. Call it once no more work will be
// added, after the tests that use the binaries have finished.
func (p *Parallel) Cleanup() {
	close(p.work)
	_ = os.RemoveAll(p.dir)
}

func (p *Parallel) compile(ctx context.Context, work Work) error {
	target, err := filepath.Abs(work.Target)
	if err != nil {
		return err
	}

	goos := runtime.GOOS
	for _, environment := range work.Environment {
		if name, value, found := strings.Cut(environment, "="); found && name == "GOOS" {
			goos = value
		}
	}

	path := filepath.Join(p.dir, work.Name)
	if goos == "windows" {
		path += ".exe"
	}

	args := []string{"build", "-ldflags=-w -s", "-o", path}
	if work.Tags != "" {
		args = append(args, "-tags", work.Tags)
	}
	args = append(args, work.Source)

	cmd := exec.CommandContext(ctx, goBinary(), args...)
	cmd.Dir = target
	// Unlike the backplane services this was copied from, the language server
	// cannot be built without cgo: its YAML parser is tree-sitter. So the
	// environment is inherited rather than pinned to CGO_ENABLED=0.
	cmd.Env = append(os.Environ(), work.Environment...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compiling %s: %w", work.Source, err)
	}

	if work.Result != nil {
		*work.Result = path
	}

	return nil
}

func goBinary() string {
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		return filepath.Join(goroot, "bin", "go")
	}

	return "go"
}
