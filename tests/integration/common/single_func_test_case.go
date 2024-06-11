package common

import (
	log "github.com/sirupsen/logrus"

	"github.com/validator-labs/validatorctl/tests/utils/test"
)

type SingleFuncTestCase struct {
	log *log.Entry
	*test.BaseTest

	testFunc func() error
}

func NewSingleFuncTest(name string, testFunc func() error) *SingleFuncTestCase {
	return &SingleFuncTestCase{
		log:      log.WithField("func", name),
		testFunc: testFunc,
		BaseTest: test.NewBaseTest(name, name, nil),
	}
}

func (r *SingleFuncTestCase) Execute(ctx *test.TestContext) (tr *test.TestResult) {
	r.log.Printf("Executing %s", r.GetName())
	if err := r.testFunc(); err != nil {
		r.log.Errorf("Failed to run test: %v", err)
		return test.Failure(err.Error())
	}
	return test.Success()
}

func (r *SingleFuncTestCase) TearDown(ctx *test.TestContext) {
	r.log.Printf("Executing TearDown for %s and %s", r.GetName(), r.GetDescription())
}
