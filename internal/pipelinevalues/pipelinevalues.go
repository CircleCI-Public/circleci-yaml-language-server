// Package pipelinevalues describes the built-in pipeline values, such as
// pipeline.git.branch, from the list the expr module keeps.
package pipelinevalues

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/CircleCI-Public/expr/domains"
	"gopkg.in/yaml.v3"
)

type Value struct {
	Name string
	// Type is the scalar type, which is string when the list gives none.
	Type       string
	Definition string
	// Private values aren't documented publicly.
	Private    bool
	ReplacedBy string
}

// Lookup finds a value by its full name. Pipeline parameters aren't in the
// list.
func Lookup(name string) (Value, bool) {
	value, ok := values()[name]
	return value, ok
}

// Public leaves out the private values and the pipeline.parameters.*
// wildcard, and sorts the rest by name.
func Public() []Value {
	public := []Value{}
	for _, name := range slices.Sorted(maps.Keys(values())) {
		if value := values()[name]; !value.Private && !strings.HasSuffix(name, "*") {
			public = append(public, value)
		}
	}
	return public
}

var values = sync.OnceValue(func() map[string]Value {
	values, err := parse(domains.PipelineValues)
	if err != nil {
		panic(err)
	}
	return values
})

func parse(data []byte) (map[string]Value, error) {
	var list struct {
		Fields []struct {
			Name       string `yaml:"name"`
			Type       string `yaml:"type"`
			Definition string `yaml:"defn"`
			Private    bool   `yaml:"private"`
			ReplacedBy string `yaml:"replaced_by"`
		} `yaml:"fields"`
	}
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("pipeline values: %w", err)
	}

	values := make(map[string]Value, len(list.Fields))
	for _, field := range list.Fields {
		valueType := field.Type
		if valueType == "" {
			valueType = "string"
		}
		values[field.Name] = Value{
			Name:       field.Name,
			Type:       valueType,
			Definition: field.Definition,
			Private:    field.Private,
			ReplacedBy: field.ReplacedBy,
		}
	}
	return values, nil
}
