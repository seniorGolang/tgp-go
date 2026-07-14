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

	return model.HTTPArgsFromRequestBody(r.project, contract, method)
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

	var expr Code
	if variable.TypeID == "string" {
		expr = Id(ToLowerCamel(variable.Name))
	} else {
		expr = Qual(PackageFmt, "Sprint").Call(Id(ToLowerCamel(variable.Name)))
	}
	if variable.NumberOfPointers > 0 {
		expr = Op("*").Add(expr)
	}
	return expr
}

func (r *ClientRenderer) contractNameToLowerCamel(contract *model.Contract) (s string) {

	if contract == nil {
		return ""
	}
	return model.LowerCamel(contract.Name)
}

func (r *ClientRenderer) methodNameToLowerCamel(method *model.Method) (s string) {

	if method == nil {
		return ""
	}
	return model.LowerCamel(method.Name)
}

func (r *ClientRenderer) jsonRPCWireMethod(contract *model.Contract, method *model.Method) (s string) {

	if contract == nil || method == nil {
		return ""
	}
	return model.JsonRPCWireMethod(contract.Name, method.Name)
}
