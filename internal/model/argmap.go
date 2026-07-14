// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"fmt"
	"strings"
)

const (
	ArgModeExplicit = "explicit"
	ArgModeImplicit = "implicit"
	ArgModeBody     = "body"

	typeIDContext      = "context:Context"
	typeIDIOReader     = "io:Reader"
	typeIDIOReadCloser = "io:ReadCloser"
)

func isContextArg(arg *Variable) (ok bool) {

	if arg == nil {
		return false
	}
	return arg.TypeID == typeIDContext || arg.TypeID == "context.Context"
}

// ArgMapItem is one entry from http-headers, http-cookies or http-args (arg|key or arg|key|mode).
type ArgMapItem struct {
	Arg  string
	Key  string
	Mode string
}

// ParseArgMapEntries parses comma-separated "arg|key" or "arg|key|mode" pairs.
// Missing mode defaults to ArgModeBody.
func ParseArgMapEntries(value string) (items []ArgMapItem) {

	if value == "" {
		return
	}

	for _, pair := range strings.Split(value, ",") {
		parts := strings.Split(strings.TrimSpace(pair), "|")
		if len(parts) < 2 {
			continue
		}

		arg := strings.TrimSpace(parts[0])
		key := strings.TrimSpace(parts[1])
		if arg == "" || key == "" {
			continue
		}

		mode := ArgModeBody
		if len(parts) >= 3 {
			mode = strings.TrimSpace(parts[2])
			if mode != ArgModeExplicit && mode != ArgModeImplicit && mode != ArgModeBody {
				mode = ArgModeBody
			}
		}

		items = append(items, ArgMapItem{
			Arg:  arg,
			Key:  key,
			Mode: mode,
		})
	}

	return items
}

// ParseArgMapEntriesStrict парсит http-args, http-headers и http-cookies с ошибкой на неверный формат.
func ParseArgMapEntriesStrict(value string) (items []ArgMapItem, err error) {

	if value == "" {
		return nil, nil
	}

	for index, pair := range strings.Split(value, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.Split(pair, "|")
		if len(parts) < 2 {
			return nil, fmt.Errorf("entry %q must be arg|key or arg|key|mode", pair)
		}

		arg := strings.TrimSpace(parts[0])
		key := strings.TrimSpace(parts[1])
		if arg == "" || key == "" {
			return nil, fmt.Errorf("entry %q must be arg|key or arg|key|mode", pair)
		}

		mode := ArgModeBody
		if len(parts) >= 3 {
			mode = strings.TrimSpace(parts[2])
			if mode != ArgModeExplicit && mode != ArgModeImplicit && mode != ArgModeBody {
				return nil, fmt.Errorf("entry %d %q: mode must be explicit, implicit or body", index+1, pair)
			}
		}

		items = append(items, ArgMapItem{
			Arg:  arg,
			Key:  key,
			Mode: mode,
		})
	}
	return items, nil
}

// ArgMapItemsByArg returns map arg -> ArgMapItem for the first occurrence of each arg.
func ArgMapItemsByArg(items []ArgMapItem) (itemsByArg map[string]ArgMapItem) {

	itemsByArg = make(map[string]ArgMapItem, len(items))

	for _, it := range items {
		if _, ok := itemsByArg[it.Arg]; !ok {
			itemsByArg[it.Arg] = it
		}
	}

	return itemsByArg
}

// HTTPArgMappings aggregates parsed http-headers, http-cookies and http-args entries
// for a конкретный метод.
type HTTPArgMappings struct {
	HeaderItems []ArgMapItem
	CookieItems []ArgMapItem
	ArgItems    []ArgMapItem
}

// BuildHTTPArgMappings returns parsed HTTPArgMappings for the given method.
func BuildHTTPArgMappings(project *Project, contract *Contract, method *Method) (mappings HTTPArgMappings) {

	headerValue := GetAnnotationValue(project, contract, method, nil, TagHttpHeader, "")
	cookieValue := GetAnnotationValue(project, contract, method, nil, TagHttpCookies, "")
	argValue := GetAnnotationValue(project, contract, method, nil, TagHttpArg, "")

	mappings.HeaderItems = ParseArgMapEntries(headerValue)
	mappings.CookieItems = ParseArgMapEntries(cookieValue)
	mappings.ArgItems = ParseArgMapEntries(argValue)

	return mappings
}

// HTTPImplicitArgSet returns set of arg names with mode implicit in http-headers, http-cookies or http-args (excluding path).
func HTTPImplicitArgSet(mappings HTTPArgMappings) (implicitArgs map[string]struct{}) {

	implicitArgs = make(map[string]struct{})

	for _, it := range mappings.HeaderItems {
		if it.Mode == ArgModeImplicit {
			implicitArgs[it.Arg] = struct{}{}
		}
	}

	for _, it := range mappings.CookieItems {
		if it.Mode == ArgModeImplicit {
			implicitArgs[it.Arg] = struct{}{}
		}
	}

	for _, it := range mappings.ArgItems {
		if it.Mode == ArgModeImplicit && it.Arg != "path" {
			implicitArgs[it.Arg] = struct{}{}
		}
	}

	return implicitArgs
}

// HTTPExcludeFromExchangeRequestSet returns set of аргументов, которые должны
// быть исключены из JSON body exchange-структуры запроса (explicit/implicit + path).
func HTTPExcludeFromExchangeRequestSet(mappings HTTPArgMappings) (excludeArgs map[string]struct{}) {

	excludeArgs = make(map[string]struct{})

	for _, it := range mappings.HeaderItems {
		if it.Mode == ArgModeExplicit || it.Mode == ArgModeImplicit {
			excludeArgs[it.Arg] = struct{}{}
		}
	}

	for _, it := range mappings.CookieItems {
		if it.Mode == ArgModeExplicit || it.Mode == ArgModeImplicit {
			excludeArgs[it.Arg] = struct{}{}
		}
	}

	for _, it := range mappings.ArgItems {
		if it.Mode == ArgModeExplicit || it.Mode == ArgModeImplicit {
			if it.Arg == "path" {
				continue
			}
			excludeArgs[it.Arg] = struct{}{}
		}
	}

	return excludeArgs
}

// HTTPOmitFromRequestJSON объединяет explicit/implicit exclude с path-параметрами (json:"-").
func HTTPOmitFromRequestJSON(project *Project, contract *Contract, method *Method) (omit map[string]struct{}) {

	omit = HTTPExcludeFromExchangeRequestSet(BuildHTTPArgMappings(project, contract, method))
	for argName := range HTTPPathParamArgSet(project, contract, method) {
		omit[argName] = struct{}{}
	}

	return omit
}

func HTTPHeaderArgMapForRequest(project *Project, contract *Contract, method *Method) (headerMap map[string]string) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	headerMap = make(map[string]string)

	for _, it := range mappings.HeaderItems {
		if it.Mode != ArgModeExplicit && it.Mode != ArgModeImplicit {
			continue
		}

		if argByName(method, it.Arg) != nil {
			headerMap[it.Arg] = it.Key
		}
	}

	return headerMap
}

func HTTPCookieArgMapForRequest(project *Project, contract *Contract, method *Method) (cookieMap map[string]string) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	cookieMap = make(map[string]string)

	for _, it := range mappings.CookieItems {
		if it.Mode != ArgModeExplicit && it.Mode != ArgModeImplicit {
			continue
		}

		if argByName(method, it.Arg) != nil {
			cookieMap[it.Arg] = it.Key
		}
	}

	return cookieMap
}

// За исключением специального arg == "path".
func HTTPArgQueryMapForRequest(project *Project, contract *Contract, method *Method) (queryMap map[string]string) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	queryMap = make(map[string]string)

	for _, it := range mappings.ArgItems {
		if it.Arg == "path" {
			continue
		}

		queryMap[it.Arg] = it.Key
	}

	return queryMap
}

// ArgByPathSegment ищет аргумент метода по имени сегмента http-path (:name): exact или LowerCamel.
func ArgByPathSegment(method *Method, segmentName string) (variable *Variable) {

	if method == nil || segmentName == "" {
		return nil
	}

	segmentName = strings.TrimPrefix(strings.TrimSpace(segmentName), "!")
	for _, arg := range method.Args {
		if arg.Name == segmentName || LowerCamel(arg.Name) == segmentName {
			return arg
		}
	}

	return nil
}

// HTTPPathParamArgSet returns argument names bound to :segments in http-path.
func HTTPPathParamArgSet(project *Project, contract *Contract, method *Method) (pathArgs map[string]struct{}) {

	pathArgs = make(map[string]struct{})
	for argName := range HTTPPathParamArgMap(project, contract, method) {
		pathArgs[argName] = struct{}{}
	}

	return pathArgs
}

// HTTPPathParamArgMap возвращает arg.Name → имя сегмента пути для Fiber Params / client escape.
func HTTPPathParamArgMap(project *Project, contract *Contract, method *Method) (pathMap map[string]string) {

	pathMap = make(map[string]string)
	if project == nil || contract == nil || method == nil {
		return pathMap
	}

	urlPath := GetAnnotationValue(project, contract, method, nil, TagHttpPath, "")
	for _, token := range strings.Split(urlPath, "/") {
		token = strings.TrimSpace(token)
		if !strings.HasPrefix(token, ":") {
			continue
		}
		segmentName := strings.TrimPrefix(token, ":")
		if arg := ArgByPathSegment(method, segmentName); arg != nil {
			pathMap[arg.Name] = segmentName
		}
	}

	return pathMap
}

// HTTPArgsFromRequestBody returns arguments populated from the JSON request body.
// Same rules as generated HTTP clients: body-mode headers/cookies and plain fields;
// not context, io.Reader, path, http-args query, or explicit/implicit header/cookie transport.
func HTTPArgsFromRequestBody(project *Project, contract *Contract, method *Method) (args []*Variable) {

	if method == nil {
		return nil
	}

	mappings := BuildHTTPArgMappings(project, contract, method)
	implicit := HTTPImplicitArgSet(mappings)
	pathParams := HTTPPathParamArgSet(project, contract, method)
	headerExplicit := HTTPHeaderArgMapForRequest(project, contract, method)
	cookieExplicit := HTTPCookieArgMapForRequest(project, contract, method)
	queryArgs := HTTPArgQueryMapForRequest(project, contract, method)

	for _, arg := range method.Args {
		if isContextArg(arg) {
			continue
		}
		if _, ok := implicit[arg.Name]; ok {
			continue
		}
		if arg.TypeID == typeIDIOReader {
			continue
		}
		if _, inPath := pathParams[arg.Name]; inPath {
			continue
		}
		if _, inHeader := headerExplicit[arg.Name]; inHeader {
			continue
		}
		if _, inCookie := cookieExplicit[arg.Name]; inCookie {
			continue
		}
		if _, inQuery := queryArgs[arg.Name]; inQuery {
			continue
		}
		args = append(args, arg)
	}

	return args
}

// HTTPNeedsRequestBodyDecode reports whether the HTTP handler should decode the request body into the exchange struct.
func HTTPNeedsRequestBodyDecode(project *Project, contract *Contract, method *Method) (needs bool) {

	return len(HTTPArgsFromRequestBody(project, contract, method)) > 0
}

// HTTPAllowsEmptyRequestBody reports whether an empty request body is valid for REST decode.
// True for GET methods that populate the exchange from the JSON body (including body-mode headers/cookies),
// so transport overlay can fill fields without a non-empty body.
func HTTPAllowsEmptyRequestBody(project *Project, contract *Contract, method *Method) (allows bool) {

	if !HTTPNeedsRequestBodyDecode(project, contract, method) {
		return false
	}

	return strings.EqualFold(GetHTTPMethod(project, contract, method), "GET")
}

// HTTPResultNamesExcludeFromBody — results, которых нет в JSON body ответа клиента:
// только mode explicit/implicit. Mode body (default) остаётся в теле (как на server).
func HTTPResultNamesExcludeFromBody(project *Project, contract *Contract, method *Method) (names map[string]struct{}) {

	return HTTPResultNamesOmitFromExchangeBody(project, contract, method)
}

// HTTPResultNamesOmitFromExchangeBody — результаты без поля в JSON body (server/swagger/client):
// только explicit/implicit. Body-mode остаётся в JSON; server может ещё писать header/cookie.
func HTTPResultNamesOmitFromExchangeBody(project *Project, contract *Contract, method *Method) (names map[string]struct{}) {

	return httpResultNamesFromHeaderCookie(project, contract, method)
}

// HTTPResultsForExchangeBody — results для JSON body на server/swagger (без error, io.ReadCloser и omit transport-only).
func HTTPResultsForExchangeBody(project *Project, contract *Contract, method *Method) (results []*Variable) {

	if method == nil {
		return nil
	}

	exclude := HTTPResultNamesOmitFromExchangeBody(project, contract, method)
	for _, res := range method.Results {
		if res.TypeID == "error" || res.TypeID == typeIDIOReadCloser {
			continue
		}
		if _, ok := exclude[res.Name]; ok {
			continue
		}
		results = append(results, res)
	}

	return results
}

func httpResultNamesFromHeaderCookie(project *Project, contract *Contract, method *Method) (names map[string]struct{}) {

	mappings := BuildHTTPArgMappings(project, contract, method)
	names = make(map[string]struct{})

	for _, it := range mappings.HeaderItems {
		if it.Mode != ArgModeExplicit && it.Mode != ArgModeImplicit {
			continue
		}
		if resultByName(method, it.Arg) != nil {
			names[it.Arg] = struct{}{}
		}
	}

	for _, it := range mappings.CookieItems {
		if it.Mode != ArgModeExplicit && it.Mode != ArgModeImplicit {
			continue
		}
		if resultByName(method, it.Arg) != nil {
			names[it.Arg] = struct{}{}
		}
	}

	return names
}

func HTTPResultHeaderMapForResponse(project *Project, contract *Contract, method *Method) (headerMap map[string]string) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	headerMap = make(map[string]string)

	for _, it := range mappings.HeaderItems {
		if resultByName(method, it.Arg) != nil {
			headerMap[it.Arg] = it.Key
		}
	}

	return headerMap
}

func HTTPResultCookieMapForResponse(project *Project, contract *Contract, method *Method) (cookieMap map[string]string) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	cookieMap = make(map[string]string)

	for _, it := range mappings.CookieItems {
		if resultByName(method, it.Arg) != nil {
			cookieMap[it.Arg] = it.Key
		}
	}

	return cookieMap
}

// HTTPIsArgInHeader сообщает, замаплен ли аргумент в заголовок (explicit, implicit или body — при body сервер может читать заголовок с fallback в тело).
func HTTPIsArgInHeader(project *Project, contract *Contract, method *Method, arg *Variable) (inHeader bool) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	for _, it := range mappings.HeaderItems {
		if it.Arg == arg.Name && (it.Mode == ArgModeExplicit || it.Mode == ArgModeImplicit || it.Mode == ArgModeBody) {
			return true
		}
	}

	return false
}

// HTTPIsArgInCookie сообщает, замаплен ли аргумент в cookie (explicit, implicit или body — при body сервер может читать куку с fallback в тело).
func HTTPIsArgInCookie(project *Project, contract *Contract, method *Method, arg *Variable) (inCookie bool) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	for _, it := range mappings.CookieItems {
		if it.Arg == arg.Name && (it.Mode == ArgModeExplicit || it.Mode == ArgModeImplicit || it.Mode == ArgModeBody) {
			return true
		}
	}

	return false
}

// argByName ищет аргумент метода по имени.
func argByName(method *Method, argName string) (variable *Variable) {

	if method == nil {
		return nil
	}

	argName = strings.TrimPrefix(argName, "!")
	for _, arg := range method.Args {
		if arg.Name == argName {
			return arg
		}
	}

	return nil
}

// resultByName ищет результат метода по имени (не error).
func resultByName(method *Method, resultName string) (variable *Variable) {

	if method == nil {
		return nil
	}

	for _, res := range method.Results {
		if res.Name == resultName && res.TypeID != "error" {
			return res
		}
	}

	return nil
}
