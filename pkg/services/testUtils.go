package languageservice

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/expect"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

type ExpDiagInfo struct {
	content    parser.YamlDocument
	parseError error
	t          *testing.T
}

type ExpDiag struct {
	t *testing.T
}

type ExpDiagStruct struct {
	info ExpDiagInfo

	To ExpDiagTo
}

type ExpDiagTo struct {
	info ExpDiagInfo
	Not  ExpDiagToNot
	Have ExpDiagToHave
}

type ExpDiagToHave struct {
	info ExpDiagInfo
}

type ExpDiagToNot struct {
	info ExpDiagInfo

	Have ExpDiagToNotHave
}

type ExpDiagToNotHave struct {
	info ExpDiagInfo
}

func ExpectDiagnostic(t *testing.T) ExpDiag {
	return ExpDiag{
		t: t,
	}
}

// ExpectDiagnostic.File
func (root ExpDiag) File(context *utils.LsContext, uri protocol.URI) ExpDiagStruct {
	yamlDocument, err := parser.ParseFromURI(uri, context)

	return buildExDiag(root.t, yamlDocument, err)
}

// ExpectDiagnostic.String
func (root ExpDiag) String(context *utils.LsContext, content string) ExpDiagStruct {
	yamlDocument, err := parser.ParseFromContent([]byte(content), context)

	return buildExDiag(root.t, yamlDocument, err)
}

// ExpectDiagnostic.Yaml
func (root ExpDiag) Yaml(yamlDocument parser.YamlDocument) ExpDiagStruct {
	return buildExDiag(root.t, yamlDocument, nil)
}

// ExpectDiagnostic.<type>.To.Include
func (exp ExpDiagTo) Include(context *utils.LsContext, expected protocol.Diagnostic) {
	exp.info.ensureNoError()

	diagnostics, err := DiagnosticYAML(
		exp.info.content,
		utils.CreateCache(),
		context,
	)

	assert.Nil(exp.info.t, err)

	expect.DiagnosticList(exp.info.t, diagnostics).To.Include(expected)
}

// ExpectDiagnostic.<type>.To.Not.Include
func (exp ExpDiagToNot) Include(context *utils.LsContext, expected protocol.Diagnostic) {
	exp.info.ensureNoError()

	diagnostics, err := DiagnosticYAML(
		exp.info.content,
		utils.CreateCache(),
		context,
	)

	assert.Nil(exp.info.t, err)

	expect.DiagnosticList(exp.info.t, diagnostics).To.Not.Include(expected)
}

// ExpectDiagnostic.<type>.To.IncludeAll
func (exp ExpDiagTo) IncludeAll(context *utils.LsContext, expected []protocol.Diagnostic) {
	exp.info.ensureNoError()

	diagnostics, err := DiagnosticYAML(exp.info.content, utils.CreateCache(), context)

	assert.Nil(exp.info.t, err)

	expect.DiagnosticList(exp.info.t, diagnostics).To.IncludeAll(expected)
}

// ExpectDiagnostic.<type>.To.Not.IncludeAll
func (exp ExpDiagToNot) IncludeAll(context *utils.LsContext, expected []protocol.Diagnostic) {
	exp.info.ensureNoError()

	diagnostics, err := DiagnosticYAML(
		exp.info.content,
		utils.CreateCache(),
		context,
	)

	assert.Nil(exp.info.t, err)

	expect.DiagnosticList(exp.info.t, diagnostics).To.Not.IncludeAll(expected)
}

// ExpectDiagnostic.<type>.To.Have.AnyParseError()
func (exp ExpDiagToHave) AnyParseError() {
	if exp.info.parseError != nil {
		return
	}

	exp.info.t.Error("No parse error")
}

// ExpectDiagnostic.<type>.To.Not.Have.AnyParseError()
func (exp ExpDiagToNotHave) AnyParseError() {
	if exp.info.parseError == nil {
		return
	}

	exp.info.t.Errorf(
		"Parse error during validation: %s",
		exp.info.parseError.Error(),
	)
}

// ExpectDiagnostic.<type>.To.Have.ParseError()
func (exp ExpDiagToHave) ParseError(parseError string) {
	if exp.info.parseError == nil {
		exp.info.t.Error("Parse error expected")
	}

	if exp.info.parseError.Error() == parseError {
		return
	}

	message := `Invalid parse error.
Expected: %s
Actual: %s
`

	exp.info.t.Errorf(message, parseError, exp.info.parseError.Error())
}

// ExpectDiagnostic.<type>.To.Not.Have.ParseError()
func (exp ExpDiagToNotHave) ParseError(parseError string) {
	if exp.info.parseError == nil {
		return
	}

	if exp.info.parseError.Error() != parseError {
		return
	}

	message := `Invalid parse error.
Expected: %s
Should not be: %s
`

	exp.info.t.Errorf(message, parseError, exp.info.parseError.Error())
}

func Test(t *testing.T) {
}

func buildExDiag(t *testing.T, yamlDocument parser.YamlDocument, parseError error) ExpDiagStruct {
	info := ExpDiagInfo{
		content:    yamlDocument,
		parseError: parseError,
		t:          t,
	}

	return ExpDiagStruct{
		info: info,

		To: ExpDiagTo{
			info: info,

			Have: ExpDiagToHave{
				info: info,
			},

			Not: ExpDiagToNot{
				info: info,

				Have: ExpDiagToNotHave{
					info: info,
				},
			},
		},
	}
}

func (info ExpDiagInfo) ensureNoError() {
	if info.parseError == nil {
		return
	}

	info.t.Errorf(info.parseError.Error())
}
