package utils

func GetYAMLBooleanValue(str string) bool {
	if str == "true" || str == "yes" || str == "y" || str == "1" || str == "on" {
		return true
	}
	return false
}

func IsValidYAMLBooleanValue(str string) bool {
	if str == "true" || str == "yes" || str == "y" || str == "1" || str == "on" ||
		str == "false" || str == "no" || str == "n" || str == "0" || str == "off" {
		return true
	}
	return false
}
