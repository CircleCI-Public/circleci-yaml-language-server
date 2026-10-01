package definition

import (
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Link is a definition, and the text that it was found from.
type Link struct {
	// Origin is the text at the position asked about that names the
	// definition, such as a job's name in a workflow.
	Origin protocol.Range
	URI    uri.URI
	// Range is the whole definition, and NameRange its name. NameRange is
	// empty for a definition with no name of its own, such as an orb's file.
	Range     protocol.Range
	NameRange protocol.Range
}

// Location is where the definition is, without its origin or name.
func (link Link) Location() protocol.Location {
	return protocol.Location{URI: link.URI, Range: link.Range}
}

// DefinitionLink is the link in the form a client that supports links reads.
func (link Link) DefinitionLink() protocol.DefinitionLink {
	selection := link.NameRange
	if selection == (protocol.Range{}) {
		selection = link.Range
	}
	return protocol.DefinitionLink{
		OriginSelectionRange: &link.Origin,
		TargetURI:            link.URI,
		TargetRange:          link.Range,
		TargetSelectionRange: selection,
	}
}

// from gives links the origin that they were found from.
func from(origin protocol.Range, links []Link) []Link {
	for i := range links {
		links[i].Origin = origin
	}
	return links
}
