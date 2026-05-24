// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"
)

// JoinEndpointHTTPPrefix дописывает @tg http-prefix к endpoint (как HTTP-клиент: origin + "/" + prefix).
func JoinEndpointHTTPPrefix(endpoint string, prefix string) (joined string) {

	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return endpoint
	}

	return strings.TrimRight(endpoint, "/") + "/" + prefix
}

// JSONRPCClientEndpoint возвращает endpoint с http-prefix первого JSON-RPC контракта.
func JSONRPCClientEndpoint(project *Project, endpoint string) (rpcEndpoint string) {

	if project == nil {
		return endpoint
	}

	for _, contract := range project.Contracts {
		if !IsAnnotationSet(project, contract, nil, nil, TagServerJsonRPC) {
			continue
		}
		prefix := GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, "")

		return JoinEndpointHTTPPrefix(endpoint, prefix)
	}

	return endpoint
}
