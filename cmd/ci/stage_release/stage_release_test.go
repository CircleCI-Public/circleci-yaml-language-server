package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
)

// The shape goreleaser 2.16 writes, trimmed to one target.
const artifactsJSON = `[
  {"name": "metadata.json", "path": "dist/metadata.json", "type": "Metadata"},
  {"name": "circleci-yaml-lsp", "path": "dist/b_linux_amd64_v1/circleci-yaml-lsp", "type": "Binary",
   "extra": {"ID": "circleci-yaml-language-server"}},
  {"name": "linux-amd64-lsp", "path": "dist/b_linux_amd64_v1/circleci-yaml-lsp", "type": "Binary",
   "extra": {"ID": "binary", "Format": "binary"}},
  {"name": "p_1.2.3_linux_amd64.tar.gz", "path": "dist/p_1.2.3_linux_amd64.tar.gz", "type": "Archive",
   "extra": {"ID": "archive", "Format": "tar.gz"}},
  {"name": "checksums.txt", "path": "dist/checksums.txt", "type": "Checksum"}
]`

func TestStage(t *testing.T) {
	root := fs.NewDir(t, "repo",
		fs.WithFile("schema.json", "{}"),
		fs.WithDir("dist",
			fs.WithFile("artifacts.json", artifactsJSON),
			fs.WithFile("metadata.json", "{}"),
			fs.WithFile("checksums.txt", "sums"),
			fs.WithFile("p_1.2.3_linux_amd64.tar.gz", "archive"),
			fs.WithDir("b_linux_amd64_v1",
				fs.WithFile("circleci-yaml-lsp", "binary", fs.WithMode(0o755)),
			),
		),
	)
	t.Chdir(root.Path())

	assert.NilError(t, stage("dist", filepath.Join("dist", "release"), "schema.json"))

	t.Run("stages the release under its published names", func(t *testing.T) {
		entries, err := os.ReadDir(filepath.Join("dist", "release"))
		assert.NilError(t, err)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		assert.Check(t, cmp.DeepEqual(names, []string{
			"checksums.txt",
			"linux-amd64-lsp",
			"p_1.2.3_linux_amd64.tar.gz",
			"schema.json",
		}))
	})

	t.Run("keeps the binary executable", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows has no executable bit")
		}
		info, err := os.Stat(filepath.Join("dist", "release", "linux-amd64-lsp"))
		assert.NilError(t, err)
		assert.Check(t, info.Mode().Perm()&0o100 != 0, "mode %v", info.Mode())
	})
}
