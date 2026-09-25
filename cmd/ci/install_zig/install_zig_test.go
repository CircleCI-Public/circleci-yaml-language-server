package main

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestZigHost(t *testing.T) {
	t.Run("every release build host has a checksum", func(t *testing.T) {
		for _, p := range [][2]string{{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "arm64"}, {"windows", "amd64"}} {
			_, err := zigHost(p[0], p[1])
			assert.Check(t, err, "%s/%s", p[0], p[1])
		}
	})

	t.Run("an unpinned host is an error", func(t *testing.T) {
		_, err := zigHost("windows", "arm64")
		assert.Check(t, cmp.ErrorContains(err, "no Zig 0.16.0 pinned for windows/arm64"))
	})
}

func TestWithin(t *testing.T) {
	dir := t.TempDir()

	t.Run("a nested entry is allowed", func(t *testing.T) {
		got, err := within(dir, "zig/lib/std.zig")
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(got, filepath.Join(dir, "zig", "lib", "std.zig")))
	})

	t.Run("an entry that escapes is refused", func(t *testing.T) {
		for _, name := range []string{"../evil", "zig/../../evil", "/etc/passwd", "."} {
			_, err := within(dir, name)
			assert.Check(t, cmp.ErrorContains(err, "escapes"), "entry %q", name)
		}
	})
}
