package utils

func FindInArray[T comparable](array []T, elem T) int {
	for i, v := range array {
		if v == elem {
			return i
		}
	}
	return -1
}
