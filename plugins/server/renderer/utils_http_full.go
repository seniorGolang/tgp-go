// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) varHeaderMap(method *model.Method) map[string]string {

	headers := make(map[string]string)
	if httpHeaders := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHttpHeader, ""); httpHeaders != "" {
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

func (r *contractRenderer) varCookieMap(method *model.Method) map[string]string {

	cookies := make(map[string]string)
	if httpCookies := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHttpCookies, ""); httpCookies != "" {
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

func usedHeaderNamesForMethod(project *model.Project, contract *model.Contract, method *model.Method) []string {

	if project == nil || contract == nil || method == nil {
		return nil
	}
	headers := make(map[string]struct{})
	if httpHeaders := model.GetAnnotationValue(project, contract, method, nil, TagHttpHeader, ""); httpHeaders != "" {
		for _, pair := range strings.Split(httpHeaders, ",") {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				header := strings.TrimSpace(pairTokens[1])
				if header != "" {
					headers[header] = struct{}{}
				}
			}
		}
	}
	return common.SortedKeys(headers)
}

func usedCookieNamesForMethod(project *model.Project, contract *model.Contract, method *model.Method) []string {

	if project == nil || contract == nil || method == nil {
		return nil
	}
	cookies := make(map[string]struct{})
	if httpCookies := model.GetAnnotationValue(project, contract, method, nil, TagHttpCookies, ""); httpCookies != "" {
		for _, pair := range strings.Split(httpCookies, ",") {
			if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
				cookie := strings.TrimSpace(pairTokens[1])
				if cookie != "" {
					cookies[cookie] = struct{}{}
				}
			}
		}
	}
	return common.SortedKeys(cookies)
}

func (r *contractRenderer) argPathMap(method *model.Method) map[string]string {

	paths := make(map[string]string)
	if urlPath := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHttpPath, ""); urlPath != "" {
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

func (r *contractRenderer) argParamMap(method *model.Method) map[string]string {

	params := make(map[string]string)
	if urlArgs := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHttpArg, ""); urlArgs != "" {
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

func (r *contractRenderer) argByName(method *model.Method, argName string) *model.Variable {

	argName = strings.TrimPrefix(argName, "!")
	for _, arg := range method.Args {
		if arg.Name == argName {
			return arg
		}
	}
	return nil
}

func (r *contractRenderer) resultByName(method *model.Method, retName string) *model.Variable {

	for _, ret := range method.Results {
		if ret.Name == retName {
			return ret
		}
	}
	return nil
}

func (r *contractRenderer) retCookieMap(method *model.Method) map[string]string {

	cookies := make(map[string]string)
	cookieMap := r.varCookieMap(method)
	for varName, cookieName := range common.SortedPairs(cookieMap) {
		if r.resultByName(method, varName) != nil {
			cookies[varName] = cookieName
		}
	}
	return cookies
}

func (r *contractRenderer) argsWithoutSpecialArgs(method *model.Method) []*model.Variable {

	vars := make([]*model.Variable, 0)
	argsAll := argsWithoutContext(method)
	pathMap := r.argPathMap(method)
	paramMap := r.argParamMap(method)
	headerMap := r.varHeaderMap(method)
	cookieMap := r.varCookieMap(method)

	for _, arg := range argsAll {
		_, inPath := pathMap[arg.Name]
		_, inArgs := paramMap[arg.Name]
		_, inHeader := headerMap[arg.Name]
		_, inCookie := cookieMap[arg.Name]

		if !inArgs && !inPath && !inHeader && !inCookie {
			vars = append(vars, arg)
		}
	}
	return vars
}

func (r *contractRenderer) methodRequestBodyStreamArg(method *model.Method) *model.Variable {

	for _, arg := range argsWithoutContext(method) {
		if arg.TypeID == TypeIDIOReader {
			return arg
		}
	}
	return nil
}

func (r *contractRenderer) methodResponseBodyStreamResult(method *model.Method) *model.Variable {

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
	if len(readerArgs) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, TagHttpMultipart) {
		return true
	}
	return false
}

func (r *contractRenderer) methodResponseMultipart(method *model.Method) bool {

	readCloserResults := r.methodResponseBodyStreamResults(method)
	if len(readCloserResults) > 1 {
		return true
	}
	if len(readCloserResults) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, TagHttpMultipart) {
		return true
	}
	return false
}

func (r *contractRenderer) streamPartName(method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[TagHttpPartName]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[TagHttpPartName]; found && val != "" {
			if partName := r.varValueFromMethodMap(val, v.Name); partName != "" {
				return partName
			}
		}
	}
	return v.Name
}

func (r *contractRenderer) streamPartContent(method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[TagHttpPartContent]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[TagHttpPartContent]; found && val != "" {
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

func isBuiltinTypeID(typeID string) bool {

	switch typeID {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "error", "any":
		return true
	}
	return false
}

func (r *contractRenderer) argFromString(srcFile *GoFile, typeGen *types.Generator, method *model.Method, typeName string, varMap map[string]string, srcCode func(srcName string) Code, errStatement func(arg, header string) *Statement) *Statement {

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

			var argTypeName string
			var typ *model.Type
			var fieldTypeID string
			var fieldIsSlice bool
			var fieldElementPointers int
			var fieldNumberOfPointers int
			var fieldMapKey *model.TypeRef
			var fieldMapValue *model.TypeRef

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
				if isBuiltinTypeID(elementTypeID) {
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
			case isBuiltinTypeID(fieldTypeID):
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

			argID := Id(argVarName)

			// Обработка указателей (если нужно)
			if fieldNumberOfPointers > 0 {
				argID = Op("&").Add(argID)
			}

			// Всегда используем проверку на пустоту, как в эталонной реализации
			block.If(Id("_" + argVarName).Op(":=").Add(srcCode(srcName)).Op(";").Id("_" + argVarName).Op("!=").Lit("")).
				BlockFunc(func(bg *Group) {
					// Объявляем переменную нужного типа
					if typ != nil && typ.ImportPkgPath != "" {
						// Если тип импортирован, используем полное имя (может содержать точку)
						typeParts := strings.Split(argTypeName, ".")
						if len(typeParts) > 1 {
							bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, typeParts[1])
						} else {
							bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, argTypeName)
						}
					} else {
						// Встроенный или локальный тип
						bg.Var().Id(argVarName).Id(argTypeName)
					}

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
					bg.Add(r.argToTypeConverter(srcFile, typeGen, Id("_"+argVarName), fieldVar, Id(argVarName), errStatement(argVarName, srcName)))

					// Присваиваем значение в request
					reqID := bg.Id("request").Dot(toCamel(argName))
					if len(argTokens) > 1 {
						for _, token := range argTokens[1:] {
							reqID = reqID.Dot(toCamel(token))
						}
						reqID.Op("=").Add(argID)
					} else {
						reqID.Op("=").Add(argID)
					}
				}).Line()
		}
	}
	return block
}

func (r *contractRenderer) argFromStringOrdered(srcFile *GoFile, typeGen *types.Generator, method *model.Method, typeName string, varMap map[string]string, orderedArgs []string, srcCode func(srcName string) Code, errStatement func(arg, header string) *Statement) *Statement {

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

			var argTypeName string
			var typ *model.Type
			var fieldTypeID string
			var fieldIsSlice bool
			var fieldElementPointers int
			var fieldNumberOfPointers int
			var fieldMapKey *model.TypeRef
			var fieldMapValue *model.TypeRef

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
				if isBuiltinTypeID(elementTypeID) {
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
			case isBuiltinTypeID(fieldTypeID):
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

			argID := Id(argVarName)

			// Обработка указателей (если нужно)
			if fieldNumberOfPointers > 0 {
				argID = Op("&").Add(argID)
			}

			// Всегда используем проверку на пустоту, как в эталонной реализации
			block.If(Id("_" + argVarName).Op(":=").Add(srcCode(srcName)).Op(";").Id("_" + argVarName).Op("!=").Lit("")).
				BlockFunc(func(bg *Group) {
					// Объявляем переменную нужного типа
					if typ != nil && typ.ImportPkgPath != "" {
						// Если тип импортирован, используем полное имя (может содержать точку)
						typeParts := strings.Split(argTypeName, ".")
						if len(typeParts) > 1 {
							bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, typeParts[1])
						} else {
							bg.Var().Id(argVarName).Qual(typ.ImportPkgPath, argTypeName)
						}
					} else {
						// Встроенный или локальный тип
						bg.Var().Id(argVarName).Id(argTypeName)
					}

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
					bg.Add(r.argToTypeConverter(srcFile, typeGen, Id("_"+argVarName), fieldVar, Id(argVarName), errStatement(argVarName, srcName)))

					// Присваиваем значение в request
					reqID := bg.Id("request").Dot(toCamel(argName))
					if len(argTokens) > 1 {
						for _, token := range argTokens[1:] {
							reqID = reqID.Dot(toCamel(token))
						}
						reqID.Op("=").Add(argID)
					} else {
						reqID.Op("=").Add(argID)
					}
				}).Line()
		}
	}
	return block
}

func (r *contractRenderer) argToTypeConverter(srcFile *GoFile, typeGen *types.Generator, from *Statement, arg *model.Variable, id *Statement, errStatement *Statement) *Statement {

	op := "="

	// Обработка слайсов
	if arg.IsSlice {
		// Для слайсов парсим через strings.Split и конвертируем каждый элемент
		// TypeID уже содержит тип элемента без префикса []
		elementTypeID := arg.TypeID
		// Создаем переменную для элемента
		elementVar := &model.Variable{
			TypeRef: model.TypeRef{
				TypeID:           elementTypeID,
				NumberOfPointers: arg.ElementPointers,
				IsSlice:          false,
			},
			Name: "elem",
		}
		srcFile.ImportName(PackageStrings, "strings")
		elementTypeCode := typeGen.FieldType(elementTypeID, arg.ElementPointers, false)
		return BlockFunc(func(bg *Group) {
			bg.Id("parts").Op(":=").Qual(PackageStrings, "Split").Call(from, Lit(","))
			bg.Id("result").Op(":=").Make(Index().Add(elementTypeCode), Lit(0), Len(Id("parts")))
			bg.For(List(Id("_"), Id("elemStr")).Op(":=").Range().Id("parts")).BlockFunc(func(ig *Group) {
				ig.Id("elemStr").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("elemStr"))
				ig.If(Id("elemStr").Op("==").Lit("")).Block(
					Continue(),
				)
				ig.Var().Id("elem").Add(elementTypeCode)
				ig.Add(r.argToTypeConverter(srcFile, typeGen, Id("elemStr"), elementVar, Id("elem"), errStatement))
				ig.Id("result").Op("=").Append(Id("result"), Id("elem"))
			})
			bg.Add(id).Op("=").Id("result")
		})
	}

	// Для встроенных типов TypeID - это имя типа, они не в project.Types
	if isBuiltinTypeID(arg.TypeID) {
		switch arg.TypeID {
		case "string":
			return id.Op(op).Add(from)
		case "bool":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "ParseBool").Call(from).Add(errStatement)
		case "int":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "Atoi").Call(from).Add(errStatement)
		case "int64":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "ParseInt").Call(from, Lit(10), Lit(64)).Add(errStatement)
		case "int32":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "ParseInt").Call(from, Lit(10), Lit(32)).Add(errStatement)
		case "uint":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "ParseUint").Call(from, Lit(10), Lit(64)).Add(errStatement)
		case "uint64":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "ParseUint").Call(from, Lit(10), Lit(64)).Add(errStatement)
		case "uint32":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "ParseUint").Call(from, Lit(10), Lit(32)).Add(errStatement)
		case "float64":
			return List(id, Err()).Op(op).Qual(PackageStrconv, "ParseFloat").Call(from, Lit(64)).Add(errStatement)
		case "float32":
			temp64 := Id("temp64")
			return List(temp64, Err()).Op(":=").Qual(PackageStrconv, "ParseFloat").Call(from, Lit(32)).Add(errStatement).Add(Line()).Add(id.Op(op).Float32().Call(temp64))
		default:
			// Для остальных встроенных типов (byte, rune, error, any) используем прямое преобразование
			return id.Op(op).Add(from)
		}
	}

	// Пользовательский тип: ищем в project.Types
	typ, ok := r.project.Types[arg.TypeID]
	if !ok {
		// Если тип не найден, используем прямое преобразование
		return id.Op(op).Add(from)
	}

	openAPIType, format := getSerializationFormat(typ, r.project)

	switch {
	case openAPIType == "string" && format == "uuid":
		// UUID - парсим через uuid.Parse
		uuidPackage := PackageUUID
		return List(id, Id("_")).Op(op).Qual(uuidPackage, "Parse").Call(from)
	case openAPIType == "string" && format == "date-time":
		// time.Time - парсим через time.Parse
		return List(id, Err()).Op(op).Qual(PackageTime, "Parse").Call(Qual(PackageTime, "RFC3339Nano"), from).Add(errStatement)
	case openAPIType == "string" && containsString(typ.ImplementsInterfaces, "encoding/json:Marshaler"):
		// Типы, реализующие json.Marshaler, парсим через JSON unmarshal
		jsonPkg := r.getPackageJSON()
		srcFile.ImportName(jsonPkg, "json")
		return Op("_").Op("=").Qual(jsonPkg, "Unmarshal").Call(Op("[]").Byte().Call(Op("`\"`").Op("+").Add(from).Op("+").Op("`\"`")), Op("&").Add(id))
	default:
		// Для остальных сложных типов используем JSON unmarshal
		jsonPkg := r.getPackageJSON()
		srcFile.ImportName(jsonPkg, "json")
		return Op("_").Op("=").Qual(jsonPkg, "Unmarshal").Call(Op("[]").Byte().Call(Op("`\"`").Op("+").Add(from).Op("+").Op("`\"`")), Op("&").Add(id))
	}
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func toIDWithImport(qualifiedName string, srcFile *GoFile) *Statement {
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

func (r *contractRenderer) arguments(method *model.Method) []*model.Variable {
	return r.argsWithoutSpecialArgs(method)
}

func (r *contractRenderer) urlArgs(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errStatement func(arg, header string) *Statement) *Statement {
	return r.argFromString(srcFile, typeGen, method, "urlParam", r.argPathMap(method),
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Params").Call(Lit(srcName))
		},
		errStatement,
	)
}

func (r *contractRenderer) urlParams(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errStatement func(arg, header string) *Statement) *Statement {
	queryParams := make(map[string]string)
	if urlArgs := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHttpArg, ""); urlArgs != "" {
		paramPairs := strings.Split(urlArgs, ",")
		for _, pair := range paramPairs {
			pair = strings.TrimSpace(pair)
			if strings.Contains(pair, "|") && !strings.HasPrefix(pair, "path|") && !strings.HasPrefix(pair, "header|") && !strings.HasPrefix(pair, "cookie|") {
				if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
					arg := strings.TrimSpace(pairTokens[0])
					param := strings.TrimSpace(pairTokens[1])
					if _, inPath := r.argPathMap(method)[arg]; !inPath {
						if _, inHeader := r.varHeaderMap(method)[arg]; !inHeader {
							if _, inCookie := r.varCookieMap(method)[arg]; !inCookie {
								queryParams[arg] = param
							}
						}
					}
				}
			}
		}
	}
	var orderedArgs []string
	if urlArgs := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHttpArg, ""); urlArgs != "" {
		paramPairs := strings.Split(urlArgs, ",")
		for _, pair := range paramPairs {
			pair = strings.TrimSpace(pair)
			if strings.Contains(pair, "|") && !strings.HasPrefix(pair, "path|") && !strings.HasPrefix(pair, "header|") && !strings.HasPrefix(pair, "cookie|") {
				if pairTokens := strings.Split(pair, "|"); len(pairTokens) == 2 {
					arg := strings.TrimSpace(pairTokens[0])
					if _, ok := queryParams[arg]; ok {
						orderedArgs = append(orderedArgs, arg)
					}
				}
			}
		}
	}
	return r.argFromStringOrdered(srcFile, typeGen, method, "queryParam", queryParams, orderedArgs,
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Query").Call(Lit(srcName))
		},
		errStatement,
	)
}

func (r *contractRenderer) httpArgHeaders(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errStatement func(arg, header string) *Statement) *Statement {
	return r.argFromString(srcFile, typeGen, method, "header", r.varHeaderMap(method),
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Get").Call(Lit(srcName))
		},
		errStatement,
	)
}

func (r *contractRenderer) applyOverlayFromContext(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errStatement func(arg, header string) *Statement, overlayAsStruct bool) *Statement {

	headerMap := r.varHeaderMap(method)
	cookieMap := r.varCookieMap(method)
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
	inner.Add(r.argFromString(srcFile, typeGen, method, "header", headerMap, overlayFromMap, errStatement))
	inner.Add(r.argFromString(srcFile, typeGen, method, "cookie", cookieMap, overlayFromMap, errStatement))
	var assertBlock *Statement
	if overlayAsStruct {
		assertBlock = If(List(Id("overlay"), Id("ok")).Op(":=").Id("overlayVal").Assert(Id("requestOverlay")).Op(";").Id("ok")).Block(inner)
	} else {
		assertBlock = If(List(Id("overlay"), Id("ok")).Op(":=").Id("overlayVal").Assert(Map(String()).String()).Op(";").Id("ok")).Block(inner)
	}
	return Line().
		Id("overlayVal").Op(":=").Id(VarNameCtx).Dot("Value").Call(Id("keyRequestOverlay")).
		Line().If(Id("overlayVal").Op("!=").Nil()).Block(assertBlock)
}

func (r *contractRenderer) httpCookies(srcFile *GoFile, typeGen *types.Generator, method *model.Method, errStatement func(arg, header string) *Statement) *Statement {
	return r.argFromString(srcFile, typeGen, method, "cookie", r.varCookieMap(method),
		func(srcName string) Code {
			return Id(VarNameFtx).Dot("Cookies").Call(Lit(srcName))
		},
		errStatement,
	)
}

func (r *contractRenderer) httpRetHeaders(method *model.Method) *Statement {
	ex := Line()
	headerMap := r.varHeaderMap(method)
	for varName, headerName := range common.SortedPairs(headerMap) {
		if ret := r.resultByName(method, varName); ret != nil {
			ex.Id(VarNameFtx).Dot("Set").Call(Lit(headerName), Id("response").Dot(toCamel(varName)))
		}
	}
	return ex
}

func getSerializationFormat(typ *model.Type, project *model.Project) (openAPIType string, format string) {

	if typ.ImportPkgPath == "time" && typ.TypeName == "Time" {
		return "string", "date-time"
	}

	if strings.Contains(typ.TypeName, "UUID") || strings.Contains(typ.ImportPkgPath, "uuid") {
		return "string", "uuid"
	}

	if containsString(typ.ImplementsInterfaces, "encoding/json:Marshaler") {
		return "string", ""
	}

	// Для остальных типов возвращаем пустые значения
	return "", ""
}
