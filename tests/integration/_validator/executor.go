package validator

import (
	"log"

	validator "github.com/validator-labs/validatorctl/tests/integration/_validator/testcases"
	"github.com/validator-labs/validatorctl/tests/integration/common"
	"github.com/validator-labs/validatorctl/tests/utils/test"
)

func Execute(ctx *test.TestContext) error {
	log.Printf("-----------------------------------")
	log.Printf("--------- Validator Suite ----------")
	log.Printf("-----------------------------------")
	return test.Flow(ctx).
		Test(common.NewSingleFuncTest("validator-test", validator.Execute)).
		Summarize().TearDown().Audit()
}
