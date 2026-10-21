package ast_test

import (
	"testing"
	"unsafe"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

// TestSizes pins the size of the structs the editor builds on every change,
// since they are copied by value. Compile-only data belongs off these structs.
func TestSizes(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("sizes are pinned for 64-bit architectures")
	}

	sizes := []struct {
		name      string
		got, want uintptr
	}{
		{"Job", unsafe.Sizeof(ast.Job{}), 1296},
		{"Command", unsafe.Sizeof(ast.Command{}), 152},
		{"JobInvocation", unsafe.Sizeof(ast.JobInvocation{}), 424},
		{"Workflow", unsafe.Sizeof(ast.Workflow{}), 208},
		{"Run", unsafe.Sizeof(ast.Run{}), 240},
		{"NamedStep", unsafe.Sizeof(ast.NamedStep{}), 88},
		{"Steps", unsafe.Sizeof(ast.Steps{}), 80},
		{"ParameterValue", unsafe.Sizeof(ast.ParameterValue{}), 80},
	}
	for _, s := range sizes {
		assert.Check(t, cmp.Equal(s.got, s.want), s.name)
	}
}
