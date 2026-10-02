package circleci

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

const CurrentLinuxImage = "ubuntu-2404:current"

type Offerings struct {
	Linux   map[string][]string `json:"linux"`
	Windows map[string][]string `json:"windows"`
	MacOS   map[string][]string `json:"macos"`
	// RemoteDocker is the Docker versions setup_remote_docker takes, by the
	// resource class of the remote machine.
	RemoteDocker map[string][]string `json:"remote_docker"`
	// Docker is the Docker executor's resource classes. They take any image,
	// so their lists are empty.
	Docker map[string][]string `json:"docker"`
	// Unlike the lists above, Deprecated is keyed by executor, not resource class, and
	// excludes images already present there.
	Deprecated map[string][]string `json:"deprecated"`
}

type MachinePair struct {
	ResourceClass string
	Images        []string
}

// FetchOfferings reads the machine catalog: the images and resource classes
// each machine executor offers. It returns nil on any failure, and for a
// catalog with nothing in it, so that callers skip validation rather than flag
// valid config.
func FetchOfferings(ctx context.Context, api Config) *Offerings {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// The V3 response wraps the catalog in a data entity: {"data": {"attributes": {...}}}.
	var body struct {
		Data struct {
			Attributes Offerings `json:"attributes"`
		} `json:"data"`
	}

	// This is a V3 route, but it authenticates with Circle-Token as the V2
	// routes do, rather than with the bearer token V3Client sends.
	client := client.New(httpcl.Config{
		BaseURL:    api.HostUrl + "/api/v3",
		AuthToken:  api.Token,
		AuthHeader: "Circle-Token",
	})
	status, err := client.Call(ctx, httpcl.NewRequest(http.MethodGet, "/catalog/offerings", httpcl.JSONDecoder(&body)))
	if err != nil || status != http.StatusOK {
		return nil
	}

	o := body.Data.Attributes
	if len(o.Linux)+len(o.Windows)+len(o.MacOS) == 0 {
		return nil
	}
	return &o
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
