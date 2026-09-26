package paramref

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

var paramRegex = regexp.MustCompile(`<<\s*(parameters|pipeline.parameters)\.([A-Za-z0-9-_]*)\s*>>`)

func Contains(content string) bool {
	return paramRegex.MatchString(content)
}

// Return the name of the parameter used at the given position
func NameUsedAtPos(content []byte, pos protocol.Position) (string, bool) {
	isPipelineParam := false

	posIndex := position.ToIndex(pos, content)
	if posIndex > len(content) {
		posIndex = len(content)
	}

	lineStart := bytes.LastIndex(content[:posIndex], []byte("\n"))
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++
	}

	lineEndRel := bytes.Index(content[lineStart:], []byte("\n"))
	var lineEnd int
	if lineEndRel == -1 {
		lineEnd = len(content)
	} else {
		lineEnd = lineStart + lineEndRel
	}

	if !paramRegex.Match(content[lineStart:lineEnd]) {
		return "", isPipelineParam
	}

	startOfParam := bytes.LastIndex(content[:posIndex], []byte("<<"))
	if startOfParam == -1 {
		return "", isPipelineParam
	}

	endOfParamRel := bytes.Index(content[startOfParam:], []byte(">>"))
	if endOfParamRel == -1 {
		return "", isPipelineParam
	}

	endOfParam := startOfParam + endOfParamRel + 2

	param := paramRegex.Find(content[startOfParam:endOfParam])

	// Not a parameter if the regex does not match
	if param == nil {
		return "", isPipelineParam
	}

	fullParamName, paramName := ExtractName(string(param))
	isPipelineParam = strings.HasPrefix(fullParamName, "pipeline.")

	return paramName, isPipelineParam
}

// Search the right parameters that is defined in the given position and return its name
func NameDefinedAtPos(parameters map[string]ast.Parameter, pos protocol.Position) string {
	for _, param := range parameters {
		rng := param.GetRange()
		if position.InRange(rng, pos) {
			return param.GetName()
		}
	}

	return ""
}

func ReferencesInRange(content []byte, paramName string, rng protocol.Range) ([][]int, error) {
	startIndex := position.ToIndex(rng.Start, content)
	endIndex := position.ToIndex(rng.End, content)

	paramRegex, err := regexp.Compile(fmt.Sprintf("<<\\s*(parameters|pipeline.parameters).%s\\s*>>", paramName))
	if err != nil {
		return [][]int{}, fmt.Errorf("error while compiling regex: %s", err)
	}

	allRef := paramRegex.FindAllIndex(content[startIndex:endIndex], -1)

	for i := range allRef {
		allRef[i][0] += startIndex
		allRef[i][1] += startIndex
	}

	return allRef, nil
}

var referenceRegex = regexp.MustCompile(`<<.*?>>`)

// ContainsReference reports whether content holds a `<< ... >>` reference of
// any kind, so that its value is only known once the config is compiled.
func ContainsReference(content string) bool {
	return referenceRegex.MatchString(content)
}

// CouldExpandTo reports whether content, once each of its references is
// replaced by some value, could be s.
func CouldExpandTo(content, s string) bool {
	if !ContainsReference(content) {
		return content == s
	}
	literals := referenceRegex.Split(content, -1)
	for i, literal := range literals {
		literals[i] = regexp.QuoteMeta(literal)
	}
	return regexp.MustCompile("^" + strings.Join(literals, ".*") + "$").MatchString(s)
}

var onlyParamRegex = regexp.MustCompile(`^<<\s*(parameters|pipeline.parameters)\.([A-Za-z0-9-_]*)\s*>>$`)

// Returns true if the string is *only* a parameter
// Example:
//
//	param: << parameters.paramName >> -> true
//	param: << pipeline.parameters.paramName >> -> true
//	param: `/home/<< parameters.paramName >>/Downloads` -> false
func IsOnlyParameter(content string) bool {
	return onlyParamRegex.MatchString(content)
}

var onlyReferenceRegex = regexp.MustCompile(`^<<\s*[A-Za-z][A-Za-z0-9_.-]*\s*>>$`)

// IsOnlyReference reports whether content is nothing but a single `<< ... >>`
// reference of any kind: a parameter, a pipeline value or a matrix value.
// What such a value will be is only known once the config is compiled.
func IsOnlyReference(content string) bool {
	return onlyReferenceRegex.MatchString(unquote(content))
}

var onlyPipelineValueRegex = regexp.MustCompile(`^<<\s*(pipeline\.[A-Za-z0-9_.-]+)\s*>>$`)

// OnlyPipelineValue returns the name of the pipeline value, such as
// pipeline.number, that content is nothing but a reference to. Pipeline
// parameters are declared in the config, so they are not pipeline values.
func OnlyPipelineValue(content string) (string, bool) {
	match := onlyPipelineValueRegex.FindStringSubmatch(unquote(content))
	if match == nil || strings.HasPrefix(match[1], "pipeline.parameters.") {
		return "", false
	}

	return match[1], true
}

func unquote(content string) string {
	content = strings.TrimSpace(content)
	if len(content) >= 2 && (content[0] == '"' || content[0] == '\'') && content[len(content)-1] == content[0] {
		return content[1 : len(content)-1]
	}

	return content
}

var partialParamRegex = regexp.MustCompile(`<<\s*(parameters|pipeline.parameters|pipeline.git)\.\s*>?>?`)

func IsPartiallyReferenced(content string) (bool, bool) {
	isPipelineParam := strings.Contains(content, "pipeline.")
	return partialParamRegex.Find([]byte(content)) != nil, isPipelineParam
}

var partialMatrixRegex = regexp.MustCompile(`<<\s*matrix\.\s*>?>?`)

func IsMatrixPartiallyReferenced(content string) bool {
	return partialMatrixRegex.Find([]byte(content)) != nil
}

var paramInStringRegex = regexp.MustCompile(`<<\s*(parameters|pipeline.parameters)\.([A-Za-z0-9-_]*)\s*>>`)

func InString(content string) ([]struct {
	Name       string
	FullName   string
	ParamRange protocol.Range
}, error,
) {
	byteContent := []byte(content)
	params := paramInStringRegex.FindAllIndex(byteContent, -1)

	results := []struct {
		Name       string
		FullName   string
		ParamRange protocol.Range
	}{}

	for _, param := range params {
		length := param[1] - param[0]

		if length < 0 {
			continue
		}

		paramFullName, paramName := ExtractName(string(byteContent[param[0]:param[1]]))

		startPos := position.FromIndex(param[0], byteContent)
		endPos := protocol.Position{
			Line:      startPos.Line,
			Character: startPos.Character + uint32(length),
		}

		totalRange := protocol.Range{
			Start: startPos,
			End:   endPos,
		}

		result := struct {
			Name       string
			FullName   string
			ParamRange protocol.Range
		}{
			Name:       paramName,
			FullName:   paramFullName,
			ParamRange: totalRange,
		}

		results = append(results, result)
	}

	return results, nil
}

// Given a correct parameter string (example: << parameters.something >>)
// will return a pair of strings
//
// The first returned value is the full path to the parameter (in example above: parameters.something)
// The second returned value is the parameter name
func ExtractName(parameter string) (string, string) {
	full := strings.Trim(parameter, "<")
	full = strings.Trim(full, ">")
	full = strings.Trim(full, " ")

	splittedName := strings.Split(full, ".")

	return full, splittedName[len(splittedName)-1]
}
