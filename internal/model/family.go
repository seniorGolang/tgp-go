// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

// ContractMarks возвращает enable-семьи контракта по маркерам аннотаций.
func ContractMarks(project *Project, contract *Contract) (httpFamily bool, kafka bool) {

	if contract == nil {
		return false, false
	}
	httpFamily = ContractIsHTTPFamily(project, contract)
	kafka = ContractIsKafka(project, contract)
	return httpFamily, kafka
}

// ContractIsHTTPFamily — контракт HTTP-семьи: http-server / jsonRPC-server / ws-server / sse-server.
func ContractIsHTTPFamily(project *Project, contract *Contract) (ok bool) {

	if contract == nil {
		return false
	}
	return IsAnnotationSet(project, contract, nil, nil, TagServerHTTP) ||
		IsAnnotationSet(project, contract, nil, nil, TagServerJsonRPC) ||
		ContractHasWS(project, contract) ||
		ContractHasSSE(project, contract)
}

// ContractIsKafka — контракт с @tg kafka.
func ContractIsKafka(project *Project, contract *Contract) (ok bool) {

	if contract == nil {
		return false
	}
	return IsAnnotationSet(project, contract, nil, nil, TagKafka)
}
