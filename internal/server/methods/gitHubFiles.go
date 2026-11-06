package methods

import (
	"context"
	"errors"
	"time"
)

// methodFetchGitHubFile asks a client that fetches GitHub files for the
// server (the fetchesGitHubFiles initialization option) for one.
const methodFetchGitHubFile = "circleci/fetchGitHubFile"

type fetchGitHubFileParams struct {
	URL string `json:"url"`
}

type fetchGitHubFileResult struct {
	Found  bool    `json:"found"`
	Source *string `json:"source,omitempty"`
}

// fetchGitHubFileTimeout bounds how long a fetch waits on the client, so that
// a client that never answers can't hold up checking a document for good.
const fetchGitHubFileTimeout = 30 * time.Second

var errMalformedGitHubFile = errors.New(methodFetchGitHubFile + ": the client's answer is neither a source nor not found")

// fetchGitHubFile asks the client for a file on GitHub, which it fetches as
// the user's GitHub account. found and err mean what they do for
// orburl.Fetch: an answer that isn't one the client could mean is an error,
// so that it isn't remembered as missing.
func (methods *Methods) fetchGitHubFile(ctx context.Context, address string) (source string, found bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, fetchGitHubFileTimeout)
	defer cancel()

	var result *fetchGitHubFileResult
	if _, err := methods.conn.Call(ctx, methodFetchGitHubFile, fetchGitHubFileParams{URL: address}, &result); err != nil {
		return "", false, err
	}

	switch {
	case result == nil, result.Found && result.Source == nil:
		return "", false, errMalformedGitHubFile
	case !result.Found:
		return "", false, nil
	default:
		return *result.Source, true, nil
	}
}
