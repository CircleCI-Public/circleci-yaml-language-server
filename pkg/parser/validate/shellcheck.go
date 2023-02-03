package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type ShellCheckReplacement struct {
	Precedence     int
	Line           int
	EndLine        int
	Column         int
	EndColumn      int
	InsertionPoint string
	Replacement    string
}

type ShellCheckResult struct {
	Comments []struct {
		File      string
		Line      int
		EndLine   int
		Column    int
		EndColumn int
		Level     string
		Code      int
		Message   string
		Fix       struct {
			Replacements []ShellCheckReplacement
		}
	}
}

func getSeverity(level string) protocol.DiagnosticSeverity {
	switch level {
	case "error":
		return protocol.DiagnosticSeverityError
	case "warning":
		return protocol.DiagnosticSeverityWarning
	case "info":
		return protocol.DiagnosticSeverityInformation
	case "style":
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityError
	}
}

func getShellCheckBinary() (string, error) {
	dir, err := os.Executable()
	if err != nil {
		return "", err
	}

	dir = path.Dir(dir)
	if os.Getenv("CCI_DEV") == "true" {
		dir = path.Join(dir, "..")
	}

	currentOs := runtime.GOOS + "/" + runtime.GOARCH
	switch currentOs {
	case "linux/amd64":
		return path.Join(dir, "shellcheck", "shellcheck-linux.x86_64"), nil
	case "linux/arm64":
		return path.Join(dir, "shellcheck", "shellcheck-linux.aarch64"), nil
	case "darwin/amd64":
		return path.Join(dir, "shellcheck", "shellcheck-darwin.x86_64"), nil
	case "darwin/arm64":
		return path.Join(dir, "shellcheck", "shellcheck-darwin.x86_64"), nil
	case "windows/amd64":
		return path.Join(dir, "shellcheck", "shellcheck-windows.exe"), nil
	}

	return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

func (val Validate) shellCheck(step ast.Run) {
	if step.Shell != "" && step.Shell != "/bin/bash" {
		return
	}

	if hasParam, _ := utils.CheckIfParamIsPartiallyReferenced(step.RawCommand); hasParam {
		return
	}

	binary, err := getShellCheckBinary()
	if err != nil {
		return
	}
	subCmd := exec.Command(binary, "-s", "bash", "-f", "json1", "-")

	stdin, err := subCmd.StdinPipe()
	stdout, err2 := subCmd.StdoutPipe()
	if err != nil || err2 != nil {
		return
	}

	if err := subCmd.Start(); err != nil {
		return
	}

	formattedCommand, lineRemoved, characterRemoved := val.Doc.GetCommandTextForShellCheck(step.RawCommand)
	io.WriteString(stdin, "#!/bin/sh\n"+formattedCommand+"\n")
	stdin.Close()
	res, err := io.ReadAll(stdout)
	if err != nil {
		return
	}

	if len(res) == 0 {
		return
	}

	var shellCheckResult ShellCheckResult
	err = json.Unmarshal(res, &shellCheckResult)
	if err != nil {
		return
	}

	for _, result := range shellCheckResult.Comments {
		var start protocol.Position
		var end protocol.Position

		if result.Line > 0 {
			result.Line -= 1
		}
		if result.EndLine > 0 {
			result.EndLine -= 1
		}
		if result.Column > 0 {
			result.Column -= 1
		}
		if result.EndColumn > 0 {
			result.EndColumn -= 1
		}

		if lineRemoved != 0 || characterRemoved != 0 {
			var startCharacter uint32
			var endCharacter uint32

			if lineRemoved > 0 {
				startCharacter = uint32(result.Column) + uint32(characterRemoved)
				endCharacter = uint32(result.EndColumn) + uint32(characterRemoved)
			} else {
				startCharacter = step.CommandRange.Start.Character + uint32(result.Column) + uint32(characterRemoved)
				endCharacter = step.CommandRange.Start.Character + uint32(result.EndColumn) + uint32(characterRemoved)
			}

			start = protocol.Position{
				Line:      step.CommandRange.Start.Line + uint32(result.Line) + uint32(lineRemoved) - 1,
				Character: startCharacter,
			}
			end = protocol.Position{
				Line:      step.CommandRange.Start.Line + uint32(result.EndLine) + uint32(lineRemoved) - 1,
				Character: endCharacter,
			}
		} else {
			start = protocol.Position{
				Line:      step.CommandRange.Start.Line + uint32(result.Line) - 1,
				Character: step.CommandRange.Start.Character + uint32((result.Column)),
			}
			end = protocol.Position{
				Line:      step.CommandRange.Start.Line + uint32(result.EndLine) - 1,
				Character: step.CommandRange.Start.Character + uint32((result.EndColumn)),
			}
		}

		val.addDiagnostic(protocol.Diagnostic{
			Range: protocol.Range{
				Start: start,
				End:   end,
			},
			Message:  result.Message,
			Severity: getSeverity(result.Level),
			CodeDescription: &protocol.CodeDescription{
				Href: uri.URI(fmt.Sprintf("https://www.shellcheck.net/wiki/SC%d", result.Code)),
			},
			Code: fmt.Sprintf("SC%d", result.Code),
		})
	}
}
