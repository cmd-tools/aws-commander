package helpers

import "github.com/cmd-tools/aws-commander/constants"

func IsStringEmpty(value string) bool {
	return value == constants.EmptyString
}

func AppendUniqueLast(slice []string, item string) []string {
	if len(slice) == 0 || slice[len(slice)-1] != item {
		return append(slice, item)
	}
	return slice
}
