package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

type LsContext struct {
	Api                ApiContext
	UserIdForTelemetry string
	IsCciExtension     bool
}

type ApiContext struct {
	Token   string
	HostUrl string
	// RunnerHost is where self-hosted runner requests go. Empty means the API
	// host with "runner." prefixed to it, which is how this repository has
	// always addressed the runner service — a service with versioning of its
	// own, not the CircleCI V3 API, as methods.getResourceClassOfOrg explains.
	// A self-hosted install serving it elsewhere, or a test pointing it at a
	// fake, says so here.
	RunnerHost string
}

// RunnerHostUrl resolves the host self-hosted runner requests are made
// against.
//
// The "runner." prefix is the non-standard part: the CLI calls the same paths
// on the API host itself. This exists so that the prefix can be overridden
// rather than assumed, and it should become unnecessary once those calls move
// onto the standard API.
func (apiContext ApiContext) RunnerHostUrl() (string, error) {
	if apiContext.RunnerHost != "" {
		return apiContext.RunnerHost, nil
	}

	hostUrl, err := url.Parse(apiContext.HostUrl)
	if err != nil {
		return "", err
	}
	hostUrl.Host = "runner." + hostUrl.Host

	return hostUrl.String(), nil
}

func (apiContext ApiContext) UseDefaultInstance() bool {
	return apiContext.HostUrl == CIRCLE_CI_APP_HOST_URL
}

func (apiContext ApiContext) IsLoggedIn() bool {
	return apiContext.Token != ""
}

type MeRes struct {
	Id    string
	Login string
	Name  string
}

// userIds memoises the account id a token resolves to on a host.
//
// It is keyed rather than held on ApiContext for two reasons: LsContext carries
// its ApiContext by value, so anything memoised into the struct is lost the
// first time the context is copied; and requests are served on goroutines that
// share one LsContext, so a field written without a lock would be a race. It
// follows the same shape as the V3 orb route capability cache in
// orbregistry.go.
var userIds = struct {
	mutex sync.RWMutex
	known map[string]string
}{known: map[string]string{}}

func lookupUserId(key string) (userId string, known bool) {
	userIds.mutex.RLock()
	defer userIds.mutex.RUnlock()

	userId, known = userIds.known[key]

	return userId, known
}

func recordUserId(key, userId string) {
	userIds.mutex.Lock()
	defer userIds.mutex.Unlock()

	userIds.known[key] = userId
}

// resetUserIds forgets every memoised account id. It exists for tests, which
// stand up a fresh server per case, and reaches them through export_test.go
// rather than this package's public API.
func resetUserIds() {
	userIds.mutex.Lock()
	defer userIds.mutex.Unlock()

	userIds.known = map[string]string{}
}

// GetUserId reports the id of the account the token belongs to, fetching it at
// most once per host and token.
//
// Anything short of an id — a rejected token, an error status, an unreachable
// host, a body without one — is reported as "" and is not memoised, so a later
// call tries again rather than caching a non-answer.
func (apiContext ApiContext) GetUserId() string {
	key := apiContext.HostUrl + "\x00" + apiContext.Token
	if userId, known := lookupUserId(key); known {
		return userId
	}

	url := fmt.Sprintf("%s/api/v2/me", apiContext.HostUrl)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Add("Circle-Token", apiContext.Token)
	req.Header.Set("User-Agent", UserAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ""
	}

	// The status has to be checked rather than inferred from the decoded body:
	// an error body carrying an id of its own — a rate limit reports one —
	// would otherwise be memoised as the account's.
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return ""
	}

	var userRes MeRes
	err = json.Unmarshal(body, &userRes)
	if err != nil {
		return ""
	}

	if userRes.Id == "" {
		return ""
	}

	recordUserId(key, userRes.Id)

	return userRes.Id
}
