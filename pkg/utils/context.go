package utils

type LsContext struct {
	Api    ApiContext
	UserId string
}

type ApiContext struct {
	Token   string
	HostUrl string
}

func (apiContext ApiContext) UseDefaultInstance() bool {
	return apiContext.HostUrl == CIRCLE_CI_APP_HOST_URL
}
