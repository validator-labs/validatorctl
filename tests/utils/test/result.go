package test

type TestResult struct {
	succeeded bool
	Errors    []string
}

func (t *TestResult) IsSucceeded() bool {
	return t.Errors == nil
}

func (t *TestResult) IsFailed() bool {
	return t.Errors != nil
}

func (t *TestResult) Succeeded() {
	t.succeeded = true
}

func (t *TestResult) Failed() {
	t.succeeded = false
}

func Success() *TestResult {
	return &TestResult{
		succeeded: true,
		Errors:    nil,
	}
}

func Failure(errors ...string) *TestResult {
	return &TestResult{
		succeeded: false,
		Errors:    errors,
	}
}
