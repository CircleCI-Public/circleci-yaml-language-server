package methods

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/rollbar/rollbar-go"
)

func init() {
	rollbar.SetEnabled(false)
	rollbar.SetToken("acbd150ea5714049add26080daf522bf")
	rollbar.SetCaptureIp(rollbar.CaptureIpNone)
	rollbar.SetCodeVersion(utils.ServerVersion)
	rollbar.SetPlatform("client")
	rollbar.SetServerHost("localhost")
	rollbar.SetServerRoot("github.com/CircleCI-Public/circleci-yaml-language-server")
}
