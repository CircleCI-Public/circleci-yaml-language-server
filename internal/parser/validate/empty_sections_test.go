package validate

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestEmptySections(t *testing.T) {
	for _, section := range []string{"orbs", "parameters", "executors", "commands"} {
		t.Run(section, func(t *testing.T) {
			diags := diagnosticMessages(validateYAML(t, "version: 2.1\n"+section+":\n"))
			assert.Check(t, cmp.Contains(diags, "`"+section+"` is empty"))
		})
	}
}
