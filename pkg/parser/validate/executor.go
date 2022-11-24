package validate

import (
	"fmt"
	"strings"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (val Validate) ValidateExecutors() {
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
	"14.1.0",
	"14.0.1",
	"13.4.1",
	"13.3.1",
	"13.2.1",
	"13.1.0",
	"13.0.0",
	"12.5.1",
	"11.7.0",
}

var ValidMacOSResourceClasses = []string{
	"medium",
	"macos.x86.medium.gen2",
	"large",
	"macos.x86.metal.gen1",
}

func (val Validate) validateMacOSExecutor(executor ast.MacOSExecutor) {
	if utils.FindInArray(ValidXCodeVersions, executor.Xcode) == -1 {
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
	} else {
		val.validateLinuxMachineExecutor(executor)
	}
}

var ValidARMResourceClasses = []string{
	"arm.medium",
	"arm.large",
	"arm.xlarge",
	"arm.2xlarg",
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

var ValidLinuxResourceClasses = []string{
	"medium",
	"large",
	"xlarge",
	"2xlarge",
}

func (val Validate) validateLinuxMachineExecutor(executor ast.MachineExecutor) {
	val.checkIfValidResourceClass(executor.ResourceClass, ValidLinuxResourceClasses, executor.ResourceClassRange)

	if executor.Image != "" {
		val.validateImage(executor.Image, executor.ImageRange)
	} else if !executor.IsDeprecated {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			executor.Range,
			"Missing image",
		))
	}
}

var ValidARMOrMachineImages = []string{
	"ubuntu-2004:current",
	"ubuntu-2004:2022.04.1",
	"ubuntu-2004:202201-02",
	"ubuntu-2004:202201-01",
	"ubuntu-2004:202111-02",
	"ubuntu-2004:202111-01",
	"ubuntu-2004:202107-01",
	"ubuntu-2004:202104-01",
	"ubuntu-2004:202101-01",
	"ubuntu-2004:202011-01",
}

func (val Validate) validateImage(img string, imgRange protocol.Range) {
	if utils.FindInArray(ValidARMOrMachineImages, img) == -1 {
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
		isValid, errMessage := ValidateDockerImage(&img, &val.Cache.DockerCache)

		if !isValid {
			val.addDiagnostic(
				utils.CreateErrorDiagnosticFromRange(
					img.ImageRange,
					errMessage,
				),
			)
		}
	}
}

// WindowsExecutor

func (val Validate) validateWindowsExecutor(executor ast.WindowsExecutor) {
	// Same resource class as Linux
	val.checkIfValidResourceClass(executor.ResourceClass, ValidLinuxResourceClasses, executor.ResourceClassRange)
}

func (val Validate) checkIfValidResourceClass(resourceClass string, validResourceClasses []string, resourceClassRange protocol.Range) {
	if !utils.CheckIfOnlyParamUsed(resourceClass) && resourceClass != "" && utils.FindInArray(validResourceClasses, resourceClass) == -1 {

		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			resourceClassRange,
			fmt.Sprintf("Invalid resource class: \"%s\"", resourceClass),
		))
	}
}
