package complete

import (
	"regexp"
)

var (
	listItemKey = regexp.MustCompile(`^\s*-\s+([\w-]+)\s*:`)
	dockerKey   = regexp.MustCompile(`^\s*docker\s*:\s*$`)
)

// dockerEntryKeys are the keys an image of a docker executor takes.
var dockerEntryKeys = []string{"image", "name", "entrypoint", "command", "user", "environment", "auth", "aws_auth"}

// dockerAuthKeys are the keys of each of a docker image's credentials.
var dockerAuthKeys = map[string][]string{
	"auth":     {"username", "password"},
	"aws_auth": {"aws_access_key_id", "aws_secret_access_key", "oidc_role_arn"},
}

// completeDockerEntry offers the keys an image of a docker executor, or its
// credentials, don't have yet, when the cursor is at a key of one, and says
// whether it is.
func (ch *CompletionHandler) completeDockerEntry() bool {
	lines, parent := ch.keyParent()
	if parent == -1 {
		return false
	}

	var keys []string
	present := ch.stepBodyKeys(parent)
	if match := listItemKey.FindStringSubmatch(lines[parent]); match != nil {
		if docker := parentLine(lines, parent); docker == -1 || !dockerKey.MatchString(lines[docker]) {
			return false
		}
		keys = dockerEntryKeys
		present[match[1]] = true
	} else if match := mappingKey.FindStringSubmatch(lines[parent]); match != nil && dockerAuthKeys[match[1]] != nil {
		entry := parentLine(lines, parent)
		if entry == -1 || !listItemKey.MatchString(lines[entry]) {
			return false
		}
		if docker := parentLine(lines, entry); docker == -1 || !dockerKey.MatchString(lines[docker]) {
			return false
		}
		keys = dockerAuthKeys[match[1]]
	} else {
		return false
	}

	for _, key := range keys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
	return true
}
