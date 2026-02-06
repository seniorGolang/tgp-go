//go:build pluginInfo

package main

import (
	"tgp/core/manifest"
)

func init() {

	// При сборке с тегом pluginInfo генерируем манифест
	// translator уже инициализирован в translate.go через init()
	manifest.GenerateFromArgs(&BaseGoPlugin{})
}
