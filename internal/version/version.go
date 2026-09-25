// Package version identifies the language server build.
package version

// Server is the version of this build, stamped at build time by
// goreleaser (see .goreleaser.yml).
var Server string = "<dev build>"

// UserAgent is what the language server sends with every request it makes.
var UserAgent string = "CircleCI-Language-Server/" + Server
