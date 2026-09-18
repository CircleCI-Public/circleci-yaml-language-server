package fakes

// This file serves the signed-in account: GET /api/v2/me, which the language
// server reads to attach a user id to its telemetry.

import "net/http"

// accountState is the stored account.
type accountState struct {
	user *User
}

// User is the account GET /api/v2/me reports.
type User struct {
	ID    string
	Login string
	Name  string
}

// SetUser registers the account GET /api/v2/me reports. Until it is called the
// route answers 401, as it does for a request carrying no usable token.
func (f *CircleCI) SetUser(id, login, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.account.user = &User{ID: id, Login: login, Name: name}
}

func (f *CircleCI) handleGetMe(w http.ResponseWriter, _ *http.Request) {
	f.mu.RLock()
	user := f.account.user
	f.mu.RUnlock()

	if user == nil {
		writeV2Error(w, http.StatusUnauthorized, "You must log in first.")

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.ID,
		"login": user.Login,
		"name":  user.Name,
	})
}
