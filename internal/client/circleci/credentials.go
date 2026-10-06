package circleci

import (
	"net/http"
	"net/url"
)

// Credentials are who a request to CircleCI is made as, and where it is sent.
type Credentials struct {
	// HostURL is the scheme and authority of the CircleCI instance, e.g.
	// "https://circleci.com".
	HostURL string
	// Token is a personal API token. When empty, no Authorization header is
	// sent: the orb and namespace routes answer unauthenticated requests for
	// public orbs, which is how the language server serves users who have not
	// logged in.
	Token string
	// UserID is sent as the user_id header for telemetry.
	UserID string
}

// Credentials makes a fixed set of credentials a CredentialSource of its own,
// for a client that only ever talks to one host as one user.
func (credentials Credentials) Credentials() Credentials {
	return credentials
}

// A CredentialSource provides the credentials for a request. It is asked on
// every request, so whatever it returns when a request is sent is what that
// request is sent with.
type CredentialSource interface {
	Credentials() Credentials
}

// credentialsTransport adds the credentials of the moment to each request.
//
// They are only added to a request for the host they belong to. The transport
// sees every hop of a redirect, unlike headers set on the request, which
// net/http drops when a redirect leaves the host; and the host is read for the
// address a moment before the token is read here, so the two could disagree
// across a change of host.
type credentialsTransport struct {
	credentials CredentialSource
	next        http.RoundTripper
}

func (t *credentialsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	credentials := t.credentials.Credentials()
	if !sameOrigin(req.URL, credentials.HostURL) || (credentials.Token == "" && credentials.UserID == "") {
		return t.next.RoundTrip(req)
	}

	// A RoundTripper must not modify the request it is given.
	req = req.Clone(req.Context())
	if credentials.Token != "" {
		req.Header.Set("Authorization", "Bearer "+credentials.Token)
	}
	if credentials.UserID != "" {
		req.Header.Set("user_id", credentials.UserID)
	}

	return t.next.RoundTrip(req)
}

// sameOrigin reports whether a request is addressed to hostURL's scheme and
// authority.
func sameOrigin(address *url.URL, hostURL string) bool {
	host, err := url.Parse(hostURL)
	if err != nil {
		return false
	}

	return address.Scheme == host.Scheme && address.Host == host.Host
}
