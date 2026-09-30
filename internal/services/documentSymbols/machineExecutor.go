package documentSymbols

import (
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// machineExecutorSymbols is at the machine's image, or, for `machine: true`,
// which has none, at given, where the machine is given.
func machineExecutorSymbols(machineExec ast.MachineExecutor, given protocol.Range) protocol.DocumentSymbol {
	splits := strings.Split(machineExec.Image, ":")

	machineName := ""
	machineVersion := ""
	deprecated := false

	if machineExec.Machine {
		// There is no image when using machine: true
		// set the name & version to different values to reflect this
		machineName = "default machine"
		machineVersion = "[deprecated]"
		deprecated = true
	} else {
		machineName = splits[0]

		if len(splits) > 1 {
			machineVersion = splits[1]
		}
	}

	rng := machineExec.ImageRange
	if position.IsDefaultRange(rng) {
		rng = given
	}

	symbol := protocol.DocumentSymbol{
		Name:           machineName,
		Range:          rng,
		SelectionRange: rng,
		Detail:         unlessZero(machineVersion),
		Kind:           DockerSymbol,
		Deprecated:     unlessZero(deprecated), //nolint:staticcheck
	}

	return symbol
}
