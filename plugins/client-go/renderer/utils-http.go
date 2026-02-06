// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *ClientRenderer) varHeaderMap(contract *model.Contract, method *model.Method) map[string]string {

	headers := make(map[string]string)
	if httpHeaders := model.GetAnnotationValue(r.project, contract, method, nil, TagHttpHeader, ""); httpHeaders != "" {
		headerPairs := strings.Split(httpHeaders, ",")
		for _, pair := range headerPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				header := strings.TrimSpace(pairTokens[1])
				headers[arg] = header
			}
		}
	}
	return headers
}

func (r *ClientRenderer) varCookieMap(contract *model.Contract, method *model.Method) map[string]string {

	cookies := make(map[string]string)
	if httpCookies := model.GetAnnotationValue(r.project, contract, method, nil, TagHttpCookies, ""); httpCookies != "" {
		cookiePairs := strings.Split(httpCookies, ",")
		for _, pair := range cookiePairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				cookie := strings.TrimSpace(pairTokens[1])
				cookies[arg] = cookie
			}
		}
	}
	return cookies
}

func (r *ClientRenderer) argPathMap(contract *model.Contract, method *model.Method) map[string]string {

	paths := make(map[string]string)
	if urlPath := model.GetAnnotationValue(r.project, contract, method, nil, TagHttpPath, ""); urlPath != "" {
		urlTokens := strings.Split(urlPath, "/")
		for _, token := range urlTokens {
			if strings.HasPrefix(token, ":") {
				arg := strings.TrimSpace(strings.TrimPrefix(token, ":"))
				paths[arg] = arg
			}
		}
	}
	return paths
}

func (r *ClientRenderer) argParamMap(contract *model.Contract, method *model.Method) map[string]string {

	params := make(map[string]string)
	if urlArgs := model.GetAnnotationValue(r.project, contract, method, nil, TagHttpArg, ""); urlArgs != "" {
		paramPairs := strings.Split(urlArgs, ",")
		for _, pair := range paramPairs {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				arg := strings.TrimSpace(pairTokens[0])
				param := strings.TrimSpace(pairTokens[1])
				params[arg] = param
			}
		}
	}
	return params
}

func (r *ClientRenderer) argByName(method *model.Method, argName string) *model.Variable {
	argName = strings.TrimPrefix(argName, "!")
	for _, arg := range method.Args {
		if arg.Name == argName {
			return arg
		}
	}
	return nil
}

func (r *ClientRenderer) varToString(ctx context.Context, variable *model.Variable) Code {
	if variable.TypeID == "string" {
		return Id(ToLowerCamel(variable.Name))
	}
	// Для остальных типов используем fmt.Sprint
	return Qual(PackageFmt, "Sprint").Call(Id(ToLowerCamel(variable.Name)))
}

func (r *ClientRenderer) contractNameToLowerCamel(contract *model.Contract) string {
	if contract == nil {
		return ""
	}
	return ToLowerCamel(contract.Name)
}

func (r *ClientRenderer) methodNameToLowerCamel(method *model.Method) string {
	if method == nil {
		return ""
	}
	return ToLowerCamel(method.Name)
}

func (r *ClientRenderer) contractNameToLower(contract *model.Contract) string {
	if contract == nil {
		return ""
	}
	return strings.ToLower(contract.Name)
}

func (r *ClientRenderer) methodNameToLower(method *model.Method) string {
	if method == nil {
		return ""
	}
	return strings.ToLower(method.Name)
}
