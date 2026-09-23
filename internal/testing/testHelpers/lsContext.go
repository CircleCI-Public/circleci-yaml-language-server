package testHelpers

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func DefaultSettings() *session.Settings {
	return &session.Settings{
		Api: circleci.Config{
			Token:   "XXXXXXXXXXXX",
			HostUrl: "https://circleci.com",
		},
	}
}

func SettingsForHost(hostUrl string) *session.Settings {
	return &session.Settings{
		Api: circleci.Config{
			Token:   "XXXXXXXXXXXX",
			HostUrl: hostUrl,
		},
	}
}
