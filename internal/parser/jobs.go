package parser

import (
	"strconv"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

func (doc *YamlDocument) parseJobs(jobsNode *sitter.Node) {
	// jobsNode is of type block_node
	blockMappingNode := GetChildMapping(jobsNode)
	if blockMappingNode == nil {
		return
	}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		job := doc.parseSingleJob(child)
		if definedJob, ok := doc.Jobs[job.Name]; ok {
			doc.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Range:    job.NameRange,
				Message:  protocol.String("Job already defined"),
				Source:   protocol.NewOptional("cci-language-server"),
			})
			doc.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Range:    definedJob.NameRange,
				Message:  protocol.String("Job already defined"),
				Source:   protocol.NewOptional("cci-language-server"),
			})
			return
		}

		doc.Jobs[job.Name] = job
	})
}

func (doc *YamlDocument) parseSingleJob(jobNode *sitter.Node) ast2.Job {
	// jobNode is a block_mapping_pair
	jobNameNode, valueNode := doc.GetKeyValueNodes(jobNode)
	res := ast2.Job{CompletionItem: &[]protocol.CompletionItem{}, Parallelism: -1, Contexts: &[]string{}, Parameters: map[string]ast2.Parameter{}}

	if jobNameNode == nil || valueNode == nil {
		return res
	}
	jobName := doc.GetNodeText(jobNameNode)
	blockMappingNode := GetChildMapping(valueNode)

	if blockMappingNode == nil { //TODO: deal with errors
		return res
	}
	res.Name = doc.getAttributeName(jobName)
	res.Range = doc.NodeToRange(jobNode)
	res.NameRange = doc.NodeToRange(jobNameNode)

	machineNode := &sitter.Node{}
	machineNodeFound := false

	// keys are the job's keys, including those with no value yet, for
	// completion to leave out.
	keys := map[string]bool{}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		if child.Kind() == "block_mapping_pair" || child.Kind() == "flow_pair" {
			keyNode, valueNode := doc.GetKeyValueNodes(child)
			if keyNode != nil {
				keys[doc.GetNodeText(keyNode)] = true
			}
			if keyNode == nil || valueNode == nil {
				return
			}

			keyName := doc.GetNodeText(keyNode)
			switch keyName {
			case "shell":
				res.Shell = doc.GetNodeText(valueNode)
				res.ShellRange = doc.NodeToRange(keyNode)

			case "working_directory":
				res.WorkingDirectory = doc.GetNodeText(valueNode)

			case "retention":
				res.RetentionRange = doc.NodeToRange(child)
				res.Retention = doc.parseRetention(valueNode)

			case "description":
				res.Description = doc.GetNodeText(valueNode)

			case "parallelism":
				parsedInt, err := strconv.ParseInt(doc.GetNodeText(valueNode), 10, 8)
				if err != nil {
					return
				}

				res.Parallelism = int(parsedInt)
				res.ParallelismRange = doc.NodeToRange(child)
			case "resource_class":
				res.ResourceClass = doc.GetNodeText(valueNode)
				res.ResourceClassRange = doc.NodeToRange(keyNode)

			case "steps":
				res.StepsRange = doc.NodeToRange(child)
				res.Steps = doc.parseSteps(valueNode)

			case "executor":
				res.Executor, res.ExecutorRange, res.ExecutorParameters = doc.parseExecutorRef(valueNode, child)

			case "parameters":
				res.ParametersRange = doc.NodeToRange(child)
				res.Parameters = doc.parseParameters(valueNode)

			case "docker":
				res.Docker = doc.parseSingleExecutorDocker(keyNode, blockMappingNode)
				res.DockerRange = doc.NodeToRange(child)

			case "machine":
				machineNode = child
				machineNodeFound = true

				res.Machine = doc.parseSingleExecutorMachine(keyNode, blockMappingNode)
				res.MachineRange = doc.NodeToRange(child)

			case "macos":
				res.MacOS = doc.parseSingleExecutorMacOS(keyNode, blockMappingNode)
				res.MacOSRange = doc.NodeToRange(child)

			case "environment":
				blockMapping := GetChildMapping(valueNode)
				res.Environment = doc.parseDictionary(blockMapping)
				res.EnvironmentRange = doc.NodeToRange(child)

			case "type":
				res.Type = doc.GetNodeText(valueNode)
				res.TypeRange = doc.NodeToRange(child)

			case "plan_name":
				res.PlanName = doc.GetNodeText(valueNode)
				res.PlanNameRange = doc.NodeToRange(child)

			case "key":
				res.Key = doc.GetNodeText(valueNode)
				res.KeyRange = doc.NodeToRange(child)

			}
		}
	})

	if machineNodeFound {
		doc.addedMachineTrueDeprecatedDiag(machineNode, res.ResourceClass)
	}
	doc.jobCompletionItem(res, keys)

	return res
}

// jobCompletionItem sets the keys completion offers in a job: those its type
// allows that it doesn't have.
func (doc *YamlDocument) jobCompletionItem(job ast2.Job, has map[string]bool) {
	block := []string{":", "\n", "\t"}
	scalar := []string{":", " "}
	offer := func(key string, commitCharacters []string) {
		if !has[key] {
			job.AddCompletionItem(key, commitCharacters)
		}
	}

	switch job.Type {
	case "release":
		offer("plan_name", scalar)
	case "lock", "unlock":
		offer("key", scalar)
		offer("parameters", block)
	case "approval", "no-op":
	case "", "build":
		offer("steps", block)
		offer("description", scalar)
		// A job runs on one executor, named or given in place.
		if !has["executor"] && !has["docker"] && !has["machine"] && !has["macos"] {
			offer("executor", scalar)
			offer("docker", block)
			offer("machine", block)
			offer("macos", block)
		}
		offer("resource_class", scalar)
		offer("shell", scalar)
		offer("working_directory", scalar)
		offer("environment", block)
		offer("parameters", block)
		offer("parallelism", scalar)
		offer("circleci_ip_ranges", scalar)
		offer("retention", block)
	}

	offer("type", scalar)
}
