/**
 * Before explaining this code, it is important to know that compared to a normal yaml, an orb can
 * only declare three type of things:
 *  - commands
 *  - executors
 *  - jobs
 *
 *
 * The parsing of local orbs is as is:
 *  - convert the orb's content into a string
 *  - remove the orb's indentation so that is has the format of a real yaml
 *  - parse the orb's yaml with GetParsedYAMLWithContent
 *  - go through all the entities that are declared inside the orb and move their range according
 *    to the orb's position
 *    We do this because when GetParsedYAMLWithContent parsed the orb's yaml it does so by starting
 *    at line 0 and character 0, so all the entities need to be offseted well to match their real
 *    position in the global yaml config file
 *  - prefix the name of all the commands, executors and jobs with `{orbName}/` and add them to the
 *    global yaml scope
 *
 * Now the entities declared in the orb can be accessed with `{orbName}/{entityName}`
 *
 * This solution is possible because you can not create any commands, executor or jobs that starts
 * with `prefix/`
 * To convince yourself of this:
 *  - for commands: https://app.circleci.com/pipelines/github/circleci/circleci-vscode-extension/422
 *  - for executors: https://app.circleci.com/pipelines/github/circleci/circleci-vscode-extension/420
 *  - for jobs: we don't allow '/' in job names
 */

package parser

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type LocalOrb struct {
	Name   string
	Offset protocol.Position
}

func (doc *YamlDocument) parseLocalOrb(name string, orbNode *sitter.Node) error {
	orb := LocalOrb{
		Name:   name,
		Offset: NodeToRange(orbNode).Start,
	}
	if orbNode.Type() != "block_node" {
		return fmt.Errorf("Invalid orb body")
	}
	orbContent := doc.GetNodeText(orbNode)
	deindentedContent := removeIndentationFromText(orbContent, orb.Offset.Character)
	orbDoc, err := ParseFromContent([]byte(deindentedContent), doc.Context, doc.URI)
	if err != nil {
		return err
	}
	for name, command := range orbDoc.Commands {
		doc.Commands[fmt.Sprintf("%s/%s", orb.Name, name)] = doc.adaptCommand(orb, command)
	}
	for name, executor := range orbDoc.Executors {
		doc.Executors[fmt.Sprintf("%s/%s", orb.Name, name)] = doc.adaptExecutor(orb, executor)
	}
	for name, job := range orbDoc.Jobs {
		doc.Jobs[fmt.Sprintf("%s/%s", orb.Name, name)] = doc.adaptJob(orb, job)
	}
	for _, diagnostic := range *orbDoc.Diagnostics {
		utils.OffsetRange(&diagnostic.Range, orb.Offset)
		doc.addDiagnostic(diagnostic)
	}
	return nil
}

func removeIndentationFromText(text string, indent uint32) string {
	indentation := int(indent)
	lines := strings.Split(text, "\n")
	prefix := strings.Repeat(" ", indentation)
	for i, s := range lines[1:] {
		if len(s) > indentation && strings.HasPrefix(s, prefix) {
			lines[i+1] = s[indentation:]
		}
	}
	return strings.Join(lines, "\n")
}

func (doc *YamlDocument) adaptCommand(orb LocalOrb, command ast.Command) ast.Command {
	command.Name = fmt.Sprintf("%s/%s", orb.Name, command.Name)

	utils.OffsetRange(&command.Range, orb.Offset)
	utils.OffsetRange(&command.NameRange, orb.Offset)
	utils.OffsetRange(&command.DescriptionRange, orb.Offset)
	utils.OffsetRange(&command.StepsRange, orb.Offset)
	utils.OffsetRange(&command.ParametersRange, orb.Offset)
	for i, step := range command.Steps {
		command.Steps[i] = doc.adaptStep(orb, step)
	}
	for i, parameter := range command.Parameters {
		command.Parameters[i] = doc.adaptParameter(orb, parameter)
	}
	return command
}

func (doc *YamlDocument) adaptExecutor(orb LocalOrb, executor ast.Executor) ast.Executor {

	switch e := executor.(type) {
	case ast.BaseExecutor:
		doc.adaptBaseExecutor(orb, &e)
		return e
	case ast.DockerExecutor:
		doc.adaptBaseExecutor(orb, &e.BaseExecutor)
		for i := range e.Image {
			utils.OffsetRange(&e.Image[i].ImageRange, orb.Offset)
		}
		for i := range e.ServiceImages {
			utils.OffsetRange(&e.ServiceImages[i].ImageRange, orb.Offset)
		}
		return e
	case ast.MachineExecutor:
		doc.adaptBaseExecutor(orb, &e.BaseExecutor)
		utils.OffsetRange(&e.ImageRange, orb.Offset)
		return e
	case ast.MacOSExecutor:
		doc.adaptBaseExecutor(orb, &e.BaseExecutor)
		utils.OffsetRange(&e.XcodeRange, orb.Offset)
		return e
	case ast.WindowsExecutor:
		doc.adaptBaseExecutor(orb, &e.BaseExecutor)
		return e
	default:
		doc.addDiagnostic(utils.CreateHintDiagnosticFromRange(executor.GetRange(), "This kind of executor is not supported for local orbs"))
		return nil
	}
}

func (doc *YamlDocument) adaptBaseExecutor(orb LocalOrb, e *ast.BaseExecutor) {
	e.Name = fmt.Sprintf("%s/%s", orb.Name, e.Name)
	utils.OffsetRange(&e.Range, orb.Offset)
	utils.OffsetRange(&e.NameRange, orb.Offset)
	utils.OffsetRange(&e.ResourceClassRange, orb.Offset)
	utils.OffsetRange(&e.UserParametersRange, orb.Offset)
	for k, parameter := range e.UserParameters {
		e.UserParameters[k] = doc.adaptParameter(orb, parameter)
	}
}

func (doc *YamlDocument) adaptJob(orb LocalOrb, job ast.Job) ast.Job {
	job.Name = fmt.Sprintf("%s/%s", orb.Name, job.Name)

	if job.Executor != "" {
		job.Executor = fmt.Sprintf("%s/%s", orb.Name, job.Executor)
	}
	utils.OffsetRange(&job.Range, orb.Offset)
	utils.OffsetRange(&job.NameRange, orb.Offset)
	utils.OffsetRange(&job.ResourceClassRange, orb.Offset)
	utils.OffsetRange(&job.StepsRange, orb.Offset)
	utils.OffsetRange(&job.ExecutorRange, orb.Offset)
	utils.OffsetRange(&job.ParametersRange, orb.Offset)
	utils.OffsetRange(&job.DockerRange, orb.Offset)
	for i, step := range job.Steps {
		job.Steps[i] = doc.adaptStep(orb, step)
	}
	for k, executorParameter := range job.ExecutorParameters {
		job.ExecutorParameters[k] = doc.adaptParameterValue(orb, executorParameter)
	}
	for k, parameter := range job.Parameters {
		job.Parameters[k] = doc.adaptParameter(orb, parameter)
	}
	oldName := job.Docker.Name
	job.Docker = doc.adaptExecutor(orb, job.Docker).(ast.DockerExecutor)
	job.Docker.Name = oldName
	return job
}

func (doc *YamlDocument) adaptStep(orb LocalOrb, step ast.Step) ast.Step {
	switch s := step.(type) {
	case ast.Run:
		utils.OffsetRange(&s.Range, orb.Offset)
		utils.OffsetRange(&s.CommandRange, orb.Offset)
		return s
	case ast.Checkout:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.SetupRemoteDocker:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.SaveCache:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.RestoreCache:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.StoreArtifacts:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.StoreTestResults:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.PersistToWorkspace:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.AttachWorkspace:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.AddSSHKey:
		utils.OffsetRange(&s.Range, orb.Offset)
		return s
	case ast.NamedStep:
		if !doc.IsBuiltIn(s.Name) {
			s.Name = fmt.Sprintf("%s/%s", orb.Name, s.Name)
		}
		utils.OffsetRange(&s.ParametersRange, orb.Offset)
		utils.OffsetRange(&s.Range, orb.Offset)
		for i, parameter := range s.Parameters {
			s.Parameters[i] = doc.adaptParameterValue(orb, parameter)
		}
		return s
	default:
		doc.addDiagnostic(utils.CreateHintDiagnosticFromRange(s.GetRange(), "This kind of step is not yet supported in local orbs"))
		return s
	}
}

func (doc *YamlDocument) adaptParameter(orb LocalOrb, parameter ast.Parameter) ast.Parameter {
	switch p := parameter.(type) {
	case ast.StringParameter:
		doc.adaptBaseParameter(orb, &p.BaseParameter)
		return p
	case ast.BooleanParameter:
		doc.adaptBaseParameter(orb, &p.BaseParameter)
		return p
	case ast.IntegerParameter:
		doc.adaptBaseParameter(orb, &p.BaseParameter)
		return p
	case ast.EnumParameter:
		doc.adaptBaseParameter(orb, &p.BaseParameter)
		return p
	case ast.ExecutorParameter:
		doc.adaptBaseParameter(orb, &p.BaseParameter)
		return p
	case ast.StepsParameter:
		doc.adaptBaseParameter(orb, &p.BaseParameter)
		return p
	case ast.EnvVariableParameter:
		doc.adaptBaseParameter(orb, &p.BaseParameter)
		return p
	default:
		doc.addDiagnostic(utils.CreateHintDiagnosticFromRange(parameter.GetRange(), "This kind of parameters is not yet supported in local orbs"))
		return p
	}
}

func (doc *YamlDocument) adaptBaseParameter(orb LocalOrb, p *ast.BaseParameter) {
	utils.OffsetRange(&p.NameRange, orb.Offset)
	utils.OffsetRange(&p.Range, orb.Offset)
	utils.OffsetRange(&p.TypeRange, orb.Offset)
	utils.OffsetRange(&p.DefaultRange, orb.Offset)
}

func (doc *YamlDocument) adaptParameterValue(orb LocalOrb, parameter ast.ParameterValue) ast.ParameterValue {
	utils.OffsetRange(&parameter.Range, orb.Offset)
	utils.OffsetRange(&parameter.ValueRange, orb.Offset)
	return parameter
}
