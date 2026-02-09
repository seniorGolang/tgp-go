package srvctx

import (
	"reflect"
	"sync"
)

var keyCache sync.Map

func cachedKey(rt reflect.Type) string {

	if v, ok := keyCache.Load(rt); ok {
		return v.(string)
	}
	s := rt.String()
	keyCache.Store(rt, s)
	return s
}
