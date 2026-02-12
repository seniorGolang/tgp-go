// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"tgp/internal/model"
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

	isJsonRPC := model.IsAnnotationSet(g.project, contract, nil, nil, model.TagServerJsonRPC) && !model.IsAnnotationSet(g.project, contract, method, nil, model.TagHTTPMethod)
	isHTTP := model.IsAnnotationSet(g.project, contract, nil, nil, model.TagServerHTTP) && (!model.IsAnnotationSet(g.project, contract, nil, nil, model.TagServerJsonRPC) || model.IsAnnotationSet(g.project, contract, method, nil, model.TagHTTPMethod))

	if isJsonRPC {
		g.generateJsonRPCPath(paths, contract, method, serviceTags)
	} else if isHTTP {
		g.generateHTTPPath(paths, contract, method, serviceTags)
	}
}

func (g *generator) generateJsonRPCPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	prefix := model.GetAnnotationValue(g.project, contract, nil, nil, model.TagHttpPrefix, "")
	urlPath := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpPath, "/"+types.ToLowerCamel(method.Name))
	urlPath = strings.Split(urlPath, ":")[0]
	jsonrpcPath := path.Join("/", prefix, urlPath)

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	g.registerStruct(requestStructName, contract.PkgPath, method.Annotations, method.Args, contentJSON)
	g.registerStruct(responseStructName, contract.PkgPath, method.Annotations, method.Results, contentJSON)

	operation := &types.Operation{
		OperationID: types.ToCamel(contract.Name) + types.ToCamel(method.Name),
		Summary:     model.GetAnnotationValue(g.project, contract, method, nil, tagSummary, ""),
		Description: model.GetAnnotationValue(g.project, contract, method, nil, tagDesc, ""),
		Tags:        serviceTags,
		Deprecated:  model.IsAnnotationSet(g.project, contract, method, nil, tagDeprecated),
		RequestBody: &types.RequestBody{
			Content: types.Content{
				contentJSON: types.Media{
					Schema: types.JSONRPCSchemaPerPath("params", g.toSchema(requestStructName)),
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
								types.JSONRPCSchemaPerPath("result", g.toSchema(responseStructName)),
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
	g.fillErrors(operation.Responses, method)

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
	if n == 1 && model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (g *generator) responseMultipart(contract *model.Contract, method *model.Method) bool {

	n := g.countIOReadCloserResults(method)
	if n > 1 {
		return true
	}
	if n == 1 && model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (g *generator) streamPartName(contract *model.Contract, method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[model.TagHttpPartName]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[model.TagHttpPartName]; found && val != "" {
			if partName := g.varValueFromMethodMap(val, v.Name); partName != "" {
				return partName
			}
		}
	}
	return v.Name
}

func (g *generator) streamPartContent(contract *model.Contract, method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[model.TagHttpPartContent]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[model.TagHttpPartContent]; found && val != "" {
			return g.varValueFromMethodMap(val, v.Name)
		}
	}
	return ""
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
	encoding := make(map[string]types.Encoding)
	for _, arg := range method.Args {
		if arg.TypeID == typeIDIOReader {
			partName := g.streamPartName(contract, method, arg)
			properties[partName] = types.Schema{Type: "string", Format: "binary"}
			if ct := g.streamPartContent(contract, method, arg); ct != "" {
				encoding[partName] = types.Encoding{ContentType: ct}
			}
		}
	}
	media := types.Media{Schema: types.Schema{Type: "object", Properties: properties}}
	if len(encoding) > 0 {
		media.Encoding = encoding
	}
	return &types.RequestBody{
		Content: types.Content{
			contentMultipartFormData: media,
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

func (g *generator) effectiveResponseSchema(contract *model.Contract, method *model.Method, responseStructName string) types.Schema {

	var results []*model.Variable
	for _, r := range method.Results {
		if r.TypeID != "error" {
			results = append(results, r)
		}
	}
	if len(results) == 0 {
		return g.toSchema(responseStructName)
	}
	hasAnyInline := false
	for _, r := range results {
		if model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpEnableInlineSingle) && len(results) == 1 {
			hasAnyInline = true
			break
		}
		if g.resultHasJsonInline(method, r) {
			hasAnyInline = true
			break
		}
	}
	if !hasAnyInline {
		return g.toSchema(responseStructName)
	}
	merged := types.Schema{Type: "object", Properties: make(types.Properties), Required: []string{}}
	for _, r := range results {
		inline := (model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpEnableInlineSingle) && len(results) == 1) || g.resultHasJsonInline(method, r)
		if inline {
			s := g.variableToSchema(r, contract.PkgPath, false)
			if s != nil {
				toMerge := g.resolveSchemaForMerge(s)
				if len(toMerge.Properties) > 0 {
					g.mergeSchema(&merged, toMerge)
				}
			}
		} else {
			jsonName := g.getJSONFieldName(r)
			if jsonName == "" || jsonName == "-" {
				jsonName = types.ToLowerCamel(r.Name)
			}
			if s := g.variableToSchema(r, contract.PkgPath, false); s != nil {
				merged.Properties[jsonName] = *s
			}
		}
	}
	return merged
}

func (g *generator) resolveSchemaForMerge(s *types.Schema) types.Schema {

	if s == nil {
		return types.Schema{}
	}
	if s.Ref != "" {
		resolved, ok := g.resolveRefToSchema(s.Ref)
		if ok {
			return resolved
		}
	}
	return *s
}

func (g *generator) mergeSchema(dst *types.Schema, src types.Schema) {

	if dst.Properties == nil {
		dst.Properties = make(types.Properties)
	}
	for k, v := range src.Properties {
		dst.Properties[k] = v
	}
	dst.Required = append(dst.Required, src.Required...)
}

func (g *generator) resultHasJsonInline(method *model.Method, v *model.Variable) bool {

	sub := method.Annotations.Sub(v.Name)
	for key, value := range sub {
		if key != model.TagParamTags {
			continue
		}
		for _, item := range strings.Split(value, "|") {
			tokens := strings.SplitN(strings.TrimSpace(item), ":", 2)
			if len(tokens) < 2 {
				continue
			}
			tagName := strings.TrimSpace(tokens[0])
			tagValue := strings.TrimSpace(tokens[1])
			if tagName == "json" && (tagValue == "inline" || strings.Contains(tagValue, ",inline")) {
				return true
			}
		}
	}
	return false
}

func (g *generator) bodyArgs(method *model.Method, contract *model.Contract, httpPath string) (out []*model.Variable) {

	for _, arg := range method.Args {
		if arg.TypeID == "context:Context" || arg.TypeID == typeIDIOReader {
			continue
		}
		if g.isArgInPath(arg, method, httpPath) || g.isArgInQuery(arg, contract, method) || g.isArgInHeader(arg, contract, method) || g.isArgInCookie(arg, contract, method) {
			continue
		}
		out = append(out, arg)
	}
	return out
}

func (g *generator) effectiveRequestBodySchema(contract *model.Contract, method *model.Method, requestStructName string, bodyArgs []*model.Variable) types.Schema {

	if len(bodyArgs) == 0 {
		return g.toSchema(requestStructName)
	}
	hasAnyInline := false
	for _, a := range bodyArgs {
		if g.resultHasJsonInline(method, a) {
			hasAnyInline = true
			break
		}
	}
	if !hasAnyInline {
		return g.toSchema(requestStructName)
	}
	merged := types.Schema{Type: "object", Properties: make(types.Properties), Required: []string{}}
	for _, a := range bodyArgs {
		if g.resultHasJsonInline(method, a) {
			s := g.variableToSchema(a, contract.PkgPath, true)
			if s != nil {
				toMerge := g.resolveSchemaForMerge(s)
				if len(toMerge.Properties) > 0 {
					g.mergeSchema(&merged, toMerge)
				}
			}
		} else {
			jsonName := g.getJSONFieldName(a)
			if jsonName == "" || jsonName == "-" {
				jsonName = types.ToLowerCamel(a.Name)
			}
			if s := g.variableToSchema(a, contract.PkgPath, true); s != nil {
				merged.Properties[jsonName] = *s
			}
		}
	}
	return merged
}

func (g *generator) generateHTTPPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	prefix := model.GetAnnotationValue(g.project, contract, nil, nil, model.TagHttpPrefix, "")
	methodPath := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpPath, "/"+types.ToLowerCamel(method.Name))
	httpPath := methodPath
	if prefix != "" {
		httpPath = path.Join("/", prefix, methodPath)
	}

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	httpMethod := strings.ToLower(model.GetHTTPMethod(g.project, contract, method))
	successCode := model.GetAnnotationValueInt(g.project, contract, method, nil, model.TagHttpSuccess, 200)
	requestContentType := model.GetAnnotationValue(g.project, contract, method, nil, model.TagRequestContentType, contentJSON)
	responseContentType := model.GetAnnotationValue(g.project, contract, method, nil, model.TagResponseContentType, contentJSON)

	g.registerStruct(requestStructName, contract.PkgPath, method.Annotations, method.Args, requestContentType)
	g.registerStruct(responseStructName, contract.PkgPath, method.Annotations, method.Results, contentJSON)


	reqMultipart := g.requestMultipart(contract, method)
	respMultipart := g.responseMultipart(contract, method)
	customResponse := model.IsAnnotationSet(g.project, contract, method, nil, tagHttpResponse)

	var successContent types.Content
	if !customResponse {
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
			successSchema := g.effectiveResponseSchema(contract, method, responseStructName)
			successContent = types.Content{
				responseContentType: types.Media{
					Schema: successSchema,
				},
			}
		}
	}

	successDesc := types.CodeToText(successCode)
	if customResponse {
		successDesc = "Ответ определяется кастомным обработчиком"
	}
	operation := &types.Operation{
		OperationID:  types.ToCamel(contract.Name) + types.ToCamel(method.Name),
		Summary:      model.GetAnnotationValue(g.project, contract, method, nil, tagSummary, ""),
		Description:  model.GetAnnotationValue(g.project, contract, method, nil, tagDesc, ""),
		Tags:         serviceTags,
		Deprecated:   model.IsAnnotationSet(g.project, contract, method, nil, tagDeprecated),
		Responses: types.Responses{
			fmt.Sprintf("%d", successCode): types.Response{
				Description: successDesc,
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
		bodyArgs := g.bodyArgs(method, contract, httpPath)
		if len(bodyArgs) > 0 {
			requestSchema := g.effectiveRequestBodySchema(contract, method, requestStructName, bodyArgs)
			operation.RequestBody = &types.RequestBody{
				Content: types.Content{
					requestContentType: types.Media{
						Schema: requestSchema,
					},
				},
			}
		}
	}

	g.addResponseHeaders(operation, contract, method, successCode)
	g.fillErrors(operation.Responses, method)

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

	httpArgs := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpArg, "")
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

	httpHeaders := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpHeader, "")
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
					required := arg.Annotations != nil && arg.Annotations.IsSet(model.TagRequired)
					operation.Parameters = append(operation.Parameters, types.Parameter{
						In:       "header",
						Name:     headerName,
						Required: required,
						Schema:   schema,
					})
					break
				}
			}
		}
	}
}

func (g *generator) addCookieParameters(operation *types.Operation, contract *model.Contract, method *model.Method) {

	httpCookies := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpCookies, "")
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
					required := arg.Annotations != nil && arg.Annotations.IsSet(model.TagRequired)
					operation.Parameters = append(operation.Parameters, types.Parameter{
						In:       "cookie",
						Name:     cookieName,
						Required: required,
						Schema:   schema,
					})
					break
				}
			}
		}
	}
}

func (g *generator) addResponseHeaders(operation *types.Operation, contract *model.Contract, method *model.Method, successCode int) {

	httpHeaders := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpHeader, "")
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

func (g *generator) fillErrors(responses types.Responses, method *model.Method) {

	if len(method.Errors) == 0 {
		return
	}

	byCode := make(map[int][]*model.ErrorInfo)
	for _, errInfo := range method.Errors {
		code := errInfo.HTTPCode
		if code == 0 || !types.IsValidHTTPCode(code) {
			code = 500
		}
		byCode[code] = append(byCode[code], errInfo)
	}

	for code, errInfos := range byCode {
		var schemas []types.Schema
		for _, errInfo := range errInfos {
			if typeInfo := g.errorInfoToType(errInfo); typeInfo != nil {
				if p := g.structTypeToSchema(typeInfo, nil); p != nil {
					schemas = append(schemas, *p)
				}
			}
		}
		var schema types.Schema
		switch len(schemas) {
		case 0:
		case 1:
			schema = schemas[0]
		default:
			schema = types.Schema{OneOf: schemas}
		}
		key := strconv.Itoa(code)
		desc := errInfos[0].HTTPCodeText
		if desc == "" {
			desc = types.CodeToText(code)
		}
		responses[key] = types.Response{
			Description: desc,
			Content: types.Content{
				contentJSON: types.Media{Schema: schema},
			},
		}
	}
}

func (g *generator) errorInfoToType(errInfo *model.ErrorInfo) (typeInfo *model.Type) {

	if typeInfo = g.project.Types[errInfo.TypeID]; typeInfo != nil {
		return typeInfo
	}
	for typeID, ti := range g.project.Types {
		if ti.TypeName == errInfo.TypeName && strings.Contains(typeID, errInfo.PkgPath) {
			return ti
		}
	}
	return nil
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

	httpArgs := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpArg, "")
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

	httpHeaders := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpHeader, "")
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

	httpCookies := model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpCookies, "")
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
