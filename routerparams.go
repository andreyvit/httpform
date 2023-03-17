package httpform

import (
	"fmt"
)

func interpretPathParams(pathParams any) pathParamsImpl {
	if pathParams == nil {
		return noPathParamsImpl{}
	} else if v, ok := pathParams.(bunRouterParams); ok {
		return &bunRouterParamsImpl{v}
	} else {
		panic(fmt.Errorf("unsupported pathParams %T", pathParams))
	}
}

type pathParamsImpl interface {
	Get(key string) string
	Keys() []string
}

type noPathParamsImpl struct{}

func (_ noPathParamsImpl) Get(key string) string {
	return ""
}

func (_ noPathParamsImpl) Keys() []string {
	return nil
}

type bunRouterParamsImpl struct {
	params bunRouterParams
}

func (impl bunRouterParamsImpl) Get(key string) string {
	v, _ := impl.params.Get(key)
	return v
}

func (impl bunRouterParamsImpl) Keys() []string {
	m := impl.params.Map()
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

type bunRouterParams interface {
	Get(name string) (string, bool)
	Map() map[string]string
}
