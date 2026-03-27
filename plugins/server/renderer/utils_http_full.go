// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/converter"
	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) headerEntries(method *model.Method) (out []model.ArgMapItem) {

	return model.ParseArgMapEntries(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpHeader, ""))
}

func (r *contractRenderer) cookieEntries(method *model.Method) (out []model.ArgMapItem) {

	return model.ParseArgMapEntries(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpCookies, ""))
}

func (r *contractRenderer) varHeaderMap(method *model.Method) (out map[string]string) {

	out = make(map[string]string)
	for _, it := range r.headerEntries(method) {
		out[it.Arg] = it.Key
	}
	return
}

func (r *contractRenderer) varHeaderMapForRequest(method *model.Method) (out map[string]string) {

	out = make(map[string]string)
	for _, it := range r.headerEntries(method) {
		if r.argByName(method, it.Arg) != nil {
			out[it.Arg] = it.Key
		}
	}
	return
}

func (r *contractRenderer) varCookieMap(method *model.Method) (out map[string]string) {

	out = make(map[string]string)
	for _, it := range r.cookieEntries(method) {
		out[it.Arg] = it.Key
	}
	return
}

func (r *contractRenderer) varCookieMapForRequest(method *model.Method) (out map[string]string) {

	out = make(map[string]string)
	for _, it := range r.cookieEntries(method) {
		if r.argByName(method, it.Arg) != nil {
			out[it.Arg] = it.Key
		}
	}
	return
}

func usedHeaderNamesForRequestOverlay(project *model.Project, contract *model.Contract, method *model.Method) (out []string) {

	if project == nil || contract == nil || method == nil {
		return nil
	}
	headers := make(map[string]struct{})
	argSet := make(map[string]struct{})
	for _, a := range argsWithoutContext(method) {
		argSet[a.Name] = struct{}{}
	}
	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(project, contract, method, nil, model.TagHttpHeader, "")) {
		if _, isArg := argSet[it.Arg]; isArg && it.Key != "" {
			headers[it.Key] = struct{}{}
		}
	}
	return common.SortedKeys(headers)
}

func usedCookieNamesForRequestOverlay(project *model.Project, contract *model.Contract, method *model.Method) (out []string) {

	if project == nil || contract == nil || method == nil {
		return nil
	}
	cookies := make(map[string]struct{})
	argSet := make(map[string]struct{})
	for _, a := range argsWithoutContext(method) {
		argSet[a.Name] = struct{}{}
	}
	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(project, contract, method, nil, model.TagHttpCookies, "")) {
		if _, isArg := argSet[it.Arg]; isArg && it.Key != "" {
			cookies[it.Key] = struct{}{}
		}
	}
	return common.SortedKeys(cookies)
}

func (r *contractRenderer) argPathMap(method *model.Method) (out map[string]string) {

	out = make(map[string]string)
	if urlPath := model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpPath, ""); urlPath != "" {
		urlTokens := strings.Split(urlPath, "/")
		for _, token := range urlTokens {
			if strings.HasPrefix(token, ":") {
				arg := strings.TrimSpace(strings.TrimPrefix(token, ":"))
				out[arg] = arg
			}
		}
	}
	return
}

func (r *contractRenderer) argParamMap(method *model.Method) (out map[string]string) {

	out = make(map[string]string)
	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpArg, "")) {
		if it.Arg != "path" {
			out[it.Arg] = it.Key
		}
	}
	return
}

func (r *contractRenderer) resultNamesExcludeFromBody(method *model.Method) (out map[string]struct{}) {

	out = make(map[string]struct{})
	for _, it := range r.headerEntries(method) {
		if (it.Mode == model.ArgModeExplicit || it.Mode == model.ArgModeImplicit) && r.resultByName(method, it.Arg) != nil {
			out[it.Arg] = struct{}{}
		}
	}
	for _, it := range r.cookieEntries(method) {
		if (it.Mode == model.ArgModeExplicit || it.Mode == model.ArgModeImplicit) && r.resultByName(method, it.Arg) != nil {
			out[it.Arg] = struct{}{}
		}
	}
	return
}

func (r *contractRenderer) resultsForBody(method *model.Method) (out []*model.Variable) {

	exclude := r.resultNamesExcludeFromBody(method)
	var list []*model.Variable
	for _, res := range resultsWithoutError(method) {
		if res.TypeID == TypeIDIOReadCloser {
			continue
		}
		if _, ok := exclude[res.Name]; ok {
			continue
		}
		list = append(list, res)
	}
	return list
}

func (r *contractRenderer) argByName(method *model.Method, argName string) (v *model.Variable) {

	argName = strings.TrimPrefix(argName, "!")
	for _, arg := range method.Args {
		if arg.Name == argName {
			return arg
		}
	}
	return nil
}

func (r *contractRenderer) resultByName(method *model.Method, retName string) (v *model.Variable) {

	for _, ret := range method.Results {
		if ret.Name == retName {
			return ret
		}
	}
	return nil
}

func (r *contractRenderer) retCookieMap(method *model.Method) (out map[string]string) {

	cookies := make(map[string]string)
	cookieMap := r.varCookieMap(method)
	for varName, cookieName := range common.SortedPairs(cookieMap) {
		if r.resultByName(method, varName) != nil {
			cookies[varName] = cookieName
		}
	}
	return cookies
}

func (r *contractRenderer) argsWithoutSpecialArgs(method *model.Method) (out []*model.Variable) {

	vars := make([]*model.Variable, 0)
	argsAll := argsWithoutContext(method)
	pathMap := r.argPathMap(method)
	paramMap := r.argParamMap(method)
	headerRequest := r.varHeaderMapForRequest(method)
	cookieRequest := r.varCookieMapForRequest(method)

	for _, arg := range argsAll {
		_, inPath := pathMap[arg.Name]
		_, inArgs := paramMap[arg.Name]
		_, inHeader := headerRequest[arg.Name]
		_, inCookie := cookieRequest[arg.Name]

		if !inArgs && !inPath && !inHeader && !inCookie {
			vars = append(vars, arg)
		}
	}
	return vars
}

func (r *contractRenderer) methodRequestBodyStreamArg(method *model.Method) (v *model.Variable) {

	for _, arg := range argsWithoutContext(method) {
		if arg.TypeID == TypeIDIOReader {
			return arg
		}
	}
	return nil
}

func (r *contractRenderer) methodResponseBodyStreamResult(method *model.Method) (v *model.Variable) {

	for _, res := range resultsWithoutError(method) {
		if res.TypeID == TypeIDIOReadCloser {
			return res
		}
	}
	return nil
}

func (r *contractRenderer) methodRequestBodyStreamArgs(method *model.Method) (args []*model.Variable) {

	for _, arg := range argsWithoutContext(method) {
		if arg.TypeID == TypeIDIOReader {
			args = append(args, arg)
		}
	}
	return args
}

func (r *contractRenderer) methodResponseBodyStreamResults(method *model.Method) (results []*model.Variable) {

	for _, res := range resultsWithoutError(method) {
		if res.TypeID == TypeIDIOReadCloser {
			results = append(results, res)
		}
	}
	return results
}

func (r *contractRenderer) methodRequestMultipart(method *model.Method) bool {

	readerArgs := r.methodRequestBodyStreamArgs(method)
	if len(readerArgs) > 1 {
		return true
	}
	if len(readerArgs) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (r *contractRenderer) methodResponseMultipart(method *model.Method) bool {

	readCloserResults := r.methodResponseBodyStreamResults(method)
	if len(readCloserResults) > 1 {
		return true
	}
	if len(readCloserResults) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (r *contractRenderer) streamPartName(method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[model.TagHttpPartName]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[model.TagHttpPartName]; found && val != "" {
			if partName := r.varValueFromMethodMap(val, v.Name); partName != "" {
				return partName
			}
		}
	}
	return v.Name
}

func (r *contractRenderer) streamPartContent(method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[model.TagHttpPartContent]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[model.TagHttpPartContent]; found && val != "" {
			return r.varValueFromMethodMap(val, v.Name)
		}
	}
	return ""
}

func (r *contractRenderer) varValueFromMethodMap(annotationValue string, varName string) string {

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

func (r *contractRenderer) argFromString(srcFile *GoFile, typeGen *types.Generator, method *model.Method, typeName string, varMap map[string]string, srcCode func(srcName string) Code, errBody func(arg, header string) []Code, getTarget func(arg *model.Variable) *Statement) *Statement {

	block := Line()
	if len(varMap) != 0 {
		for fullArgName, srcName := range common.SortedPairs(varMap) {
			fullArgName = strings.TrimPrefix(fullArgName, "!")
			argTokens := strings.Split(fullArgName, ".")
			argName := argTokens[0]
			argVarName := strings.Join(argTokens, "")
			arg := r.argByName(method, argName)
			if arg == nil {
				continue
			}
			srcName = strings.TrimPrefix(srcName, "!")

			var typ *model.Type
			var fieldTypeID string
			var argTypeName string
			var fieldMapKey *model.TypeRef
			var fieldIsSlice bool
			var fieldMapValue *model.TypeRef
			var fieldElementPointers int
			var fieldNumberOfPointers int

			if len(argTokens) > 1 {
				argType, ok := r.project.Types[arg.TypeID]
				if !ok {
					continue
				}
				if argType.Kind == model.TypeKindAlias && argType.AliasOf != "" {
					if baseType, ok := r.project.Types[argType.AliasOf]; ok {
						argType = baseType
					}
				}
				currentType := argType
				var field *model.StructField
				for i := 1; i < len(argTokens); i++ {
					for _, f := range currentType.StructFields {
						if f.Name == argTokens[i] {
							field = f
							break
						}
					}
					if field == nil {
						break
					}
					// Если это не последний элемент пути, переходим к следующему типу
					if i+1 < len(argTokens) {
						nextType, ok := r.project.Types[field.TypeID]
						if !ok {
							field = nil
							break
						}
						if nextType.Kind == model.TypeKindAlias && nextType.AliasOf != "" {
							if baseType, ok := r.project.Types[nextType.AliasOf]; ok {
								currentType = baseType
							} else {
								field = nil
								break
							}
						} else {
							currentType = nextType
						}
					}
				}
				if field == nil {
					continue
				}
				fieldTypeID = field.TypeID
				fieldIsSlice = field.IsSlice
				fieldElementPointers = field.ElementPointers
				fieldNumberOfPointers = field.NumberOfPointers
				fieldMapKey = field.MapKey
				fieldMapValue = field.MapValue
			} else {
				fieldTypeID = arg.TypeID
				fieldIsSlice = arg.IsSlice
				fieldElementPointers = arg.ElementPointers
				fieldNumberOfPointers = arg.NumberOfPointers
				fieldMapKey = arg.MapKey
				fieldMapValue = arg.MapValue
			}

			switch {
			case fieldIsSlice:
				// Для слайсов TypeID уже содержит тип элемента без префикса []
				elementTypeID := fieldTypeID
				if converter.IsBuiltinTypeID(elementTypeID) {
					argTypeName = "[]" + elementTypeID
				} else {
					var ok bool
					typ, ok = r.project.Types[elementTypeID]
					if !ok {
						continue
					}
					argTypeName = "[]" + typ.TypeName
					if typ.TypeName == "" {
						argTypeName = "[]" + string(typ.Kind)
					}
				}
			case converter.IsBuiltinTypeID(fieldTypeID):
				argTypeName = fieldTypeID
			default:
				var ok bool
				typ, ok = r.project.Types[fieldTypeID]
				if !ok {
					continue
				}
				argTypeName = typ.TypeName
				if argTypeName == "" {
					argTypeName = string(typ.Kind)
				}
			}

			rawVarName := "_" + argName + "_"
			fieldVar := &model.Variable{
				TypeRef: model.TypeRef{
					TypeID:           fieldTypeID,
					IsSlice:          fieldIsSlice,
					ElementPointers:  fieldElementPointers,
					NumberOfPointers: fieldNumberOfPointers,
					MapKey:           fieldMapKey,
					MapValue:         fieldMapValue,
				},
			}

			if fieldTypeID == "string" && !fieldIsSlice {
				block.If(Id(rawVarName).Op(":=").Add(srcCode(srcName)).Op(";").Id(rawVarName).Op("!=").Lit("")).
					BlockFunc(func(bg *Group) {
						var reqID *Statement
						if len(argTokens) == 1 {
							reqID = getTarget(arg)
						} else {
							reqID = Id("request").Dot(r.requestStructFieldName(method, arg))
							for _, token := range argTokens[1:] {
								reqID = reqID.Dot(toCamel(token))
							}
						}
						if fieldNumberOfPointers == 0 {
							bg.Add(reqID.Op("=").Id(rawVarName))
						} else {
							bg.Add(reqID.Op("=").Op("&").Id(rawVarName))
						}
					}).Line()
				continue
			}

			typeForArg, ok := r.project.Types[arg.TypeID]
			useConverterTarget := ok && typeForArg != nil && (typeForArg.ParseFromString != nil || converter.HasBuiltinScalarBase(r.project, arg.TypeID))

			block.If(Id(rawVarName).Op(":=").Add(srcCode(srcName)).Op(";").Id(rawVarName).Op("!=").Lit("")).
				BlockFunc(func(bg *Group) {
					var reqID *Statement
					if len(argTokens) == 1 {
						reqID = getTarget(arg)
					} else {
						reqID = Id("request").Dot(r.requestStructFieldName(method, arg))
						for _, token := range argTokens[1:] {
							reqID = reqID.Dot(toCamel(token))
						}
					}
					if useConverterTarget {
						bg.Add(r.argToTypeConverter(srcFile, typeGen, Id(rawVarName), fieldVar, reqID, errBody(argVarName, srcName)))
					} else {
						if typ != nil && typ.ImportPkgPath != "" {
							typeParts := strings.Split(argTypeName, ".")
							if len(typeParts) > 1 {
								bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, typeParts[1])
							} else {
								bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, argTypeName)
							}
						} else {
							bg.Var().Id(argVarName).Id(argTypeName)
						}
						bg.Add(r.argToTypeConverter(srcFile, typeGen, Id(rawVarName), fieldVar, Id(argVarName), errBody(argVarName, srcName)))
						argID := Id(argVarName)
						if fieldNumberOfPointers > 0 {
							argID = Op("&").Add(argID)
						}
						bg.Add(reqID.Op("=").Add(argID))
					}
				}).Line()
		}
	}
	return block
}

func (r *contractRenderer) argFromStringOrdered(srcFile *GoFile, typeGen *types.Generator, method *model.Method, typeName string, varMap map[string]string, orderedArgs []string, srcCode func(srcName string) Code, errBody func(arg, header string) []Code, getTarget func(arg *model.Variable) *Statement) *Statement {

	block := Line()
	if len(varMap) != 0 {
		var argsToProcess []string
		if len(orderedArgs) > 0 {
			argsToProcess = orderedArgs
		} else {
			// Если порядок не указан, используем отсортированные ключи для детерминированного порядка
			for argName := range common.SortedPairs(varMap) {
				argsToProcess = append(argsToProcess, argName)
			}
		}
		for _, fullArgName := range argsToProcess {
			srcName, ok := varMap[fullArgName]
			if !ok {
				continue
			}
			fullArgName = strings.TrimPrefix(fullArgName, "!")
			argTokens := strings.Split(fullArgName, ".")
			argName := argTokens[0]
			argVarName := strings.Join(argTokens, "")
			arg := r.argByName(method, argName)
			if arg == nil {
				continue
			}
			srcName = strings.TrimPrefix(srcName, "!")

			var typ *model.Type
			var fieldTypeID string
			var argTypeName string
			var fieldMapKey *model.TypeRef
			var fieldIsSlice bool
			var fieldMapValue *model.TypeRef
			var fieldElementPointers int
			var fieldNumberOfPointers int

			// Если это вложенное поле (например, "data.name"), используем данные из StructField напрямую
			if len(argTokens) > 1 {
				argType, ok := r.project.Types[arg.TypeID]
				if !ok {
					continue
				}
				if argType.Kind == model.TypeKindAlias && argType.AliasOf != "" {
					if baseType, ok := r.project.Types[argType.AliasOf]; ok {
						argType = baseType
					}
				}
				currentType := argType
				var field *model.StructField
				for i := 1; i < len(argTokens); i++ {
					for _, f := range currentType.StructFields {
						if f.Name == argTokens[i] {
							field = f
							break
						}
					}
					if field == nil {
						break
					}
					// Если это не последний элемент пути, переходим к следующему типу
					if i+1 < len(argTokens) {
						nextType, ok := r.project.Types[field.TypeID]
						if !ok {
							field = nil
							break
						}
						if nextType.Kind == model.TypeKindAlias && nextType.AliasOf != "" {
							if baseType, ok := r.project.Types[nextType.AliasOf]; ok {
								currentType = baseType
							} else {
								field = nil
								break
							}
						} else {
							currentType = nextType
						}
					}
				}
				if field == nil {
					continue
				}
				fieldTypeID = field.TypeID
				fieldIsSlice = field.IsSlice
				fieldElementPointers = field.ElementPointers
				fieldNumberOfPointers = field.NumberOfPointers
				fieldMapKey = field.MapKey
				fieldMapValue = field.MapValue
			} else {
				fieldTypeID = arg.TypeID
				fieldIsSlice = arg.IsSlice
				fieldElementPointers = arg.ElementPointers
				fieldNumberOfPointers = arg.NumberOfPointers
				fieldMapKey = arg.MapKey
				fieldMapValue = arg.MapValue
			}

			switch {
			case fieldIsSlice:
				// Для слайсов TypeID уже содержит тип элемента без префикса []
				elementTypeID := fieldTypeID
				if converter.IsBuiltinTypeID(elementTypeID) {
					argTypeName = "[]" + elementTypeID
				} else {
					var ok bool
					typ, ok = r.project.Types[elementTypeID]
					if !ok {
						continue
					}
					argTypeName = "[]" + typ.TypeName
					if typ.TypeName == "" {
						argTypeName = "[]" + string(typ.Kind)
					}
				}
			case converter.IsBuiltinTypeID(fieldTypeID):
				argTypeName = fieldTypeID
			default:
				var ok bool
				typ, ok = r.project.Types[fieldTypeID]
				if !ok {
					continue
				}
				argTypeName = typ.TypeName
				if argTypeName == "" {
					argTypeName = string(typ.Kind)
				}
			}

			rawVarNameOrdered := "_" + argName + "_"
			fieldVarOrdered := &model.Variable{
				TypeRef: model.TypeRef{
					TypeID:           fieldTypeID,
					IsSlice:          fieldIsSlice,
					ElementPointers:  fieldElementPointers,
					NumberOfPointers: fieldNumberOfPointers,
					MapKey:           fieldMapKey,
					MapValue:         fieldMapValue,
				},
			}

			if fieldTypeID == "string" && !fieldIsSlice {
				block.If(Id(rawVarNameOrdered).Op(":=").Add(srcCode(srcName)).Op(";").Id(rawVarNameOrdered).Op("!=").Lit("")).
					BlockFunc(func(bg *Group) {
						var reqID *Statement
						if len(argTokens) == 1 {
							reqID = getTarget(arg)
						} else {
							reqID = Id("request").Dot(r.requestStructFieldName(method, arg))
							for _, token := range argTokens[1:] {
								reqID = reqID.Dot(toCamel(token))
							}
						}
						if fieldNumberOfPointers == 0 {
							bg.Add(reqID.Op("=").Id(rawVarNameOrdered))
						} else {
							bg.Add(reqID.Op("=").Op("&").Id(rawVarNameOrdered))
						}
					}).Line()
				continue
			}

			typeForArgOrdered, ok := r.project.Types[arg.TypeID]
			useConverterTargetOrdered := ok && typeForArgOrdered != nil && (typeForArgOrdered.ParseFromString != nil || converter.HasBuiltinScalarBase(r.project, arg.TypeID))

			block.If(Id(rawVarNameOrdered).Op(":=").Add(srcCode(srcName)).Op(";").Id(rawVarNameOrdered).Op("!=").Lit("")).
				BlockFunc(func(bg *Group) {
					var reqID *Statement
					if len(argTokens) == 1 {
						reqID = getTarget(arg)
					} else {
						reqID = Id("request").Dot(r.requestStructFieldName(method, arg))
						for _, token := range argTokens[1:] {
							reqID = reqID.Dot(toCamel(token))
						}
					}
					if useConverterTargetOrdered {
						bg.Add(r.argToTypeConverter(srcFile, typeGen, Id(rawVarNameOrdered), fieldVarOrdered, reqID, errBody(argVarName, srcName)))
					} else {
						if typ != nil && typ.ImportPkgPath != "" {
							typeParts := strings.Split(argTypeName, ".")
							if len(typeParts) > 1 {
								bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, typeParts[1])
							} else {
								bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, argTypeName)
							}
						} else {
							bg.Var().Id(argVarName).Id(argTypeName)
						}
						bg.Add(r.argToTypeConverter(srcFile, typeGen, Id(rawVarNameOrdered), fieldVarOrdered, Id(argVarName), errBody(argVarName, srcName)))
						argID := Id(argVarName)
						if fieldNumberOfPointers > 0 {
							argID = Op("&").Add(argID)
						}
						bg.Add(reqID.Op("=").Add(argID))
					}
				}).Line()
		}
	}
	return block
}

func (r *contractRenderer) argToTypeConverter(srcFile *GoFile, typeGen *types.Generator, from *Statement, arg *model.Variable, id *Statement, errBody []Code) *Statement {

	return converter.BuildStringToType(converter.StringToTypeConfig{
		Project:        r.project,
		From:           from,
		Arg:            arg,
		Id:             id,
		ErrBody:        errBody,
		OptionalAssign: false,
		FieldType:      typeGen.FieldType,
		AddImport:      srcFile.ImportName,
		JSONPkg:        r.getPackageJSON(),
	})
}

func containsString(slice []string, s string) (ok bool) {

	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func toIDWithImport(qualifiedName string, srcFile *GoFile) (stmt *Statement) {

	// Формат: "package/path:FunctionName"
	if tokens := strings.Split(qualifiedName, ":"); len(tokens) == 2 {
		pkgPath := tokens[0]
		funcName := tokens[1]
		baseName := filepath.Base(pkgPath)
		srcFile.ImportName(pkgPath, baseName)
		return Qual(pkgPath, funcName)
	}
	// Если формат неверный, возвращаем как есть
	return Id(qualifiedName)
}

func (r *contractRenderer) arguments(method *model.Method) (out []*model.Variable) {

	if method == nil {
		return nil
	}
	return r.argsWithoutSpecialArgs(method)
}

func (r *contractRenderer) urlArgs(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errBody func(arg, header string) []Code) *Statement {
	return r.argFromString(srcFile, typeGen, method, "urlParam", r.argPathMap(method),
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Params").Call(Lit(srcName))
		},
		errBody,
		func(arg *model.Variable) *Statement {
			return Id("request").Dot(r.requestStructFieldName(method, arg))
		},
	)
}

func (r *contractRenderer) urlParams(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errBody func(arg, header string) []Code) *Statement {

	pathMap := r.argPathMap(method)
	headerMap := r.varHeaderMap(method)
	cookieMap := r.varCookieMap(method)
	queryParams := make(map[string]string)
	var orderedArgs []string
	for _, it := range model.ParseArgMapEntries(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpArg, "")) {
		if it.Arg == "path" {
			continue
		}
		if _, inPath := pathMap[it.Arg]; inPath {
			continue
		}
		if _, inHeader := headerMap[it.Arg]; inHeader {
			continue
		}
		if _, inCookie := cookieMap[it.Arg]; inCookie {
			continue
		}
		queryParams[it.Arg] = it.Key
		orderedArgs = append(orderedArgs, it.Arg)
	}
	return r.argFromStringOrdered(srcFile, typeGen, method, "queryParam", queryParams, orderedArgs,
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Query").Call(Lit(srcName))
		},
		errBody,
		func(arg *model.Variable) *Statement {
			return Id("request").Dot(r.requestStructFieldName(method, arg))
		},
	)
}

func (r *contractRenderer) httpArgHeaders(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errBody func(arg, header string) []Code) *Statement {
	return r.argFromString(srcFile, typeGen, method, "header", r.varHeaderMapForRequest(method),
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Get").Call(Lit(srcName))
		},
		errBody,
		func(arg *model.Variable) *Statement {
			return Id("request").Dot(r.requestStructFieldName(method, arg))
		},
	)
}

func (r *contractRenderer) applyOverlayFromContext(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errBody func(arg, header string) []Code, overlayAsStruct bool) *Statement {

	headerMap := r.varHeaderMapForRequest(method)
	cookieMap := r.varCookieMapForRequest(method)
	if len(headerMap) == 0 && len(cookieMap) == 0 {
		return Line()
	}
	hasValidArgs := false
	for fullArgName := range headerMap {
		argName := strings.Split(strings.TrimPrefix(fullArgName, "!"), ".")[0]
		if r.argByName(method, argName) != nil {
			hasValidArgs = true
			break
		}
	}
	if !hasValidArgs {
		for fullArgName := range cookieMap {
			argName := strings.Split(strings.TrimPrefix(fullArgName, "!"), ".")[0]
			if r.argByName(method, argName) != nil {
				hasValidArgs = true
				break
			}
		}
	}
	if !hasValidArgs {
		return Line()
	}
	var overlayFromMap func(srcName string) Code
	if overlayAsStruct {
		overlayFromMap = func(srcName string) Code {
			return Id("overlay").Dot("Get").Call(Lit(srcName))
		}
	} else {
		overlayFromMap = func(srcName string) Code {
			return Id("overlay").Index(Lit(srcName))
		}
	}
	inner := Line()
	getTarget := func(arg *model.Variable) *Statement {
		return Id("request").Dot(r.requestStructFieldName(method, arg))
	}
	inner.Add(r.argFromString(srcFile, typeGen, method, "header", headerMap, overlayFromMap, errBody, getTarget))
	inner.Add(r.argFromString(srcFile, typeGen, method, "cookie", cookieMap, overlayFromMap, errBody, getTarget))
	var block *Statement
	if overlayAsStruct {
		block = Id("getterVal").Op(":=").Id(VarNameCtx).Dot("Value").Call(Id("keyRequestOverlay")).
			Line().If(Id("getterVal").Op("!=").Nil()).Block(
			Id("getter").Op(":=").Id("getterVal").Assert(Id("requestOverlayGetter")),
			Id("overlay").Op(":=").Id("getter").Call(),
			inner,
		)
	} else {
		assertBlock := If(List(Id("overlay"), Id("ok")).Op(":=").Id("overlayVal").Assert(Map(String()).String()).Op(";").Id("ok")).Block(inner)
		block = Id("overlayVal").Op(":=").Id(VarNameCtx).Dot("Value").Call(Id("keyRequestOverlay")).
			Line().If(Id("overlayVal").Op("!=").Nil()).Block(assertBlock)
	}
	return Line().Add(block)
}

func (r *contractRenderer) httpCookies(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errBody func(arg, header string) []Code) *Statement {
	return r.argFromString(srcFile, typeGen, method, "cookie", r.varCookieMapForRequest(method),
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Cookies").Call(Lit(srcName))
		},
		errBody,
		func(arg *model.Variable) *Statement {
			return Id("request").Dot(r.requestStructFieldName(method, arg))
		},
	)
}

func (r *contractRenderer) resultToHeaderStringExpr(ret *model.Variable, valueExpr Code) Code {

	deref := func(expr Code) Code { return Op("*").Add(expr) }
	if ret.TypeID == "string" {
		if ret.NumberOfPointers > 0 {
			return deref(valueExpr)
		}
		return valueExpr
	}
	if converter.IsBuiltinTypeID(ret.TypeID) {
		if ret.NumberOfPointers > 0 {
			return Qual(PackageFmt, "Sprint").Call(deref(valueExpr))
		}
		return Qual(PackageFmt, "Sprint").Call(valueExpr)
	}
	typ, ok := r.project.Types[ret.TypeID]
	if ok && (containsString(typ.ImplementsInterfaces, "fmt:Stringer") ||
		(strings.Contains(typ.ImportPkgPath, "time") && (typ.TypeName == "Duration" || typ.TypeName == "Time")) ||
		(strings.Contains(typ.ImportPkgPath, "uuid") && typ.TypeName == "UUID")) {
		return (&Statement{}).Add(valueExpr).Dot("String").Call()
	}
	if ret.NumberOfPointers > 0 {
		return Qual(PackageFmt, "Sprint").Call(deref(valueExpr))
	}
	return Qual(PackageFmt, "Sprint").Call(valueExpr)
}

func (r *contractRenderer) httpRetHeaders(method *model.Method) *Statement {

	ex := Line()
	headerMap := r.varHeaderMap(method)
	for varName, headerName := range common.SortedPairs(headerMap) {
		ret := r.resultByName(method, varName)
		if ret == nil {
			continue
		}
		fieldName := r.responseStructFieldName(method, ret)
		valueExpr := Id("response").Dot(fieldName)
		setHeaderValue := func(expr Code) Code {
			strExpr := r.resultToHeaderStringExpr(ret, expr)
			if headerName == "Content-Disposition" {
				// RFC 2183: браузеры ожидают attachment; filename="..."
				return Qual(PackageFmt, "Sprintf").Call(Lit("attachment; filename=%q"), strExpr)
			}
			return strExpr
		}
		if ret.NumberOfPointers > 0 {
			ex.If(valueExpr.Clone().Op("!=").Nil()).Block(
				Id(VarNameFtx).Dot("Set").Call(Lit(headerName), setHeaderValue(valueExpr.Clone())),
			)
		} else {
			ex.Id(VarNameFtx).Dot("Set").Call(Lit(headerName), setHeaderValue(valueExpr))
		}
	}
	return ex
}

func (r *contractRenderer) httpRetCookies(method *model.Method) *Statement {

	ex := Line()
	for retName, cookieName := range common.SortedPairs(r.retCookieMap(method)) {
		ret := r.resultByName(method, retName)
		if ret == nil {
			continue
		}
		fieldName := r.responseStructFieldName(method, ret)
		valueExpr := Id("response").Dot(fieldName)
		cookieTypeBlock := If(List(Id("rCookie"), Id("ok")).Op(":=").
			Qual(PackageReflect, "ValueOf").Call(valueExpr.Clone()).Dot("Interface").Call().
			Op(".").Call(Id("cookieType")).Op(";").Id("ok")).Block(
			Id("cookie").Op(":=").Id("rCookie").Dot("Cookie").Call(),
			Id(VarNameFtx).Dot("Cookie").Call(Op("&").Id("cookie")),
		)
		simpleCookieStmts := func(vExpr Code) []Code {
			return []Code{
				Id("cookie").Op(":=").Qual(PackageFiber, "Cookie").Values(Dict{
					Id("Name"):  Lit(cookieName),
					Id("Value"): r.resultToHeaderStringExpr(ret, vExpr),
				}),
				Id(VarNameFtx).Dot("Cookie").Call(Op("&").Id("cookie")),
			}
		}
		if ret.NumberOfPointers > 0 {
			cookieTypeBlock.Else().Block(
				If(valueExpr.Clone().Op("!=").Nil()).Block(simpleCookieStmts(valueExpr.Clone())...),
			)
		} else {
			cookieTypeBlock.Else().Block(simpleCookieStmts(valueExpr)...)
		}
		ex.Add(cookieTypeBlock)
	}
	return ex
}
