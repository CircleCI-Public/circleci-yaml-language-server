// Command release_version prints the version the release job publishes for
// the commit being built: the stable version release-please has drafted, or
// else a prerelease whose tag has never been used, so no tag moves and no
// asset is replaced.
//
// Drafts are only listed for a token that can push, so GITHUB_TOKEN must be
// set.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/Masterminds/semver/v3"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

// release is the part of a GitHub release this command reads.
type release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Draft   bool   `json:"draft"`
}

func main() {
	api := flag.String("api", "https://api.github.com", "GitHub API URL")
	repo := flag.String("repo", "CircleCI-Public/circleci-yaml-language-server", "repository to release")
	manifest := flag.String("manifest", ".circleci/release/release-please-manifest.json", "release-please manifest")
	flag.Parse()

	if err := run(*api, *repo, *manifest); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "release_version:", err)
		os.Exit(1)
	}
}

func run(api, repo, manifest string) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN is not set, and without it drafts aren't listed")
	}
	current, err := manifestVersion(manifest)
	if err != nil {
		return err
	}
	cl := httpcl.New(httpcl.Config{BaseURL: api, AuthToken: token})
	releases, err := listReleases(context.Background(), cl, repo)
	if err != nil {
		return err
	}
	version, err := nextVersion(current, releases)
	if err != nil {
		return err
	}
	fmt.Println(version)
	return nil
}

func manifestVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var packages map[string]string
	if err := json.Unmarshal(data, &packages); err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	version, ok := packages["."]
	if !ok {
		return "", fmt.Errorf("%s has no version for %q", path, ".")
	}
	return version, nil
}

// listReleases reads every page of the repository's releases, drafts and
// prereleases included.
func listReleases(ctx context.Context, cl *httpcl.Client, repo string) ([]release, error) {
	var all []release
	for page := 1; ; page++ {
		var releases []release
		_, err := cl.Call(ctx, httpcl.NewRequest(http.MethodGet, "/repos/"+repo+"/releases",
			httpcl.Header("Accept", "application/vnd.github+json"),
			httpcl.QueryParam("per_page", "100"),
			httpcl.QueryParam("page", strconv.Itoa(page)),
			httpcl.JSONDecoder(&releases),
		))
		if err != nil {
			return nil, fmt.Errorf("listing releases: %w", err)
		}
		if len(releases) == 0 {
			return all, nil
		}
		all = append(all, releases...)
	}
}

var prereleaseNumber = regexp.MustCompile(`^pre\.(\d+)$`)

// nextVersion is the manifest's version if release-please has drafted it,
// and otherwise the patch after it at one more than its highest pre.N.
func nextVersion(current string, releases []release) (string, error) {
	v, err := semver.StrictNewVersion(current)
	if err != nil {
		return "", fmt.Errorf("manifest version: %w", err)
	}
	for _, r := range releases {
		if r.Draft && r.Name == "v"+v.String() {
			return v.String(), nil
		}
	}

	next := v.IncPatch()
	highest := 0
	for _, r := range releases {
		released, err := semver.NewVersion(r.TagName)
		if err != nil {
			continue
		}
		if released.Major() != next.Major() || released.Minor() != next.Minor() || released.Patch() != next.Patch() {
			continue
		}
		m := prereleaseNumber.FindStringSubmatch(released.Prerelease())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		highest = max(highest, n)
	}
	return fmt.Sprintf("%s-pre.%d", next.String(), highest+1), nil
}
