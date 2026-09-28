package validate

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

// localExecutor returns the executor defined in the config that a name
// stands for, directly or through an alias such as `alias-exec: real-exec`.
func (val Validate) localExecutor(name string) (ast.Executor, bool) {
	if alias, ok := val.Doc.ExecutorAlias(name); ok {
		name = alias.Target
	}
	executor, ok := val.Doc.Executors[name]
	return executor, ok
}

// validateExecutorAliases reports an executor alias that names nothing. Its
// target is either an orb's executor, `orb-alias/executor-name`, or an
// executor the config defines. The problem is an error where the alias is
// used, and only a warning where nothing uses it.
func (val Validate) validateExecutorAliases(used map[string]bool) {
	for _, alias := range val.Doc.Aliases.Executors {
		if _, ok := val.Doc.Executors[alias.Name]; ok {
			continue
		}

		message := val.executorAliasProblem(alias)
		if message == "" {
			continue
		}
		if used[alias.Name] {
			val.addDiagnostic(diagnostic.Error(alias.TargetRange, message))
		} else {
			val.addDiagnostic(diagnostic.Warning(alias.TargetRange, message))
		}
	}
}

func (val Validate) executorAliasProblem(alias ast.Alias) string {
	if orbName, executorName, ok := alias.OrbTarget(); ok {
		if _, ok := val.Doc.Orbs[orbName]; !ok {
			return fmt.Sprintf("Unable to determine target for executor invocation %s (renamed from local executor %s)",
				alias.Target, alias.Name)
		}
		if exists, err := val.doesOrbExecutorExist(alias.Target, alias.TargetRange); !exists && err == nil {
			return fmt.Sprintf("Cannot find executor %s in orb %s", executorName, orbName)
		}
		return ""
	}

	if strings.Contains(alias.Target, "/") {
		return malformedAliasMessage
	}
	if _, ok := val.Doc.Executors[alias.Target]; !ok {
		return fmt.Sprintf("Executor alias '%s' refers to '%s', which is not a known local executor",
			alias.Name, alias.Target)
	}
	return ""
}

// malformedAliasMessage is the compiler's error for an alias whose target
// isn't a single `orb-alias/element-name`.
const malformedAliasMessage = "Invalid orb element alias, expected a single 'orb-alias/element-name' reference"
