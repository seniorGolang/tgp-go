// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"tgp/plugins/server/renderer/types"
)

type cacheLoggerImpl struct{}

func (c *cacheLoggerImpl) OnCacheHit() {

	incrementCacheHits()
}

func (c *cacheLoggerImpl) OnCacheMiss() {

	incrementCacheMisses()
}

func setupCacheLogger() {

	types.SetCacheLogger(&cacheLoggerImpl{})
}
