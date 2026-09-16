package validate

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
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

func TestOutdatedComponent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		pin    string
		latest string
		want   string
	}{
		{name: "major pin behind on major", pin: "6", latest: "7.2.1", want: "major"},
		{name: "major pin current despite newer minor", pin: "5", latest: "5.4.2", want: ""},
		{name: "major pin current despite newer patch", pin: "0", latest: "0.1.11", want: ""},
		{name: "major.minor pin behind on minor", pin: "7.1", latest: "7.2.1", want: "minor"},
		{name: "major.minor pin behind on major", pin: "7.1", latest: "8.0.0", want: "major"},
		{name: "major.minor pin current despite newer patch", pin: "7.1", latest: "7.1.9", want: ""},
		{name: "exact pin behind on patch", pin: "7.1.0", latest: "7.1.9", want: "patch"},
		{name: "exact pin behind on minor", pin: "7.1.0", latest: "7.2.1", want: "minor"},
		{name: "exact pin behind on major", pin: "7.1.0", latest: "8.0.0", want: "major"},
		{name: "exact pin current", pin: "7.2.1", latest: "7.2.1", want: ""},
		{name: "invalid pin", pin: "volatile", latest: "1.0.0", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := outdatedComponent(tt.pin, tt.latest)
			assert.Check(t, cmp.Equal(got, tt.want))
		})
	}
}

func TestDiagnosticVersionPartialPins(t *testing.T) {
	t.Parallel()

	t.Run("major-only pin ignores newer patch and minor", func(t *testing.T) {
		t.Parallel()

		message, severity := DiagnosticVersion("0", InfoVersions{
			LatestVersion:      "0.1.11",
			LatestMinorVersion: "0.1.11",
			LatestPatchVersion: "0.1.11",
		})
		assert.Check(t, cmp.Equal(message, ""))
		assert.Check(t, cmp.Equal(severity, protocol.DiagnosticSeverityInformation))
	})

	t.Run("major-only pin reports a newer major", func(t *testing.T) {
		t.Parallel()

		message, severity := DiagnosticVersion("6", InfoVersions{
			LatestVersion:      "7.2.1",
			LatestMinorVersion: "6.9.0",
			LatestPatchVersion: "6.9.0",
		})
		assert.Check(t, cmp.Contains(message, "newer major"))
		assert.Check(t, cmp.Contains(message, "7.2.1"))
		assert.Check(t, cmp.Equal(severity, protocol.DiagnosticSeverityInformation))
	})

	t.Run("major.minor pin ignores newer patch", func(t *testing.T) {
		t.Parallel()

		message, severity := DiagnosticVersion("7.1", InfoVersions{
			LatestVersion:      "7.1.9",
			LatestMinorVersion: "7.1.9",
			LatestPatchVersion: "7.1.9",
		})
		assert.Check(t, cmp.Equal(message, ""))
		assert.Check(t, cmp.Equal(severity, protocol.DiagnosticSeverityInformation))
	})

	t.Run("major.minor pin reports a newer minor", func(t *testing.T) {
		t.Parallel()

		message, severity := DiagnosticVersion("7.1", InfoVersions{
			LatestVersion:      "7.2.1",
			LatestMinorVersion: "7.2.1",
			LatestPatchVersion: "7.1.0",
		})
		assert.Check(t, cmp.Contains(message, "newer minor"))
		assert.Check(t, cmp.Contains(message, "7.2.1"))
		assert.Check(t, cmp.Equal(severity, protocol.DiagnosticSeverityInformation))
	})

	t.Run("exact pin still warns on a newer patch", func(t *testing.T) {
		t.Parallel()

		message, severity := DiagnosticVersion("7.1.0", InfoVersions{
			LatestVersion:      "7.1.1",
			LatestMinorVersion: "7.1.1",
			LatestPatchVersion: "7.1.1",
		})
		assert.Check(t, cmp.Contains(message, "newer patched"))
		assert.Check(t, cmp.Equal(severity, protocol.DiagnosticSeverityWarning))
	})
}
