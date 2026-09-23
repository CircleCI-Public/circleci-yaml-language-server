package fakes_test

// getJSON is shared by the route tests in this package.

import (
	"encoding/json"
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
)

// getJSON reads a route of the fake into target and reports the status.
func getJSON(t *testing.T, url string, target any) int {
	t.Helper()

	res, err := http.Get(url)
	assert.NilError(t, err)
	defer func() { _ = res.Body.Close() }()

	err = json.NewDecoder(res.Body).Decode(target)
	assert.NilError(t, err)

	return res.StatusCode
}
