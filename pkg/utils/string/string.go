package string

import (
	"strings"

	"github.com/google/uuid"
)

func Capitalize(s string) string {
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func MultiTrim(str string, prefixes, suffixes []string) string {
	for _, p := range prefixes {
		str = strings.TrimPrefix(str, p)
	}
	for _, s := range suffixes {
		str = strings.TrimSuffix(str, s)
	}
	return str
}

// RandStr generates a random string of a given length, which is the first
// N chars of a UUID of the form: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx, per RFC4122.
func RandStr(len int) string {
	return uuid.NewString()[:len]
}

// IsDevVersion checks if a given CLI version is a development version.
func IsDevVersion(cliVersion interface{}) bool {
	return strings.HasSuffix(cliVersion.(string), "-dev")
}
