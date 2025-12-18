package complete

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (ch *CompletionHandler) completeSteps(entityName string, inJob bool, includeJobSteps bool, completionNode *sitter.Node) {
	if ch.isWritingAnEnvVariableInRunStep(entityName, inJob) {
		return
	}
	if ch.isWritingCheckoutMethod(entityName, inJob) {
		ch.addCheckoutMethodCompletion()
		return
	}
	// We have two ifs to keep the order of the steps in the completion list.
	ch.userDefinedCommands()
	if includeJobSteps {
		ch.userDefinedJobs()
		ch.orbsJobs()
	}
	ch.builtInSteps()
	ch.orbCommands(completionNode)
}

func (ch *CompletionHandler) builtInSteps() {
	BUILT_IN_STEPS := []string{
		"run",
		"checkout",
		"setup_remote_docker",
		"save_cache",
		"restore_cache",
		"store_artifacts",
		"store_test_results",
		"persist_to_workspace",
		"attach_workspace",
		"add_ssh_keys",
		"unless",
		"when",
	}
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
	var parameters map[string]ast.Parameter
	environmentField := map[string]string{}
	if inJob {
		contexts = *ch.Doc.Jobs[entityName].Contexts
		parameters = ch.Doc.Jobs[entityName].Parameters
		environmentField = ch.Doc.Jobs[entityName].Environment
	} else {
		contexts = *ch.Doc.Commands[entityName].Contexts
		parameters = ch.Doc.Commands[entityName].Parameters
	}

	for _, step := range steps {
		switch step := step.(type) {
		case ast.Run:
			if utils.PosInRange(step.CommandRange, ch.Params.Position) {
				idx := utils.PosToIndex(ch.Params.Position, ch.Doc.Content)
				if idx > 0 && string(ch.Doc.Content[idx-1]) == "$" {
					ch.addCompleteEnvVariables(contexts, parameters, environmentField)
					return true
				}
			}
		}
	}

	return false
}

func (ch *CompletionHandler) isWritingCheckoutMethod(entityName string, inJob bool) bool {
	var steps []ast.Step
	if inJob {
		steps = ch.Doc.Jobs[entityName].Steps
	} else {
		steps = ch.Doc.Commands[entityName].Steps
	}

	for _, step := range steps {
		switch step := step.(type) {
		case ast.Checkout:
			if utils.PosInRange(step.MethodRange, ch.Params.Position) {
				return true
			}
		}
	}

	return false
}

func (ch *CompletionHandler) addCheckoutMethodCompletion() {
	for _, method := range utils.CheckoutMethods {
		ch.addCompletionItem(method)
	}
}

func (ch *CompletionHandler) addCompleteEnvVariables(contexts []string, parameters map[string]ast.Parameter, environmentField map[string]string) {
	for _, param := range parameters {
		switch param.(type) {
		case ast.EnvVariableParameter:
			ch.addCompletionItemFieldWithCustomText(param.GetName(), "<< parameters.", " >>", "From parameter "+param.GetName(), "A")
		}
	}

	for env := range environmentField {
		ch.addCompletionItemWithDetail(env, "From environment defined in the job", "A")
	}

	if cachedFile := ch.Cache.FileCache.GetFile(ch.Doc.URI); cachedFile != nil {
		for _, env := range cachedFile.EnvVariables {
			ch.addCompletionItemWithDetail(env, "From project "+cachedFile.Project.Name, "B")
		}

		contextEnvVariables := utils.GetAllContextEnvVariables(ch.Cache, cachedFile.Project.OrganizationId, contexts)
		for _, env := range contextEnvVariables {
			ch.addCompletionItemWithDetail(env.Name, "From context "+env.AssociatedContext, "B")
		}
	}

	for _, env := range BUILT_IN_ENV {
		ch.addCompletionItemWithDetail(env, "Built-in environment variable", "C")
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
}
