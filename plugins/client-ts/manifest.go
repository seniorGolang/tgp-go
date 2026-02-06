// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

//go:build pluginInfo

package main

import (
	"tgp/core/manifest"
)

func init() {

	// При сборке с тегом pluginInfo генерируем манифест
	// translator уже инициализирован в translate.go через init()
	manifest.GenerateFromArgs(&ClientTsPlugin{})
}
