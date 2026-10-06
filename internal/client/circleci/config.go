package circleci

type Config struct {
	Token   string
	HostUrl string
}

func (apiContext Config) UseDefaultInstance() bool {
	return apiContext.HostUrl == DefaultHostURL
}

func (apiContext Config) IsLoggedIn() bool {
	return apiContext.Token != ""
}
