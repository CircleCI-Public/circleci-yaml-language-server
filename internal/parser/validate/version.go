package validate

import (
	"strings"

	"go.lsp.dev/protocol"
	"golang.org/x/mod/semver"
)

// semverCore strips prerelease and build metadata so "1.2.3-rc.1+build"
// compares as "1.2.3".
func semverCore(version string) string {
	core := version
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}
	return core
}

// isExactOrbVersionPin reports whether version is a fully specified
// major.minor.patch pin (optional prerelease/build metadata allowed).
// Partial pins such as "1" or "1.2" already track the latest compatible
// release; code actions must not rewrite them to an exact x.y.z.
func isExactOrbVersionPin(version string) bool {
	if !semver.IsValid("v" + version) {
		return false
	}

	return strings.Count(semverCore(version), ".") == 2
}

type InfoVersions struct {
	LatestVersion      string
	LatestMinorVersion string
	LatestPatchVersion string
}

// outdatedComponent reports whether latest is newer than pin in a component
// the pin actually specifies.
//
// CircleCI orb pins are prefix ranges:
//
//	"1"     tracks any 1.y.z
//	"1.2"   tracks any 1.2.z
//	"1.2.3" is exact
//
// Comparison walks major → minor → patch, but only as deep as the pin.
// Differences in unspecified components are in-range and return "".
// A shorter pin that matches the corresponding prefix of latest is current.
func outdatedComponent(pin, latest string) string {
	pinV := "v" + semverCore(pin)
	latestV := "v" + semverCore(latest)
	if !semver.IsValid(pinV) || !semver.IsValid(latestV) {
		return ""
	}

	levels := []struct {
		name  string
		trunc func(string) string
	}{
		{"major", semver.Major},
		{"minor", semver.MajorMinor},
		{"patch", func(v string) string { return v }},
	}

	depth := strings.Count(semverCore(pin), ".")
	for i := 0; i <= depth && i < len(levels); i++ {
		if semver.Compare(levels[i].trunc(pinV), levels[i].trunc(latestV)) < 0 {
			return levels[i].name
		}
	}

	return ""
}

/**
 * Calculate diagnostic information about a package version.
 *    version: Version of the package to diagnostic (the pin as written)
 *    infoVersions: Several information about the given package
 */
func DiagnosticVersion(version string, infoVersions InfoVersions) (string, protocol.DiagnosticSeverity) {
	// Displaying a warning if the version is a pre-release (ex: 0.x.x),
	// and a release version exists (ex: 1.20.3)*
	if semver.Major("v"+version) == "v0" && semver.Major("v"+infoVersions.LatestVersion) != "v0" {
		return "A production version has been released. Latest: " + infoVersions.LatestVersion,
			protocol.DiagnosticSeverityWarning
	}

	// Displaying a warning if a patched version exists
	if outdatedComponent(version, infoVersions.LatestPatchVersion) == "patch" {
		text := "A newer patched version exists.\n"
		text += "- Current: " + version + "\n"

		if infoVersions.LatestVersion != infoVersions.LatestMinorVersion {
			if infoVersions.LatestMinorVersion != infoVersions.LatestPatchVersion {
				text += "- Patch:   " + infoVersions.LatestPatchVersion + "\n"
			}

			text += "- Minor:   " + infoVersions.LatestMinorVersion + "\n"
		}

		text += "- Latest:  " + infoVersions.LatestVersion

		return text, protocol.DiagnosticSeverityWarning
	}

	// Displaying an info if a new minor exists
	if outdatedComponent(version, infoVersions.LatestMinorVersion) == "minor" {
		text := "A newer minor version exists.\n"
		text += "- Current: " + version + "\n"
		if infoVersions.LatestVersion != infoVersions.LatestMinorVersion {
			text += "- Minor:   " + infoVersions.LatestMinorVersion + "\n"
		}
		text += "- Latest:  " + infoVersions.LatestVersion

		return text, protocol.DiagnosticSeverityInformation
	}

	// Displaying an info if a new major exists
	if outdatedComponent(version, infoVersions.LatestVersion) == "major" {
		text := "A newer major version exists.\n"
		text += "- Current: " + version + "\n"
		text += "- Latest:  " + infoVersions.LatestVersion

		return text, protocol.DiagnosticSeverityInformation
	}

	return "", protocol.DiagnosticSeverityInformation
}
