package utils

// Shared setup for the tests that live inside the package, which cannot import
// pkg/testHelpers: that package imports this one.

// testToken is the token the in-package tests send. Point a fake's
// RequireToken at something else to exercise an unauthorized call.
const testToken = "test-token"

// lsContextFor builds an LsContext aimed at a host, carrying testToken.
func lsContextFor(hostUrl string) *LsContext {
	return &LsContext{
		Api: ApiContext{
			Token:   testToken,
			HostUrl: hostUrl,
		},
	}
}
