package validate

import (
	"context"
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
func (val Validate) validateExecutorAliases(ctx context.Context, used map[string]int) {
	for _, alias := range val.Doc.Aliases.Executors {
		if _, ok := val.Doc.Executors[alias.Name]; ok {
			continue
		}

		val.reportAliasProblem(alias, val.executorAliasProblem(ctx, alias), used[alias.Name] > 0)
	}
}

func (val Validate) reportAliasProblem(alias ast.Alias, message string, used bool) {
	switch {
	case message == "":
	case used:
		val.addDiagnostic(diagnostic.Error(alias.TargetRange, message))
	default:
		val.addDiagnostic(diagnostic.Warning(alias.TargetRange, message))
	}
}

func (val Validate) executorAliasProblem(ctx context.Context, alias ast.Alias) string {
	if orbName, executorName, ok := alias.OrbTarget(); ok {
		if _, ok := val.Doc.Orbs[orbName]; !ok {
			return fmt.Sprintf("Unable to determine target for executor invocation %s (renamed from local executor %s)",
				alias.Target, alias.Name)
		}
		if exists, err := val.doesOrbExecutorExist(ctx, alias.Target, alias.TargetRange); !exists && err == nil {
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

// validateCommandAliases reports a command alias that names nothing. Its
// target must be an orb's command, `orb-alias/command-name`.
func (val Validate) validateCommandAliases(ctx context.Context) {
	for _, alias := range val.Doc.Aliases.Commands {
		if _, ok := val.Doc.Commands[alias.Name]; ok {
			continue
		}

		val.warnShadowedPrimitive(alias.Name, alias.NameRange)

		used := val.checkIfCommandIsUsed(ast.Command{Name: alias.Name})
		if !used && !val.IsLocalOrb {
			val.commandIsUnused(ast.Command{Name: alias.Name, NameRange: alias.NameRange})
		}

		// The compiler rejects a malformed alias named after the `run` step
		// even when nothing uses it.
		if _, _, ok := alias.OrbTarget(); !ok && !used && alias.Name != "run" {
			val.addDiagnostic(diagnostic.Warning(alias.TargetRange, malformedAliasWarning(alias)))
			continue
		}
		val.reportAliasProblem(alias, val.commandAliasProblem(ctx, alias), used || alias.Name == "run")
	}
}

func (val Validate) commandAliasProblem(ctx context.Context, alias ast.Alias) string {
	orbName, _, ok := alias.OrbTarget()
	if !ok {
		return malformedAliasMessage
	}
	if _, ok := val.Doc.Orbs[orbName]; !ok {
		return fmt.Sprintf("Unable to determine target for step invocation %s (renamed from local command %s)",
			alias.Target, alias.Name)
	}
	return val.unknownStepMessage(ctx, alias.Target)
}

// malformedAliasWarning is the compiler's warning for a malformed alias
// that nothing uses.
func malformedAliasWarning(alias ast.Alias) string {
	return fmt.Sprintf("`%s` is not a valid orb element alias, so this entry is ignored unless it is invoked. "+
		"An alias must be a single `orb-alias/element-name` reference.", alias.Target)
}

// validateJobAliases reports a job alias that names nothing. Its target must
// be an orb's job, `orb-alias/job-name`.
func (val Validate) validateJobAliases(ctx context.Context) {
	for _, alias := range val.Doc.Aliases.Jobs {
		if _, ok := val.Doc.Jobs[alias.Name]; ok {
			continue
		}

		if !val.IsLocalOrb {
			val.checkAndReportUnusedJob(ast.Job{Name: alias.Name, NameRange: alias.NameRange})
		}

		used, _ := val.jobUse(alias.Name)
		if _, _, ok := alias.OrbTarget(); !ok && !used {
			val.addDiagnostic(diagnostic.Warning(alias.TargetRange, malformedAliasWarning(alias)))
			continue
		}
		val.reportAliasProblem(alias, val.jobAliasProblem(ctx, alias), used)
	}
}

func (val Validate) jobAliasProblem(ctx context.Context, alias ast.Alias) string {
	orbName, _, ok := alias.OrbTarget()
	if !ok {
		return malformedAliasMessage
	}
	if _, ok := val.Doc.Orbs[orbName]; !ok {
		return fmt.Sprintf("Unable to determine target for job invocation %s (renamed from local job %s)",
			alias.Target, alias.Name)
	}
	if val.Doc.IsFromUnfetchableOrb(ctx, alias.Target, val.Cache) {
		return ""
	}
	return val.unknownJobMessage(ctx, alias.Target)
}
