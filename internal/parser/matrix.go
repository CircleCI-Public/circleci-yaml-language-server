package parser

import (
	"regexp"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// maxMatrixJobs is the compiler's limit on the jobs one matrix expands to.
const maxMatrixJobs = 10000

var matrixReferenceRegex = regexp.MustCompile(`<<\s*matrix\.([A-Za-z0-9_-]+)\s*>>`)

type matrixParameter struct {
	name   string
	values []string
}

// matrixMemberNames returns the names of the jobs a matrix expands to, in the
// order the compiler produces them. name is the invocation's `name:`, if it
// has one. The compiler names each member, in order of precedence:
//
//  1. by a `name:` holding `<< matrix.x >>`, expanded for the member;
//  2. by any other `name:`, as written when only one combination survives
//     `exclude`, and otherwise suffixed `-1`, `-2`, … in the order produced;
//  3. by a matrix parameter called `name`;
//  4. as `job-value1-value2`, in the parameters' declared order.
func (doc *YamlDocument) matrixMemberNames(matrixNode *sitter.Node, jobName, name string) []string {
	parameters, excludes := doc.parseMatrixCombinations(matrixNode)

	combinations := matrixCartesianProduct(parameters)
	surviving := make([]map[string]string, 0, len(combinations))
	for _, combination := range combinations {
		if !isExcludedCombination(combination, excludes) {
			surviving = append(surviving, combination)
		}
	}

	names := make([]string, 0, len(surviving))
	for i, combination := range surviving {
		switch {
		case strings.Contains(name, "<<"):
			names = append(names, matrixReferenceRegex.ReplaceAllStringFunc(name, func(reference string) string {
				parameter := matrixReferenceRegex.FindStringSubmatch(reference)[1]
				if value, ok := combination[parameter]; ok {
					return value
				}
				return reference
			}))
		case name != "":
			if len(surviving) == 1 {
				names = append(names, name)
			} else {
				names = append(names, name+"-"+strconv.Itoa(i+1))
			}
		case combination["name"] != "":
			names = append(names, combination["name"])
		default:
			values := []string{jobName}
			for _, parameter := range parameters {
				values = append(values, combination[parameter.name])
			}
			names = append(names, strings.Join(values, "-"))
		}
	}

	return names
}

func (doc *YamlDocument) parseMatrixCombinations(matrixNode *sitter.Node) ([]matrixParameter, []map[string]string) {
	parameters := []matrixParameter{}
	excludes := []map[string]string{}

	doc.iterateOnBlockMapping(GetChildMapping(matrixNode), func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}

		switch doc.GetNodeText(keyNode) {
		case "parameters":
			doc.iterateOnBlockMapping(GetChildMapping(valueNode), func(child *sitter.Node) {
				keyNode, valueNode := doc.GetKeyValueNodes(child)
				if keyNode == nil || valueNode == nil {
					return
				}
				values := []string{}
				for _, value := range doc.getNodeTextArray(valueNode) {
					values = append(values, unquoteScalar(value))
				}
				parameters = append(parameters, matrixParameter{name: doc.GetNodeText(keyNode), values: values})
			})
		case "exclude":
			iterateOnBlockSequence(GetChildSequence(valueNode), func(item *sitter.Node) {
				if item.Kind() == "block_sequence_item" {
					item = item.Child(1)
				}
				exclude := map[string]string{}
				for key, value := range doc.parseDictionary(GetChildMapping(item)) {
					exclude[key] = unquoteScalar(value)
				}
				excludes = append(excludes, exclude)
			})
		}
	})

	return parameters, excludes
}

// matrixCartesianProduct returns every combination of the parameters'
// values, the first parameter varying slowest. A parameter with no values
// leaves no combinations.
func matrixCartesianProduct(parameters []matrixParameter) []map[string]string {
	count := 1
	for _, parameter := range parameters {
		count *= len(parameter.values)
		if count > maxMatrixJobs {
			return nil
		}
	}

	combinations := []map[string]string{{}}
	for _, parameter := range parameters {
		next := make([]map[string]string, 0, len(combinations)*len(parameter.values))
		for _, combination := range combinations {
			for _, value := range parameter.values {
				extended := make(map[string]string, len(combination)+1)
				for k, v := range combination {
					extended[k] = v
				}
				extended[parameter.name] = value
				next = append(next, extended)
			}
		}
		combinations = next
	}

	return combinations
}

func isExcludedCombination(combination map[string]string, excludes []map[string]string) bool {
	for _, exclude := range excludes {
		matches := true
		for key, value := range exclude {
			if actual, ok := combination[key]; !ok || actual != value {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func unquoteScalar(text string) string {
	text = strings.TrimSpace(text)
	if len(text) >= 2 && (text[0] == '"' || text[0] == '\'') && text[len(text)-1] == text[0] {
		return text[1 : len(text)-1]
	}
	return text
}
