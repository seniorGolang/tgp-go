// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"strings"
	"unicode"

	"tgp/internal/model"
)

type ClientRenderer struct {
	project        *model.Project
	outDir         string
	contract       *model.Contract
	knownTypes     map[string]int
	typeDefTs      map[string]typeDefTs
	typeAnchorsSet map[string]bool // множество якорей типов из секции «Общие типы» (заполняется при генерации readme)
}

func NewClientRenderer(project *model.Project, outDir string) *ClientRenderer {
	return &ClientRenderer{
		project:    project,
		outDir:     outDir,
		knownTypes: make(map[string]int),
		typeDefTs:  make(map[string]typeDefTs),
	}
}

func (r *ClientRenderer) HasJsonRPC() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) HasHTTP() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) isTypeFromCurrentProject(importPkgPath string) bool {
	// Если ImportPkgPath начинается с ModulePath проекта, это тип из текущего проекта
	if r.project.ModulePath != "" && strings.HasPrefix(importPkgPath, r.project.ModulePath) {
		return true
	}
	return false
}

func (r *ClientRenderer) tsFileName(contract *model.Contract) string {
	name := contract.Name
	if len(name) == 0 {
		return ""
	}
	// Для случаев типа "HTTPService" -> "http_service", "JsonRPCService" -> "json_rpc_service", "JWKS" -> "jwks"
	result := make([]rune, 0, len(name)*2)
	for i, r := range name {
		if i > 0 {
			prevR := rune(name[i-1])
			if unicode.IsUpper(r) && unicode.IsLower(prevR) {
				result = append(result, '_')
			}
			// если следующая буква маленькая
			if unicode.IsUpper(r) && unicode.IsUpper(prevR) {
				if i+1 < len(name) && unicode.IsLower(rune(name[i+1])) {
					result = append(result, '_')
				}
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

func (r *ClientRenderer) lcName(s string) string {
	if len(s) == 0 {
		return ""
	}
	return toLowerCamel(s)
}

func (r *ClientRenderer) requestTypeName(contract *model.Contract, method *model.Method) string {
	return fmt.Sprintf("Request%s%s", contract.Name, method.Name)
}

func (r *ClientRenderer) responseTypeName(contract *model.Contract, method *model.Method) string {
	return fmt.Sprintf("Response%s%s", contract.Name, method.Name)
}

func toLowerCamel(s string) string {
	if s == "" {
		return s
	}
	isAllUpper := true
	for _, v := range s {
		if v >= 'a' && v <= 'z' {
			isAllUpper = false
			break
		}
	}
	if isAllUpper {
		return s
	}
	if len(s) > 0 {
		first := rune(s[0])
		if first >= 'A' && first <= 'Z' {
			s = strings.ToLower(string(first)) + s[1:]
		}
	}
	// Убираем подчеркивания и преобразуем в camelCase
	parts := strings.Split(s, "_")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	return result
}

// Все методы Render* реализованы в соответствующих файлах:
// - RenderClientOptions - options.go
// - RenderVersion - version.go
// - RenderClient - client.go
// - RenderClientError - error.go
// - RenderClientBatch - batch.go
// - CollectTypeIDsForExchange - collector.go
// - RenderClientTypes - types.go
// - RenderExchangeTypes - exchange.go
// - RenderJsonRPCClientClass - jsonrpc-client.go
// - RenderHTTPClientClass - http-client.go
// - RenderReadmeTS - readme.go
// - RenderTsConfig - tsconfig.go
