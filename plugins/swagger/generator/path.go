// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/swagger/types"
)

// generatePaths генерирует пути для всех контрактов.
func (g *generator) generatePaths(contracts []*model.Contract, ifaces []string) (paths map[string]types.Path) {

	paths = make(map[string]types.Path)

	var include, exclude []string
	for _, iface := range ifaces {
		if strings.HasPrefix(iface, "!") {
			exclude = append(exclude, strings.TrimPrefix(iface, "!"))
		} else {
			include = append(include, iface)
		}
	}

	for _, contract := range contracts {
		if len(include) > 0 {
			found := false
			for _, iface := range include {
				if contract.Name == iface || contract.ID == iface {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(exclude) > 0 {
			excluded := false
			for _, iface := range exclude {
				if contract.Name == iface || contract.ID == iface {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		for _, method := range contract.Methods {
			g.generateMethodPath(paths, contract, method)
		}
	}

	return
}

func (g *generator) generateMethodPath(paths map[string]types.Path, contract *model.Contract, method *model.Method) {

	serviceTags := strings.Split(contract.Annotations.Value(tagSwaggerTags, contract.Name), ",")
	if method.Annotations.Contains(tagSwaggerTags) {
		serviceTags = strings.Split(method.Annotations.Value(tagSwaggerTags), ",")
	}

	isJsonRPC := contract.Annotations.Contains(tagServerJsonRPC) && !method.Annotations.Contains(tagMethodHTTP)
	isHTTP := contract.Annotations.Contains(tagServerHTTP) && method.Annotations.Contains(tagMethodHTTP)

	if isJsonRPC {
		g.generateJsonRPCPath(paths, contract, method, serviceTags)
	} else if isHTTP {
		g.generateHTTPPath(paths, contract, method, serviceTags)
	}
}

func (g *generator) generateJsonRPCPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	prefix := contract.Annotations.Value(tagHttpPrefix, "")
	urlPath := method.Annotations.Value(tagHttpPath, "/"+types.ToLowerCamel(contract.Name)+"/"+types.ToLowerCamel(method.Name))
	urlPath = strings.Split(urlPath, ":")[0]
	jsonrpcPath := path.Join("/", prefix, urlPath)

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	g.registerStruct(requestStructName, contract.PkgPath, method.Annotations, method.Args)
	g.registerStruct(responseStructName, contract.PkgPath, method.Annotations, method.Results)

	operation := &types.Operation{
		Summary:     method.Annotations.Value(tagSummary),
		Description: method.Annotations.Value(tagDesc),
		Tags:        serviceTags,
		Deprecated:  method.Annotations.Contains(tagDeprecated),
		RequestBody: &types.RequestBody{
			Content: types.Content{
				contentJSON: types.Media{
					Schema: types.JSONRPCSchema("params", g.toSchema(requestStructName)),
				},
			},
		},
		Responses: types.Responses{
			"200": types.Response{
				Description: types.CodeToText(200),
				Content: types.Content{
					contentJSON: types.Media{
						Schema: types.Schema{
							OneOf: []types.Schema{
								types.JSONRPCSchema("result", g.toSchema(responseStructName)),
								types.JSONRPCErrorSchema(),
							},
						},
					},
				},
			},
		},
	}

	g.addHeaderParameters(operation, contract, method)
	g.addCookieParameters(operation, contract, method)
	g.addResponseHeaders(operation, contract, method, 200)
	g.fillErrors(operation.Responses, method.Annotations.Merge(contract.Annotations))

	paths[jsonrpcPath] = types.Path{Post: operation}
}

func (g *generator) generateHTTPPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	prefix := contract.Annotations.Value(tagHttpPrefix, "")
	methodPath := method.Annotations.Value(tagHttpPath, "/"+types.ToLowerCamel(contract.Name)+"/"+types.ToLowerCamel(method.Name))
	httpPath := methodPath
	if prefix != "" {
		httpPath = path.Join("/", prefix, methodPath)
	}

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	g.registerStruct(requestStructName, contract.PkgPath, method.Annotations, method.Args)
	g.registerStruct(responseStructName, contract.PkgPath, method.Annotations, method.Results)

	httpMethod := strings.ToLower(method.Annotations.Value(tagMethodHTTP, defaultHTTPMethod))
	successCode := method.Annotations.ValueInt(tagHttpSuccess, 200)
	requestContentType := method.Annotations.Value(tagRequestContentType, contentJSON)
	responseContentType := method.Annotations.Value(tagResponseContentType, contentJSON)

	operation := &types.Operation{
		Summary:     method.Annotations.Value(tagSummary),
		Description: method.Annotations.Value(tagDesc),
		Tags:        serviceTags,
		Deprecated:  method.Annotations.Contains(tagDeprecated),
		Responses: types.Responses{
			fmt.Sprintf("%d", successCode): types.Response{
				Description: types.CodeToText(successCode),
				Content: types.Content{
					responseContentType: types.Media{
						Schema: g.toSchema(responseStructName),
					},
				},
			},
		},
	}

	g.addPathParameters(operation, contract, method, httpPath)
	g.addQueryParameters(operation, contract, method)
	g.addHeaderParameters(operation, contract, method)
	g.addCookieParameters(operation, contract, method)

	if len(method.Args) > 0 {
		hasBodyArgs := false
		for _, arg := range method.Args {
			if arg.TypeID != "context:Context" && !g.isArgInPath(arg, method, httpPath) && !g.isArgInQuery(arg, method) && !g.isArgInHeader(arg, method) && !g.isArgInCookie(arg, method) {
				hasBodyArgs = true
				break
			}
		}
		if hasBodyArgs {
			operation.RequestBody = &types.RequestBody{
				Content: types.Content{
					requestContentType: types.Media{
						Schema: g.toSchema(requestStructName),
					},
				},
			}
		}
	}

	g.addResponseHeaders(operation, contract, method, successCode)
	g.fillErrors(operation.Responses, method.Annotations.Merge(contract.Annotations))

	pathValue, found := paths[httpPath]
	if !found {
		pathValue = types.Path{}
	}

	switch httpMethod {
	case "get":
		pathValue.Get = operation
	case defaultHTTPMethod:
		pathValue.Post = operation
	case "put":
		pathValue.Put = operation
	case "patch":
		pathValue.Patch = operation
	case "delete":
		pathValue.Delete = operation
	default:
		pathValue.Post = operation
	}

	paths[httpPath] = pathValue
}

func (g *generator) addPathParameters(operation *types.Operation, contract *model.Contract, method *model.Method, httpPath string) {

	pathParts := strings.Split(httpPath, "/")
	for _, part := range pathParts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			for _, arg := range method.Args {
				if types.ToLowerCamel(arg.Name) == paramName || arg.Name == paramName {
					var schema types.Schema
					if schemaPtr := g.variableToSchema(arg, contract.PkgPath, true); schemaPtr != nil {
						schema = *schemaPtr
					}
					operation.Parameters = append(operation.Parameters, types.Parameter{
						In:       "path",
						Name:     paramName,
						Required: true,
						Schema:   schema,
					})
					break
				}
			}
		}
	}
}

func (g *generator) addQueryParameters(operation *types.Operation, contract *model.Contract, method *model.Method) {

	httpArgs := method.Annotations.Value(tagHttpArg, "")
	if httpArgs == "" {
		return
	}

	argPairs := strings.Split(httpArgs, ",")
	for _, pair := range argPairs {
		pairTokens := strings.Split(pair, "|")
		if len(pairTokens) == 2 {
			argName := strings.TrimSpace(pairTokens[0])
			queryName := strings.TrimSpace(pairTokens[1])
			for _, arg := range method.Args {
				if arg.Name == argName {
					var schema types.Schema
					if schemaPtr := g.variableToSchema(arg, contract.PkgPath, true); schemaPtr != nil {
						schema = *schemaPtr
					}
					operation.Parameters = append(operation.Parameters, types.Parameter{
						In:     "query",
						Name:   queryName,
						Schema: schema,
					})
					break
				}
			}
		}
	}
}

func (g *generator) addHeaderParameters(operation *types.Operation, contract *model.Contract, method *model.Method) {

	httpHeaders := method.Annotations.Value(tagHttpHeader, "")
	if httpHeaders == "" {
		return
	}

	headerPairs := strings.Split(httpHeaders, ",")
	for _, pair := range headerPairs {
		pairTokens := strings.Split(pair, "|")
		if len(pairTokens) == 2 {
			argName := strings.TrimSpace(pairTokens[0])
			headerName := strings.TrimSpace(pairTokens[1])
			for _, arg := range method.Args {
				if arg.Name == argName {
					var schema types.Schema
					if schemaPtr := g.variableToSchema(arg, contract.PkgPath, true); schemaPtr != nil {
						schema = *schemaPtr
					}
					operation.Parameters = append(operation.Parameters, types.Parameter{
						In:       "header",
						Name:     headerName,
						Required: true,
						Schema:   schema,
					})
					break
				}
			}
		}
	}
}

func (g *generator) addCookieParameters(operation *types.Operation, contract *model.Contract, method *model.Method) {

	httpCookies := method.Annotations.Value(tagHttpCookies, "")
	if httpCookies == "" {
		return
	}

	cookiePairs := strings.Split(httpCookies, ",")
	for _, pair := range cookiePairs {
		pairTokens := strings.Split(pair, "|")
		if len(pairTokens) == 2 {
			argName := strings.TrimSpace(pairTokens[0])
			cookieName := strings.TrimSpace(pairTokens[1])
			for _, arg := range method.Args {
				if arg.Name == argName {
					var schema types.Schema
					if schemaPtr := g.variableToSchema(arg, contract.PkgPath, true); schemaPtr != nil {
						schema = *schemaPtr
					}
					operation.Parameters = append(operation.Parameters, types.Parameter{
						In:       "cookie",
						Name:     cookieName,
						Required: true,
						Schema:   schema,
					})
					break
				}
			}
		}
	}
}

func (g *generator) addResponseHeaders(operation *types.Operation, contract *model.Contract, method *model.Method, successCode int) {

	httpHeaders := method.Annotations.Value(tagHttpHeader, "")
	if httpHeaders == "" {
		return
	}

	headerPairs := strings.Split(httpHeaders, ",")
	for _, pair := range headerPairs {
		pairTokens := strings.Split(pair, "|")
		if len(pairTokens) == 2 {
			argName := strings.TrimSpace(pairTokens[0])
			headerName := strings.TrimSpace(pairTokens[1])
			for _, result := range method.Results {
				if result.Name == argName {
					schemaPtr := g.variableToSchema(result, contract.PkgPath, false)
					if schemaPtr == nil {
						schemaPtr = &types.Schema{Type: "string"}
					}
					schema := *schemaPtr
					successKey := fmt.Sprintf("%d", successCode)
					if operation.Responses[successKey].Headers == nil {
						response := operation.Responses[successKey]
						response.Headers = make(map[string]types.Header)
						operation.Responses[successKey] = response
					}
					operation.Responses[successKey].Headers[headerName] = types.Header{
						Schema: schema,
					}
					break
				}
			}
		}
	}
}

func (g *generator) fillErrors(responses types.Responses, methodTags tags.DocTags) {

	// Используем отсортированные пары для детерминированного порядка
	for key, value := range common.SortedPairs(methodTags) {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		code, err := strconv.Atoi(key)
		if err != nil {
			continue
		}

		if value == "skip" {
			continue
		}

		if types.IsValidHTTPCode(code) {
			var content types.Content
			if value != "" {
				if tokens := strings.Split(value, ":"); len(tokens) == 2 {
					pkgPath := tokens[0]
					typeName := tokens[1]
					// Используем отсортированные пары для детерминированного порядка
					for typeID, typeInfo := range common.SortedPairs(g.project.Types) {
						if typeInfo.TypeName == typeName && strings.Contains(typeID, pkgPath) {
							var schema types.Schema
							if schemaPtr := g.structTypeToSchema(typeInfo, nil); schemaPtr != nil {
								schema = *schemaPtr
							}
							content = types.Content{
								contentJSON: types.Media{Schema: schema},
							}
							break
						}
					}
				}
			}
			responses[key] = types.Response{
				Description: types.CodeToText(code),
				Content:     content,
			}
		} else if key == "defaultError" {
			var content types.Content
			if value != "" {
				if tokens := strings.Split(value, ":"); len(tokens) == 2 {
					pkgPath := tokens[0]
					typeName := tokens[1]
					// Используем отсортированные пары для детерминированного порядка
					for typeID, typeInfo := range common.SortedPairs(g.project.Types) {
						if typeInfo.TypeName == typeName && strings.Contains(typeID, pkgPath) {
							var schema types.Schema
							if schemaPtr := g.structTypeToSchema(typeInfo, nil); schemaPtr != nil {
								schema = *schemaPtr
							}
							content = types.Content{
								contentJSON: types.Media{Schema: schema},
							}
							break
						}
					}
				}
			}
			responses["default"] = types.Response{
				Description: "Generic error",
				Content:     content,
			}
		}
	}
}

func (g *generator) requestStructName(contract *model.Contract, method *model.Method) (name string) {
	return types.ToCamel(contract.Name) + types.ToCamel(method.Name) + "Request"
}

func (g *generator) responseStructName(contract *model.Contract, method *model.Method) (name string) {
	return types.ToCamel(contract.Name) + types.ToCamel(method.Name) + "Response"
}

func (g *generator) isArgInPath(arg *model.Variable, method *model.Method, httpPath string) (found bool) {

	pathParts := strings.Split(httpPath, "/")
	argName := types.ToLowerCamel(arg.Name)
	for _, part := range pathParts {
		if strings.TrimPrefix(part, ":") == argName {
			return true
		}
	}
	return
}

func (g *generator) isArgInQuery(arg *model.Variable, method *model.Method) (found bool) {

	httpArgs := method.Annotations.Value(tagHttpArg, "")
	if httpArgs == "" {
		return
	}
	argPairs := strings.Split(httpArgs, ",")
	for _, pair := range argPairs {
		pairTokens := strings.Split(pair, "|")
		if len(pairTokens) == 2 && strings.TrimSpace(pairTokens[0]) == arg.Name {
			return true
		}
	}
	return
}

func (g *generator) isArgInHeader(arg *model.Variable, method *model.Method) (found bool) {

	httpHeaders := method.Annotations.Value(tagHttpHeader, "")
	if httpHeaders == "" {
		return
	}
	headerPairs := strings.Split(httpHeaders, ",")
	for _, pair := range headerPairs {
		pairTokens := strings.Split(pair, "|")
		if len(pairTokens) == 2 && strings.TrimSpace(pairTokens[0]) == arg.Name {
			return true
		}
	}
	return
}

func (g *generator) isArgInCookie(arg *model.Variable, method *model.Method) (found bool) {

	httpCookies := method.Annotations.Value(tagHttpCookies, "")
	if httpCookies == "" {
		return
	}
	cookiePairs := strings.Split(httpCookies, ",")
	for _, pair := range cookiePairs {
		pairTokens := strings.Split(pair, "|")
		if len(pairTokens) == 2 && strings.TrimSpace(pairTokens[0]) == arg.Name {
			return true
		}
	}
	return
}
