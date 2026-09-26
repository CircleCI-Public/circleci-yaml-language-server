package complete

import (
	"regexp"
	"slices"
	"strings"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
)

// builtInStepKeys are the keys each built-in step takes, except
// setup_remote_docker's resource_class, which is ignored, with a warning.
var builtInStepKeys = map[string][]string{
	"run": {
		"command", "name", "shell", "environment", "background", "working_directory",
		"no_output_timeout", "when", "max_auto_reruns", "auto_rerun_delay", "teardown",
	},
	"checkout":             {"method", "depth", "path", "when"},
	"setup_remote_docker":  {"docker_layer_caching", "version", "prefer_same_region", "when"},
	"save_cache":           {"key", "paths", "name", "when"},
	"restore_cache":        {"key", "keys", "name", "when"},
	"store_artifacts":      {"path", "destination", "name", "when"},
	"store_test_results":   {"path", "name", "when"},
	"persist_to_workspace": {"root", "paths", "name", "when"},
	"attach_workspace":     {"at", "name", "when"},
	"add_ssh_keys":         {"fingerprints", "when"},
	"with_tool_cache":      {"tool", "steps"},
	"when":                 {"condition", "steps"},
	"unless":               {"condition", "steps"},
}

// builtInStepKeyValues are the values of the built-in steps' keys that take
// only a few, other than `when`, which every step with a `when` takes.
var builtInStepKeyValues = map[string]map[string][]string{
	"run": {"background": {"true", "false"}},
	"setup_remote_docker": {
		"version":              {"default", "24.0.9"},
		"docker_layer_caching": {"true", "false"},
		"prefer_same_region":   {"true", "false"},
	},
	"with_tool_cache": {"tool": {"gradle", "bazel", "turborepo", "xcode"}},
}

// builtInStepValues is empty for a key whose values can't be listed, such
// as run's command.
func builtInStepValues(step, key string) []string {
	if key == "when" && slices.Contains(builtInStepKeys[step], "when") {
		return []string{"always", "on_success", "on_fail"}
	}
	return builtInStepKeyValues[step][key]
}

// teardownSteps are the steps a run step's teardown can hold. A teardown run
// can't run in the background or have a teardown of its own.
var (
	teardownSteps      = []string{"run", "save_cache", "persist_to_workspace", "store_artifacts", "store_test_results"}
	teardownRunLeftOut = []string{"background", "teardown"}
)

// isInTeardown goes by indentation, not the parsed steps: a teardown's range
// doesn't reach the blank item being typed at its end.
func isInTeardown(lines []string, line int) bool {
	parent := parentLine(lines, line)
	return parent != -1 && strings.TrimSpace(lines[parent]) == "teardown:"
}

// functionStepKeys are the keys a function step takes.
var functionStepKeys = []string{"id", "with"}

// stepPosition is where in a list of steps the cursor is.
type stepPosition int

const (
	// atStepName is where a step's name is written: after a "- ".
	atStepName stepPosition = iota
	// inStepBody is a key of a step's body, indented under its name.
	inStepBody
	// elsewhere is anywhere else, such as in a value of a step's body.
	elsewhere
)

var (
	stepNameBeingWritten = regexp.MustCompile(`^\s*-\s*[\w/.-]*$`)
	stepWithBody         = regexp.MustCompile(`^(\s*)-\s+([\w/.-]+)\s*:\s*$`)
	// stepWithValue is a step and the start of the value written after it,
	// such as `- run: make`.
	stepWithValue = regexp.MustCompile(`^\s*-\s+[\w/.-]+\s*:`)
	bodyKey       = regexp.MustCompile(`^(\s*)([A-Za-z_][\w-]*)\s*:`)
	stepsKey      = regexp.MustCompile(`^\s*(-\s+)?(steps|pre-steps|post-steps)\s*:\s*$`)
)

// stepAt says where the cursor is among steps, and when it is in a step's
// body, which step's and the line its name is on. It reads the text, not
// the parsed steps: a step's range doesn't reach a blank line after its
// body, so the indentation is what says the cursor belongs to it.
func (ch *CompletionHandler) stepAt() (stepPosition, string, int) {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return elsewhere, "", 0
	}

	line := lines[pos.Line]
	before := line[:min(int(pos.Character), len(line))]
	if stepNameBeingWritten.MatchString(before) {
		return atStepName, "", 0
	}
	if stepWithValue.MatchString(before) {
		return elsewhere, "", 0
	}

	indent := len(before) - len(strings.TrimLeft(before, " "))
	for parent := int(pos.Line) - 1; parent >= 0; parent-- {
		text := lines[parent]
		if strings.TrimSpace(text) == "" {
			continue
		}
		if len(text)-len(strings.TrimLeft(text, " ")) >= indent {
			continue
		}
		if match := stepWithBody.FindStringSubmatch(text); match != nil {
			return inStepBody, match[2], parent
		}
		if stepsKey.MatchString(text) {
			return atStepName, "", 0
		}
		return elsewhere, "", 0
	}

	return elsewhere, "", 0
}

// completeStepBody offers the keys a step doesn't have yet: a built-in
// step's own keys, a function step's, or the parameters of the command it
// runs.
func (ch *CompletionHandler) completeStepBody(name string, nameLine int) {
	var keys []string

	if builtIn, ok := builtInStepKeys[name]; ok {
		keys = builtIn
		if name == "run" && isInTeardown(strings.Split(string(ch.Doc.Content), "\n"), nameLine) {
			keys = slices.DeleteFunc(slices.Clone(keys), func(key string) bool {
				return slices.Contains(teardownRunLeftOut, key)
			})
		}
	} else if _, _, isFunction := ch.Doc.FunctionForStep(name); isFunction {
		keys = functionStepKeys
	} else {
		for param := range ch.Doc.GetDefinedParams(name, yamlparser.CommandEntity, ch.Cache) {
			keys = append(keys, param)
		}
		slices.Sort(keys)
	}

	present := ch.stepBodyKeys(nameLine)
	for _, key := range keys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
}

// stepBodyKeys are the keys at the top of the body of the step or job
// invocation named on a line: the lines after it indented deeper than it, at the shallowest of
// their indentations.
func (ch *CompletionHandler) stepBodyKeys(nameLine int) map[string]bool {
	lines := strings.Split(string(ch.Doc.Content), "\n")
	stepIndent := len(lines[nameLine]) - len(strings.TrimLeft(lines[nameLine], " "))

	keys := map[string]bool{}
	indent := -1
	for _, text := range lines[nameLine+1:] {
		if strings.TrimSpace(text) == "" {
			continue
		}
		match := bodyKey.FindStringSubmatch(text)
		if match == nil || len(match[1]) <= stepIndent {
			if len(text)-len(strings.TrimLeft(text, " ")) <= stepIndent {
				break
			}
			continue
		}
		if indent == -1 || len(match[1]) < indent {
			indent = len(match[1])
			clear(keys)
		}
		if len(match[1]) == indent {
			keys[match[2]] = true
		}
	}

	return keys
}
