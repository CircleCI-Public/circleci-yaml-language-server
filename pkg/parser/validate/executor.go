package validate

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

func (val Validate) ValidateExecutors() {
	if len(val.Doc.Executors) == 0 && !position.IsDefaultRange(val.Doc.ExecutorsRange) {
		val.addDiagnostic(
			diagnostic.EmptyAssignationWarning(val.Doc.ExecutorsRange),
		)

		return
	}

	for _, executor := range val.Doc.Executors {
		switch executor := executor.(type) {
		case ast.MacOSExecutor:
			val.validateMacOSExecutor(executor)
		case ast.MachineExecutor:
			val.validateMachineExecutor(executor)
		case ast.DockerExecutor:
			val.validateDockerExecutor(executor)
		}
	}
}

// MacOSExecutor

func (val Validate) validateMacOSExecutor(executor ast.MacOSExecutor) {
	xcodeVersions := val.Cache.Offerings(val.Context.Api).XcodeVersions()
	if xcodeVersions == nil {
		return
	}

	if slices.Contains(xcodeVersions, executor.Xcode) {
		val.checkIfValidResourceClass(
			executor.ResourceClass,
			val.Cache.Offerings(val.Context.Api).MacOSResourceClasses(),
			executor.ResourceClassRange,
			fmt.Sprintf("Xcode version \"%s\"", executor.Xcode),
		)
	} else if slices.Contains(val.Cache.Offerings(val.Context.Api).DeprecatedXcodeVersions(), executor.Xcode) {
		val.addDiagnostic(diagnostic.Deprecated(
			executor.XcodeRange,
			fmt.Sprintf("Xcode version \"%s\" is deprecated", executor.Xcode),
		))
	} else {
		val.addDiagnostic(diagnostic.Error(
			executor.XcodeRange,
			fmt.Sprintf("Unknown Xcode version \"%s\"", executor.Xcode),
		))
	}
}

// MachineExecutor

func (val Validate) validateMachineExecutor(executor ast.MachineExecutor) {
	if executor.IsDeprecated {
		return
	}

	pairs := val.Cache.Offerings(val.Context.Api).MachinePairs()
	if pairs == nil {
		return
	}

	rcParam := paramref.Contains(executor.ResourceClass)
	imgParam := paramref.Contains(executor.Image)

	if executor.Image == "" {
		if executor.ResourceClass != "" &&
			!circleci.IsSelfHostedRunner(executor.ResourceClass) &&
			!rcParam &&
			!slices.Contains(val.Cache.Offerings(val.Context.Api).MachineResourceClasses(), executor.ResourceClass) {

			val.addDiagnostic(diagnostic.Error(
				executor.ResourceClassRange,
				fmt.Sprintf("Unknown resource class \"%s\"", executor.ResourceClass),
			))
		}
		return
	}

	if circleci.IsSelfHostedRunner(executor.ResourceClass) {
		val.addDiagnostic(diagnostic.Error(
			executor.Range,
			fmt.Sprintf(
				"Extraneous image \"%s\" for self-hosted runner \"%s\"",
				executor.Image,
				executor.ResourceClass,
			),
		))
		return
	}

	var validResourceClass bool
	var validImage bool
	for _, pair := range pairs {
		hasRC := pair.ResourceClass == executor.ResourceClass ||
			rcParam || executor.ResourceClass == ""
		hasImg := slices.Contains(pair.Images, executor.Image) ||
			imgParam

		if hasRC || rcParam {
			validResourceClass = true
		}
		if hasImg || imgParam {
			validImage = true
		}
		if hasRC && hasImg {
			// Valid (rc, img) pair, no diagnostics to add
			return
		}
	}

	if !validResourceClass {
		val.addDiagnostic(diagnostic.Error(
			executor.ResourceClassRange,
			fmt.Sprintf(
				"Unknown resource class \"%s\"",
				executor.ResourceClass,
			),
		))
	}

	if !validImage {
		if slices.Contains(val.Cache.Offerings(val.Context.Api).DeprecatedMachineImages(), executor.Image) {
			val.addDiagnostic(diagnostic.Deprecated(
				executor.ImageRange,
				fmt.Sprintf(
					"Machine image \"%s\" is deprecated",
					executor.Image,
				),
			))
		} else {
			val.addDiagnostic(diagnostic.Error(
				executor.ImageRange,
				fmt.Sprintf(
					"Unknown machine image \"%s\"",
					executor.Image,
				),
			))
		}
	}

	if validResourceClass && validImage {
		// rc and img exist, but do not form a valid pair
		val.addDiagnostic(diagnostic.Error(
			executor.Range,
			fmt.Sprintf(
				"Machine image \"%s\" is not available for resource class \"%s\"",
				executor.Image,
				executor.ResourceClass,
			),
		))
	}
}

// DockerExecutor

func (val Validate) validateDockerExecutor(executor ast.DockerExecutor) {
	if dockerResourceClasses := val.Cache.Offerings(val.Context.Api).DockerResourceClasses(); dockerResourceClasses != nil {
		val.checkIfValidResourceClass(
			executor.ResourceClass,
			dockerResourceClasses,
			executor.ResourceClassRange,
			"Docker executor",
		)
	}

	for _, img := range executor.Image {

		if !isDockerImageCheckable(&img) {
			// When a Docker image can't be checked, skip it (consider it valid)
			continue
		}

		imageExists := DoesDockerImageExists(&img, &val.Cache.DockerCache, val.APIs.DockerHub)
		if !imageExists {
			val.addDiagnostic(
				diagnostic.Error(
					img.ImageRange,
					fmt.Sprintf(
						"Docker image not found \"%s\"",
						img.Image.FullPath,
					),
				),
			)
		} else {
			// Validate digest format if present
			if img.Image.Digest != "" && !isValidDockerDigest(img.Image.Digest) {
				val.addDiagnostic(
					diagnostic.Error(
						img.ImageRange,
						fmt.Sprintf(
							"Invalid Docker image digest format \"%s\". Expected format: sha256:<64 hex characters>",
							img.Image.Digest,
						),
					),
				)
				continue
			}

			// Exclude cases with only digest without tags, such as node@sha256:...
			if img.Image.Tag != "" || img.Image.Digest == "" {
				// Validate image tag
				imgTag := img.Image.Tag

				if imgTag == "" {
					imgTag = "latest"
				}

				tagExists := DoesTagExist(&img, imgTag, &val.Cache.DockerTagsCache, val.APIs.DockerHub)

				if !tagExists {
					actions := GetImageTagActions(&val.Doc, &img, &val.Cache.DockerTagsCache, val.APIs.DockerHub)
					val.addDiagnostic(
						diagnostic.New(
							img.ImageRange,
							protocol.DiagnosticSeverityError,
							fmt.Sprintf("Docker image \"%s\" has no tag \"%s\"", img.Image.FullPath, imgTag),
							actions,
						),
					)
				}

				if tagExists && img.Image.Tag == "" {
					actions := GetImageTagActions(&val.Doc, &img, &val.Cache.DockerTagsCache, val.APIs.DockerHub)
					val.addDiagnostic(
						diagnostic.New(
							img.ImageRange,
							protocol.DiagnosticSeverityHint,
							"It is recommended to set explicit tags",
							actions,
						),
					)
				}
			}
		}

		if img.Image.Namespace == "circleci" {
			val.addDiagnostic(
				diagnostic.New(
					img.ImageRange,
					protocol.DiagnosticSeverityWarning,
					"Docker images from `circleci` namespace are deprecated. Please use its `cimg` namespace's alternative.",
					[]protocol.CodeAction{
						codeaction.TextEdit(
							"Use `cimg` namespace's alternative",
							val.Doc.URI, []protocol.TextEdit{
								{
									Range:   img.ImageRange,
									NewText: fmt.Sprintf("image: %s", strings.Replace(img.Image.FullPath, "circleci", "cimg", 1)),
								},
							}, true,
						),
					},
				),
			)
		}
	}
}

func (val Validate) checkIfValidResourceClass(
	resourceClass string,
	validResourceClasses []string,
	resourceClassRange protocol.Range,
	context string,
) {
	if !paramref.IsOnlyParameter(resourceClass) &&
		resourceClass != "" &&
		!slices.Contains(validResourceClasses, resourceClass) &&
		!circleci.IsSelfHostedRunner(resourceClass) {

		var message string
		if context == "" {
			message = fmt.Sprintf("Invalid resource class \"%s\"", resourceClass)
		} else {
			message = fmt.Sprintf(
				"Invalid resource class \"%s\" for %s",
				resourceClass,
				context,
			)
		}
		val.addDiagnostic(diagnostic.Error(
			resourceClassRange,
			message,
		))
	}

	if circleci.IsSelfHostedRunner(resourceClass) {
		namespace := strings.Split(resourceClass, "/")[0]
		val.validateExecutorNamespace(namespace, resourceClassRange)
	}
}

func (val Validate) validateExecutorNamespace(resourceClass string, resourceClassRange protocol.Range) {
	registry := val.Context.OrbRegistry()

	_, err := registry.FetchNamespace(context.Background(), resourceClass)
	if err == nil {
		return
	}

	// Only a definitive "no such namespace" earns a diagnostic. A request that
	// simply failed is not evidence the namespace is missing, and reporting one
	// would mean flagging valid configs whenever the API is unreachable.
	if circleci.IsNotFound(err) {
		val.addDiagnostic(diagnostic.Error(
			resourceClassRange,
			fmt.Sprintf("Namespace \"%s\" does not exist", resourceClass),
		))
	}
}

func (val Validate) validateExecutorReference(executor string, rng protocol.Range) {
	if !val.Doc.DoesExecutorExist(executor) {
		if val.Doc.IsOrbReference(executor) {
			val.validateOrbExecutor(executor, rng)
		} else {
			if possibleOrbName, couldBeOrbReference := val.Doc.CouldBeOrbReference(executor); couldBeOrbReference &&
				!val.Doc.IsOrbReference(executor) {
				val.addDiagnostic(
					protocol.Diagnostic{
						Range:    rng,
						Message:  fmt.Sprintf("Cannot find orb \"%s\". Looking for executor named \"%s\".", possibleOrbName, executor),
						Severity: protocol.DiagnosticSeverityError,
					},
				)
			} else {
				val.addDiagnostic(
					protocol.Diagnostic{
						Range:    rng,
						Message:  fmt.Sprintf("Executor \"%s\" does not exist", executor),
						Severity: protocol.DiagnosticSeverityError,
					},
				)
			}
		}
	}
}
