package documentSymbols

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

func machineExecutorSymbols(machineExec ast.MachineExecutor) protocol.DocumentSymbol {
	splits := strings.Split(machineExec.Image, ":")

	machineName := ""
	machineVersion := ""

	if machineExec.IsDeprecated {
		// There is no image when using machine: true
		// set the name & version to different values to reflect this
		machineName = "default machine"
		machineVersion = "[deprecated]"
	} else {
		machineName = splits[0]

		if len(splits) > 1 {
			machineVersion = splits[1]
		}
	}

	symbol := protocol.DocumentSymbol{
		Name:           machineName,
		Range:          machineExec.Range,
		SelectionRange: machineExec.Range,
		Detail:         machineVersion,
		Kind:           protocol.SymbolKind(DockerSymbol),
		Deprecated:     true,
	}

	return symbol
}
