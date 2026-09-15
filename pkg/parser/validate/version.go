package validate

import (
	"strings"

	"go.lsp.dev/protocol"
	"golang.org/x/mod/semver"
)

// isExactOrbVersionPin reports whether version is a fully specified
// major.minor.patch pin (optional prerelease/build metadata allowed).
// Partial pins such as "1" or "1.2" already track the latest compatible
// release, so upgrade diagnostics must not treat them as 1.0.0 / 1.2.0.
func isExactOrbVersionPin(version string) bool {
	if !semver.IsValid("v" + version) {
		return false
	}

	core := version
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}

	return strings.Count(core, ".") == 2
}

type InfoVersions struct {
	LatestVersion      string
	LatestMinorVersion string
	LatestPatchVersion string
}

/**
 * Calculate diagnostic information about a package version.
 *    version: Version of the package to diagnostic
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
	if infoVersions.LatestPatchVersion != version {
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
	if infoVersions.LatestMinorVersion != version {
		text := "A newer minor version exists.\n"
		text += "- Current: " + version + "\n"
		if infoVersions.LatestVersion != infoVersions.LatestMinorVersion {
			text += "- Minor:   " + infoVersions.LatestMinorVersion + "\n"
		}
		text += "- Latest:  " + infoVersions.LatestVersion

		return text, protocol.DiagnosticSeverityInformation
	}

	// Displaying an info if a new major exists
	if infoVersions.LatestVersion != version {
		text := "A newer major version exists.\n"
		text += "- Current: " + version + "\n"
		text += "- Latest:  " + infoVersions.LatestVersion

		return text, protocol.DiagnosticSeverityInformation
	}

	return "", protocol.DiagnosticSeverityInformation
}
