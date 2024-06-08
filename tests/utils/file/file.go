package file

import "fmt"

func UnitTestFile(name string) string {
	return fmt.Sprintf("%s/%s/%s", HomePath("pkg"), "tests/unit-test-data", name)
}
