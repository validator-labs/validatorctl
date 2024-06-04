package string

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var (
	dnsNameRegex = regexp.MustCompile(`[^a-zA-Z0-9\-\.]`)
	dotsRegex    = regexp.MustCompile(`\.{2,}`)
	dashesRegex  = regexp.MustCompile(`\-{2,}`)
)

func Contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

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

func ConvertToDNSName(input string) string {
	// Lowercase & remove leading/trailing whitespaces
	input = strings.ToLower(strings.TrimSpace(input))

	// Remove invalid characters
	input = dnsNameRegex.ReplaceAllString(input, "-")

	// Replace consecutive dots with a single dot
	input = dotsRegex.ReplaceAllString(input, ".")

	// Replace consecutive dashes with a single dash
	input = dashesRegex.ReplaceAllString(input, "-")

	// Truncate the string to 253 characters
	if len(input) > 253 {
		input = input[:253]
	}

	// Remove leading/trailing dots
	input = strings.Trim(input, ".")

	return input
}

func GetAirgapValues(f *os.File, airgapKeys []string) ([]string, error) {
	_, err := f.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	airgapValue := make([]string, len(airgapKeys))
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for i, key := range airgapKeys {
			if strings.Contains(line, key) {
				airgapValue[i] = getEnvValue(line)
			}
		}
	}
	return airgapValue, nil
}

func getEnvValue(line string) string {
	arr := strings.Split(line, "=")
	return strings.TrimSpace(arr[1])
}
