package validate

import "go.lsp.dev/protocol"

func (val Validate) ValidateAnchors() {

	// Searching for all unused anchors
	for _, anchor := range val.Doc.YamlAnchors {
		if len(*anchor.References) > 0 {
			continue
		}

		val.addDiagnostic(protocol.Diagnostic{
			Severity: protocol.DiagnosticSeverityInformation,
			Range:    anchor.DefinitionRange,
			Message:  "Anchor never used",
			Source:   "cci-language-server",
		})
	}
}
