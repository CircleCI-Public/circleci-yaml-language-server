package hover

import (
	"fmt"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
)

func HoverInCommands(doc yamlparser.YamlDocument, path []string) string {
	if len(path) == 0 {
		return commands
	}

	commandName := path[0]
	if len(path) == 1 {
		return commandDefinition(commandName)
	}

	return ""
}

var commands string = "A command definition defines a sequence of steps as a map to be executed in a job, enabling you to reuse a single command definition across multiple jobs.\n\n" +
	"For more information see the [Reusable Config Reference Guide](https://circleci.com/docs//2.0/reusing-config/)."

func commandDefinition(name string) string {
	return fmt.Sprintf("`%s` - Command definition\n\n", name) +
		"Allowed keys:\n\n" +
		"- `steps` (required): A list of steps to run.\n" +
		"- `parameters` (optional): A list of parameters to pass to the command.\n" +
		"- `description` (optional): A description of the command.\n"
}
