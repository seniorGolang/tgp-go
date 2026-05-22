// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package types

type CacheLogger interface {
	OnCacheHit()
	OnCacheMiss()
}

var cacheLogger CacheLogger

func SetCacheLogger(logger CacheLogger) {

	cacheLogger = logger
}
