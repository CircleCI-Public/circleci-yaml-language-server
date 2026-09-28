package validate

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

func (val Validate) ValidateExecutors() {
	if len(val.Doc.Executors) == 0 && len(val.Doc.Aliases.Executors) == 0 && !position.IsDefaultRange(val.Doc.ExecutorsRange) {
		val.addDiagnostic(
			diagnostic.EmptySectionWarning(val.Doc.ExecutorsRange, "executors"),
		)

		return
	}

	for _, executor := range val.Doc.Executors {
		switch executor := executor.(type) {
		case ast.MacOSExecutor:
			val.validateMacOSExecutor(executor)
		case ast.MachineExecutor:
			val.validateMachineExecutor(executor)
			val.validateMachineMapClashes(executor, executor.ResourceClassBeside, executor.ShellBeside, true)
		case ast.DockerExecutor:
			val.validateDockerExecutor(executor)
		}
	}

	values := val.valuesOutsideExecutors()
	val.validateExecutorAliases(values)

	// Local orbs do not need unused checks because those checks collides with the overall YAML unused checks
	if !val.IsLocalOrb {
		val.checkUnusedExecutors(values)
	}
}

// checkUnusedExecutors warns about an executor or an alias whose name is not
// among values, once there are jobs to use it. An alias that is used uses its
// target.
func (val Validate) checkUnusedExecutors(values map[string]bool) {
	if len(val.Doc.Jobs) == 0 {
		return
	}

	for _, alias := range val.Doc.Aliases.Executors {
		if values[alias.Name] {
			values[alias.Target] = true
		} else {
			val.addDiagnostic(diagnostic.Warning(alias.NameRange, "Executor is unused"))
		}
	}

	for _, executor := range val.Doc.Executors {
		if !values[executor.GetName()] {
			val.addDiagnostic(diagnostic.Warning(executor.GetNameRange(), "Executor is unused"))
		}
	}
}

// valuesOutsideExecutors are the config's scalar values outside its
// executors. An executor's name can reach a job through a parameter, as an
// enum, a default or a matrix's values, so any of them is taken as a use.
func (val Validate) valuesOutsideExecutors() map[string]bool {
	values := map[string]bool{}
	for node := range yamltree.Walk(val.Doc.RootNode) {
		switch node.Kind() {
		case "plain_scalar", "single_quote_scalar", "double_quote_scalar":
		default:
			continue
		}
		if isKey(node) || position.InRange(val.Doc.ExecutorsRange, position.Start(node)) {
			continue
		}
		values[strings.Trim(val.Doc.GetNodeText(node), `"'`)] = true
	}
	return values
}

func isKey(scalar *sitter.Node) bool {
	node := scalar.Parent()
	if node == nil {
		return false
	}
	pair := node.Parent()
	if pair == nil || (pair.Kind() != "block_mapping_pair" && pair.Kind() != "flow_pair") {
		return false
	}
	key := pair.ChildByFieldName("key")
	return key != nil && key.Id() == node.Id()
}

// MacOSExecutor

func (val Validate) validateMacOSExecutor(executor ast.MacOSExecutor) {
	// A version from a parameter is only known once the config is compiled.
	if paramref.ContainsReference(executor.Xcode) {
		return
	}

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
		// The compiler doesn't check Xcode versions, and the catalog can lag
		// behind what can be scheduled, so a missing one is only a warning.
		val.addDiagnostic(diagnostic.Warning(
			executor.XcodeRange,
			fmt.Sprintf("Unknown Xcode version \"%s\"", executor.Xcode),
		))
	}
}

// MachineExecutor

// validateMachineMapClashes reports a resource_class or shell given inside a
// machine map when the job, or the executor, also gives it.
func (val Validate) validateMachineMapClashes(executor ast.MachineExecutor, resourceClassBeside, shellBeside bool, onExecutor bool) {
	message := "%s is set both on the job and inside the `machine` map; remove the one inside `machine`"
	if onExecutor {
		message = "%s is set both on the executor and inside its `machine` map; remove the one inside `machine`"
	}
	if resourceClassBeside && !position.IsDefaultRange(executor.InMapResourceClassRange) {
		val.addDiagnostic(diagnostic.Error(executor.InMapResourceClassRange, fmt.Sprintf(message, "resource_class")))
	}
	if shellBeside && !position.IsDefaultRange(executor.InMapShellRange) {
		val.addDiagnostic(diagnostic.Error(executor.InMapShellRange, fmt.Sprintf(message, "shell")))
	}
}

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

			val.addDiagnostic(diagnostic.Warning(
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

	// The compiler checks neither resource classes nor images, and the catalog
	// can lag behind what can be scheduled (canary tags, for one), so one the
	// catalog lacks is only a warning, unless its family is missing too. A pair
	// it rules out is an error.
	if !validResourceClass {
		val.addDiagnostic(diagnostic.Warning(
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
			message := fmt.Sprintf("Unknown machine image \"%s\"", executor.Image)
			if isMistakenImage(executor.Image, val.Cache.Offerings(val.Context.Api)) {
				val.addDiagnostic(diagnostic.Error(executor.ImageRange, message))
			} else {
				val.addDiagnostic(diagnostic.Warning(executor.ImageRange, message))
			}
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

// isMistakenImage reports whether a machine image the catalog lacks is of a
// family it lacks too, rather than a tag it has yet to catch up with. That
// makes CircleCI's own images outside the catalog errors as well.
func isMistakenImage(image string, offerings *circleci.Offerings) bool {
	family, _, _ := strings.Cut(image, ":")
	return !slices.Contains(offerings.MachineImageFamilies(), family)
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
		// Needs no lookup, so it holds for the images the lookups skip too.
		val.checkDeprecatedNamespace(img)

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
	}
}

// legacyImages maps each deprecated legacy convenience image in the
// `circleci` namespace to its `cimg` successor, or to "" when it has none.
// Other `circleci` images, such as circleci/circleci-cli, are current.
var legacyImages = map[string]string{
	"android":        "android",
	"buildpack-deps": "base",
	"clojure":        "clojure",
	"dynamodb":       "",
	"elixir":         "elixir",
	"golang":         "go",
	"jruby":          "",
	"mariadb":        "mariadb",
	"mongo":          "",
	"mysql":          "mysql",
	"node":           "node",
	"openjdk":        "openjdk",
	"php":            "php",
	"postgres":       "postgres",
	"python":         "python",
	"redis":          "redis",
	"ruby":           "ruby",
	"rust":           "rust",
}

func (val Validate) checkDeprecatedNamespace(img ast.DockerImage) {
	if img.Image.Namespace != "circleci" {
		return
	}
	successor, legacy := legacyImages[img.Image.Name]
	if !legacy {
		return
	}

	image := "circleci/" + img.Image.Name
	if successor == "" {
		val.addDiagnostic(diagnostic.Warning(img.ImageRange, fmt.Sprintf(
			"The legacy `%s` image is deprecated, and has no `cimg` successor.", image)))
		return
	}

	replacement := "cimg/" + successor
	val.addDiagnostic(
		diagnostic.New(
			img.ImageRange,
			protocol.DiagnosticSeverityWarning,
			fmt.Sprintf("The legacy `%s` image is deprecated. Use `%s` instead.", image, replacement),
			[]protocol.CodeAction{
				codeaction.TextEdit(
					fmt.Sprintf("Use `%s`", replacement),
					val.Doc.URI, []protocol.TextEdit{
						{
							Range:   img.ImageRange,
							NewText: "image: " + strings.Replace(img.Image.FullPath, image, replacement, 1),
						},
					}, true,
				),
			},
		),
	)
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
		if !val.validateRunnerResourceClass(resourceClass, resourceClassRange) {
			return
		}
		namespace := strings.Split(resourceClass, "/")[0]
		val.validateExecutorNamespace(namespace, resourceClassRange)
	}
}

// validateRunnerResourceClass reports a self-hosted runner's resource class
// that isn't namespace/name, returning whether it is.
func (val Validate) validateRunnerResourceClass(resourceClass string, resourceClassRange protocol.Range) bool {
	if !circleci.IsSelfHostedRunner(resourceClass) || paramref.ContainsReference(resourceClass) ||
		runnerResourceClassRegex.MatchString(resourceClass) {
		return true
	}
	val.addDiagnostic(diagnostic.Error(resourceClassRange,
		fmt.Sprintf("Invalid format in resource class or classes: %s.", resourceClass)))
	return false
}

// runnerResourceClassRegex is the compiler's format for a self-hosted runner's
// resource class, namespace/name.
var runnerResourceClassRegex = regexp.MustCompile(`^[a-z0-9_\-]+/[a-zA-Z0-9:_\-+]+$`)

func (val Validate) validateExecutorNamespace(resourceClass string, resourceClassRange protocol.Range) {
	exists, err := val.Cache.NamespaceCache.Exists(resourceClass, func() (bool, error) {
		_, err := val.Context.OrbRegistry().FetchNamespace(context.Background(), resourceClass)
		switch {
		case err == nil:
			return true, nil
		case circleci.IsNotFound(err):
			return false, nil
		default:
			return false, err
		}
	})

	// Only a definitive "no such namespace" earns a diagnostic. A request that
	// simply failed is not evidence the namespace is missing, and reporting one
	// would mean flagging valid configs whenever the API is unreachable.
	if err == nil && !exists {
		val.addDiagnostic(diagnostic.Error(
			resourceClassRange,
			fmt.Sprintf("Namespace \"%s\" does not exist", resourceClass),
		))
	}
}

func (val Validate) validateExecutorReference(executor string, rng protocol.Range) {
	// A name built from a reference is only known once the config is compiled.
	if paramref.ContainsReference(executor) {
		return
	}

	if !val.Doc.DoesExecutorExist(executor) {
		if val.Doc.IsOrbReference(executor) {
			val.validateOrbExecutor(executor, rng)
		} else {
			if possibleOrbName, couldBeOrbReference := val.Doc.CouldBeOrbReference(executor); couldBeOrbReference &&
				!val.Doc.IsOrbReference(executor) {
				val.addDiagnostic(
					protocol.Diagnostic{
						Range:    rng,
						Message:  protocol.String(fmt.Sprintf("Cannot find orb \"%s\". Looking for executor named \"%s\".", possibleOrbName, executor)),
						Severity: protocol.DiagnosticSeverityError,
					},
				)
			} else {
				val.addDiagnostic(
					protocol.Diagnostic{
						Range:    rng,
						Message:  protocol.String(fmt.Sprintf("Executor \"%s\" does not exist", executor)),
						Severity: protocol.DiagnosticSeverityError,
					},
				)
			}
		}
	}
}
