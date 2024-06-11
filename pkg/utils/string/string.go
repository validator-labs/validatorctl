package string

import "strings"

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
