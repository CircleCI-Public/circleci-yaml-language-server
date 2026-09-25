package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const gitHubAPI = "https://api.github.com"

type repository struct {
	FullName string    `json:"full_name"`
	Archived bool      `json:"archived"`
	Fork     bool      `json:"fork"`
	PushedAt time.Time `json:"pushed_at"`
}

type contentEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

var errNotFound = errors.New("not found")

func fetch(args []string) error {
	flags := flag.NewFlagSet("fetch", flag.ExitOnError)
	out := flags.String("out", defaultDir, "directory to write the configs to")
	since := flags.String("since", time.Now().AddDate(-1, 0, 0).Format(time.DateOnly), "only repositories pushed to since this date")
	_ = flags.Parse(args)

	sinceDate, err := time.Parse(time.DateOnly, *since)
	if err != nil {
		return fmt.Errorf("-since: %w", err)
	}

	orgs := flags.Args()
	if len(orgs) == 0 {
		orgs = []string{"circleci", "CircleCI-Public"}
	}

	gh := gitHubClient{token: gitHubToken()}
	ctx := context.Background()

	var repos []repository
	for _, org := range orgs {
		found, err := gh.orgRepositories(ctx, org)
		if err != nil {
			return fmt.Errorf("listing %s: %w", org, err)
		}
		for _, repo := range found {
			if !repo.Archived && !repo.Fork && !repo.PushedAt.Before(sinceDate) {
				repos = append(repos, repo)
			}
		}
	}

	// A handful of workers is quick, and well within GitHub's rate limits.
	work := make(chan repository)
	var wg sync.WaitGroup
	var mu sync.Mutex
	files := 0
	for range 8 {
		wg.Go(func() {
			for repo := range work {
				n, err := gh.fetchConfigs(ctx, repo.FullName, *out)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", repo.FullName, err)
				}
				mu.Lock()
				files += n
				mu.Unlock()
			}
		})
	}
	for _, repo := range repos {
		work <- repo
	}
	close(work)
	wg.Wait()

	fmt.Fprintf(os.Stderr, "%d files from %d repositories in %s\n", files, len(repos), *out)
	return nil
}

type gitHubClient struct {
	token string
}

// get reads a GitHub API route, as JSON into out, or raw when out is a
// *[]byte.
func (gh gitHubClient) get(ctx context.Context, route string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gitHubAPI+route, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if _, raw := out.(*[]byte); raw {
		req.Header.Set("Accept", "application/vnd.github.raw")
	}
	if gh.token != "" {
		req.Header.Set("Authorization", "Bearer "+gh.token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return errNotFound
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("GET %s: %d %s", route, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if raw, ok := out.(*[]byte); ok {
		*raw = body
		return nil
	}
	return json.Unmarshal(body, out)
}

func (gh gitHubClient) orgRepositories(ctx context.Context, org string) ([]repository, error) {
	var all []repository
	for page := 1; ; page++ {
		var repos []repository
		if err := gh.get(ctx, fmt.Sprintf("/orgs/%s/repos?type=all&per_page=100&page=%d", org, page), &repos); err != nil {
			return nil, err
		}
		all = append(all, repos...)
		if len(repos) < 100 {
			return all, nil
		}
	}
}

// fetchConfigs writes the YAML files directly under a repository's
// .circleci/ to out/owner__repo/, and returns how many it wrote.
func (gh gitHubClient) fetchConfigs(ctx context.Context, fullName, out string) (int, error) {
	var entries []contentEntry
	err := gh.get(ctx, "/repos/"+fullName+"/contents/.circleci", &entries)
	if errors.Is(err, errNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	dir := filepath.Join(out, strings.ReplaceAll(fullName, "/", "__"))
	written := 0
	for _, entry := range entries {
		if entry.Type != "file" || !isYAML(entry.Name) {
			continue
		}

		var content []byte
		if err := gh.get(ctx, "/repos/"+fullName+"/contents/"+entry.Path, &content); err != nil {
			return written, err
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return written, err
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(entry.Name)), content, 0o600); err != nil {
			return written, err
		}
		written++
	}

	return written, nil
}
