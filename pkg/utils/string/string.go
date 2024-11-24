// Package string contains utility functions for string manipulation.
package string

import (
	"strings"

	"github.com/google/uuid"
)

// Capitalize capitalizes the first letter of a string.
func Capitalize(s string) string {
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

// MultiTrim trims a string of multiple prefixes and suffixes.
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
func RandStr(strLen int) string {
	return uuid.NewString()[:strLen]
}

// IsDevVersion checks if a given CLI version is a development version.
func IsDevVersion(cliVersion interface{}) bool {
	return strings.HasSuffix(cliVersion.(string), "-dev")
}
