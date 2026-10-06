package circleci

import (
	"context"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

const CurrentLinuxImage = "ubuntu-2404:current"

// The executors the catalog lists resource classes for.
const (
	ExecutorLinux        = "linux"
	ExecutorWindows      = "windows"
	ExecutorMacOS        = "macos"
	ExecutorRemoteDocker = "remote_docker"
	ExecutorDocker       = "docker"
)

// MachineExecutors are the executors a machine job runs on: a class is one or
// the other's.
var MachineExecutors = []string{ExecutorLinux, ExecutorWindows}

type Offerings struct {
	Linux   map[string][]string
	Windows map[string][]string
	MacOS   map[string][]string
	// RemoteDocker is the Docker versions setup_remote_docker takes, by the
	// resource class of the remote machine.
	RemoteDocker map[string][]string
	// Docker is the Docker executor's resource classes. They take any image,
	// so their lists are empty.
	Docker map[string][]string
	// Unlike the lists above, Deprecated is keyed by executor, not resource class, and
	// excludes images already present there.
	Deprecated map[string][]string
	// ResourceClasses is what the catalog says of each class besides what it
	// runs, by executor and then by class.
	ResourceClasses map[string]map[string]ResourceClass
}

// ResourceClass is a resource class as the catalog describes it: its name and
// its size.
type ResourceClass struct {
	// Name is what CircleCI calls the class, such as "Linux Medium Gen2".
	Name  string `json:"name"`
	CPU   int    `json:"cpu"`
	RAMMB int    `json:"ram_mb"`
}

// Size is the class's CPUs and memory, such as "2 vCPUs, 8 GB RAM".
func (rc ResourceClass) Size() string {
	cpus := "vCPUs"
	if rc.CPU == 1 {
		cpus = "vCPU"
	}
	gb := strconv.FormatFloat(math.Round(float64(rc.RAMMB)/1024*10)/10, 'f', -1, 64)
	return fmt.Sprintf("%d %s, %s GB RAM", rc.CPU, cpus, gb)
}

// Summary is the class's name and size, such as "Linux Medium: 2 vCPUs, 8 GB
// RAM", or just its name for a class with no size, and nothing for a class
// the catalog doesn't describe.
func (rc ResourceClass) Summary() string {
	switch {
	case rc.CPU == 0 || rc.RAMMB == 0:
		return rc.Name
	case rc.Name == "":
		return rc.Size()
	}
	return rc.Name + ": " + rc.Size()
}

type MachinePair struct {
	ResourceClass string
	Images        []string
}

// catalogClass is a resource class in the catalog's response.
type catalogClass struct {
	ResourceClass
	Images           []string `json:"images"`
	DeprecatedImages []string `json:"deprecated_images"`
}

// FetchOfferings reads the machine catalog: the resource classes each
// executor offers, and the images, Xcode versions or Docker versions each
// class takes. It returns nil on any failure, and for a catalog with nothing
// in it, so that callers skip validation rather than flag valid config.
func FetchOfferings(ctx context.Context, cl *Client) *Offerings {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// The response is a list of executors, each with its classes.
	var executors []struct {
		Attributes struct {
			Executor        string                  `json:"executor"`
			ResourceClasses map[string]catalogClass `json:"resource_classes"`
		} `json:"attributes"`
	}
	if err := cl.Get(ctx, "catalog/resource-classes", nil, &executors); err != nil {
		return nil
	}

	o := &Offerings{
		Linux:           map[string][]string{},
		Windows:         map[string][]string{},
		MacOS:           map[string][]string{},
		RemoteDocker:    map[string][]string{},
		Docker:          map[string][]string{},
		Deprecated:      map[string][]string{},
		ResourceClasses: map[string]map[string]ResourceClass{},
	}
	groups := map[string]map[string][]string{
		ExecutorLinux:        o.Linux,
		ExecutorWindows:      o.Windows,
		ExecutorMacOS:        o.MacOS,
		ExecutorRemoteDocker: o.RemoteDocker,
		ExecutorDocker:       o.Docker,
	}
	for _, entry := range executors {
		executor := entry.Attributes.Executor
		group, ok := groups[executor]
		if !ok {
			continue
		}
		classes := map[string]ResourceClass{}
		// The catalog deprecates images class by class; validation only asks
		// whether an executor has deprecated an image.
		deprecated := []string{}
		for name, class := range entry.Attributes.ResourceClasses {
			group[name] = append([]string{}, class.Images...)
			classes[name] = class.ResourceClass
			for _, image := range class.DeprecatedImages {
				if !slices.Contains(deprecated, image) {
					deprecated = append(deprecated, image)
				}
			}
		}
		slices.Sort(deprecated)
		o.ResourceClasses[executor] = classes
		o.Deprecated[executor] = deprecated
	}

	if len(o.Linux)+len(o.Windows)+len(o.MacOS) == 0 {
		return nil
	}
	return o
}

// Class is what the catalog says of a resource class on the first of
// executors that offers it. It is false for a class none of them offers, such
// as a self-hosted runner's.
func (o *Offerings) Class(name string, executors ...string) (ResourceClass, bool) {
	if o == nil {
		return ResourceClass{}, false
	}
	for _, executor := range executors {
		if class, ok := o.ResourceClasses[executor][name]; ok {
			return class, true
		}
	}
	return ResourceClass{}, false
}

// MachinePairs covers Linux and Windows machine executors; macOS is handled separately.
func (o *Offerings) MachinePairs() []MachinePair {
	if o == nil {
		return nil
	}
	pairs := []MachinePair{}
	for _, group := range []map[string][]string{o.Linux, o.Windows} {
		for class, images := range group {
			pairs = append(pairs, MachinePair{ResourceClass: class, Images: images})
		}
	}
	return pairs
}

func (o *Offerings) MachineImages() []string {
	pairs := o.MachinePairs()
	if pairs == nil {
		return nil
	}
	images := map[string]bool{}
	for _, pair := range pairs {
		for _, image := range pair.Images {
			images[image] = true
		}
	}
	return slices.Collect(maps.Keys(images))
}

// MachineImageFamilies is each Linux and Windows image in the catalog,
// deprecated ones included, without its tag.
func (o *Offerings) MachineImageFamilies() []string {
	if o == nil {
		return nil
	}
	families := []string{}
	for _, image := range append(o.MachineImages(), o.DeprecatedMachineImages()...) {
		family, _, _ := strings.Cut(image, ":")
		if !slices.Contains(families, family) {
			families = append(families, family)
		}
	}
	return families
}

func (o *Offerings) MachineResourceClasses() []string {
	pairs := o.MachinePairs()
	if pairs == nil {
		return nil
	}
	classes := []string{}
	for _, pair := range pairs {
		classes = append(classes, pair.ResourceClass)
	}
	return classes
}

func (o *Offerings) DeprecatedMachineImages() []string {
	if o == nil {
		return nil
	}
	images := []string{}
	images = append(images, o.Deprecated["linux"]...)
	images = append(images, o.Deprecated["windows"]...)
	return images
}

func (o *Offerings) DeprecatedXcodeVersions() []string {
	if o == nil {
		return nil
	}
	versions := []string{}
	for _, image := range o.Deprecated["macos"] {
		// API returns "xcode:<version>"; the config field is the bare version.
		versions = append(versions, strings.TrimPrefix(image, "xcode:"))
	}
	return versions
}

func (o *Offerings) XcodeVersions() []string {
	if o == nil {
		return nil
	}
	versions := map[string]bool{}
	for _, images := range o.MacOS {
		for _, image := range images {
			// API returns "xcode:<version>"; the config field is the bare version.
			versions[strings.TrimPrefix(image, "xcode:")] = true
		}
	}
	return slices.Collect(maps.Keys(versions))
}

func (o *Offerings) MacOSResourceClasses() []string {
	if o == nil {
		return nil
	}
	return slices.Collect(maps.Keys(o.MacOS))
}

// DockerResourceClasses is nil for a catalog that doesn't list the Docker
// executor's classes, so that they aren't checked.
func (o *Offerings) DockerResourceClasses() []string {
	if o == nil || len(o.Docker) == 0 {
		return nil
	}
	return slices.Collect(maps.Keys(o.Docker))
}

// remoteDockerClasses are the Docker executor's classes that run
// setup_remote_docker on a machine of another class, as the machine
// provisioner routes them.
var remoteDockerClasses = map[string]string{
	"small":   "medium",
	"medium+": "large",
}

// RemoteDockerVersions is the Docker versions setup_remote_docker takes on a
// Docker job of a resource class, except deprecated ones. It is empty for a
// class that can't run setup_remote_docker, and nil when the catalog doesn't
// say: it doesn't list the class as a Docker one, or has no remote Docker
// versions at all.
func (o *Offerings) RemoteDockerVersions(dockerClass string) []string {
	if o == nil || len(o.RemoteDocker) == 0 {
		return nil
	}
	if _, ok := o.Docker[dockerClass]; !ok {
		return nil
	}
	if class, ok := remoteDockerClasses[dockerClass]; ok {
		dockerClass = class
	}
	versions, ok := o.RemoteDocker[dockerClass]
	if !ok {
		return []string{}
	}
	return versions
}

// DeprecatedRemoteDockerVersions isn't by resource class: the catalog lists
// them for remote Docker as a whole.
func (o *Offerings) DeprecatedRemoteDockerVersions() []string {
	if o == nil {
		return nil
	}
	return o.Deprecated["remote_docker"]
}
