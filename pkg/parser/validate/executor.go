package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (val Validate) ValidateExecutors() {
	if len(val.Doc.Executors) == 0 && !utils.IsDefaultRange(val.Doc.ExecutorsRange) {
		val.addDiagnostic(
			utils.CreateEmptyAssignationWarning(val.Doc.ExecutorsRange),
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
	if slices.Contains(utils.ValidXcodeAppleSiliconVersions, executor.Xcode) {
		val.checkIfValidResourceClass(
			executor.ResourceClass,
			utils.ValidMacOSAppleSiliconResourceClasses,
			executor.ResourceClassRange,
			fmt.Sprintf("Xcode version \"%s\"", executor.Xcode),
		)
	} else if slices.Contains(utils.ValidXcodeIntelVersions, executor.Xcode) {
		val.checkIfValidResourceClass(
			executor.ResourceClass,
			utils.ValidMacOSIntelResourceClasses,
			executor.ResourceClassRange,
			fmt.Sprintf("Xcode version \"%s\"", executor.Xcode),
		)
	} else {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executor.XcodeRange,
			fmt.Sprintf("Invalid Xcode version \"%s\"", executor.Xcode),
		))
	}
}

// MachineExecutor

func (val Validate) validateMachineExecutor(executor ast.MachineExecutor) {
	if executor.IsDeprecated {
		return
	}

	rcParam := utils.ContainsParam(executor.ResourceClass)
	imgParam := utils.ContainsParam(executor.Image)

	if executor.Image == "" {
		if executor.ResourceClass != "" &&
			!utils.IsSelfHostedRunner(executor.ResourceClass) &&
			!rcParam &&
			!slices.Contains(utils.ValidMachineResourceClasses, executor.ResourceClass) {

			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				executor.ResourceClassRange,
				fmt.Sprintf("Unknown resource class \"%s\"", executor.ResourceClass),
			))
		}
		return
	}

	if utils.IsSelfHostedRunner(executor.ResourceClass) {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
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
	for _, pair := range utils.ValidMachinePairs {
		hasRC := slices.Contains(pair.ResourceClasses, executor.ResourceClass) ||
			rcParam
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
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executor.ResourceClassRange,
			fmt.Sprintf(
				"Unknown resource class \"%s\"",
				executor.ResourceClass,
			),
		))
	}

	if !validImage {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executor.ImageRange,
			fmt.Sprintf(
				"Unknown machine image \"%s\"",
				executor.Image,
			),
		))
	}

	if validResourceClass && validImage {
		// rc and img exist, but do not form a valid pair
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executor.Range,
			fmt.Sprintf(
				"Machine image \"%s\" is not available for resource class \"%s\"",
				executor.Image,
				executor.ResourceClass,
			),
		))
	}

	// if executor.ResourceClass != "" {
	// 	// Resource class is defined, check that the resource class/image pair
	// 	// is valid
	// 	for _, pair := range utils.ValidPairs {
	// 		if slices.Contains(pair.Images, executor.Image) {
	// 			val.checkIfValidResourceClass(
	// 				executor.ResourceClass,
	// 				pair.ResourceClasses,
	// 				executor.ResourceClassRange,
	// 				fmt.Sprintf("machine image \"%s\"", executor.Image),
	// 			)
	// 			return
	// 		}
	// 	}
	// } else if executor.Image != "" {
	// 	// No resource class, assume linux, check for valid image
	// 	if !slices.Contains(utils.ValidLinuxImages, executor.Image) {
	// 		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
	// 			executor.ImageRange,
	// 			"Invalid or deprecated machine image",
	// 		))
	// 	}
	// } else if !executor.IsDeprecated && !utils.IsSelfHostedRunner(executor.ResourceClass) {
	// 	val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
	// 		executor.Range,
	// 		"Missing machine image",
	// 	))
	// }
}

// DockerExecutor

func (val Validate) validateDockerExecutor(executor ast.DockerExecutor) {
	val.checkIfValidResourceClass(
		executor.ResourceClass,
		utils.ValidDockerResourceClasses,
		executor.ResourceClassRange,
		"Docker executor",
	)

	for _, img := range executor.Image {

		if !isDockerImageCheckable(&img) {
			// When a Docker image can't be checked, skip it (consider it valid)
			continue
		}

		imageExists := DoesDockerImageExists(&img, &val.Cache.DockerCache, val.APIs.DockerHub)
		if !imageExists {
			val.addDiagnostic(
				utils.CreateErrorDiagnosticFromRange(
					img.ImageRange,
					fmt.Sprintf(
						"Docker image not found \"%s\"",
						img.Image.FullPath,
					),
				),
			)
		} else {
			// Validate image tag
			imgTag := img.Image.Tag

			if imgTag == "" {
				imgTag = "latest"
			}

			tagExists := DoesTagExist(&img, imgTag, &val.Cache.DockerTagsCache, val.APIs.DockerHub)

			if !tagExists {
				actions := GetImageTagActions(&val.Doc, &img, &val.Cache.DockerTagsCache, val.APIs.DockerHub)
				val.addDiagnostic(
					utils.CreateDiagnosticFromRange(
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
					utils.CreateDiagnosticFromRange(
						img.ImageRange,
						protocol.DiagnosticSeverityHint,
						"It is recommended to set explicit tags",
						actions,
					),
				)
			}
		}

		if img.Image.Namespace == "circleci" {
			val.addDiagnostic(
				utils.CreateDiagnosticFromRange(
					img.ImageRange,
					protocol.DiagnosticSeverityWarning,
					"Docker images from `circleci` namespace are deprecated. Please use its `cimg` namespace's alternative.",
					[]protocol.CodeAction{
						utils.CreateCodeActionTextEdit(
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
	if !utils.CheckIfOnlyParamUsed(resourceClass) &&
		resourceClass != "" &&
		!slices.Contains(validResourceClasses, resourceClass) &&
		!utils.IsSelfHostedRunner(resourceClass) {

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
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			resourceClassRange,
			message,
		))
	}

	if utils.IsSelfHostedRunner(resourceClass) {
		namespace := strings.Split(resourceClass, "/")[0]
		val.validateExecutorNamespace(namespace, resourceClassRange)
	}
}

type RegistryNamespace struct {
	RegistryNameSpace *struct {
		Name string
	}
}

func (val Validate) validateExecutorNamespace(resourceClass string, resourceClassRange protocol.Range) {
	client := utils.NewClient(val.Context.Api.HostUrl, "graphql-unstable", val.Context.Api.Token, false)

	query := `query($name: String!) {
		registryNamespace(name: $name) {
			name
		}
	}`

	request := utils.NewRequest(query)
	request.SetToken(val.Context.Api.Token)
	request.Var("name", resourceClass)
	request.SetUserId(val.Context.UserIdForTelemetry)

	var response RegistryNamespace
	err := client.Run(request, &response)
	if err != nil {
		return
	}

	if response.RegistryNameSpace == nil {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
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
