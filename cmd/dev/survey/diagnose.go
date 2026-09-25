package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/orburl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// row is one diagnostic, as diagnose writes it and compare reads it.
type row struct {
	File     string `json:"file"`
	Line     uint32 `json:"line"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	// Text is the line the diagnostic starts on, to read a result without
	// opening the file.
	Text string `json:"text"`
}

func diagnose(args []string) error {
	dir := defaultDir
	if len(args) > 0 {
		dir = args[0]
	}

	settings := &session.Settings{
		Api: circleci.Config{
			HostUrl: circleci.DefaultHostURL,
			Token:   os.Getenv("CIRCLE_TOKEN"),
		},
		OrbURLs: orburl.Config{GitHubToken: gitHubToken()},
	}
	c := cache.New()
	defer c.Close()
	enc := json.NewEncoder(os.Stdout)

	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !isYAML(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		fileURI := uri.File(abs)
		c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: fileURI, Text: string(content)}})

		diags, err := languageservice.DiagnosticFile(fileURI, c, settings, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, path, err)
		}

		lines := strings.Split(string(content), "\n")
		for _, d := range diags {
			text := ""
			if int(d.Range.Start.Line) < len(lines) {
				text = strings.TrimSpace(lines[d.Range.Start.Line])
			}
			if err := enc.Encode(row{path, d.Range.Start.Line + 1, int(d.Severity), diagnostic.MessageText(d), text}); err != nil {
				return err
			}
		}
		return nil
	})
}
