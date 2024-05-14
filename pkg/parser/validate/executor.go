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
		val.validateSingleExecutor(executor)
	}
}

func (val Validate) validateSingleExecutor(executor ast.Executor) {
	switch executor := executor.(type) {
	case ast.MacOSExecutor:
		val.validateMacOSExecutor(executor)
	case ast.MachineExecutor:
		val.validateMachineExecutor(executor)
	case ast.DockerExecutor:
		val.validateDockerExecutor(executor)
	case ast.WindowsExecutor:
		val.validateWindowsExecutor(executor)
	}
}

// MacOSExecutor

var ValidXCodeVersions = []string{
	"15.3.0",
	"15.2.0",
	"15.1.0",
	"15.0.0",
	"14.3.1",
	"14.2.0",
	"14.1.0",
	"14.0.1",
	"13.4.1",
	"12.5.1",
}

var ValidMacOSResourceClasses = []string{
	"macos.x86.medium.gen2",
	"macos.m1.medium.gen1",
	"macos.m1.large.gen1",
	"macos.x86.metal.gen1",
}

func (val Validate) validateMacOSExecutor(executor ast.MacOSExecutor) {
	if !slices.Contains(ValidXCodeVersions, executor.Xcode) {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executor.XcodeRange,
			fmt.Sprintf("Invalid Xcode version %s", executor.Xcode),
		))
	}

	val.checkIfValidResourceClass(executor.ResourceClass, ValidMacOSResourceClasses, executor.ResourceClassRange)
}

// MachineExecutor

func (val Validate) validateMachineExecutor(executor ast.MachineExecutor) {
	if strings.HasPrefix(executor.ResourceClass, "arm.") {
		val.validateARMMachineExecutor(executor)
	} else if strings.HasPrefix(executor.ResourceClass, "gpu.nvidia") || strings.HasPrefix(executor.ResourceClass, "windows.gpu.nvidia") {
		val.validateNvidiaGPUMachineExecutor(executor)
	} else if strings.HasPrefix(executor.ResourceClass, "windows.") { // this is not catching all windows resource classes
		val.validateWindowsExecutor(ast.WindowsExecutor{
			BaseExecutor: executor.BaseExecutor,
			Image:        executor.Image,
		})
	} else {
		val.validateLinuxMachineExecutor(executor)
	}
}

var ValidARMResourceClasses = []string{
	"arm.medium",
	"arm.large",
	"arm.xlarge",
	"arm.2xlarge",
}

func (val Validate) validateARMMachineExecutor(executor ast.MachineExecutor) {
	val.validateImage(executor.Image, executor.ImageRange)
	val.checkIfValidResourceClass(executor.ResourceClass, ValidARMResourceClasses, executor.ResourceClassRange)
}

var ValidNvidiaGPUResourceClasses = []string{
	"gpu.nvidia.small",
	"gpu.nvidia.medium",
	"gpu.nvidia.large",
	"windows.gpu.nvidia.medium",
}

func (val Validate) validateNvidiaGPUMachineExecutor(executor ast.MachineExecutor) {
	val.checkIfValidResourceClass(executor.ResourceClass, ValidNvidiaGPUResourceClasses, executor.ResourceClassRange)
}

var ValidCommonResourceClasses = []string{
	"medium",
	"large",
	"xlarge",
	"2xlarge",
	"2xlarge+",
}

var ValidLinuxResourceClasses = ValidCommonResourceClasses

var ValidWindowsResourceClasses = append(ValidCommonResourceClasses, "windows.medium", "windows.large")

func (val Validate) validateLinuxMachineExecutor(executor ast.MachineExecutor) {
	val.checkIfValidResourceClass(executor.ResourceClass, ValidLinuxResourceClasses, executor.ResourceClassRange)

	if executor.Image != "" {
		val.validateImage(executor.Image, executor.ImageRange)
	} else if !executor.IsDeprecated && !val.Doc.IsSelfHostedRunner(executor.ResourceClass) {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executor.Range,
			"Missing image",
		))
	}
}

func (val Validate) validateImage(img string, imgRange protocol.Range) {
	if !slices.Contains(utils.ValidARMOrMachineImages, img) {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			imgRange,
			"Invalid or deprecated image",
		))
	}
}

// DockerExecutor

var ValidDockerResourceClasses = []string{
	"small",
	"medium",
	"medium+",
	"large",
	"xlarge",
	"2xlarge",
	"2xlarge+",
}

func (val Validate) validateDockerExecutor(executor ast.DockerExecutor) {
	val.checkIfValidResourceClass(executor.ResourceClass, ValidDockerResourceClasses, executor.ResourceClassRange)

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
					fmt.Sprintf("Docker image not found %s", img.Image.FullPath),
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
						fmt.Sprintf("Docker image %s has no tag %s", img.Image.FullPath, imgTag),
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

func (val Validate) validateWindowsExecutor(executor ast.WindowsExecutor) {
	// Same resource class as Linux
	val.checkIfValidResourceClass(executor.ResourceClass, ValidWindowsResourceClasses, executor.ResourceClassRange)
}

func (val Validate) checkIfValidResourceClass(resourceClass string, validResourceClasses []string, resourceClassRange protocol.Range) {
	if !utils.CheckIfOnlyParamUsed(resourceClass) && resourceClass != "" &&
		!slices.Contains(validResourceClasses, resourceClass) &&
		!val.Doc.IsSelfHostedRunner(resourceClass) {

		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			resourceClassRange,
			fmt.Sprintf("Invalid resource class: \"%s\"", resourceClass),
		))
	}

	if val.Doc.IsSelfHostedRunner(resourceClass) {
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
						Message:  fmt.Sprintf("Cannot find orb %s. Looking for executor named %s.", possibleOrbName, executor),
						Severity: protocol.DiagnosticSeverityError,
					},
				)
			} else {
				val.addDiagnostic(
					protocol.Diagnostic{
						Range:    rng,
						Message:  fmt.Sprintf("Executor `%s` does not exist", executor),
						Severity: protocol.DiagnosticSeverityError,
					},
				)
			}
		}
	}
}
