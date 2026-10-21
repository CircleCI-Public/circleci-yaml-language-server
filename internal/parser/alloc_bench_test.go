package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

// The configs match bench/'s sizes, plus the conditional steps, logic
// statements and filters that only some configs use.
var benchConfigs = []struct {
	name   string
	config string
}{
	{"small", generateConfig(5, 5)},
	{"large", generateConfig(200, 25)},
}

func BenchmarkParseFromContent(b *testing.B) {
	for _, c := range benchConfigs {
		b.Run(c.name, func(b *testing.B) {
			content := []byte(c.config)
			b.ReportAllocs()
			for b.Loop() {
				doc, err := parser.ParseFromContent(content, testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
				if err != nil {
					b.Fatal(err)
				}
				doc.Close()
			}
		})
	}
}

// generateConfig writes a config with jobs jobs of steps run steps each, every
// job requiring the one before it in a single workflow.
func generateConfig(jobs, steps int) string {
	var g strings.Builder
	line := func(format string, args ...any) {
		_, _ = fmt.Fprintf(&g, format+"\n", args...)
	}

	commands := min(jobs, 10)

	line("version: 2.1")
	line("")
	line("parameters:")
	line("  deploy:")
	line("    type: boolean")
	line("    default: false")
	line("")
	line("commands:")
	for c := range commands {
		line("  setup-%d:", c)
		line("    description: Set up the tools, variant %d.", c)
		line("    parameters:")
		line("      version:")
		line("        type: string")
		line("        default: latest")
		line("    steps:")
		line("      - run: echo setting up << parameters.version >>")
		line("      - when:")
		line("          condition:")
		line("            equal: [latest, << parameters.version >>]")
		line("          steps:")
		line("            - run: echo latest")
	}
	line("")
	line("jobs:")
	for j := range jobs {
		line("  job-%d:", j)
		line("    machine:")
		line("      image: ubuntu-2404:current")
		line("    steps:")
		line("      - checkout")
		line("      - setup-%d", j%commands)
		for s := range steps {
			line("      - run:")
			line("          name: Step %d", s)
			line("          command: echo job %d step %d", j, s)
		}
		line("      - unless:")
		line("          condition: << pipeline.parameters.deploy >>")
		line("          steps:")
		line("            - when:")
		line("                condition:")
		line("                  and:")
		line("                    - << pipeline.git.branch >>")
		line("                    - not: << pipeline.parameters.deploy >>")
		line("                steps:")
		line("                  - run: echo job %d checks only", j)
	}
	line("")
	line("workflows:")
	line("  main:")
	line("    when:")
	line("      not:")
	line("        equal: [scheduled_pipeline, << pipeline.trigger_source >>]")
	line("    jobs:")
	line("      - job-0")
	for j := 1; j < jobs; j++ {
		line("      - job-%d:", j)
		line("          requires:")
		line("            - job-%d", j-1)
		line("          filters:")
		line("            branches:")
		line("              only: [main, /release-.*/]")
		line("            tags:")
		line("              ignore: /.*/")
	}

	return g.String()
}
