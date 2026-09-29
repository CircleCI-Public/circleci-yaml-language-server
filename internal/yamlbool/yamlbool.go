package yamlbool

import "slices"

func Value(str string) bool {
	if str == "true" || str == "yes" || str == "y" || str == "1" || str == "on" {
		return true
	}
	return false
}

func IsValid(str string) bool {
	if str == "true" || str == "yes" || str == "y" || str == "1" || str == "on" ||
		str == "false" || str == "no" || str == "n" || str == "0" || str == "off" {
		return true
	}
	return false
}

// yaml11 are the plain scalars the compiler's YAML 1.1 parser reads as
// booleans, and a YAML 1.2 parser reads as strings.
var yaml11 = []string{
	"yes", "Yes", "YES", "no", "No", "NO",
	"on", "On", "ON", "off", "Off", "OFF",
}

// IsYAML11 reports whether a plain scalar is a boolean only in YAML 1.1.
func IsYAML11(text string) bool {
	return slices.Contains(yaml11, text)
}
