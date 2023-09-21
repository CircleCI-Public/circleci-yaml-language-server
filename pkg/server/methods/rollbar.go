package methods

import (
	"github.com/rollbar/rollbar-go"
)

func init() {
	rollbar.SetEnabled(false)
	rollbar.SetToken("acbd150ea5714049add26080daf522bf")
	rollbar.SetCaptureIp(rollbar.CaptureIpNone)
	rollbar.SetCodeVersion(ServerVersion)
	rollbar.SetPlatform("client")
	rollbar.SetServerHost("localhost")
	rollbar.SetServerRoot("github.com/CircleCI-Public/circleci-yaml-language-server")
}
