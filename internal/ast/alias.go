package ast

import (
	"strings"

	"go.lsp.dev/protocol"
)

// Alias is an entry of `commands`, `jobs` or `executors` whose value is a
// string, such as `build: orb/build`. It defines nothing itself: the name
// stands for the element the string names.
type Alias struct {
	Name        string
	NameRange   protocol.Range
	Target      string
	TargetRange protocol.Range
	// Range covers the whole entry, name and target.
	Range protocol.Range
}

// OrbTarget splits a target written as `orb-alias/element-name`. Anything
// else, with no `/`, more than one, or a blank side, is not an orb element.
func (alias Alias) OrbTarget() (orb string, element string, ok bool) {
	orb, element, ok = strings.Cut(alias.Target, "/")
	if !ok || strings.TrimSpace(orb) == "" || strings.TrimSpace(element) == "" || strings.Contains(element, "/") {
		return "", "", false
	}
	return orb, element, true
}

// Aliases are the aliases a config or an orb declares, by name. A name
// defined as a real element as well is only that element.
type Aliases struct {
	Commands  map[string]Alias
	Jobs      map[string]Alias
	Executors map[string]Alias
}

func NewAliases() Aliases {
	return Aliases{
		Commands:  map[string]Alias{},
		Jobs:      map[string]Alias{},
		Executors: map[string]Alias{},
	}
}
