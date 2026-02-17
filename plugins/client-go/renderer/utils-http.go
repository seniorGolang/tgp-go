// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *ClientRenderer) resultByName(method *model.Method, retName string) (v *model.Variable) {

	for _, ret := range method.Results {
		if ret.Name == retName {
			return ret
		}
	}
	return nil
}

func (r *ClientRenderer) argsForClient(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	mappings := model.BuildHTTPArgMappings(r.project, contract, method)
	implicit := model.HTTPImplicitArgSet(mappings)
	var list []*model.Variable
	for _, arg := range r.argsWithoutContext(method) {
		if _, ok := implicit[arg.Name]; !ok {
			list = append(list, arg)
		}
	}
	return list
}

func (r *ClientRenderer) argsForExchangeRequest(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	mappings := model.BuildHTTPArgMappings(r.project, contract, method)
	exclude := model.HTTPExcludeFromExchangeRequestSet(mappings)
	var list []*model.Variable
	for _, arg := range r.argsWithoutContext(method) {
		if _, ok := exclude[arg.Name]; !ok {
			list = append(list, arg)
		}
	}
	return list
}

func (r *ClientRenderer) argsForRequestBody(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	pathParams := r.argPathMap(contract, method)
	headerForReq := model.HTTPHeaderArgMapForRequest(r.project, contract, method)
	cookieForReq := model.HTTPCookieArgMapForRequest(r.project, contract, method)
	queryForReq := model.HTTPArgQueryMapForRequest(r.project, contract, method)

	var list []*model.Variable
	for _, arg := range r.argsForClient(contract, method) {
		if arg.TypeID == TypeIDIOReader {
			continue
		}
		if _, inPath := pathParams[arg.Name]; inPath {
			continue
		}
		if _, inHeader := headerForReq[arg.Name]; inHeader {
			continue
		}
		if _, inCookie := cookieForReq[arg.Name]; inCookie {
			continue
		}
		if _, inQuery := queryForReq[arg.Name]; inQuery {
			continue
		}
		list = append(list, arg)
	}
	return list
}

func (r *ClientRenderer) argPathMap(contract *model.Contract, method *model.Method) (out map[string]struct{}) {

	out = make(map[string]struct{})
	if urlPath := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, ""); urlPath != "" {
		for _, token := range strings.Split(urlPath, "/") {
			token = strings.TrimSpace(token)
			if !strings.HasPrefix(token, ":") {
				continue
			}
			segmentName := strings.TrimPrefix(token, ":")
			if arg := r.argByPathParamName(contract, method, segmentName); arg != nil {
				out[arg.Name] = struct{}{}
			}
		}
	}
	return
}

func (r *ClientRenderer) argByPathParamName(contract *model.Contract, method *model.Method, pathSegmentName string) (v *model.Variable) {

	for _, arg := range method.Args {
		if arg.Name == pathSegmentName || ToLowerCamel(arg.Name) == pathSegmentName {
			return arg
		}
	}
	return nil
}

func (r *ClientRenderer) argByName(method *model.Method, argName string) (v *model.Variable) {

	argName = strings.TrimPrefix(argName, "!")
	for _, arg := range method.Args {
		if arg.Name == argName {
			return arg
		}
	}
	return nil
}

func (r *ClientRenderer) varToString(ctx context.Context, variable *model.Variable) (c Code) {

	if variable.TypeID == "string" {
		return Id(ToLowerCamel(variable.Name))
	}
	// Для остальных типов используем fmt.Sprint
	return Qual(PackageFmt, "Sprint").Call(Id(ToLowerCamel(variable.Name)))
}

func (r *ClientRenderer) contractNameToLowerCamel(contract *model.Contract) (s string) {

	if contract == nil {
		return ""
	}
	return ToLowerCamel(contract.Name)
}

func (r *ClientRenderer) methodNameToLowerCamel(method *model.Method) (s string) {

	if method == nil {
		return ""
	}
	return ToLowerCamel(method.Name)
}

func (r *ClientRenderer) defaultMethodHTTPPath(contract *model.Contract, method *model.Method) (s string) {

	return "/" + r.contractNameToLowerCamel(contract) + "/" + r.methodNameToLowerCamel(method)
}

func (r *ClientRenderer) contractNameToLower(contract *model.Contract) (s string) {

	if contract == nil {
		return ""
	}
	return strings.ToLower(contract.Name)
}

func (r *ClientRenderer) methodNameToLower(method *model.Method) (s string) {

	if method == nil {
		return ""
	}
	return strings.ToLower(method.Name)
}
