package test

type TestContext struct {
	ctx GenericMap
}

func NewTestContext() *TestContext {
	return &TestContext{}
}

func (t *TestContext) Put(key string, value interface{}) *TestContext {
	t.ctx.Put(key, value)
	return t
}

func (t *TestContext) Get(key string) interface{} {
	return t.ctx.Get(key)
}

func (t *TestContext) GetStr(key string) string {
	return t.ctx.GetStr(key)
}
