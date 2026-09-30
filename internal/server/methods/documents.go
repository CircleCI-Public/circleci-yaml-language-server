package methods

import (
	"path"
	"slices"
	"strings"

	"go.lsp.dev/uri"
)

// Serves reports whether the server answers for a document: one in a
// .circleci directory, or the source of an orb that the server wrote out
// itself for go-to-definition to open. Other YAML shares keys such as `jobs`
// and `version` with pipeline config, so it would get errors that aren't
// there if it were checked as config.
func (methods *Methods) Serves(document uri.URI) bool {
	if isOrb, _ := methods.isOrb(document); isOrb {
		return true
	}
	return inCircleCIDirectory(document)
}

// inCircleCIDirectory reports whether a document is a file with a .circleci
// directory somewhere above it, such as .circleci/config.yml, or a config a
// setup workflow continues with from .circleci/continue/.
func inCircleCIDirectory(document uri.URI) bool {
	if !document.IsFile() {
		return false
	}
	directories := strings.Split(path.Dir(document.Path()), "/")
	return slices.Contains(directories, ".circleci")
}
