package logging_test

import (
	"bytes"
	"errors"
	"log"
	"log/slog"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/logging"
)

func TestNew(t *testing.T) {
	t.Run("writes a message with its attributes", func(t *testing.T) {
		var out bytes.Buffer
		logger := logging.New(&out, false)

		logger.Warn("listing contexts", "err", errors.New("500 Internal Server Error"))

		assert.Check(t, cmp.Contains(out.String(), "WARN"))
		assert.Check(t, cmp.Contains(out.String(), "listing contexts"))
		assert.Check(t, cmp.Contains(out.String(), "err="))
		assert.Check(t, cmp.Contains(out.String(), "500 Internal Server Error"))
	})

	t.Run("leaves out debug messages by default", func(t *testing.T) {
		var out bytes.Buffer
		logger := logging.New(&out, false)

		logger.Debug("called method", "method", "initialize")

		assert.Check(t, cmp.Equal(out.String(), ""))
	})

	t.Run("writes debug messages when asked to", func(t *testing.T) {
		var out bytes.Buffer
		logger := logging.New(&out, true)

		logger.Debug("called method", "method", "initialize")

		assert.Check(t, cmp.Contains(out.String(), "DEBU"))
		assert.Check(t, cmp.Contains(out.String(), "called method"))
		assert.Check(t, cmp.Contains(out.String(), "initialize"))
	})
}

// A dependency that logs through the standard library's log package ends up
// in the same place, in the same format, once a logger is the default.
func TestDefaultTakesTheLogPackage(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })

	var out bytes.Buffer
	slog.SetDefault(logging.New(&out, false))

	log.Printf("from %s", "a dependency")

	assert.Check(t, cmp.Contains(out.String(), "INFO"))
	assert.Check(t, cmp.Contains(out.String(), "from a dependency"))
}
