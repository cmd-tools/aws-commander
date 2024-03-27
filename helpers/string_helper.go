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
