// Command dockerhub probes the Docker Hub API the language server reads images
// and tags from, and checks that it still looks the way
// internal/testing/fakes pretends it does.
//
// Usage:
//
//	go run ./cmd/dockerhub [namespace/repository]
//
// The repository defaults to cimg/node.
//
// It exits 0 when Docker Hub looks as expected, 1 when it has drifted, and 2
// when it could not be reached. See internal/probe.
//
// Anonymous requests are rate limited, so a probe that is run often will
// eventually be refused; a refusal reads as an empty answer here, because this
// package does not check response statuses — see PLAN.md.
package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/probe"
)

// The defaults are a namespace and a repository large enough to page: cimg has
// far more repositories than a page holds, and cimg/node far more tags.
const (
	defaultNamespace  = "cimg"
	defaultRepository = "node"
)

// knownTag is a tag cimg/node has, and absentTag one it does not. There is no
// "latest" to use: cimg images are versioned, and asking for one 404s. If this
// version is ever withdrawn the probe will say so, and the fix is this line.
const (
	knownTag  = "26"
	absentTag = "definitely-not-a-tag-9f3a"
)

// absentRepository is a repository cimg does not have.
const absentRepository = "definitely-not-a-repository-9f3a"

// defaultPageSize is what Docker Hub serves when a request does not ask for a
// size. Reading more than this many items proves a cursor was followed.
const defaultPageSize = 10

// tagPageSize is what this package asks for when it lists tags, so reading
// more than this many proves the same thing for tags.
const tagPageSize = 100

// probeTimeout bounds the whole probe. The repository cursor loops until it is
// satisfied and cannot be given a deadline, so this is the only bound there is.
const probeTimeout = 3 * time.Minute

func main() {
	os.Exit(probe.Within(probeTimeout, run))
}

func run() int {
	namespace, repository := defaultNamespace, defaultRepository
	if len(os.Args) > 1 {
		parts := strings.SplitN(os.Args[1], "/", 2)
		if len(parts) != 2 {
			fmt.Fprintln(os.Stderr, "usage: dockerhub [namespace/repository]")

			return probe.ExitUnavailable
		}
		namespace, repository = parts[0], parts[1]
	}

	hubProbe := probe.New("docker hub", os.Stdout)
	hubProbe.Note("repository: %s/%s", namespace, repository)

	// The searches go through the package's default API, the one production
	// completion uses, and the rest through an API of its own. That is not
	// tidiness: a search loads the namespace into whichever API ran it, and
	// DoesImageExist answers from that without asking Docker Hub, so sharing
	// one would leave the existence checks proving nothing.
	api := dockerhub.NewAPI()

	hubProbe.Check("a namespace lists its repositories, paging past the first", func() error {
		cursor := dockerhub.Search(namespace + "/")

		found := 0
		for cursor.HasNext() {
			if cursor.Next() == nil {
				return errors.New("the cursor reported a repository and then produced nothing")
			}
			found++
		}

		hubProbe.Note("repositories: %d in %s, %d per page", found, namespace, defaultPageSize)

		switch {
		case found == 0:
			// Either the namespace is gone, or nothing could be read at all;
			// this package cannot tell those apart, so neither can the probe.
			return probe.Unavailable(fmt.Errorf("no repositories found in %s", namespace))
		case found <= defaultPageSize:
			return fmt.Errorf("found %d repositories, no more than a page holds, so no cursor was followed", found)
		}

		return nil
	})

	hubProbe.Check("a repository that exists is confirmed", func() error {
		exists, err := api.DoesImageExist(namespace, repository)
		if err != nil {
			return probe.Unavailable(err)
		}
		if !exists {
			return fmt.Errorf("%s/%s was not found", namespace, repository)
		}

		return nil
	})

	hubProbe.Check("a repository that does not exist is denied", func() error {
		exists, err := api.DoesImageExist(namespace, absentRepository)
		if err != nil {
			return probe.Unavailable(err)
		}
		if exists {
			return fmt.Errorf("%s/%s was reported to exist", namespace, absentRepository)
		}

		return nil
	})

	hubProbe.Check("a repository reports its active tags", func() error {
		tags, err := api.GetImageTags(namespace, repository)
		if err != nil {
			return probe.Unavailable(err)
		}

		hubProbe.Note("tags: %d active on the first page", len(tags))

		if len(tags) == 0 {
			return errors.New("no tags")
		}

		// Completion offers whatever is in this list, so a tag without a
		// name is drift in what Docker Hub reports.
		if slices.Contains(tags, "") {
			return errors.New("a tag has no name")
		}

		return nil
	})

	hubProbe.Check("a tag that exists is confirmed", func() error {
		hasTag, err := api.ImageHasTag(namespace, repository, knownTag)
		if err != nil {
			return probe.Unavailable(err)
		}
		if !hasTag {
			return fmt.Errorf("%s/%s:%s was not found", namespace, repository, knownTag)
		}

		return nil
	})

	hubProbe.Check("a tag that does not exist is denied", func() error {
		hasTag, err := api.ImageHasTag(namespace, repository, absentTag)
		if err != nil {
			return probe.Unavailable(err)
		}
		if hasTag {
			return fmt.Errorf("%s/%s:%s was reported to exist", namespace, repository, absentTag)
		}

		return nil
	})

	// Docker Hub reports the next page as an absolute URL rather than as a
	// cursor to put in a query, which is the other claim the fake makes about
	// this API.
	hubProbe.Check("tags page past the first, on an absolute next url", func() error {
		cursor, err := dockerhub.SearchTags(namespace, repository, "")
		if err != nil {
			return probe.Unavailable(err)
		}

		walked := 0
		for cursor.HasNext() && walked <= tagPageSize+1 {
			if cursor.Next() == nil {
				return errors.New("the cursor reported a tag and then produced nothing")
			}
			walked++
		}

		hubProbe.Note("tags walked: %d, %d per page", walked, tagPageSize)

		if walked <= tagPageSize {
			return fmt.Errorf("walked %d tags of a page of %d, so no next url was followed", walked, tagPageSize)
		}

		return nil
	})

	return hubProbe.Report()
}
