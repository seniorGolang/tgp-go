// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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
		if !model.ContractIsHTTPFamily(g.project, contract) {
			continue
		}
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

	isWS := model.MethodIsWS(g.project, contract, method)
	isSSE := model.MethodIsSSE(g.project, contract, method)
	isJsonRPC := model.MethodIsJSONRPC(g.project, contract, method)
	isHTTP := model.MethodIsHTTP(g.project, contract, method)

	if isWS {
		g.generateWebSocketPath(paths, contract, method, serviceTags)
	}
	if isSSE {
		g.generateSSEPath(paths, contract, method, serviceTags)
	}
	if isWS || isSSE {
		return
	}
	if isJsonRPC {
		g.generateJsonRPCPath(paths, contract, method, serviceTags)
	} else if isHTTP {
		g.generateHTTPPath(paths, contract, method, serviceTags)
	}
}

func (g *generator) generateWebSocketPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	requestName := g.requestStructName(contract, method)
	args := streamNonChannelVariables(g.project, method.Args)
	g.registerStruct(requestName, contract.PkgPath, method, args, contentJSON, true)
	wireMethod := model.JsonRPCWireMethod(contract.Name, method.Name)
	methodBlock := fmt.Sprintf("%s (`%s`): %s", method.Name, wireMethod, strings.TrimSpace(descriptionFromMethod(method)))
	wsPath := model.ContractWSPath(g.project, contract)
	pathValue := paths[wsPath]
	if pathValue.Get == nil {
		pathValue.Get = &types.Operation{
			OperationID: types.ToCamel(contract.Name) + "WebSocket",
			Description: "WebSocket JSON-RPC 2.0 streaming profile: open with id and params; chunks use $/stream; client ends with $/stream.end; cancellation uses $/cancel.\n\n" + methodBlock,
			Tags:        serviceTags,
			XWebSocket:  true,
			Responses: types.Responses{
				"101": {Description: "Switching Protocols"},
			},
		}
	} else {
		pathValue.Get.Description += "\n\n" + methodBlock
	}
	paths[wsPath] = pathValue
}

func (g *generator) generateSSEPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	requestName := g.requestStructName(contract, method)
	args := streamNonChannelVariables(g.project, method.Args)
	g.registerStruct(requestName, contract.PkgPath, method, args, contentJSON, true)
	operation := &types.Operation{
		OperationID: types.ToCamel(contract.Name) + types.ToCamel(method.Name) + "SSE",
		Description: descriptionFromMethod(method) + "\n\nServer-Sent Events JSON-RPC streaming profile: each event contains a $/stream notification; the final event is the JSON-RPC result.",
		Tags:        serviceTags,
		RequestBody: &types.RequestBody{Content: types.Content{contentJSON: types.Media{Schema: g.toSchema(requestName)}}},
		Responses: types.Responses{
			"200": {Description: "Event stream", Content: types.Content{"text/event-stream": {Schema: types.Schema{Type: "string"}}}},
		},
	}
	paths[model.MethodSSEPath(g.project, contract, method)] = types.Path{Post: operation}
}

func streamNonChannelVariables(project *model.Project, variables []*model.Variable) (out []*model.Variable) {

	for _, variable := range variables {
		if variable.TypeID == "context:Context" || model.TypeRefIsChan(project, &variable.TypeRef) {
			continue
		}
		out = append(out, variable)
	}
	return
}

func (g *generator) generateJsonRPCPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	prefix := model.GetAnnotationValue(g.project, contract, nil, nil, model.TagHttpPrefix, "")
	pathValue := model.MethodHTTPPathValue(g.project, contract, method)
	pathBase := strings.TrimPrefix(strings.Split(pathValue, ":")[0], "/")
	jsonrpcPath := model.JoinHTTPPath(prefix, "/"+pathBase)

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	g.registerStruct(requestStructName, contract.PkgPath, method, method.Args, contentJSON, true)
	g.registerStruct(responseStructName, contract.PkgPath, method, method.Results, contentJSON, false)

	paramsArgs := g.bodyArgs(method, contract, jsonrpcPath)
	paramsSchema := g.effectiveRequestBodySchema(contract, method, requestStructName, paramsArgs)
	resultSchema := g.effectiveResponseSchema(contract, method, responseStructName, nil)

	opSummary := ""
	opDesc := descriptionFromMethod(method)
	if method.Annotations != nil {
		opSummary = method.Annotations.Value(tagSummary, "")
	}
	operation := &types.Operation{
		OperationID: types.ToCamel(contract.Name) + types.ToCamel(method.Name),
		Summary:     opSummary,
		Description: opDesc,
		Tags:        serviceTags,
		Deprecated:  model.IsAnnotationSet(g.project, contract, method, nil, tagDeprecated),
		RequestBody: &types.RequestBody{
			Description: requestBodyDescription(method),
			Content: types.Content{
				contentJSON: types.Media{
					Schema: types.JSONRPCSchemaPerPath("params", paramsSchema),
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
								types.JSONRPCSchemaPerPath("result", resultSchema),
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
	return
}

func (g *generator) countIOReadCloserResults(method *model.Method) (n int) {

	for _, res := range method.Results {
		if res.TypeID == typeIDIOReadCloser {
			n++
		}
	}
	return
}

func (g *generator) requestMultipart(contract *model.Contract, method *model.Method) (ok bool) {

	n := g.countIOReaderArgs(method)
	if n > 1 {
		return true
	}
	if n == 1 && model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (g *generator) responseMultipart(contract *model.Contract, method *model.Method) (ok bool) {

	n := g.countIOReadCloserResults(method)
	if n > 1 {
		return true
	}
	if n == 1 && model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (g *generator) streamPartName(contract *model.Contract, method *model.Method, v *model.Variable) (s string) {

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

func (g *generator) streamPartContent(contract *model.Contract, method *model.Method, v *model.Variable) (s string) {

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

func (g *generator) varValueFromMethodMap(annotationValue string, varName string) (s string) {

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
		Description: requestBodyDescription(method),
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

func (g *generator) resultNamesExcludeFromBody(contract *model.Contract, method *model.Method) (out map[string]struct{}) {

	return model.HTTPResultNamesOmitFromExchangeBody(g.project, contract, method)
}

func (g *generator) resultsForBody(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	return model.HTTPResultsForExchangeBody(g.project, contract, method)
}

func (g *generator) responseBodyStructName(contract *model.Contract, method *model.Method) (s string) {

	return types.ToCamel(contract.Name) + types.ToCamel(method.Name) + "ResponseBody"
}

func (g *generator) requestBodyStructName(contract *model.Contract, method *model.Method) (s string) {

	return types.ToCamel(contract.Name) + types.ToCamel(method.Name) + "RequestBody"
}

func (g *generator) effectiveResponseSchema(contract *model.Contract, method *model.Method, responseStructName string, bodyResults []*model.Variable) types.Schema {

	var results []*model.Variable
	if bodyResults != nil {
		results = bodyResults
	} else {
		for _, r := range method.Results {
			if r.TypeID != "error" {
				results = append(results, r)
			}
		}
	}
	if len(results) == 0 {
		return g.toSchema(responseStructName)
	}
	if model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpEnableInlineSingle) && len(results) == 1 {
		effective := model.EffectiveVariable(method, results[0])
		if s := g.variableToSchema(effective, contract.PkgPath, false); s != nil && s.Type != "" && s.Type != "object" {
			return *s
		}
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
		effective := model.EffectiveVariable(method, r)
		inline := (model.IsAnnotationSet(g.project, contract, method, nil, model.TagHttpEnableInlineSingle) && len(results) == 1) || g.resultHasJsonInline(method, r)
		if inline {
			s := g.variableToSchema(effective, contract.PkgPath, false)
			if s != nil {
				toMerge := g.resolveSchemaForMerge(s)
				if len(toMerge.Properties) > 0 {
					g.mergeSchema(&merged, toMerge)
				}
			}
		} else {
			jsonName := g.getJSONFieldName(effective)
			if jsonName == "" || jsonName == "-" {
				jsonName = types.ToLowerCamel(effective.Name)
			}
			if s := g.variableToSchema(effective, contract.PkgPath, false); s != nil {
				merged.Properties[jsonName] = *s
			}
		}
	}
	return merged
}

func (g *generator) resolveSchemaForMerge(s *types.Schema) (resolved types.Schema) {

	if s == nil {
		return types.Schema{}
	}
	if s.Ref != "" {
		if refResolved, ok := g.resolveRefToSchema(s.Ref); ok {
			return g.resolveSchemaForMerge(&refResolved)
		}
		return *s
	}
	if len(s.Properties) > 0 {
		return *s
	}
	for _, item := range s.OneOf {
		if item.Nullable {
			continue
		}
		if candidate := g.resolveSchemaForMerge(&item); len(candidate.Properties) > 0 {
			return candidate
		}
	}
	for _, item := range s.AllOf {
		if candidate := g.resolveSchemaForMerge(&item); len(candidate.Properties) > 0 {
			return candidate
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

func (g *generator) resultHasJsonInline(method *model.Method, v *model.Variable) (ok bool) {

	if method == nil {
		return false
	}
	return tags.HasJSONInline(method.Annotations, v.Name)
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
	return
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
		effective := model.EffectiveVariable(method, a)
		if g.resultHasJsonInline(method, a) {
			s := g.variableToSchema(effective, contract.PkgPath, true)
			if s != nil {
				toMerge := g.resolveSchemaForMerge(s)
				if len(toMerge.Properties) > 0 {
					g.mergeSchema(&merged, toMerge)
				}
			}
		} else {
			jsonName := g.getJSONFieldName(effective)
			if jsonName == "" || jsonName == "-" {
				jsonName = types.ToLowerCamel(effective.Name)
			}
			if s := g.variableToSchema(effective, contract.PkgPath, true); s != nil {
				merged.Properties[jsonName] = *s
			}
		}
	}
	return merged
}

func (g *generator) generateHTTPPath(paths map[string]types.Path, contract *model.Contract, method *model.Method, serviceTags []string) {

	httpPath := model.MethodHTTPFullPath(g.project, contract, method)

	requestStructName := g.requestStructName(contract, method)
	responseStructName := g.responseStructName(contract, method)

	httpMethod := strings.ToLower(model.GetHTTPMethod(g.project, contract, method))
	successCode := model.GetAnnotationValueInt(g.project, contract, method, nil, model.TagHttpSuccess, 200)
	requestContentType := model.GetAnnotationValue(g.project, contract, method, nil, model.TagRequestContentType, contentJSON)
	responseContentType := model.GetAnnotationValue(g.project, contract, method, nil, model.TagResponseContentType, contentJSON)

	g.registerStruct(requestStructName, contract.PkgPath, method, method.Args, requestContentType, true)

	resultsForResponse := g.resultsWithoutError(method)
	hasPayloadResults := len(resultsForResponse) > 0
	g.registerStruct(responseStructName, contract.PkgPath, method, resultsForResponse, contentJSON, false)

	reqMultipart := g.requestMultipart(contract, method)
	respMultipart := g.responseMultipart(contract, method)
	customResponse := model.IsAnnotationSet(g.project, contract, method, nil, tagHttpResponse)

	var successContent types.Content
	if !customResponse {
		if hasPayloadResults {
			switch {
			case g.countIOReadCloserResults(method) > 0 && !respMultipart:
				streamResponseContentType := model.GetAnnotationValue(g.project, contract, method, nil, model.TagResponseContentType, contentOctetStream)
				successContent = types.Content{
					streamResponseContentType: types.Media{Schema: types.Schema{Type: "string", Format: "binary"}},
				}
			case respMultipart:
				successContent = types.Content{
					contentMultipartFormData: types.Media{
						Schema: g.multipartResponseSchema(contract, method),
					},
				}
			default:
				var successSchema types.Schema
				if len(g.resultNamesExcludeFromBody(contract, method)) > 0 {
					bodyStructName := g.responseBodyStructName(contract, method)
					g.registerStruct(bodyStructName, contract.PkgPath, method, g.resultsForBody(contract, method), contentJSON, false)
					successSchema = g.effectiveResponseSchema(contract, method, bodyStructName, g.resultsForBody(contract, method))
				} else {
					successSchema = g.effectiveResponseSchema(contract, method, responseStructName, nil)
				}
				successContent = types.Content{
					responseContentType: types.Media{
						Schema: successSchema,
					},
				}
			}
		} else {
			successContent = types.Content{
				responseContentType: types.Media{
					Schema: g.toSchema(responseStructName),
				},
			}
		}
	}

	successDesc := types.CodeToText(successCode)
	if customResponse {
		successDesc = "Ответ определяется кастомным обработчиком"
	}

	successKey := fmt.Sprintf("%d", successCode)
	responses := types.Responses{}
	if customResponse {
		responses[successKey] = types.Response{
			Description: successDesc,
		}
	} else {
		responses[successKey] = types.Response{
			Description: successDesc,
			Content:     successContent,
		}
	}

	opSummary := ""
	opDesc := descriptionFromMethod(method)
	if method.Annotations != nil {
		opSummary = method.Annotations.Value(tagSummary, "")
	}
	operation := &types.Operation{
		OperationID: types.ToCamel(contract.Name) + types.ToCamel(method.Name),
		Summary:     opSummary,
		Description: opDesc,
		Tags:        serviceTags,
		Deprecated:  model.IsAnnotationSet(g.project, contract, method, nil, tagDeprecated),
		Responses:   responses,
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
				Description: requestBodyDescription(method),
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
			bodyStructName := g.requestBodyStructName(contract, method)
			g.registerStruct(bodyStructName, contract.PkgPath, method, bodyArgs, requestContentType, true)
			requestSchema := g.effectiveRequestBodySchema(contract, method, bodyStructName, bodyArgs)
			operation.RequestBody = &types.RequestBody{
				Description: requestBodyDescription(method),
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
					effective := model.EffectiveVariable(method, arg)
					var schema types.Schema
					if schemaPtr := g.variableToSchema(effective, contract.PkgPath, true); schemaPtr != nil {
						schema = *schemaPtr
					}
					operation.Parameters = append(operation.Parameters, types.Parameter{
						In:          "path",
						Name:        paramName,
						Required:    true,
						Schema:      schema,
						Description: descriptionFromVariable(effective),
					})
					break
				}
			}
		}
	}
}

func (g *generator) addQueryParameters(operation *types.Operation, contract *model.Contract, method *model.Method, httpPath string) {

	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpArg, "")) {
		if it.Arg == "path" {
			continue
		}
		for _, arg := range method.Args {
			if arg.Name == it.Arg {
				if g.isArgInPath(arg, method, httpPath) {
					break
				}
				effective := model.EffectiveVariable(method, arg)
				var schema types.Schema
				if schemaPtr := g.variableToSchema(effective, contract.PkgPath, true); schemaPtr != nil {
					schema = *schemaPtr
				}
				operation.Parameters = append(operation.Parameters, types.Parameter{
					In:          "query",
					Name:        it.Key,
					Required:    isRequiredQueryParameter(effective),
					Schema:      schema,
					Description: descriptionFromVariable(effective),
				})
				break
			}
		}
	}
}

const transportBodyOverlayDescription = "Если задан, перезаписывает соответствующее поле JSON request body."

func (g *generator) resultsWithoutError(method *model.Method) (out []*model.Variable) {

	for _, result := range method.Results {
		if result.TypeID != "error" {
			out = append(out, result)
		}
	}
	return
}

func (g *generator) addHeaderParameters(operation *types.Operation, contract *model.Contract, method *model.Method) {

	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpHeader, "")) {
		g.appendArgMapRequestParameter(operation, contract, method, it, "header")
	}
}

func (g *generator) addCookieParameters(operation *types.Operation, contract *model.Contract, method *model.Method) {

	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpCookies, "")) {
		g.appendArgMapRequestParameter(operation, contract, method, it, "cookie")
	}
}

func (g *generator) appendArgMapRequestParameter(
	operation *types.Operation,
	contract *model.Contract,
	method *model.Method,
	it model.ArgMapItem,
	in string,
) {

	switch it.Mode {
	case model.ArgModeExplicit, model.ArgModeImplicit:
	case model.ArgModeBody:
	default:
		return
	}

	for _, arg := range method.Args {
		if arg.Name != it.Arg {
			continue
		}

		effective := model.EffectiveVariable(method, arg)
		var schema types.Schema
		if schemaPtr := g.variableToSchema(effective, contract.PkgPath, true); schemaPtr != nil {
			schema = *schemaPtr
		}

		paramDesc := descriptionFromVariable(effective)
		required := false
		switch it.Mode {
		case model.ArgModeExplicit, model.ArgModeImplicit:
			required = isRequiredHeaderOrCookie(effective)
		case model.ArgModeBody:
			if paramDesc != "" {
				paramDesc += ". "
			}
			paramDesc += transportBodyOverlayDescription
		}

		operation.Parameters = append(operation.Parameters, types.Parameter{
			In:          in,
			Name:        it.Key,
			Required:    required,
			Schema:      schema,
			Description: paramDesc,
		})
		return
	}
}

func (g *generator) addResponseHeaders(operation *types.Operation, contract *model.Contract, method *model.Method, successCode int) {

	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpHeader, "")) {
		for _, result := range method.Results {
			if result.Name != it.Arg || result.TypeID == "error" {
				continue
			}
			effective := model.EffectiveVariable(method, result)
			schemaPtr := g.variableToSchema(effective, contract.PkgPath, false)
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
			operation.Responses[successKey].Headers[it.Key] = types.Header{
				Description: descriptionFromVariable(effective),
				Schema:      schema,
			}
			break
		}
	}
}

func (g *generator) fillErrors(responses types.Responses, method *model.Method) {

	if len(method.Errors) == 0 {
		return
	}

	byCode := make(map[int][]*model.ErrorInfo)
	var withoutHTTPCode []*model.ErrorInfo

	for _, errInfo := range method.Errors {
		if errInfo.TypeID == "" {
			continue
		}
		code := errInfo.HTTPCode
		if code != 0 && types.IsValidHTTPCode(code) {
			byCode[code] = append(byCode[code], errInfo)
			continue
		}
		withoutHTTPCode = append(withoutHTTPCode, errInfo)
	}

	codes := make([]int, 0, len(byCode))
	for code := range byCode {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	for _, code := range codes {
		errInfos := byCode[code]
		schema := g.errorInfosSchema(errInfos)
		if schema == nil {
			continue
		}
		key := strconv.Itoa(code)
		desc := errInfos[0].HTTPCodeText
		if desc == "" {
			desc = types.CodeToText(code)
		}
		responses[key] = types.Response{
			Description: desc,
			Content: types.Content{
				contentJSON: types.Media{Schema: *schema},
			},
		}
	}

	schema := g.errorInfosSchema(withoutHTTPCode)
	if schema == nil {
		return
	}
	responses[responseKeyDefault] = types.Response{
		Description: "Error",
		Content: types.Content{
			contentJSON: types.Media{Schema: *schema},
		},
	}
}

func (g *generator) errorInfosSchema(errInfos []*model.ErrorInfo) (schema *types.Schema) {

	var schemas []types.Schema
	for _, errInfo := range errInfos {
		if typeInfo := g.errorInfoToType(errInfo); typeInfo != nil {
			if p := g.structTypeToSchema(typeInfo, nil); p != nil {
				schemas = append(schemas, *p)
			}
		}
	}
	switch len(schemas) {
	case 0:
		return nil
	case 1:
		return &schemas[0]
	default:
		return &types.Schema{OneOf: schemas}
	}
}

func (g *generator) errorInfoToType(errInfo *model.ErrorInfo) (typeInfo *model.Type) {

	if errInfo == nil || errInfo.TypeID == "" {
		return
	}

	typeInfo = g.project.Types[errInfo.TypeID]
	return
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

	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpArg, "")) {
		if it.Arg != "path" && it.Arg == arg.Name {
			return true
		}
	}
	return false
}

func (g *generator) isArgInHeader(arg *model.Variable, contract *model.Contract, method *model.Method) (found bool) {

	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpHeader, "")) {
		if it.Arg == arg.Name && (it.Mode == model.ArgModeExplicit || it.Mode == model.ArgModeImplicit) {
			return true
		}
	}
	return false
}

func (g *generator) isArgInCookie(arg *model.Variable, contract *model.Contract, method *model.Method) (found bool) {

	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(g.project, contract, method, nil, model.TagHttpCookies, "")) {
		if it.Arg == arg.Name && (it.Mode == model.ArgModeExplicit || it.Mode == model.ArgModeImplicit) {
			return true
		}
	}
	return false
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
