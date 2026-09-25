package hover

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// Executor is the hover for the executor a job names, whether the config's
// own, an inline orb's or an orb's: the executor's description and
// parameters.
func Executor(doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	if name, ok := executorNamedAt(pos, doc.Jobs); ok {
		if executor, ok := doc.Executors[name]; ok {
			return describeExecutor(name, executor), true
		}
		if orb, executorName, ok := orbOf(doc, c, name); ok {
			if executor, ok := orb.Executors[executorName]; ok {
				return describeExecutor(name, executor), true
			}
		}
		return "", false
	}

	// An inline orb's jobs name its executors without the orb's prefix.
	for _, orb := range doc.LocalOrbInfo {
		if name, ok := executorNamedAt(pos, orb.Jobs); ok {
			if executor, ok := orb.Executors[name]; ok {
				return describeExecutor(name, executor), true
			}
			return "", false
		}
	}
	return "", false
}

// executorNamedAt is the executor named by the job whose `executor:` line the
// cursor is on. The line alone, because the range of an executor given as a
// mapping spans its parameters too.
func executorNamedAt(pos protocol.Position, jobs map[string]ast.Job) (string, bool) {
	for _, job := range jobs {
		if job.Executor != "" && pos.Line == job.ExecutorRange.Start.Line && position.InRange(job.ExecutorRange, pos) {
			return job.Executor, true
		}
	}
	return "", false
}

func describeExecutor(name string, executor ast.Executor) string {
	return describe(name, "executor", executor.GetDescription(), executor.GetParameters())
}
