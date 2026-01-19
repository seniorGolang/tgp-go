// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"strings"

	"tgp/internal/model"
)

// ClientRenderer содержит общую функциональность для генерации клиента.
type ClientRenderer struct {
	project *model.Project
	outDir  string
}

// NewClientRenderer создает новый рендерер клиента.
func NewClientRenderer(project *model.Project, outDir string) *ClientRenderer {
	return &ClientRenderer{
		project: project,
		outDir:  outDir,
	}
}

// pkgPath возвращает путь пакета для указанной директории.
func (r *ClientRenderer) pkgPath(dir string) string {

	// В WASM файловая система монтируется в корень "/", поэтому используем относительные пути
	// dir уже является относительным путем от корня файловой системы
	// Преобразуем относительный путь в путь пакета
	pkgDir := filepath.ToSlash(dir)

	// Убираем ведущий "./" если есть
	pkgDir = strings.TrimPrefix(pkgDir, "./")

	// Если pkgDir не пустой, добавляем "/" в начало для формирования пути пакета
	if pkgDir != "" && !strings.HasPrefix(pkgDir, "/") {
		pkgDir = "/" + pkgDir
	}

	return r.project.ModulePath + pkgDir
}

// HasJsonRPC проверяет, есть ли контракты с JSON-RPC.
func (r *ClientRenderer) HasJsonRPC() bool {

	for _, contract := range r.project.Contracts {
		if contract.Annotations.IsSet(TagServerJsonRPC) {
			return true
		}
	}
	return false
}

// HasHTTP проверяет, есть ли контракты с HTTP.
func (r *ClientRenderer) HasHTTP() bool {

	for _, contract := range r.project.Contracts {
		if contract.Annotations.IsSet(TagServerHTTP) {
			return true
		}
	}
	return false
}

// HasMetrics проверяет, есть ли контракты с метриками.
func (r *ClientRenderer) HasMetrics() bool {

	for _, contract := range r.project.Contracts {
		if contract.Annotations.IsSet(TagMetrics) {
			return true
		}
	}
	return false
}

// Все методы Render* реализованы в соответствующих файлах:
// - RenderClientOptions - options.go
// - RenderVersion - version.go
// - RenderClient - client.go
// - RenderClientError - error.go
// - RenderClientBatch - batch.go
// - CollectTypeIDsForExchange - collector.go
// - RenderClientTypes - types.go
// - RenderExchange - exchange.go
// - RenderServiceClient - service.go
// - RenderClientMetrics - metrics.go
// - RenderReadmeGo - readme.go
