// Command orb probes the CircleCI orb registry: it resolves an orb reference
// against the real API and checks that what comes back still looks the way
// internal/testing/fakes pretends it does.
//
// Usage:
//
//	go run ./cmd/orb [orb-reference]
//
// The reference defaults to circleci/go@1.7.1. Environment:
//
//	CIRCLE_TOKEN    used when set; without one, public orbs still resolve
//	CIRCLECI_HOST   defaults to https://circleci.com
//	ORB_BACKEND     "graphql" forces the GraphQL fallback; anything else, or
//	                unset, probes the host and uses V3 where it answers
//	ORB_DEBUG       when set, the client logs every request
//
// Forcing the backend is how the GraphQL fallback gets checked against real
// infrastructure: circleci.com serves GraphQL as well as V3, so both paths can
// be compared without a CircleCI Server instance.
//
// It exits 0 when the registry looks as expected, 1 when it has drifted, and 2
// when it could not be reached. See internal/probe.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/logging"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/probe"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

// pagedNamespace holds more orbs than the page limit below, so that following
// a cursor is exercised rather than assumed.
const pagedNamespace = "circleci"

// pageLimit is deliberately tiny. The point is not to read a namespace
// quickly; it is to make the API hand out a cursor, and to prove that
// following it works.
const pageLimit = 2

// probeTimeout bounds the whole probe, so that a schedule is not left waiting
// on a host that accepts connections and then says nothing.
const probeTimeout = 2 * time.Minute

func main() {
	os.Exit(run())
}

func run() int {
	ref := "circleci/go@1.7.1"
	if len(os.Args) > 1 {
		ref = os.Args[1]
	}

	host := utils.CIRCLE_CI_APP_HOST_URL
	if fromEnv := os.Getenv("CIRCLECI_HOST"); fromEnv != "" {
		host = fromEnv
	}

	token := os.Getenv("CIRCLE_TOKEN")
	debug := os.Getenv("ORB_DEBUG") != ""
	backend := os.Getenv("ORB_BACKEND")

	// The probe's report goes to stdout; with ORB_DEBUG, every request and
	// response is logged to stderr alongside it.
	logging.Setup(debug)

	var registry utils.OrbRegistry
	switch backend {
	case "graphql":
		registry = utils.NewGraphQLOrbRegistry(host, token, "", debug)
	default:
		backend = "v3 where it answers, graphql where it does not"
		registry = utils.NewOrbRegistry(host, token, "", debug)
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	orbProbe := probe.New("circleci orb registry", os.Stdout)
	orbProbe.Note("host:    %s", host)
	orbProbe.Note("backend: %s", backend)
	orbProbe.Note("token:   %s", describeToken(token))
	orbProbe.Note("ref:     %s", ref)

	var orb *utils.OrbPackage

	// The first read is where an unreachable host or a rejected token shows
	// up, so it is the one that can report the probe unavailable.
	orbProbe.Check("an orb package reports an id, a name and a namespace", func() error {
		fetched, err := registry.FetchOrb(ctx, utils.OrbPackageName(ref))
		if err != nil {
			if errors.Is(err, utils.ErrNotFound) {
				return fmt.Errorf("%s is not visible: %w", utils.OrbPackageName(ref), err)
			}

			return probe.Unavailable(err)
		}

		orb = fetched
		orbProbe.Note("orb:     %s (%s) in namespace %s", orb.Name, orb.ID, orb.NamespaceID)

		switch {
		case orb.ID == "":
			return errors.New("no id")
		case orb.Name == "":
			return errors.New("no name")
		case orb.NamespaceID == "":
			return errors.New("no namespace id")
		}

		return nil
	})

	// A package carries every release it has, newest first. That is what the
	// language server reads to offer an upgrade, and what the fake reproduces.
	orbProbe.Check("an orb package carries its releases, newest first", func() error {
		if len(orb.Versions) == 0 {
			return errors.New("no versions")
		}

		orbProbe.Note("versions: %d published, latest %s", len(orb.Versions), orb.Versions[0].Version)

		sorted := make([]utils.OrbPackageVersion, len(orb.Versions))
		copy(sorted, orb.Versions)
		utils.SortOrbVersionsDesc(sorted)

		if sorted[0].Version != orb.Versions[0].Version {
			return fmt.Errorf("the newest release is %s but the list starts at %s",
				sorted[0].Version, orb.Versions[0].Version)
		}

		return nil
	})

	orbProbe.Check("a reference resolves to a version with its source", func() error {
		resolved, err := registry.ResolveVersion(ctx, ref)
		if err != nil {
			return err
		}

		orbProbe.Note("resolved: %s -> %s (%s), %d bytes of source, %d siblings",
			ref, resolved.Version, resolved.ID, len(resolved.Source), len(resolved.Versions))

		switch {
		case resolved.Version == "":
			return errors.New("no version")
		case resolved.ID == "":
			return errors.New("no id")
		case len(resolved.Source) == 0:
			return errors.New("no source")
		}

		return nil
	})

	// Only V3 paginates. The GraphQL fallback reads a single page by design,
	// so there is nothing to check when it is forced.
	if backend == "graphql" {
		orbProbe.Note("paging: not checked, the GraphQL fallback reads a single page")
	} else {
		orbProbe.Check("a collection pages with an opaque cursor", func() error {
			return checkPaging(ctx, host, token, debug, orbProbe)
		})
	}

	return orbProbe.Report()
}

// checkPaging reads a namespace's orbs two at a time. Asking for more than one
// page's worth and getting it is the only way to prove, from outside, that the
// cursors the API hands out can be followed — which is the fake's main claim
// about this API.
func checkPaging(ctx context.Context, host, token string, debug bool, orbProbe *probe.Probe) error {
	client := utils.NewV3Client(host, token, "", debug)

	namespace, err := utils.FetchNamespace(ctx, client, pagedNamespace)
	if err != nil {
		return fmt.Errorf("reading the %s namespace: %w", pagedNamespace, err)
	}

	query := url.Values{}
	query.Set("filter[namespace_id]", namespace.ID)
	query.Set("page[limit]", strconv.Itoa(pageLimit))

	orbs, err := utils.GetPaged[struct {
		ID string `json:"id"`
	}](ctx, client, "orb/packages", query)
	if err != nil {
		return fmt.Errorf("listing the orbs of %s: %w", pagedNamespace, err)
	}

	orbProbe.Note("paging: %d orbs in %s, read %d at a time", len(orbs), pagedNamespace, pageLimit)

	if len(orbs) <= pageLimit {
		return fmt.Errorf("read %d orbs asking for %d at a time, so no cursor was followed",
			len(orbs), pageLimit)
	}

	return nil
}

func describeToken(token string) string {
	if token == "" {
		return "none, reading as an anonymous caller"
	}

	return "set"
}
