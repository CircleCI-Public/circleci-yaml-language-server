package hover

import (
	"fmt"
	"strings"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func HoverInJobs(doc yamlparser.YamlDocument, path []string, cache *utils.Cache) string {
	if len(path) == 0 {
		return "Jobs are collections of steps. All of the steps in the job are executed in a single unit, either within a fresh container or VM."
	}

	if len(path) == 1 {
		jobName := path[0]
		return fmt.Sprintf("%s - User defined job", jobName)
	}

	return hoverSingleJob(doc, path[1:], cache)
}

func hoverSingleJob(doc yamlparser.YamlDocument, path []string, cache *utils.Cache) string {
	if len(path) == 0 {
		return ""
	}

	fieldName := path[0]

	switch fieldName {
	case "steps":
		return hoverSteps(doc, path[1:], cache)
	default:
		return ""
	}

}

func hoverSteps(doc yamlparser.YamlDocument, path []string, cache *utils.Cache) string {
	if len(path) == 0 {
		return "Steps are the individual units of work in a job. Each step is a command, or a job."
	}

	stepName := path[0]
	if doc.IsBuiltIn(stepName) {
		return commandsDescription[stepName]
	}

	if cmd, ok := doc.Commands[stepName]; ok {
		return cmd.Description
	}

	return hoverOrb(doc, stepName, cache)
}

func hoverOrb(doc yamlparser.YamlDocument, stepName string, cache *utils.Cache) string {
	splittedStep := strings.Split(stepName, "/")
	orbInDoc := doc.Orbs[splittedStep[0]]
	orb := doc.GetOrbInfo(cache, orbInDoc.Name)
	if orb == nil {
		return ""
	}

	return orb.Commands[splittedStep[1]].Description
}

var commandsDescription map[string]string = map[string]string{
	"checkout":             "A special step used to check out source code to the configured path",
	"run":                  "Used for invoking all command-line programs, taking either a map of configuration values, or, when called in its short-form, a string that will be used as both the command and name. Run commands are executed using non-login shells by default, so you must explicitly source any dotfiles as part of the command.",
	"setup_remote_docker":  "Creates a remote Docker environment configured to execute Docker commands.",
	"save_cache":           "Generates and stores a cache of a file or directory of files such as dependencies or source code in our object storage.",
	"restore_cache":        "Restores a previously saved cache based on a key. Cache needs to have been saved first for this key using save_cache step. ",
	"store_artifacts":      "Step to store artifacts (for example logs, binaries, etc) to be available in the web app or through the API.",
	"store_test_results":   "Special step used to upload and store test results for a build. Test results are visible on the CircleCI web application under each build's Test Summary section. Storing test results is useful for timing analysis of your test suites.",
	"persist_to_workspace": "Special step used to persist a temporary file to be used by another job in the workflow.",
	"attach_workspace":     "Special step used to attach the workflow's workspace to the current container. The full contents of the workspace are downloaded and copied into the directory the workspace is being attached at.",
	"add_ssh_keys":         "Special step that adds SSH keys from a project's settings to a container. Also configures SSH to use these keys.",
}
