package complete

import (
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// completeJobExecutor completes a job's resource_class, and the body of its
// machine or macos, and says whether the cursor was at one of them.
func (ch *CompletionHandler) completeJobExecutor(job ast2.Job) bool {
	key, _, parent := ch.valueAt()
	if parent != -1 {
		switch {
		case parent == startLine(job.NameRange) && key == "resource_class":
			ch.addJobResourceClasses(job)
			return true
		case parent == startLine(job.MachineRange) && key == "image":
			ch.addCompletionItems(ch.Cache.Offerings(ch.Context.Api).MachineImages())
			return true
		case parent == startLine(job.MacOSRange) && key == "xcode":
			ch.addCompletionItems(ch.Cache.Offerings(ch.Context.Api).XcodeVersions())
			return true
		}
		return false
	}

	_, parent = ch.keyParent()
	var keys []string
	switch parent {
	case -1:
		return false
	case startLine(job.MachineRange):
		keys = []string{"image", "docker_layer_caching"}
	case startLine(job.MacOSRange):
		keys = []string{"xcode"}
	default:
		return false
	}

	present := ch.stepBodyKeys(parent)
	for _, key := range keys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
	return true
}

func (ch *CompletionHandler) addCompletionItems(labels []string) {
	for _, label := range labels {
		ch.addCompletionItem(label)
	}
}

// startLine is the line a range starts on, or -1 for the zero range, which
// stands for something that isn't there.
func startLine(rng protocol.Range) int {
	if position.IsDefaultRange(rng) {
		return -1
	}
	return int(rng.Start.Line)
}

// addJobResourceClasses offers the resource classes of the job's executor:
// the one it gives in place, or the executor of the config it names.
func (ch *CompletionHandler) addJobResourceClasses(job ast2.Job) {
	var executor ast2.Executor
	switch {
	case !position.IsDefaultRange(job.DockerRange):
		executor = job.Docker
	case !position.IsDefaultRange(job.MachineRange):
		executor = job.Machine
	case !position.IsDefaultRange(job.MacOSRange):
		executor = job.MacOS
	default:
		executor = ch.Doc.Executors[job.Executor]
	}

	offerings := ch.Cache.Offerings(ch.Context.Api)
	switch executor.(type) {
	case ast2.DockerExecutor:
		ch.addResourceClassCompletion(offerings.DockerResourceClasses())
	case ast2.MachineExecutor:
		ch.addResourceClassCompletion(offerings.MachineResourceClasses())
		if ch.Context.Api.IsLoggedIn() {
			ch.addResourceClassCompletion(ch.Cache.ResourceClassesOfFile(ch.Context.Api, ch.Doc.URI))
		}
	case ast2.MacOSExecutor:
		ch.addResourceClassCompletion(offerings.MacOSResourceClasses())
	}
}
