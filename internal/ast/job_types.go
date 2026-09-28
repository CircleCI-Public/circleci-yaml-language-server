package ast

// JobTypes is a list of all valid job types that are supported by CircleCI
var JobTypes = []string{
	"approval",
	"build", // default
	"no-op",
	"release",
	"lock",
	"unlock",
}

// CheckoutMethods is a list of all valid checkout methods that are supported by CircleCI
var CheckoutMethods = []string{
	"blobless",
	"full", // default
	"shallow",
}
