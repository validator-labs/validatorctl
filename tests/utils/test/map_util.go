package test

import (
	"fmt"
	"reflect"
)

type GenericMap struct {
	m map[string]interface{}
}

func (g *GenericMap) Put(key string, value interface{}) *GenericMap {
	if g.m == nil {
		g.m = make(map[string]interface{})
	}
	g.m[key] = value
	return g
}

func (g *GenericMap) Get(key string) interface{} {
	if g.m != nil {
		return g.m[key]
	}
	return nil
}

func (g *GenericMap) Remove(key string) {
	if g.m != nil {
		delete(g.m, key)
	}
}

func (g *GenericMap) GetStr(key string) string {
	if g.m != nil {
		v := g.m[key]
		if reflect.ValueOf(v).Kind() == reflect.String {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func MergeStrMap(m1, m2 map[string]string) map[string]string {
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}
