// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"
)

const (
	ArgModeExplicit = "explicit"
	ArgModeImplicit = "implicit"
	ArgModeBody     = "body"

	typeIDContext  = "context:Context"
	typeIDIOReader = "io:Reader"
)

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
// быть исключены из exchange-структуры запроса (explicit и implicit).
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

// HTTPPathParamArgSet returns argument names bound to :segments in http-path.
func HTTPPathParamArgSet(project *Project, contract *Contract, method *Method) (pathArgs map[string]struct{}) {

	pathArgs = make(map[string]struct{})
	if project == nil || contract == nil || method == nil {
		return pathArgs
	}

	urlPath := GetAnnotationValue(project, contract, method, nil, TagHttpPath, "")
	for _, token := range strings.Split(urlPath, "/") {
		token = strings.TrimSpace(token)
		if !strings.HasPrefix(token, ":") {
			continue
		}
		segmentName := strings.TrimPrefix(token, ":")
		if arg := argByName(method, segmentName); arg != nil {
			pathArgs[arg.Name] = struct{}{}
		}
	}

	return pathArgs
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
		if arg.TypeID == typeIDContext {
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

// HTTPResultNamesExcludeFromBody возвращает имена результатов, которые берутся из заголовка или cookie ответа (любой mode).
func HTTPResultNamesExcludeFromBody(project *Project, contract *Contract, method *Method) (names map[string]struct{}) {

	mappings := BuildHTTPArgMappings(project, contract, method)

	names = make(map[string]struct{})

	for _, it := range mappings.HeaderItems {
		if resultByName(method, it.Arg) != nil {
			names[it.Arg] = struct{}{}
		}
	}

	for _, it := range mappings.CookieItems {
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

	argName = strings.TrimPrefix(argName, "!")

	for _, variable = range method.Args {
		if variable.Name == argName {
			return variable
		}
	}

	return
}

// resultByName ищет результат метода по имени.
func resultByName(method *Method, resultName string) (variable *Variable) {

	for _, variable = range method.Results {
		if variable.Name == resultName {
			return variable
		}
	}

	return
}
