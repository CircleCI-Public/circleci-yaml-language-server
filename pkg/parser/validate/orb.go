package validate

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"golang.org/x/mod/semver"
)

func (val Validate) ValidateOrbs() {
	if len(val.Doc.Orbs) == 0 && len(val.Doc.LocalOrbs) == 0 && !utils.IsDefaultRange(val.Doc.OrbsRange) {
		val.addDiagnostic(
			utils.CreateEmptyAssignationWarning(val.Doc.OrbsRange),
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

	if hasParam, _ := utils.CheckIfParamIsPartiallyReferenced(orb.Url.Version); hasParam {
		return
	}

	orbVersion, err := parser.GetOrbInfo(orb.Url.GetOrbID(), val.Cache, val.Context)

	if err != nil && strings.HasPrefix(err.Error(), "could not find orb") {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			orb.Range,
			fmt.Sprintf("Cannot find remote orb %s", orb.Url.GetOrbID()),
		))

		return
	}

	// Adding diagnostics based on versions
	if orbVersion.ID == "" {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			orb.Range,
			"Orb or version not found",
		))

		return
	}

	message, severity := DiagnosticVersion(
		orbVersion.Version,
		InfoVersions{
			LatestVersion:      orbVersion.LatestVersion,
			LatestMinorVersion: orbVersion.LatestMinorVersion,
			LatestPatchVersion: orbVersion.LatestPatchVersion,
		},
	)

	if message == "" {
		return
	}

	val.addDiagnostic(
		utils.CreateDiagnosticFromRange(
			orb.Range,
			severity,
			message,
			val.createCodeActions(orb, *orbVersion),
		),
	)
}

type OrbVersionCodeActionCreator struct {
	OrbVersion     string
	CodeActionText string
}

func (val Validate) createCodeActions(orb ast.Orb, cachedOrb ast.CachedOrb) []protocol.CodeAction {
	res := []protocol.CodeAction{}
	versions := []OrbVersionCodeActionCreator{
		{
			OrbVersion:     cachedOrb.LatestPatchVersion,
			CodeActionText: "Update to last patch",
		},
		{
			OrbVersion:     cachedOrb.LatestMinorVersion,
			CodeActionText: "Update to last minor",
		},
		{
			OrbVersion:     cachedOrb.LatestVersion,
			CodeActionText: "Update to last version",
		},
	}

	for _, version := range versions {
		if semver.Compare("v"+orb.Url.Version, "v"+version.OrbVersion) == -1 {
			res = append(res, utils.CreateCodeActionTextEdit(
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
		if val.checkIfStepsContainOrb(job.Steps, orb.Name) {
			return true
		}
	}

	for _, workflow := range val.Doc.Workflows {
		for _, jobRef := range workflow.JobRefs {
			if val.Doc.IsGivenOrb(jobRef.JobName, orb.Name) {
				return true
			}

			steps := jobRef.PostSteps
			steps = append(steps, jobRef.PreSteps...)

			if val.checkIfStepsContainOrb(steps, orb.Name) {
				return true
			}
		}
	}

	return false
}

func (val Validate) orbIsUnused(orb ast.Orb) {
	val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
		orb.Range,
		"Orb is unused",
	))
}

func (val Validate) validateOrbExecutor(executorName string, executorRange protocol.Range) {
	splittedName := strings.Split(executorName, "/")

	orb := val.Doc.Orbs[splittedName[0]]
	remoteOrb, err := parser.GetOrbInfo(orb.Url.GetOrbID(), val.Cache, val.Context)
	if err != nil {
		val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
			executorRange,
			fmt.Sprintf("Invalid orb or error trying to fetch it: %+v", err),
		))
		return
	}

	if _, ok := remoteOrb.Executors[splittedName[1]]; !ok {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executorRange,
			fmt.Sprintf("Cannot find executor %s in orb %s", splittedName[1], splittedName[0]),
		))
	}
}
