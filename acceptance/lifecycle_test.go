package acceptance

import (
	"context"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/lspclient"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/runner"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/workspace"
)

// exitTimeout bounds how long a server may take to end once it is told to.
const exitTimeout = 10 * time.Second

// orbSources lists the directories the server has written orb sources to, in
// the temporary directory it was given.
func orbSources(t *testing.T, server *runner.Server) []string {
	t.Helper()

	dirs, err := filepath.Glob(filepath.Join(server.TempDir(), "circleci-yaml-language-server-orbs-*"))
	assert.NilError(t, err)

	return dirs
}

// openOrbConfig opens a config using a remote orb, and waits for the server to
// write out the orb's source.
func openOrbConfig(t *testing.T, session *session) {
	t.Helper()

	session.open(t, orbConfig)

	eventually(t, "the orb's source to be written", func() bool {
		return len(orbSources(t, session.server)) != 0
	})
}

func TestExit(t *testing.T) {
	t.Run("after shutdown", func(t *testing.T) {
		session := start(t, linkedProjectFake(t), orbConfig, "")
		openOrbConfig(t, session)

		err := session.client.Shutdown()
		assert.NilError(t, err)
		err = session.client.Exit()
		assert.NilError(t, err)

		t.Run("ends the process", func(t *testing.T) {
			assert.Check(t, session.server.Wait(exitTimeout))
		})
		t.Run("removes the orb sources", func(t *testing.T) {
			assert.Check(t, cmp.Len(orbSources(t, session.server), 0))
		})
	})

	// A client may exit without shutting down first.
	t.Run("without shutdown", func(t *testing.T) {
		session := start(t, linkedProjectFake(t), orbConfig, "")
		openOrbConfig(t, session)

		err := session.client.Exit()
		assert.NilError(t, err)

		t.Run("ends the process", func(t *testing.T) {
			assert.Check(t, session.server.Wait(exitTimeout))
		})
		t.Run("removes the orb sources", func(t *testing.T) {
			assert.Check(t, cmp.Len(orbSources(t, session.server), 0))
		})
	})
}

// A socket server outlives any one client, so what stops it is a signal.
func TestSocketServerStopsOnSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no SIGTERM to send")
	}

	fake := linkedProjectFake(t)
	project := workspace.New(t, orbConfig)
	server := runner.StartSocket(t, serverBinary)
	client := lspclient.New(t, context.Background(), server.Stream())

	_, err := client.Initialize(project.RootURI())
	assert.NilError(t, err)
	err = client.ExecuteCommand("setSelfHostedUrl", fake.URL())
	assert.NilError(t, err)

	openOrbConfig(t, &session{client: client, server: server, workspace: project})

	err = server.Signal(syscall.SIGTERM)
	assert.NilError(t, err)

	t.Run("ends the process", func(t *testing.T) {
		assert.Check(t, server.Wait(exitTimeout))
	})
	t.Run("removes the orb sources", func(t *testing.T) {
		assert.Check(t, cmp.Len(orbSources(t, server), 0))
	})
}
