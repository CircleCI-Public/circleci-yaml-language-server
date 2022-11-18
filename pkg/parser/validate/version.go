package validate

import (
	"go.lsp.dev/protocol"
	"golang.org/x/mod/semver"
)

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
		text := "A newer patched version exists.\n\n"
		text += "Current: " + version + "\n"

		if infoVersions.LatestVersion != infoVersions.LatestMinorVersion {
			if infoVersions.LatestMinorVersion != infoVersions.LatestPatchVersion {
				text += "Patch:   " + infoVersions.LatestPatchVersion + "\n"
			}

			text += "Minor:   " + infoVersions.LatestMinorVersion + "\n"
		}

		text += "Latest:  " + infoVersions.LatestVersion + "\n"

		return text, protocol.DiagnosticSeverityWarning
	}

	// Displaying an info if a new minor exists
	if infoVersions.LatestMinorVersion != version {
		text := "A newer minor version exists.\n\n"
		text += "Current: " + version + "\n"
		if infoVersions.LatestVersion != infoVersions.LatestMinorVersion {
			text += "Minor:   " + infoVersions.LatestMinorVersion + "\n"
		}
		text += "Latest:  " + infoVersions.LatestVersion + "\n"

		return text, protocol.DiagnosticSeverityInformation
	}

	// Displaying an info if a new major exists
	if infoVersions.LatestVersion != version {
		text := "A newer major version exists.\n\n"
		text += "Current: " + version + "\n"
		text += "Latest:  " + infoVersions.LatestVersion + "\n"

		return text, protocol.DiagnosticSeverityInformation
	}

	return "", protocol.DiagnosticSeverityInformation
}
