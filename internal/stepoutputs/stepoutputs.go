// Package stepoutputs reads references to the outputs a function step sets,
// which a later step reads as `${{ steps.<id>.outputs.<name> }}`.
package stepoutputs

import "regexp"

// Pattern matches a reference to an output of a function step. Its groups
// are the step's id, the output's name and the index into a list output, if
// there is one. Only this form is a reference: any other `${{ }}` is left
// as written.
var Pattern = regexp.MustCompile(`\$\{\{\s*steps\.([A-Za-z][A-Za-z0-9_-]*)\.outputs\.([A-Za-z][A-Za-z0-9_-]*)(?:\[(\d+)\])?\s*\}\}`)

// Contains reports whether text holds a reference to a step's output.
func Contains(text string) bool {
	return Pattern.MatchString(text)
}
