package validate

import (
	"errors"
	"fmt"
	"strings"

	"go.lsp.dev/protocol"
	"golang.org/x/mod/semver"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/orburl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (val Validate) ValidateOrbs() {
	if len(val.Doc.Orbs) == 0 && len(val.Doc.LocalOrbs) == 0 && !position.IsDefaultRange(val.Doc.OrbsRange) {
		val.addDiagnostic(
			diagnostic.EmptyAssignationWarning(val.Doc.OrbsRange),
		)

		return
	}

	for _, orb := range val.Doc.Orbs {
		val.validateSingleOrb(orb)
	}
}

func (val Validate) validateSingleOrb(orb ast.Orb) {
	if !val.checkIfOrbIsUsed(orb) {
		val.orbIsUnused(orb)
	}

	if hasParam, _ := paramref.IsPartiallyReferenced(orb.Url.Version); hasParam {
		return
	}

	if orb.Url.IsURL {
		val.validateURLOrb(orb)
		return
	}

	if !orb.Url.IsLocal && !val.Doc.DoesOrbExist(orb, val.Cache) {
		message := fmt.Sprintf("Orb %s does not exist or is private.", orb.Url.Name)

		if val.Context.IsCciExtension && val.Context.Api.Token == "" {
			message += " Authenticate via the VS Code extension to access your private orbs."
		}

		val.addDiagnostic(
			diagnostic.Error(
				orb.Range,
				message,
			),
		)

		return
	}

	orbVersion, err := val.Doc.GetOrFetchOrbInfo(orb, val.Cache)

	if err != nil {
		if strings.HasPrefix(err.Error(), "could not find orb") {
			val.addDiagnostic(diagnostic.Error(
				orb.Range,
				fmt.Sprintf("Unknown version %s for orb %s", orb.Url.Version, orb.Url.Name),
			))
		} else {
			val.addDiagnostic(diagnostic.Error(
				orb.Range,
				fmt.Sprintf("error while retrieving orb %s", orb.Url.GetOrbID()),
			))
		}
	}

	// Adding diagnostics based on versions
	if orbVersion == nil {
		val.addDiagnostic(diagnostic.Error(
			orb.Range,
			"Orb or version not found",
		))

		return
	}

	// Compare the pin as written. Partial pins (@N, @N.M) are prefix
	// ranges: DiagnosticVersion only flags components the pin specifies
	// (PIPE-9822), so @5 is not treated as 5.0.0.
	if semver.IsValid("v" + orb.Url.Version) {
		message, severity := DiagnosticVersion(
			orb.Url.Version,
			InfoVersions{
				LatestVersion:      orbVersion.RemoteInfo.LatestVersion,
				LatestMinorVersion: orbVersion.RemoteInfo.LatestMinorVersion,
				LatestPatchVersion: orbVersion.RemoteInfo.LatestPatchVersion,
			},
		)

		if message == "" {
			return
		}

		val.addDiagnostic(
			diagnostic.New(
				orb.Range,
				severity,
				message,
				val.createCodeActions(orb, *orbVersion),
			),
		)
	}
}

// validateURLOrb warns about an orb referenced by URL that can't be fetched.
// That isn't an error: the compiler fetches it with the credential its
// organization's URL orb allow-list gives the URL's prefix, which the server
// has no way to read. Nothing used from the orb can be checked, so its
// components are skipped (IsFromUnfetchableOrb).
func (val Validate) validateURLOrb(orb ast.Orb) {
	orbInfo, err := parser.GetURLOrbInfo(orb.Url.Name, val.Cache, val.Context)

	var reason string
	switch {
	case errors.Is(err, orburl.ErrNotHTTPS):
		reason = "Orbs are only fetched over https"
	case err != nil:
		reason = fmt.Sprintf("Could not fetch this orb (%s)", err)
	case orbInfo == nil:
		reason = "This orb could not be fetched: it doesn't exist, or it is private"
	default:
		return
	}

	message := reason + ", so nothing used from it is checked."
	if orbInfo == nil && err == nil && orburl.NeedsGitHubToken(val.Context.OrbURLs, orb.Url.Name) {
		message += " Set GH_TOKEN or GITHUB_TOKEN for the language server to fetch private orbs from GitHub."
	}

	val.addDiagnostic(diagnostic.Warning(orb.ValueRange, message))
}

type OrbVersionCodeActionCreator struct {
	OrbVersion     string
	CodeActionText string
}

func (val Validate) createCodeActions(orb ast.Orb, cachedOrb ast.OrbInfo) []protocol.CodeAction {
	if !isExactOrbVersionPin(orb.Url.Version) {
		return []protocol.CodeAction{}
	}

	res := []protocol.CodeAction{}
	versions := []OrbVersionCodeActionCreator{
		{
			OrbVersion:     cachedOrb.RemoteInfo.LatestPatchVersion,
			CodeActionText: "Update to last patch",
		},
		{
			OrbVersion:     cachedOrb.RemoteInfo.LatestMinorVersion,
			CodeActionText: "Update to last minor",
		},
		{
			OrbVersion:     cachedOrb.RemoteInfo.LatestVersion,
			CodeActionText: "Update to last version",
		},
	}

	for _, version := range versions {
		if semver.Compare("v"+orb.Url.Version, "v"+version.OrbVersion) == -1 {
			res = append(res, codeaction.TextEdit(
				version.CodeActionText,
				val.Doc.URI,
				[]protocol.TextEdit{
					{
						Range:   orb.VersionRange,
						NewText: version.OrbVersion,
					},
				}, false))
		}
	}

	return res
}

func (val Validate) checkIfOrbIsUsed(orb ast.Orb) bool {
	for _, command := range val.Doc.Commands {
		if val.checkIfStepsContainOrb(command.Steps, orb.Name) {
			return true
		}
	}

	for _, job := range val.Doc.Jobs {
		if val.checkIfJobUseOrb(job, orb.Name) {
			return true
		}
	}

	for _, workflow := range val.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			if val.Doc.IsGivenOrb(jobInvocation.JobName, orb.Name) {
				return true
			}

			steps := jobInvocation.PostSteps
			steps = append(steps, jobInvocation.PreSteps...)

			if val.checkIfStepsContainOrb(steps, orb.Name) {
				return true
			}

			if val.checkIfJobParamContainOrb(jobInvocation.Parameters, orb.Name) {
				return true
			}
		}
	}

	return false
}

func (val Validate) orbIsUnused(orb ast.Orb) {
	val.addDiagnostic(diagnostic.Warning(
		orb.Range,
		"Orb is unused",
	))
}

func (val Validate) validateOrbExecutor(executorName string, executorRange protocol.Range) {
	if val.Doc.IsFromUnfetchableOrb(executorName, val.Cache) {
		return
	}

	orbExecutorExist, err := val.doesOrbExecutorExist(executorName, executorRange)
	if !orbExecutorExist && err == nil {
		splittedName := strings.Split(executorName, "/")
		val.addDiagnostic(diagnostic.Error(
			executorRange,
			fmt.Sprintf("Cannot find executor %s in orb %s", splittedName[1], splittedName[0]),
		))
	}
}

func (val Validate) doesOrbExecutorExist(executorName string, executorRange protocol.Range) (bool, error) {
	splittedName := strings.Split(executorName, "/")

	if len(splittedName) != 2 {
		// Not an orb
		return false, nil
	}

	orb, ok := val.Doc.Orbs[splittedName[0]]
	if !ok {
		err := fmt.Errorf("unknown orb referenced: %s", splittedName[0])
		val.addDiagnostic(diagnostic.Warning(
			executorRange,
			err.Error(),
		))
		return false, err
	}

	remoteOrb, err := parser.GetOrbInfo(orb.Url.GetOrbID(), val.Cache, val.Context)
	if err != nil {
		val.addDiagnostic(diagnostic.Warning(
			executorRange,
			fmt.Sprintf("Invalid orb or error trying to fetch it: %+v", err),
		))
		return false, err
	}

	_, ok = remoteOrb.Executors[splittedName[1]]
	return ok, nil
}

func (val Validate) ValidateLocalOrbs() {
	for _, orb := range val.Doc.Orbs {
		if orb.Url.IsLocal && !orb.IsPlaceholder {
			orbInfo, err := val.Doc.GetOrFetchOrbInfo(orb, val.Cache)

			if err != nil {
				continue
			}

			validateStruct := Validate{
				APIs:        val.APIs,
				Doc:         val.Doc.FromOrbParsedAttributesToYamlDocument(orbInfo.OrbParsedAttributes),
				Diagnostics: val.Diagnostics,
				Cache:       val.Cache,
				Context:     val.Context,
				IsLocalOrb:  true,
			}
			validateStruct.Validate()

			for _, job := range orbInfo.Jobs {
				job.Name = fmt.Sprintf("%s/%s", orb.Name, job.Name)
				val.checkAndReportUnusedJob(job)
			}

			for _, command := range orbInfo.Commands {
				// The orb's own commands and jobs call it by its bare name,
				// and the rest of the config by the orb's name.
				if validateStruct.checkIfCommandIsUsed(command) {
					continue
				}

				command.Name = fmt.Sprintf("%s/%s", orb.Name, command.Name)
				if !val.checkIfCommandIsUsed(command) {
					val.commandIsUnused(command)
				}
			}
		}
	}
}
