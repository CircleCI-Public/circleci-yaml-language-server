package methods

import (
	"fmt"
	"runtime"

	"github.com/rollbar/rollbar-go"
)

var (
	RollbarToken = ""
)

func isRollbarEnabled() bool {
	return len(RollbarToken) != 0
}

func init() {
	rollbar.SetEnabled(false)
	rollbar.SetToken(RollbarToken)

	rollbar.SetCaptureIp(rollbar.CaptureIpNone)
	rollbar.SetEnvironment("production")
	rollbar.SetCustom(map[string]interface{}{
		"machine": fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})
	rollbar.SetPlatform("client")
	rollbar.SetServerHost("localhost")
	rollbar.SetServerRoot("github.com/CircleCI-Public/circleci-yaml-language-server")
}
