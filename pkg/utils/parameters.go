package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

// Return the name of the parameter used at the given position
func GetParamNameUsedAtPos(content []byte, position protocol.Position) (string, bool) {
	paramRegex, _ := regexp.Compile(`<<\s*(parameters|pipeline.parameters)\.([A-z0-9-_]*)\s*>>`)
	isPipelineParam := false

	// Get the content of the YAML at the beginning of the line
	PosToIndex := PosToIndex(position, content)

	// We check if the line contain a parameter
	lineIdx := strings.LastIndex(string(content[:PosToIndex]), "\n")
	if lineIdx == -1 {
		lineIdx = 0
	}

	endOfLine := strings.Index(string(content[lineIdx+1:]), "\n")
	if endOfLine == -1 {
		endOfLine = len(content[lineIdx:]) - 1
	}

	if !paramRegex.MatchString(string(content[lineIdx : lineIdx+endOfLine+1])) {
		return "", isPipelineParam
	}

	// This is needed if two parameters are in the same string, we only want
	// the one that has been declared before the cursor
	startOfParam := strings.LastIndex(string(content[:PosToIndex]), "<<")
	if startOfParam == -1 {
		return "", isPipelineParam
	}

	endOfParam := strings.Index(string(content[startOfParam:]), ">>")
	if endOfParam == -1 {
		return "", isPipelineParam
	}

	// We add the length of the ">>" and startOfParam so that we can
	// have the right index based on the beginning of content
	endOfParam += startOfParam + 2

	param := paramRegex.Find([]byte(content[startOfParam:endOfParam]))

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

// Returns true if the string is *only* a parameter
// Example:
//
//	param: << parameters.paramName >> -> true
//	param: << pipeline.parameters.paramName >> -> true
//	param: `/home/<< parameters.paramName >>/Downloads` -> false
func CheckIfOnlyParamUsed(content string) bool {
	regex, _ := regexp.Compile(`^<<\s*(parameters|pipeline.parameters)\.([A-z0-9-_]*)\s*>>$`)
	return regex.MatchString(content)
}

func CheckIfParamIsPartiallyReferenced(content string) (bool, bool) {
	regex, _ := regexp.Compile(`<<\s*(parameters|pipeline.parameters)\.\s*>?>?`)
	isPipelineParam := strings.Contains(content, "pipeline.")
	return regex.Find([]byte(content)) != nil, isPipelineParam
}

func CheckIfMatrixParamIsPartiallyReferenced(content string) bool {
	regex, _ := regexp.Compile(`<<\s*matrix\.\s*>?>?`)
	return regex.Find([]byte(content)) != nil
}

func GetParamsInString(content string) ([]struct {
	Name       string
	FullName   string
	ParamRange protocol.Range
}, error) {
	paramRegex, err := regexp.Compile(`<<\s*(parameters|pipeline.parameters)\.([A-z0-9-_]*)\s*>>`)
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
		length := uint32(param[1] - param[0])

		if length < 0 {
			continue
		}

		paramFullName, paramName := ExtractParameterName(string(byteContent[param[0]:param[1]]))

		startPos := IndexToPos(param[0], byteContent)
		endPos := protocol.Position{
			Line:      startPos.Line,
			Character: startPos.Character + length,
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
