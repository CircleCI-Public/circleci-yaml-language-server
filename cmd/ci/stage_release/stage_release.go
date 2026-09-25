// Command stage_release copies what a release publishes out of goreleaser's
// dist/ into one flat directory, under the names it is published as.
//
// Goreleaser only gives the bare binaries their release names
// ({os}-{arch}-lsp[.exe]) when it uploads them itself, which this project
// doesn't have it do, so their names come from dist/artifacts.json. The
// archives and checksums.txt are copied beside them, with schema.json, which
// editors download on its own.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// artifact is the part of a dist/artifacts.json entry this command reads.
type artifact struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Type  string `json:"type"`
	Extra struct {
		Format string `json:"Format"`
	} `json:"extra"`
}

func main() {
	dist := flag.String("dist", "dist", "goreleaser's dist directory")
	out := flag.String("out", filepath.Join("dist", "release"), "directory to stage the release in")
	schema := flag.String("schema", "schema.json", "JSON schema to publish beside the binaries")
	flag.Parse()

	if err := stage(*dist, *out, *schema); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "stage_release:", err)
		os.Exit(1)
	}
}

func stage(dist, out, schema string) error {
	data, err := os.ReadFile(filepath.Join(dist, "artifacts.json"))
	if err != nil {
		return err
	}
	var artifacts []artifact
	if err := json.Unmarshal(data, &artifacts); err != nil {
		return fmt.Errorf("reading artifacts.json: %w", err)
	}

	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	for _, a := range published(artifacts) {
		// Paths in artifacts.json start with the dist directory's name, as
		// goreleaser was configured with it, so they resolve from the root.
		if err := copyFile(a.Path, filepath.Join(out, a.Name)); err != nil {
			return err
		}
		fmt.Println(a.Name)
	}

	if err := copyFile(schema, filepath.Join(out, filepath.Base(schema))); err != nil {
		return err
	}
	fmt.Println(filepath.Base(schema))
	return nil
}

// published picks the artifacts a release uploads: the archives, the
// checksums, and the bare binaries from the `formats: [binary]` archive. The
// plain build outputs are the same binaries under the build's name, so they
// are left out.
func published(artifacts []artifact) []artifact {
	var picked []artifact
	for _, a := range artifacts {
		switch {
		case a.Type == "Archive", a.Type == "Checksum":
		case a.Type == "Binary" && a.Extra.Format == "binary":
		default:
			continue
		}
		picked = append(picked, a)
	}
	return picked
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	outFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(outFile, in); err != nil {
		_ = outFile.Close()
		return err
	}
	return outFile.Close()
}
