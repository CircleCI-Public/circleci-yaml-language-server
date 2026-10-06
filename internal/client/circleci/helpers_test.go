package circleci

// Shared setup for the tests that live inside the package.

// testToken is the token the in-package tests send. Point a fake's
// RequireToken at something else to exercise an unauthorized call.
const testToken = "test-token"

// clientFor is a client for a host, carrying testToken.
func clientFor(hostUrl string) *Client {
	return NewClient(Credentials{HostURL: hostUrl, Token: testToken}, false)
}
