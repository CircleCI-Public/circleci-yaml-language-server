package bench

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// fixture is a generated config and the places in it the benchmarks ask
// about. It is generated rather than checked in so that its size is a
// parameter, and the same inputs always give the same config.
type fixture struct {
	name   string
	config string
	jobs   int
	// hover is on a step that runs one of the config's own commands.
	hover protocol.Position
	// completion is at the end of a job named in the workflow, where the
	// config's job names are offered.
	completion protocol.Position
	// withNewStep is the config with a step started after a job's first
	// one, as it is just after typing "- ", and newStep is at the end of it,
	// where the steps a job can run are offered.
	withNewStep string
	newStep     protocol.Position
}

// newStepLine is a step that has been started and not yet named.
const newStepLine = "      - "

// maxCommands is how many commands a fixture defines at most. Jobs take turns
// running them, so a large config does not get a command per job, and a small
// one has none that go unused.
const maxCommands = 10

var (
	// small is the size of a typical project's config.
	small = generate("small", 5, 5)
	// large is the size of the largest configs seen in practice: 200 jobs and
	// 5,000 steps.
	large = generate("large", 200, 25)

	fixtures = []fixture{small, large}
)

// generate writes a config with jobs jobs of steps run steps each, every job
// requiring the one before it in a single workflow.
func generate(name string, jobs, steps int) fixture {
	g := &generator{}
	f := fixture{name: name, jobs: jobs}

	commands := min(jobs, maxCommands)

	g.line("version: 2.1")
	g.line("")
	g.line("commands:")
	for c := range commands {
		g.line("  setup-%d:", c)
		g.line("    description: Set up the tools, variant %d.", c)
		g.line("    parameters:")
		g.line("      version:")
		g.line("        type: string")
		g.line("        default: latest")
		g.line("    steps:")
		g.line("      - run: echo setting up << parameters.version >>")
	}
	g.line("")
	g.line("jobs:")
	for j := range jobs {
		g.line("  job-%d:", j)
		g.line("    machine:")
		g.line("      image: ubuntu-2404:current")
		g.line("    steps:")
		g.line("      - checkout")
		if j == jobs/2 {
			f.hover = g.position("      - ", 2)
		}
		g.line("      - setup-%d", j%commands)
		for s := range steps {
			g.line("      - run:")
			g.line("          name: Step %d", s)
			g.line("          command: echo job %d step %d", j, s)
		}
	}
	g.line("")
	g.line("workflows:")
	g.line("  main:")
	g.line("    jobs:")
	g.line("      - job-0")
	for j := 1; j < jobs; j++ {
		g.line("      - job-%d:", j)
		g.line("          requires:")
		if j == jobs/2 {
			f.completion = g.position(fmt.Sprintf("            - job-%d", j-1), 0)
		}
		g.line("            - job-%d", j-1)
	}

	f.config = g.String()

	lines := strings.SplitAfter(f.config, "\n")
	f.withNewStep = strings.Join(slices.Insert(lines, int(f.hover.Line), newStepLine+"\n"), "")
	f.newStep = protocol.Position{Line: f.hover.Line, Character: uint32(len(newStepLine))}

	return f
}

type generator struct {
	strings.Builder
	lines uint32
}

func (g *generator) line(format string, args ...any) {
	_, _ = fmt.Fprintf(g, format+"\n", args...)
	g.lines++
}

// position is on the next line written, offset characters past prefix.
func (g *generator) position(prefix string, offset int) protocol.Position {
	return protocol.Position{Line: g.lines, Character: uint32(len(prefix) + offset)}
}

// TestFixtures checks that the positions a fixture names are on what the
// benchmarks expect to find there. It needs no server, so unlike the
// benchmarks it runs with the rest of the tests.
func TestFixtures(t *testing.T) {
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			lines := strings.Split(f.config, "\n")

			hoverLine := lines[f.hover.Line]
			assert.Check(t, cmp.Regexp(`^      - setup-\d+$`, hoverLine))
			assert.Check(t, cmp.Equal(hoverLine[f.hover.Character:f.hover.Character+2], "tu"),
				"hover should be inside the command's name")

			completionLine := lines[f.completion.Line]
			assert.Check(t, cmp.Regexp(`^            - job-\d+$`, completionLine))
			assert.Check(t, cmp.Equal(int(f.completion.Character), len(completionLine)),
				"completion should be at the end of the job's name")

			newStepLines := strings.Split(f.withNewStep, "\n")
			assert.Check(t, cmp.Equal(newStepLines[f.newStep.Line-1], "      - checkout"))
			assert.Check(t, cmp.Equal(newStepLines[f.newStep.Line], newStepLine))
			assert.Check(t, cmp.Equal(int(f.newStep.Character), len(newStepLine)))
		})
	}
}
