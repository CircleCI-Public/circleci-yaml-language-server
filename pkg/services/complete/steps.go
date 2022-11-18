package complete

import sitter "github.com/smacker/go-tree-sitter"

func (ch *CompletionHandler) completeSteps(includeJobSteps bool, completionNode *sitter.Node) {
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
