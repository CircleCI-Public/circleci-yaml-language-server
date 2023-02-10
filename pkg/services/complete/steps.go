package complete

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
)

func (ch *CompletionHandler) completeSteps(entityName string, inJob bool, includeJobSteps bool, completionNode *sitter.Node) {
	if ch.isWritingAnEnvVariableInRunStep(entityName, inJob) {
		return
	}
	// We have two ifs to keep the order of the steps in the completion list.
	ch.userDefinedCommands()
	if includeJobSteps {
		ch.userDefinedJobs()
	}
	ch.builtInSteps()
	ch.orbCommands(completionNode)
	if includeJobSteps {
		ch.orbsJobs()
	}
}

func (ch *CompletionHandler) builtInSteps() {
	BUILT_IN_STEPS := []string{"run", "checkout", "setup_remote_docker", "save_cache", "restore_cache", "store_artifacts", "store_test_results", "persist_to_workspace", "attach_workspace", "add_ssh_keys", "unless", "when"}
	for _, stepName := range BUILT_IN_STEPS {
		ch.addCompletionItem(stepName)
	}
}

func (ch *CompletionHandler) isWritingAnEnvVariableInRunStep(entityName string, inJob bool) bool {
	var steps []ast.Step
	if inJob {
		steps = ch.Doc.Jobs[entityName].Steps
	} else {
		steps = ch.Doc.Commands[entityName].Steps
	}

	var contexts []string
	if inJob {
		contexts = *ch.Doc.Jobs[entityName].Contexts
	} else {
		contexts = *ch.Doc.Commands[entityName].Contexts
	}

	for _, step := range steps {
		switch step := step.(type) {
		case ast.Run:
			if utils.PosInRange(step.CommandRange, ch.Params.Position) {
				idx := utils.PosToIndex(ch.Params.Position, ch.Doc.Content)
				if idx > 0 && string(ch.Doc.Content[idx-1]) == "$" {
					ch.addCompleteEnvVariables(contexts)
					return true
				}
			}
		}
	}

	return false
}

func (ch *CompletionHandler) addCompleteEnvVariables(contexts []string) {
	contextEnvVariables := utils.GetAllContextEnvVariables(ch.Context.Api.Token, ch.Cache, contexts)
	for _, env := range contextEnvVariables {
		ch.addCompletionItem(env)
	}

	for _, env := range BUILT_IN_ENV {
		ch.addCompletionItem(env)
	}
}

var BUILT_IN_ENV = []string{
	"CI",
	"CIRCLECI",
	"CIRCLE_BRANCH",
	"CIRCLE_BUILD_NUM",
	"CIRCLE_BUILD_URL",
	"CIRCLE_JOB",
	"CIRCLE_NODE_INDEX",
	"CIRCLE_NODE_TOTAL",
	"CIRCLE_OIDC_TOKEN",
	"CIRCLE_PR_NUMBER",
	"CIRCLE_PR_REPONAME",
	"CIRCLE_PR_USERNAME",
	"CIRCLE_PREVIOUS_BUILD_NUM",
	"CIRCLE_PROJECT_REPONAME",
	"CIRCLE_PROJECT_USERNAME",
	"CIRCLE_PULL_REQUEST",
	"CIRCLE_PULL_REQUESTS",
	"CIRCLE_REPOSITORY_URL",
	"CIRCLE_SHA1",
	"CIRCLE_TAG",
	"CIRCLE_USERNAME",
	"CIRCLE_WORKFLOW_ID",
	"CIRCLE_WORKFLOW_JOB_ID",
	"CIRCLE_WORKFLOW_WORKSPACE_ID",
	"CIRCLE_WORKING_DIRECTORY",
	"CIRCLE_INTERNAL_TASK_DATA",
}
