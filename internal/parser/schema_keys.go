package parser

import (
	"encoding/json"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// schemaKeys finds the keys a schema allows at a place in a config, so that a
// key it doesn't allow can be matched to the one that was probably meant.
type schemaKeys struct {
	root map[string]any
}

// newSchemaKeys reads a schema's JSON. It is nil when that isn't a schema,
// which only costs the suggestions.
func newSchemaKeys(data []byte) *schemaKeys {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	return &schemaKeys{root: root}
}

// at returns the keys allowed at fields, a path as gojsonschema reports it,
// such as jobs.build.steps.0.run. Every schema that might apply there counts,
// whichever branch of a oneOf the value was meant for.
func (keys *schemaKeys) at(fields []string) []string {
	if keys == nil {
		return nil
	}

	schemas := keys.expand([]map[string]any{keys.root})
	for _, field := range fields {
		if field == "(root)" {
			continue
		}
		var children []map[string]any
		for _, schema := range schemas {
			children = append(children, child(schema, field)...)
		}
		schemas = keys.expand(children)
	}

	names := []string{}
	for _, schema := range schemas {
		if properties, ok := schema["properties"].(map[string]any); ok {
			for name := range properties {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// child returns the schemas for field in a value schema describes.
func child(schema map[string]any, field string) []map[string]any {
	if _, err := strconv.Atoi(field); err == nil {
		if items, ok := schema["items"].(map[string]any); ok {
			return []map[string]any{items}
		}
	}

	if properties, ok := schema["properties"].(map[string]any); ok {
		if property, ok := properties[field].(map[string]any); ok {
			return []map[string]any{property}
		}
	}

	var matched []map[string]any
	if patterns, ok := schema["patternProperties"].(map[string]any); ok {
		for pattern, property := range patterns {
			re, err := regexp.Compile(pattern)
			if property, ok := property.(map[string]any); ok && err == nil && re.MatchString(field) {
				matched = append(matched, property)
			}
		}
	}
	if len(matched) > 0 {
		return matched
	}

	if additional, ok := schema["additionalProperties"].(map[string]any); ok {
		return []map[string]any{additional}
	}
	return nil
}

// expand adds to schemas every schema they bring in: the targets of their
// $refs, and the branches of their combinators. An `if` only picks a branch,
// so its own keys are not among those allowed.
func (keys *schemaKeys) expand(schemas []map[string]any) []map[string]any {
	var expanded []map[string]any
	seen := map[uintptr]bool{}

	queue := slices.Clone(schemas)
	for len(queue) > 0 {
		schema := queue[0]
		queue = queue[1:]

		id := reflect.ValueOf(schema).Pointer()
		if seen[id] {
			continue
		}
		seen[id] = true
		expanded = append(expanded, schema)

		if ref, ok := schema["$ref"].(string); ok {
			if target := keys.resolve(ref); target != nil {
				queue = append(queue, target)
			}
		}
		for _, combinator := range []string{"allOf", "anyOf", "oneOf"} {
			branches, _ := schema[combinator].([]any)
			for _, branch := range branches {
				if branch, ok := branch.(map[string]any); ok {
					queue = append(queue, branch)
				}
			}
		}
		for _, branch := range []string{"then", "else"} {
			if branch, ok := schema[branch].(map[string]any); ok {
				queue = append(queue, branch)
			}
		}
	}

	return expanded
}

// resolve returns the schema a $ref within the document points at, such as
// #/definitions/step.
func (keys *schemaKeys) resolve(ref string) map[string]any {
	pointer, ok := strings.CutPrefix(ref, "#/")
	if !ok {
		return nil
	}

	var node any = keys.root
	for _, token := range strings.Split(pointer, "/") {
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		object, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = object[token]
	}

	schema, _ := node.(map[string]any)
	return schema
}

// closestKey returns the one allowed key within a small edit distance of
// name, such as resource_class for resouce_class. With none, or more than one
// as close, it returns "". The keys can come from branches of the schema the
// value wasn't meant for, so name itself is not suggested: in a lock job,
// steps is refused but a build job allows it.
func closestKey(name string, allowed []string) string {
	limit := min(2, len(name)/3)
	best, bestDistance, tied := "", limit+1, false

	for _, candidate := range allowed {
		if candidate == name {
			continue
		}
		distance := editDistance(strings.ToLower(name), strings.ToLower(candidate))
		switch {
		case distance < bestDistance:
			best, bestDistance, tied = candidate, distance, false
		case distance == bestDistance:
			tied = true
		}
	}

	if tied || bestDistance > limit {
		return ""
	}
	return best
}

// editDistance counts the insertions, deletions, substitutions and swaps of
// adjacent characters that turn a into b.
func editDistance(a, b string) int {
	x, y := []rune(a), []rune(b)
	rows := make([][]int, len(x)+1)
	for i := range rows {
		rows[i] = make([]int, len(y)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}

	for i := 1; i <= len(x); i++ {
		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			rows[i][j] = min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && x[i-1] == y[j-2] && x[i-2] == y[j-1] {
				rows[i][j] = min(rows[i][j], rows[i-2][j-2]+1)
			}
		}
	}

	return rows[len(x)][len(y)]
}
