package parser

import (
	"strconv"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseJobs(jobsNode *sitter.Node) {
	// jobsNode is of type block_node
	blockMappingNode := GetChildMapping(jobsNode)
	if blockMappingNode == nil {
		return
	}

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		job := doc.parseSingleJob(child)
		if definedJob, ok := doc.Jobs[job.Name]; ok {
			doc.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Range:    job.NameRange,
				Message:  "Job already defined",
				Source:   "cci-language-server",
			})
			doc.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Range:    definedJob.NameRange,
				Message:  "Job already defined",
				Source:   "cci-language-server",
			})
			return
		}

		doc.Jobs[job.Name] = job
	})
}

func (doc *YamlDocument) parseSingleJob(jobNode *sitter.Node) ast.Job {
	// jobNode is a block_mapping_pair
	jobNameNode, valueNode := getKeyValueNodes(jobNode)
	res := ast.Job{CompletionItem: &[]protocol.CompletionItem{}, Parallelism: -1}

	if jobNameNode == nil || valueNode == nil {
		return res
	}
	jobName := doc.GetNodeText(jobNameNode)
	blockMappingNode := GetChildMapping(valueNode)

	if blockMappingNode == nil { //TODO: deal with errors
		return res
	}
	res.Name = jobName
	res.Range = NodeToRange(jobNode)
	res.NameRange = NodeToRange(jobNameNode)

	machineNode := &sitter.Node{}
	machineNodeFound := false

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		if child.Type() == "block_mapping_pair" || child.Type() == "flow_pair" {
			keyNode, valueNode := getKeyValueNodes(child)
			if keyNode == nil || valueNode == nil {
				return
			}
			keyName := doc.GetNodeText(keyNode)
			switch keyName {
			case "shell":
				res.Shell = doc.GetNodeText(valueNode)

			case "working_directory":
				res.WorkingDirectory = doc.GetNodeText(valueNode)

			case "description":
				res.Description = doc.GetNodeText(valueNode)

			case "parallelism":
				parsedInt, err := strconv.ParseInt(doc.GetNodeText(valueNode), 10, 8)
				if err != nil {
					return
				}

				res.Parallelism = int(parsedInt)
				res.ParallelismRange = NodeToRange(child)
			case "resource_class":
				res.ResourceClass = doc.GetNodeText(valueNode)

			case "steps":
				res.StepsRange = NodeToRange(child)
				res.Steps = doc.parseSteps(valueNode)

			case "executor":
				res.Executor, res.ExecutorRange, res.ExecutorParameters = doc.parseExecutorRef(valueNode, child)

			case "parameters":
				res.ParametersRange = NodeToRange(child)
				res.Parameters = doc.parseParameters(valueNode)

			case "docker":
				res.Docker = doc.parseSingleExecutorDocker(keyNode, blockMappingNode)
				res.DockerRange = NodeToRange(child)

			case "machine":
				machineNode = child
				machineNodeFound = true
			}
		}

	})

	if machineNodeFound {
		doc.addedMachineTrueDeprecatedDiag(machineNode, res.ResourceClass)
	}
	doc.jobCompletionItem(res)

	return res
}

func (doc *YamlDocument) jobCompletionItem(job ast.Job) {
	if job.Steps == nil {
		job.AddCompletionItem("steps", []string{":", "\n", "\t"})
	}
	if job.Description == "" {
		job.AddCompletionItem("description", []string{":", " "})
	}
	if job.Executor == "" {
		job.AddCompletionItem("executor", []string{":", " "})
		if job.ResourceClass == "" {
			job.AddCompletionItem("resource_class", []string{":", " "})
		}
		if job.Shell == "" {
			job.AddCompletionItem("shell", []string{":", " "})
		}
		if job.WorkingDirectory == "" {
			job.AddCompletionItem("working_directory", []string{":", " "})
		}
	}
	if job.Parameters == nil {
		job.AddCompletionItem("parameters", []string{":", "\n", " "})
	}
	if job.Parallelism == 0 {
		job.AddCompletionItem("parallelism", []string{":", " "})
	}
}
