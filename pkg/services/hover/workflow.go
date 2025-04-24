package hover

import (
	"fmt"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
)

var workflows = "Used for orchestrating all jobs.\n" +
	"Each workflow consists of the workflow name as a key and a map as a value. A name should be unique within the current `config.yml`.\n\n" +
	"Allowed Keys\n" +
	"- `version`: not required for v2.1 configuration. Should be `2` for config version 2\n" +
	"- `<workflow_name>`: a map of workflow names and their parameters.\n\n" +
	"For more information, see the [Using Workflows to Schedule Jobs](https://circleci.com/docs/2.0/workflows) page."

var jobs = "A Workflow is comprised of one or more uniquely named jobs. Jobs are specified in the `jobs` map."

var triggers = "Specifies which triggers will cause this workflow to be executed. Default behavior is to trigger the workflow when pushing to a branch.\n\n" +
	"Should currently be schedule.\n\n" +
	"Example:\n\n" +
	"```yaml\n" +
	`workflows:
	version: 2
	nightly:
	  triggers:
		- schedule:
			cron: "0 0 * * *"
			filters:
			  branches:
				only:
				  - main
				  - beta
	  jobs:
		- test` +
	"```"

func workflowDefinition(name string) string {
	return fmt.Sprintf("`%s` - User defined workflow\n\n", name) +
		"Allowed keys:\n\n" +

		"- `triggers` (optional): Should currently be `schedule`\n" +
		"- `jobs` (required): A list of jobs to run with their dependencies|\n" +
		"- `when` (optional): A logic statement to determine whether or not to run this workflow.\n" +
		"- `unless` (optional): Inverse clause of `when`\n"
}

func jobReference(name string) string {
	return fmt.Sprintf("`%s` - Job reference\n\n", name) +
		"A job can have the optional keys `requires`, `name`, `context`, `type`, and `filters`.\n\n" +
		"- `requires`: Jobs are run in parallel by default, so you must explicitly require any dependencies by their job name.\n" +
		"- `name`: can be used to invoke reusable jobs across any number of workflows. Using the `name` key ensures numbers are not appended to your job name (i.e. sayhello-1 , sayhello-2, etc.). The name you assign to the name key needs to be unique, otherwise the numbers will still be appended to the job name.\n" +
		"- `context`: Jobs may be configured to use global environment variables set for an organization, see the Contexts document for adding a context in the application settings.\n" +
		"- `type`: Job type, can be build, release, no-op, or approval. If not specified, defaults to build.\n" +
		"- `filters`: Job Filters can have the key branches or tags.+\n" +
		"- `matrix` : requires config `2.1`. The `matrix` stanza allows you to run a parameterized job multiple times with different arguments.\n" +
		"- `pre-steps` and `post-steps`: requires config `2.1`. Steps under `pre-steps` are executed before any of the other steps in the job. The steps under `post-steps` are executed after all of the other steps.\n"
	"- `plan_name`: requires release job type. Used to link your release to a release plan. https://circleci.com/docs/deploy/deploys-overview/\n"

}

func HoverInWorkflows(doc yamlparser.YamlDocument, path []string) string {
	if len(path) == 0 {
		return workflows
	}

	currentKey := path[0]

	if currentKey == "version" {
		return "The Workflows version field is used to issue warnings for deprecation or breaking changes."
	}

	if len(path) == 1 {
		return workflowDefinition(currentKey)
	}
	return hoverSingleWorklow(doc, path[1:])
}

func hoverSingleWorklow(doc yamlparser.YamlDocument, path []string) string {
	if len(path) == 0 {
		return ""
	}
	currentKey := path[0]

	if len(path) == 1 {
		if currentKey == "jobs" {
			return jobs
		}
		if currentKey == "triggers" {
			return triggers
		}
	}

	if currentKey == "jobs" {
		return hoverJobReferences(doc, path[1:])
	}
	if currentKey == "triggers" {
		return hoverSchedule(doc, path[1:])
	}

	return ""
}

func hoverJobReferences(doc yamlparser.YamlDocument, path []string) string {
	jobReferenceName := path[0]

	if len(path) == 1 {
		return jobReference(jobReferenceName)
	}

	return hoverJobReferenceParameter(doc, path[1:])
}

func hoverJobReferenceParameter(doc yamlparser.YamlDocument, path []string) string {
	parameterName := path[0]
	if len(path) > 1 {
		return ""
	}

	switch parameterName {
	case "requires":
		return "A list of jobs that must succeed for the job to start."
	case "name":
		return "A replacement for the job name. Useful when calling a job multiple times. "
	case "context":
		return "The name of the context(s). The initial default name is org-global. Each context name must be unique."
	case "type":
		return "A job may have a type of approval indicating it must be manually approved before downstream jobs may proceed."
	case "filters":
		return "A map defining rules for execution on specific branches"
	case "matrix":
		return "The matrix stanza allows you to run a parameterized job multiple times with different arguments. For more information see the how-to guide on Using Matrix Jobs."
	case "when":
		return "You may use a when clause under a workflow declaration with a logic statement to determine whether or not to run that workflow."
	case "unless":
		return "You may use an unless clause under a workflow declaration with a logic statement to determine whether or not to run that workflow."
	default:
		return ""
	}
}

func hoverSchedule(doc yamlparser.YamlDocument, path []string) string {
	if len(path) == 0 {
		return ""
	}
	currentKey := path[0]
	if currentKey != "schedule" || len(path) > 1 {
		return ""
	}

	return "A workflow may have a schedule indicating it runs at a certain time,"
}
