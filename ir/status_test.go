package ir

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestStatuses(t *testing.T) {
	all := []string{"canceled", "failed", "skipped", "success", "unauthorized"}
	tests := []struct {
		name    string
		written []string
		want    []string
	}{
		{name: "nil", want: []string{"success"}},
		{name: "empty", written: []string{}, want: []string{"success"}},
		{name: "single", written: []string{"failed"}, want: []string{"failed"}},
		{name: "terminal", written: []string{"terminal"}, want: all},
		{name: "terminal with others", written: []string{"success", "terminal"}, want: all},
		{name: "not_run", written: []string{"not_run"}, want: []string{"skipped"}},
		{name: "not_run and skipped", written: []string{"skipped", "not_run"}, want: []string{"skipped"}},
		{name: "sorted", written: []string{"unauthorized", "canceled", "success"}, want: []string{"canceled", "success", "unauthorized"}},
		{name: "duplicates", written: []string{"failed", "success", "failed"}, want: []string{"failed", "success"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Statuses(tt.written)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(got, tt.want))
		})
	}

	t.Run("invalid", func(t *testing.T) {
		got, err := Statuses([]string{"success", "pending"})
		assert.Check(t, cmp.Error(err, `invalid requires status "pending"`))
		assert.Check(t, cmp.Nil(got))
	})
}
