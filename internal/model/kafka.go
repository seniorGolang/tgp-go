// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"
)

const (
	KafkaAcksAllISR   = "allISRAcks"
	KafkaAcksLeader   = "leaderAck"
	KafkaAcksNoAck    = "noAck"
	KafkaCodecJSON    = "json"
	KafkaCodecBytes   = "bytes"
	KafkaCodecMsgpack = "msgpack"
	KafkaCodecCBOR    = "cbor"
	KafkaCodecYAML    = "yaml"
	KafkaCodecXML     = "xml"
)

// ContractHasLegacyKafkaRole — устаревшие @tg kafka-consumer / kafka-publisher.
func ContractHasLegacyKafkaRole(project *Project, contract *Contract) (ok bool) {

	return IsAnnotationSet(project, contract, nil, nil, TagKafkaConsumer) ||
		IsAnnotationSet(project, contract, nil, nil, TagKafkaPublisher)
}

// MethodKafkaTopic — топик метода (после trim).
func MethodKafkaTopic(project *Project, contract *Contract, method *Method) (topic string) {

	return strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagKafkaTopic, ""))
}

// MethodKafkaKeyArg — имя аргумента ключа записи.
func MethodKafkaKeyArg(project *Project, contract *Contract, method *Method) (argName string) {

	return strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagKafkaKey, ""))
}

// MethodKafkaHeaderItems — маппинг kafka-headers (arg|header).
func MethodKafkaHeaderItems(project *Project, contract *Contract, method *Method) (items []ArgMapItem) {

	raw := strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagKafkaHeaders, ""))
	return ParseArgMapEntries(raw)
}

// MethodKafkaCodec — итоговый кодек метода (method → interface → json).
func MethodKafkaCodec(project *Project, contract *Contract, method *Method) (codec string) {

	codec = strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagKafkaCodec, KafkaCodecJSON))
	if codec == "" {
		return KafkaCodecJSON
	}
	return codec
}

// ContractKafkaAcks — значение kafka-acks на интерфейсе (пусто = нет явного дефолта).
func ContractKafkaAcks(project *Project, contract *Contract) (acks string) {

	return strings.TrimSpace(GetAnnotationValue(project, contract, nil, nil, TagKafkaAcks, ""))
}

// MethodKafkaAcks — итоговый acks: method → interface → allISRAcks.
func MethodKafkaAcks(project *Project, contract *Contract, method *Method) (acks string) {

	if method != nil {
		if raw := strings.TrimSpace(method.Annotations.Value(TagKafkaAcks, "")); raw != "" {
			return raw
		}
	}
	if acks = ContractKafkaAcks(project, contract); acks != "" {
		return acks
	}
	return KafkaAcksAllISR
}

// MethodKafkaMessageArgName — явное имя сообщения: method → interface (без эвристики).
func MethodKafkaMessageArgName(project *Project, contract *Contract, method *Method) (argName string) {

	if method != nil {
		if raw := strings.TrimSpace(method.Annotations.Value(TagKafkaMessage, "")); raw != "" {
			return raw
		}
	}
	if contract != nil {
		return strings.TrimSpace(contract.Annotations.Value(TagKafkaMessage, ""))
	}
	return ""
}

// MethodKafkaMessageArg — аргумент сообщения: явный тег / эвристика.
func MethodKafkaMessageArg(project *Project, contract *Contract, method *Method) (arg *Variable, ok bool) {

	if method == nil {
		return nil, false
	}
	if name := MethodKafkaMessageArgName(project, contract, method); name != "" {
		for _, candidate := range method.Args {
			if candidate.Name == name {
				return candidate, true
			}
		}
		return nil, false
	}
	return methodKafkaMessageHeuristic(project, contract, method)
}

// MethodKafkaPayloadArg — алиас MethodKafkaMessageArg (совместимость хелперов).
func MethodKafkaPayloadArg(project *Project, contract *Contract, method *Method) (arg *Variable, ok bool) {

	return MethodKafkaMessageArg(project, contract, method)
}

// MethodKafkaExtraArgs — аргументы вне ctx / message / key / headers (для warn).
func MethodKafkaExtraArgs(project *Project, contract *Contract, method *Method) (extras []*Variable) {

	if method == nil {
		return nil
	}
	message, hasMessage := MethodKafkaMessageArg(project, contract, method)
	keyArg := MethodKafkaKeyArg(project, contract, method)
	headers := MethodKafkaHeaderItems(project, contract, method)
	headerArgs := make(map[string]struct{}, len(headers))
	for _, item := range headers {
		headerArgs[item.Arg] = struct{}{}
	}
	for _, candidate := range method.Args {
		if isContextArg(candidate) {
			continue
		}
		if hasMessage && candidate.Name == message.Name {
			continue
		}
		if keyArg != "" && candidate.Name == keyArg {
			continue
		}
		if _, isHeader := headerArgs[candidate.Name]; isHeader {
			continue
		}
		extras = append(extras, candidate)
	}
	return extras
}

// TypeRefIsByteSlice — []byte (не [][]byte, не ...[]byte).
func TypeRefIsByteSlice(typeRef *TypeRef) (ok bool) {

	if typeRef == nil || typeRef.IsEllipsis || typeRef.NumberOfPointers != 0 {
		return false
	}
	return typeRef.IsSlice && typeRef.TypeID == "byte" && typeRef.ElementPointers == 0
}

// TypeRefIsByteSliceSlice — [][]byte.
func TypeRefIsByteSliceSlice(typeRef *TypeRef) (ok bool) {

	if typeRef == nil || typeRef.IsEllipsis || typeRef.NumberOfPointers != 0 {
		return false
	}
	return typeRef.IsSlice && typeRef.TypeID == "[]byte" && typeRef.ElementPointers == 0
}

// TypeRefIsByteSliceEllipsis — ...[]byte.
func TypeRefIsByteSliceEllipsis(typeRef *TypeRef) (ok bool) {

	if typeRef == nil || !typeRef.IsEllipsis || typeRef.NumberOfPointers != 0 {
		return false
	}
	return typeRef.TypeID == "byte" && typeRef.ElementPointers == 0
}

// TypeRefIsKafkaKeyOrHeader — допустимый тип ключа/заголовка (§5.3).
func TypeRefIsKafkaKeyOrHeader(typeRef *TypeRef) (ok bool) {

	if typeRef == nil || typeRef.NumberOfPointers != 0 {
		return false
	}
	if typeRef.TypeID == "string" && !typeRef.IsSlice && !typeRef.IsEllipsis {
		return true
	}
	if TypeRefIsByteSlice(typeRef) {
		return true
	}
	if typeRef.IsSlice && !typeRef.IsEllipsis && typeRef.TypeID == "string" && typeRef.ElementPointers == 0 {
		return true
	}
	if TypeRefIsByteSliceSlice(typeRef) {
		return true
	}
	return false
}

// TypeRefKafkaMessageElement — элемент сообщения после снятия [] / ... (кроме атомарного []byte).
func TypeRefKafkaMessageElement(typeRef *TypeRef) (elem TypeRef, ok bool) {

	if typeRef == nil {
		return TypeRef{}, false
	}
	if TypeRefIsByteSlice(typeRef) {
		return *typeRef, true
	}
	if TypeRefIsByteSliceEllipsis(typeRef) || TypeRefIsByteSliceSlice(typeRef) {
		return TypeRef{TypeID: "byte", IsSlice: true}, true
	}
	elem = *typeRef
	if elem.IsSlice || elem.IsEllipsis {
		elem.IsSlice = false
		elem.IsEllipsis = false
		return elem, true
	}
	return elem, true
}

func methodKafkaMessageHeuristic(project *Project, contract *Contract, method *Method) (arg *Variable, ok bool) {

	skip := make(map[string]struct{})
	if keyArg := MethodKafkaKeyArg(project, contract, method); keyArg != "" {
		skip[keyArg] = struct{}{}
	}
	for _, item := range MethodKafkaHeaderItems(project, contract, method) {
		skip[item.Arg] = struct{}{}
	}
	var found *Variable
	count := 0
	for _, candidate := range method.Args {
		if isContextArg(candidate) {
			continue
		}
		if _, excluded := skip[candidate.Name]; excluded {
			continue
		}
		count++
		found = candidate
	}
	if count != 1 {
		return nil, false
	}
	return found, true
}
