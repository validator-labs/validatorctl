package test

type TestCase interface {
	GetName() string
	GetDescription() string
	Execute(ctx *TestContext) *TestResult
	TearDown(ctx *TestContext)

	// Builder methods
	Name(name string) TestCase
	Description(description string) TestCase
}

type BaseTest struct {
	name        string
	description string
}

func NewBaseTest(name string, description string, input interface{}) *BaseTest {
	return &BaseTest{name: name, description: description}
}

func (b *BaseTest) Name(name string) TestCase {
	b.name = name
	return b
}

func (b *BaseTest) GetName() string {
	return b.name
}

func (b *BaseTest) Description(description string) TestCase {
	b.description = description
	return b
}

func (b *BaseTest) GetDescription() string {
	return b.description
}

func (b *BaseTest) Execute(ctx *TestContext) *TestResult {
	return Failure("Test case not implemented")
}

func (b *BaseTest) TearDown(ctx *TestContext) {}

type test struct {
	testCase   TestCase
	TestResult *TestResult
}

func Test(testCase TestCase) *test {
	t := &test{testCase: testCase}
	return t
}

func (t *test) Execute(ctx *TestContext) *test {
	t.TestResult = t.testCase.Execute(ctx)
	return t
}
