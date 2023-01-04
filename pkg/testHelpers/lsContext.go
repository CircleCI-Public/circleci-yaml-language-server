package testHelpers

import "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"

func GetDefaultLsContext() *utils.LsContext {
	return &utils.LsContext{
		Api: utils.ApiContext{
			Token:   "XXXXXXXXXXXX",
			HostUrl: "https://circleci.com",
		},
	}
}

func GetLsContextForHost(hostUrl string) *utils.LsContext {
	return &utils.LsContext{
		Api: utils.ApiContext{
			Token:   "XXXXXXXXXXXX",
			HostUrl: hostUrl,
		},
	}
}
