package utils

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

var paramRegex = regexp.MustCompile(`<<\s*(parameters|pipeline.parameters)\.([A-Za-z0-9-_]*)\s*>>`)

func ContainsParam(content string) bool {
	return paramRegex.MatchString(content)
}

// Return the name of the parameter used at the given position
func GetParamNameUsedAtPos(content []byte, position protocol.Position) (string, bool) {
	isPipelineParam := false

	posIndex := PosToIndex(position, content)
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

	fullParamName, paramName := ExtractParameterName(string(param))
	isPipelineParam = strings.HasPrefix(fullParamName, "pipeline.")

	return paramName, isPipelineParam
}

// Search the right parameters that is defined in the given position and return its name
func GetParamNameDefinedAtPos(parameters map[string]ast.Parameter, pos protocol.Position) string {
	for _, param := range parameters {
		rng := param.GetRange()
		if PosInRange(rng, pos) {
			return param.GetName()
		}
	}

	return ""
}

func GetReferencesOfParamInRange(content []byte, paramName string, rng protocol.Range) ([][]int, error) {
	startIndex := PosToIndex(rng.Start, content)
	endIndex := PosToIndex(rng.End, content)

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

var onlyParamRegex = regexp.MustCompile(`^<<\s*(parameters|pipeline.parameters)\.([A-Za-z0-9-_]*)\s*>>$`)

// Returns true if the string is *only* a parameter
// Example:
//
//	param: << parameters.paramName >> -> true
//	param: << pipeline.parameters.paramName >> -> true
//	param: `/home/<< parameters.paramName >>/Downloads` -> false
func CheckIfOnlyParamUsed(content string) bool {
	return onlyParamRegex.MatchString(content)
}

var partialParamRegex = regexp.MustCompile(`<<\s*(parameters|pipeline.parameters|pipeline.git)\.\s*>?>?`)

func CheckIfParamIsPartiallyReferenced(content string) (bool, bool) {
	isPipelineParam := strings.Contains(content, "pipeline.")
	return partialParamRegex.Find([]byte(content)) != nil, isPipelineParam
}

func CheckIfMatrixParamIsPartiallyReferenced(content string) bool {
	regex, _ := regexp.Compile(`<<\s*matrix\.\s*>?>?`)
	return regex.Find([]byte(content)) != nil
}

func GetParamsInString(content string) ([]struct {
	Name       string
	FullName   string
	ParamRange protocol.Range
}, error,
) {
	paramRegex, err := regexp.Compile(`<<\s*(parameters|pipeline.parameters)\.([A-Za-z0-9-_]*)\s*>>`)
	if err != nil {
		return nil, fmt.Errorf("")
	}

	byteContent := []byte(content)
	params := paramRegex.FindAllIndex(byteContent, -1)

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

		paramFullName, paramName := ExtractParameterName(string(byteContent[param[0]:param[1]]))

		startPos := IndexToPos(param[0], byteContent)
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
func ExtractParameterName(parameter string) (string, string) {
	full := strings.Trim(parameter, "<")
	full = strings.Trim(full, ">")
	full = strings.Trim(full, " ")

	splittedName := strings.Split(full, ".")

	return full, splittedName[len(splittedName)-1]
}
