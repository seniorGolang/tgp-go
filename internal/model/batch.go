// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"path"
	"sort"
	"strings"
)

// NormalizeHTTPPrefix приводит @tg http-prefix к каноническому виду ("/", "/api/v1").
func NormalizeHTTPPrefix(prefix string) (normalized string) {

	prefix = strings.TrimSpace(prefix)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return "/"
	}

	return path.Join("/", prefix)
}

// HTTPPrefixAncestors возвращает сам prefix и его предков без корня ("/api/v1" → ["/api/v1", "/api"]).
func HTTPPrefixAncestors(prefix string) (ancestors []string) {

	normalized := NormalizeHTTPPrefix(prefix)
	if normalized == "/" {
		return nil
	}

	parts := strings.Split(strings.TrimPrefix(normalized, "/"), "/")
	ancestors = make([]string, 0, len(parts))
	for i := len(parts); i >= 1; i-- {
		ancestors = append(ancestors, "/"+strings.Join(parts[:i], "/"))
	}

	return ancestors
}

// ContractInJSONRPCBatchScope — контракт с prefix виден на batchPath ("/" = все; иначе path-prefix match).
func ContractInJSONRPCBatchScope(contractPrefix string, batchPath string) (inScope bool) {

	batchPath = NormalizeHTTPPrefix(batchPath)
	if batchPath == "/" {
		return true
	}

	contractPrefix = NormalizeHTTPPrefix(contractPrefix)
	if contractPrefix == "/" {
		return false
	}

	return contractPrefix == batchPath || strings.HasPrefix(contractPrefix, batchPath+"/")
}

// JSONRPCBatchMounts — пути multiplex batch: "/" ∪ exact http-prefix ∪ предки, sorted.
func JSONRPCBatchMounts(project *Project) (mounts []string) {

	seen := map[string]struct{}{"/": {}}
	if project == nil {
		return []string{"/"}
	}

	for _, contract := range project.Contracts {
		if !IsAnnotationSet(project, contract, nil, nil, TagServerJsonRPC) {
			continue
		}
		prefix := GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, "")
		for _, ancestor := range HTTPPrefixAncestors(prefix) {
			seen[ancestor] = struct{}{}
		}
	}

	mounts = make([]string, 0, len(seen))
	for mount := range seen {
		mounts = append(mounts, mount)
	}
	sort.Strings(mounts)

	return mounts
}

// JSONRPCContractPrefix возвращает нормализованный http-prefix JSON-RPC контракта.
func JSONRPCContractPrefix(project *Project, contract *Contract) (prefix string) {

	if contract == nil {
		return "/"
	}

	return NormalizeHTTPPrefix(GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, ""))
}

// JSONRPCServiceBatchPath — POST-путь batch эндпоинта контракта (prefix + http-path|/name).
func JSONRPCServiceBatchPath(project *Project, contract *Contract) (servicePath string) {

	if contract == nil {
		return "/"
	}

	prefix := GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, "")
	pathValue := GetAnnotationValue(project, contract, nil, nil, TagHttpPath, "/"+LowerCamel(contract.Name))

	return JoinHTTPPath(prefix, pathValue)
}
