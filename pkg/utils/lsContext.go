package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

type LsContext struct {
	Api                ApiContext
	UserIdForTelemetry string
	IsCciExtension     bool
}

type ApiContext struct {
	Token   string
	HostUrl string
	userId  string
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

func (apiContext ApiContext) GetUserId() string {
	if apiContext.userId != "" {
		return apiContext.userId
	}

	url := fmt.Sprintf("%s/api/v2/me", apiContext.HostUrl)
	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Circle-Token", apiContext.Token)
	req.Header.Set("User-Agent", utils.UserAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ""
	}

	var userRes MeRes
	err = json.Unmarshal(body, &userRes)
	if err != nil {
		return ""
	}

	apiContext.userId = userRes.Id

	return apiContext.userId
}
