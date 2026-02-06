// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
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

	serviceTags := strings.Split(model.GetAnnotationValue(g.project, contract, nil, nil, tagSwaggerTags, contract.Name), ",")
	if model.IsAnnotationSet(g.project, contract, method, nil, tagSwaggerTags) {
		serviceTags = strings.Split(model.GetAnnotationValue(g.project, contract, method, nil, tagSwaggerTags, ""), ",")
	}

	isJsonRPC := model.IsAnnotationSet(g.project, contract, nil, nil, tagServerJsonRPC) && !model.IsAnnotationSet(g.project, contract, method, nil, tagMethodHTTP)
	isHTTP := model.IsAnnotationSet(g.project, contract, nil, nil, tagServerHTTP) && (!model.IsAnnotationSet(g.project, contract, nil, nil, tagServerJsonRPC) || model.IsAnnotationSet(g.project, contract, method, nil, tagMethodHTTP))

	if isJsonRPC {
		g.generateJsonRPCPath(paths, contract, method, serviceTags)
	} else if isHTTP {
		g.generateHTTPPath(paths, contract, method, serviceTags)
	}
}

func (g *generator) generateJsonRPCPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	prefix := model.GetAnnotationValue(g.project, contract, nil, nil, tagHttpPrefix, "")
	urlPath := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpPath, "/"+types.ToLowerCamel(method.Name))
	urlPath = strings.Split(urlPath, ":")[0]
	jsonrpcPath := path.Join("/", prefix, urlPath)

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	g.registerStruct(requestStructName, contract.PkgPath, method.Annotations, method.Args)
	g.registerStruct(responseStructName, contract.PkgPath, method.Annotations, method.Results)

	operation := &types.Operation{
		Summary:     model.GetAnnotationValue(g.project, contract, method, nil, tagSummary, ""),
		Description: model.GetAnnotationValue(g.project, contract, method, nil, tagDesc, ""),
		Tags:        serviceTags,
		Deprecated:  model.IsAnnotationSet(g.project, contract, method, nil, tagDeprecated),
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

func (g *generator) countIOReaderArgs(method *model.Method) (n int) {

	for _, arg := range method.Args {
		if arg.TypeID == typeIDIOReader {
			n++
		}
	}
	return n
}

func (g *generator) countIOReadCloserResults(method *model.Method) (n int) {

	for _, res := range method.Results {
		if res.TypeID == typeIDIOReadCloser {
			n++
		}
	}
	return n
}

func (g *generator) requestMultipart(contract *model.Contract, method *model.Method) bool {

	n := g.countIOReaderArgs(method)
	if n > 1 {
		return true
	}
	if n == 1 && model.IsAnnotationSet(g.project, contract, method, nil, tagHttpMultipart) {
		return true
	}
	return false
}

func (g *generator) responseMultipart(contract *model.Contract, method *model.Method) bool {

	n := g.countIOReadCloserResults(method)
	if n > 1 {
		return true
	}
	if n == 1 && model.IsAnnotationSet(g.project, contract, method, nil, tagHttpMultipart) {
		return true
	}
	return false
}

func (g *generator) streamPartName(contract *model.Contract, method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[tagHttpPartName]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[tagHttpPartName]; found && val != "" {
			if partName := g.varValueFromMethodMap(val, v.Name); partName != "" {
				return partName
			}
		}
	}
	return v.Name
}

func (g *generator) varValueFromMethodMap(annotationValue string, varName string) string {

	for _, pair := range strings.Split(annotationValue, ",") {
		if pairTokens := strings.Split(strings.TrimSpace(pair), "|"); len(pairTokens) == 2 {
			arg := strings.TrimSpace(pairTokens[0])
			value := strings.TrimSpace(pairTokens[1])
			if arg == varName {
				return value
			}
		}
	}
	return ""
}

func (g *generator) multipartRequestBody(contract *model.Contract, method *model.Method) *types.RequestBody {

	properties := make(types.Properties)
	for _, arg := range method.Args {
		if arg.TypeID == typeIDIOReader {
			partName := g.streamPartName(contract, method, arg)
			properties[partName] = types.Schema{Type: "string", Format: "binary"}
		}
	}
	return &types.RequestBody{
		Content: types.Content{
			contentMultipartFormData: types.Media{
				Schema: types.Schema{Type: "object", Properties: properties},
			},
		},
	}
}

func (g *generator) multipartResponseSchema(contract *model.Contract, method *model.Method) types.Schema {

	properties := make(types.Properties)
	for _, res := range method.Results {
		if res.TypeID == typeIDIOReadCloser {
			partName := g.streamPartName(contract, method, res)
			properties[partName] = types.Schema{Type: "string", Format: "binary"}
		}
	}
	return types.Schema{Type: "object", Properties: properties}
}

func (g *generator) generateHTTPPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	prefix := model.GetAnnotationValue(g.project, contract, nil, nil, tagHttpPrefix, "")
	methodPath := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpPath, "/"+types.ToLowerCamel(method.Name))
	httpPath := methodPath
	if prefix != "" {
		httpPath = path.Join("/", prefix, methodPath)
	}

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	g.registerStruct(requestStructName, contract.PkgPath, method.Annotations, method.Args)
	g.registerStruct(responseStructName, contract.PkgPath, method.Annotations, method.Results)

	httpMethod := strings.ToLower(model.GetHTTPMethod(g.project, contract, method))
	successCode := model.GetAnnotationValueInt(g.project, contract, method, nil, tagHttpSuccess, 200)
	requestContentType := model.GetAnnotationValue(g.project, contract, method, nil, tagRequestContentType, contentJSON)
	responseContentType := model.GetAnnotationValue(g.project, contract, method, nil, tagResponseContentType, contentJSON)

	reqMultipart := g.requestMultipart(contract, method)
	respMultipart := g.responseMultipart(contract, method)
	var successContent types.Content
	switch {
	case g.countIOReadCloserResults(method) > 0 && !respMultipart:
		successContent = types.Content{
			contentOctetStream: types.Media{Schema: types.Schema{Type: "string", Format: "binary"}},
		}
	case respMultipart:
		successContent = types.Content{
			contentMultipartFormData: types.Media{
				Schema: g.multipartResponseSchema(contract, method),
			},
		}
	default:
		successContent = types.Content{
			responseContentType: types.Media{
				Schema: g.toSchema(responseStructName),
			},
		}
	}

	operation := &types.Operation{
		Summary:     model.GetAnnotationValue(g.project, contract, method, nil, tagSummary, ""),
		Description: model.GetAnnotationValue(g.project, contract, method, nil, tagDesc, ""),
		Tags:        serviceTags,
		Deprecated:  model.IsAnnotationSet(g.project, contract, method, nil, tagDeprecated),
		Responses: types.Responses{
			fmt.Sprintf("%d", successCode): types.Response{
				Description: types.CodeToText(successCode),
				Content:     successContent,
			},
		},
	}

	g.addPathParameters(operation, contract, method, httpPath)
	g.addQueryParameters(operation, contract, method, httpPath)
	g.addHeaderParameters(operation, contract, method)
	g.addCookieParameters(operation, contract, method)

	readerArgs := g.countIOReaderArgs(method)
	if readerArgs > 0 {
		if reqMultipart {
			operation.RequestBody = g.multipartRequestBody(contract, method)
		} else {
			requestContentType = contentOctetStream
			operation.RequestBody = &types.RequestBody{
				Content: types.Content{
					requestContentType: types.Media{
						Schema: types.Schema{Type: "string", Format: "binary"},
					},
				},
			}
		}
	} else if len(method.Args) > 0 {
		hasBodyArgs := false
		for _, arg := range method.Args {
			if arg.TypeID != "context:Context" && arg.TypeID != typeIDIOReader && !g.isArgInPath(arg, method, httpPath) && !g.isArgInQuery(arg, contract, method) && !g.isArgInHeader(arg, contract, method) && !g.isArgInCookie(arg, contract, method) {
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

	openAPIPath := pathParamColonToBraces(httpPath)
	pathValue, found := paths[openAPIPath]
	if !found {
		pathValue = types.Path{}
	}

	switch httpMethod {
	case "get":
		pathValue.Get = operation
	case "post":
		pathValue.Post = operation
	case "put":
		pathValue.Put = operation
	case "patch":
		pathValue.Patch = operation
	case "delete":
		pathValue.Delete = operation
	case "options":
		pathValue.Options = operation
	default:
		pathValue.Post = operation
	}

	paths[openAPIPath] = pathValue
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

func (g *generator) addQueryParameters(operation *types.Operation, contract *model.Contract, method *model.Method, httpPath string) {

	httpArgs := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpArg, "")
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
					if g.isArgInPath(arg, method, httpPath) {
						break
					}
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

	httpHeaders := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpHeader, "")
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

	httpCookies := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpCookies, "")
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

	httpHeaders := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpHeader, "")
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

func (g *generator) isArgInQuery(arg *model.Variable, contract *model.Contract, method *model.Method) (found bool) {

	httpArgs := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpArg, "")
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

func (g *generator) isArgInHeader(arg *model.Variable, contract *model.Contract, method *model.Method) (found bool) {

	httpHeaders := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpHeader, "")
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

func (g *generator) isArgInCookie(arg *model.Variable, contract *model.Contract, method *model.Method) (found bool) {

	httpCookies := model.GetAnnotationValue(g.project, contract, method, nil, tagHttpCookies, "")
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

func pathParamColonToBraces(httpPath string) (openAPIPath string) {

	parts := strings.Split(httpPath, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}
