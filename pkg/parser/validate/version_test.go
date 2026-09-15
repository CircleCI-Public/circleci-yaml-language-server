package validate

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsExactOrbVersionPin(t *testing.T) {
	t.Parallel()

	assert.Check(t, isExactOrbVersionPin("1.2.3"))
	assert.Check(t, isExactOrbVersionPin("0.1.11"))
	assert.Check(t, isExactOrbVersionPin("1.2.3-rc.1"))
	assert.Check(t, !isExactOrbVersionPin("1"))
	assert.Check(t, !isExactOrbVersionPin("0"))
	assert.Check(t, !isExactOrbVersionPin("1.2"))
	assert.Check(t, !isExactOrbVersionPin("volatile"))
	assert.Check(t, !isExactOrbVersionPin("dev:alpha"))
}
