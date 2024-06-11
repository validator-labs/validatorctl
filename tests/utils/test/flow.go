package test

import (
	"strings"
	"time"

	"emperror.dev/errors"
	log "github.com/sirupsen/logrus"
)

type TestFlow struct {
	ctx        *TestContext
	skip       bool
	tests      []interface{}
	executions map[string]time.Duration
	Results    []*TestResult
}

func Flow(ctx *TestContext) *TestFlow {
	tf := TestFlow{
		ctx:        ctx,
		tests:      make([]interface{}, 0),
		executions: make(map[string]time.Duration),
	}
	return &tf
}

func (t *TestFlow) Test(testCase TestCase) *TestFlow {
	if t.skip {
		return t
	}
	log.Printf("-------------- %s -------------- ", testCase.GetName())
	start := time.Now()
	result := testCase.Execute(t.ctx)
	end := time.Now()
	t.executions[testCase.GetName()] = end.Sub(start)
	t.add(testCase)
	if result.IsFailed() {
		t.Results = append(t.Results, result)
		t.skip = true
	}
	return t
}

func (t *TestFlow) add(testCase TestCase) *TestFlow {
	t.tests = append(t.tests, testCase)
	return t
}

func (t *TestFlow) Summarize() *TestFlow {
	for k, v := range t.executions {
		log.Printf("Duration for test case %s: %v", k, v)
	}
	return t
}

func (t *TestFlow) TearDown() *TestFlow {
	for i := len(t.tests) - 1; i >= 0; i-- {
		tc := t.tests[i]
		testCase, ok := tc.(TestCase)
		if ok {
			testCase.TearDown(t.ctx)
		} else {
			testFlow, ok := tc.(*TestFlow)
			if ok {
				testFlow.TearDown()
			}
		}
	}
	return t
}

func (t *TestFlow) Audit() error {
	var err error
	for _, res := range t.Results {
		if res.IsFailed() {
			if err == nil {
				err = errors.New("[Failed]")
			}
			err = errors.Wrap(err, strings.Join(res.Errors, ", "))
		}
	}
	return err
}
