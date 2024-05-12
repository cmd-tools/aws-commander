package helpers

import "github.com/cmd-tools/aws-commander/constants"

func IsStringEmpty(value string) bool {
	return value == constants.EmptyString
}

func RemoveEmptyStrings(strings []string) []string {
	var result []string
	for _, str := range strings {
		if IsStringEmpty(str) {
			result = append(result, str)
		}
	}
	return result
}

func RemoveItem(slice []string, item string) []string {
	index := -1
	for i, s := range slice {
		if s == item {
			index = i
			break
		}
	}
	if index != -1 {
		return append(slice[:index], slice[index+1:]...)
	}
	return slice
}
