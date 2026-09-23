package circleci

// Shared setup for the tests that live inside the package.

// testToken is the token the in-package tests send. Point a fake's
// RequireToken at something else to exercise an unauthorized call.
const testToken = "test-token"

// configFor is the API configuration for a host, carrying testToken.
func configFor(hostUrl string) Config {
	return Config{
		Token:   testToken,
		HostUrl: hostUrl,
	}
}
