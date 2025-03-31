package helpers

import "strings"

func StringBuild(input ...string) string {
	var stringBuild strings.Builder
	for _, str := range input {
		stringBuild.WriteString(str)
	}
	return stringBuild.String()
}
