package testenv

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v2"
)

type TestData struct{}

func GetTestData() (*TestData, error) {
	td := &TestData{}

	all := map[string]interface{}{}

	// always read from the unified testdata directory
	_, f, _, _ := runtime.Caller(0)
	testDataRoot := filepath.Join(filepath.Dir(f), "../testenv/testdata")

	for file, v := range all {
		filename := fmt.Sprintf("%s/%s", testDataRoot, file)
		data, err := os.ReadFile(filename) //#nosec
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		if err = yaml.Unmarshal(data, v); err != nil {
			fmt.Println(err)
			return nil, err
		}
	}

	return td, nil
}
