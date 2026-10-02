package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestParseMirrors(t *testing.T) {
	list := "https://pkg.example.org/zig\n\n# retired\nhttps://zig.example.net/\n"

	mirrors := parseMirrors(list)

	assert.Check(t, cmp.DeepEqual(mirrors, []string{"https://pkg.example.org/zig", "https://zig.example.net"}))
}

func TestFetch(t *testing.T) {
	const file = "zig-x86_64-linux-" + version + ".tar.xz"
	archive := []byte("an archive")
	sum := sha256.Sum256(archive)
	want := hex.EncodeToString(sum[:])

	serving := func(t *testing.T, body []byte) string {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+file || r.URL.Query().Get("source") != source {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(body)
		}))
		t.Cleanup(server.Close)
		return server.URL
	}

	t.Run("the first source that serves the archive is used", func(t *testing.T) {
		dir := t.TempDir()
		wrong := serving(t, []byte("something else"))
		right := serving(t, archive)

		path, err := fetch(t.Context(), dir, file, want, []string{wrong, right})
		assert.NilError(t, err)

		got, err := os.ReadFile(path)
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(got, archive))
	})

	t.Run("it fails when no source does", func(t *testing.T) {
		dir := t.TempDir()
		wrong := serving(t, []byte("something else"))

		_, err := fetch(t.Context(), dir, file, want, []string{wrong})
		assert.Check(t, cmp.ErrorContains(err, "no source served "+file))

		leftover, err := os.ReadDir(dir)
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(leftover, 0), "a failed download leaves nothing behind")
	})
}
