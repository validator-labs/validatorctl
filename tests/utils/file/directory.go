package file

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const testDir = "tests"

func ValidatorTestCasesPath() string {
	// note: 'validatorctl' was used here, rather than 'validator', due to: https://github.com/helm/helm/issues/7862
	return fmt.Sprintf("%s/%s", HomePath(testDir), "tests/integration/validatorctl/testcases")
}

func HomePath(dir string) string {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}
	return strings.TrimSuffix(pwd[:strings.Index(pwd, dir)], "/")
}
