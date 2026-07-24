// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"
)

const (
	StreamModeServer = "server"
	StreamModeClient = "client"
	StreamModeBidi   = "bidi"
)

const (
	JSONRPCStreamMethod    = "$/stream"
	JSONRPCStreamEndMethod = "$/stream.end"
	JSONRPCCancelMethod    = "$/cancel"
)

// MethodStreamMode возвращает режим потока метода или пустую строку.
func MethodStreamMode(project *Project, contract *Contract, method *Method) (mode string) {

	mode = strings.ToLower(strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagStream, "")))
	switch mode {
	case StreamModeServer, StreamModeClient, StreamModeBidi:
		return mode
	default:
		return ""
	}
}

// MethodIsStream — метод с аннотацией stream=server|client|bidi.
func MethodIsStream(project *Project, contract *Contract, method *Method) (ok bool) {

	return MethodStreamMode(project, contract, method) != ""
}

// MethodIsHTTP — REST-метод (http-server и явный http-method при hybrid с jsonRPC).
func MethodIsHTTP(project *Project, contract *Contract, method *Method) (ok bool) {

	if contract == nil || method == nil {
		return false
	}
	if MethodIsStream(project, contract, method) {
		return false
	}
	if !IsAnnotationSet(project, contract, nil, nil, TagServerHTTP) {
		return false
	}
	if !IsAnnotationSet(project, contract, nil, nil, TagServerJsonRPC) {
		return true
	}
	return IsAnnotationSet(project, contract, method, nil, TagHTTPMethod)
}

// MethodIsJSONRPC — unary JSON-RPC метод.
func MethodIsJSONRPC(project *Project, contract *Contract, method *Method) (ok bool) {

	if contract == nil || method == nil {
		return false
	}
	if MethodIsStream(project, contract, method) {
		return false
	}
	if !IsAnnotationSet(project, contract, nil, nil, TagServerJsonRPC) {
		return false
	}
	return !IsAnnotationSet(project, contract, method, nil, TagHTTPMethod)
}

// MethodIsWS — stream-метод на контракте с ws-server.
func MethodIsWS(project *Project, contract *Contract, method *Method) (ok bool) {

	if !MethodIsStream(project, contract, method) {
		return false
	}
	return IsAnnotationSet(project, contract, nil, nil, TagServerWS)
}

// MethodIsSSE — server-stream на контракте с sse-server.
func MethodIsSSE(project *Project, contract *Contract, method *Method) (ok bool) {

	if MethodStreamMode(project, contract, method) != StreamModeServer {
		return false
	}
	return IsAnnotationSet(project, contract, nil, nil, TagServerSSE)
}

// ContractHasWS — на контракте включён WebSocket-транспорт.
func ContractHasWS(project *Project, contract *Contract) (ok bool) {

	return IsAnnotationSet(project, contract, nil, nil, TagServerWS)
}

// ContractHasSSE — на контракте включён SSE-транспорт.
func ContractHasSSE(project *Project, contract *Contract) (ok bool) {

	return IsAnnotationSet(project, contract, nil, nil, TagServerSSE)
}

// ContractWSPath — путь upgrade WebSocket для контракта.
func ContractWSPath(project *Project, contract *Contract) (path string) {

	path = strings.TrimSpace(GetAnnotationValue(project, contract, nil, nil, TagWSPath, ""))
	if path != "" {
		return JoinHTTPPath(GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, ""), path)
	}
	prefix := GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, "")
	return JoinHTTPPath(prefix, "/ws/"+LowerCamel(contract.Name))
}

// MethodSSEPath — путь SSE endpoint для server-stream метода.
func MethodSSEPath(project *Project, contract *Contract, method *Method) (path string) {

	path = strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagSSEPath, ""))
	if path != "" {
		return JoinHTTPPath(GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, ""), path)
	}
	prefix := GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, "")
	return JoinHTTPPath(prefix, "/sse/"+LowerCamel(contract.Name)+"/"+LowerCamel(method.Name))
}

// TypeRefIsChan — TypeRef описывает канал (напрямую или через TypeKindChan).
func TypeRefIsChan(project *Project, typeRef *TypeRef) (ok bool) {

	if typeRef == nil {
		return false
	}
	if typeRef.ChanOf != nil {
		return true
	}
	if typeRef.TypeID == "" || project == nil {
		return false
	}
	typ, found := project.Types[typeRef.TypeID]
	return found && typ != nil && typ.Kind == TypeKindChan
}

// TypeRefChanElement — элемент канала из TypeRef.
func TypeRefChanElement(project *Project, typeRef *TypeRef) (element *TypeRef, ok bool) {

	if typeRef == nil {
		return nil, false
	}
	if typeRef.ChanOf != nil {
		return typeRef.ChanOf, true
	}
	if typeRef.TypeID == "" || project == nil {
		return nil, false
	}
	typ, found := project.Types[typeRef.TypeID]
	if !found || typ == nil || typ.Kind != TypeKindChan || typ.ChanOfID == "" {
		return nil, false
	}
	return &TypeRef{TypeID: typ.ChanOfID}, true
}

// MethodStreamInChan — входной канал метода (client/bidi).
func MethodStreamInChan(project *Project, method *Method) (arg *Variable, element *TypeRef, ok bool) {

	if method == nil {
		return nil, nil, false
	}
	for _, candidate := range method.Args {
		if candidate.TypeID == "context:Context" {
			continue
		}
		var elem *TypeRef
		if elem, ok = TypeRefChanElement(project, &candidate.TypeRef); ok {
			return candidate, elem, true
		}
	}
	return nil, nil, false
}

// MethodStreamOutChan — выходной канал метода (server/bidi).
func MethodStreamOutChan(project *Project, method *Method) (result *Variable, element *TypeRef, ok bool) {

	if method == nil {
		return nil, nil, false
	}
	for _, candidate := range method.Results {
		if candidate.TypeID == "error" {
			continue
		}
		var elem *TypeRef
		if elem, ok = TypeRefChanElement(project, &candidate.TypeRef); ok {
			return candidate, elem, true
		}
	}
	return nil, nil, false
}
