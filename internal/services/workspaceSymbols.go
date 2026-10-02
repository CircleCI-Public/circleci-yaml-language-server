package languageservice

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/services/documentSymbols"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// WorkspaceSymbols are the elements the documents declare, such as their
// jobs, commands and executors, whose names contain the query, in any case.
// Each is in the section that declares it, such as Jobs. An open document is
// read as it is in the editor, and any other from disk; one that can't be
// read is left out.
func WorkspaceSymbols(ctx context.Context, query string, documents []uri.URI, cache *cache.Cache, context *session.Settings) []protocol.SymbolInformation {
	query = strings.ToLower(query)
	symbols := []protocol.SymbolInformation{}

	for _, document := range documents {
		yamlDocument, err := yamlparser.ParseFromUriWithCache(ctx, document, cache, context)
		if errors.Is(err, yamlparser.ErrCacheMissing) {
			yamlDocument, err = yamlparser.ParseFromURI(ctx, document, context)
		}
		if err != nil {
			continue
		}

		for _, section := range documentSymbols.SymbolsForDocument(&yamlDocument) {
			for _, element := range section.Children {
				if !strings.Contains(strings.ToLower(element.Name), query) {
					continue
				}
				symbols = append(symbols, protocol.SymbolInformation{
					BaseSymbolInformation: protocol.BaseSymbolInformation{
						Name:          element.Name,
						Kind:          element.Kind,
						ContainerName: &section.Name,
					},
					Location: protocol.Location{URI: document, Range: element.Range},
				})
			}
		}
		yamlDocument.Close()
	}

	slices.SortFunc(symbols, func(a, b protocol.SymbolInformation) int {
		return cmp.Or(
			strings.Compare(string(a.Location.URI), string(b.Location.URI)),
			cmp.Compare(a.Location.Range.Start.Line, b.Location.Range.Start.Line),
			cmp.Compare(a.Location.Range.Start.Character, b.Location.Range.Start.Character),
		)
	})
	return symbols
}
