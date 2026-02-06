// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"strconv"
	"strings"

	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/plugins/client-ts/tsg"
)

type errorInfo struct {
	code     int    // HTTP статус код (0 если не указан)
	codeText string // Текстовое описание статус кода
	pkgPath  string // Путь пакета
	typeName string // Имя типа ошибки
}

func (r *ClientRenderer) collectMethodErrors(method *model.Method, contract *model.Contract) map[string]errorInfo {

	errorsMap := make(map[string]errorInfo)

	// 1. Ищем ошибки из аннотаций (@tg 401=package:TypeName)
	for key, value := range common.SortedPairs(method.Annotations) {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Пытаемся преобразовать ключ в число (HTTP статус код)
		if code, err := strconv.Atoi(key); err == nil {
			if code >= 400 && code < 600 {
				if value != "" && value != "skip" {
					// Парсим значение формата "package:TypeName"
					if tokens := strings.Split(value, ":"); len(tokens) == 2 {
						pkgPath := tokens[0]
						typeName := tokens[1]
						key := fmt.Sprintf("%s:%s", pkgPath, typeName)

						codeText := r.getHTTPStatusText(code)
						errorsMap[key] = errorInfo{
							code:     code,
							codeText: codeText,
							pkgPath:  pkgPath,
							typeName: typeName,
						}
					}
				}
			}
		}
	}

	// 2. Используем ошибки из shared.Method.Errors (уже проанализированные)
	for _, errInfo := range method.Errors {
		key := fmt.Sprintf("%s:%s", errInfo.PkgPath, errInfo.TypeName)
		if _, exists := errorsMap[key]; !exists {
			errorsMap[key] = errorInfo{
				code:     errInfo.HTTPCode,
				codeText: errInfo.HTTPCodeText,
				pkgPath:  errInfo.PkgPath,
				typeName: errInfo.TypeName,
			}
		}
	}

	return errorsMap
}

func (r *ClientRenderer) renderErrorType(errInfo errorInfo) *tsg.Statement {

	stmt := tsg.NewStatement()

	var structType *model.Type
	typeID := fmt.Sprintf("%s:%s", errInfo.pkgPath, errInfo.typeName)
	if typ, ok := r.project.Types[typeID]; ok && typ.Kind == model.TypeKindStruct {
		structType = typ
	}

	// Если не нашли по полному пути, пробуем найти по имени типа
	if structType == nil {
		for _, typ := range common.SortedPairs(r.project.Types) {
			if typ.Kind == model.TypeKindStruct && typ.TypeName == errInfo.typeName {
				structType = typ
				break
			}
		}
	}

	pkgParts := strings.Split(errInfo.pkgPath, "/")
	pkgName := pkgParts[len(pkgParts)-1]
	typeName := fmt.Sprintf("%s%s", pkgName, errInfo.typeName)

	stmt.Comment(fmt.Sprintf("Error type: %s.%s", pkgName, errInfo.typeName))
	if errInfo.code != 0 {
		stmt.Comment(fmt.Sprintf("HTTP status code: %d (%s)", errInfo.code, errInfo.codeText))
	}

	stmt.Export().Interface(typeName, func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().Id("message").Colon().Id("string").Semicolon())

		if errInfo.code != 0 {
			grp.Add(tsg.NewStatement().Id("code").Colon().Lit(errInfo.code).Semicolon())
		} else {
			grp.Add(tsg.NewStatement().Id("code").Colon().Id("number").Semicolon())
		}

		if structType != nil && structType.StructFields != nil {
			for _, field := range structType.StructFields {
				fieldName, inline := r.jsonName(field)
				if fieldName == "-" || inline {
					continue
				}

				if fieldName == "message" || fieldName == "code" {
					continue
				}

				fieldVar := &model.Variable{
					TypeRef: field.TypeRef,
					Name:    field.Name,
				}
				pkgPath := errInfo.pkgPath
				if structType != nil && structType.ImportPkgPath != "" {
					pkgPath = structType.ImportPkgPath
				}
				// Для полей ошибок используем isArgument=false, так как это возвращаемые значения
				fieldTags := parseTagsFromDocs(field.Docs)
				typeStr := r.walkVariable(field.Name, pkgPath, fieldVar, fieldTags, false).typeLink()

				fieldStmt := tsg.NewStatement()
				fieldStmt.Id(fieldName)

				isOptional := false
				if field.NumberOfPointers > 0 {
					isOptional = true
				} else {
					if tagValues, ok := field.Tags["json"]; ok {
						for _, val := range tagValues {
							if val == "omitempty" {
								isOptional = true
								break
							}
						}
					}
				}

				if isOptional {
					fieldStmt.Optional()
				}

				fieldStmt.Colon()
				fieldStmt.Add(tsg.TypeFromString(typeStr))
				fieldStmt.Semicolon()
				grp.Add(fieldStmt)
			}
		}
	})

	return stmt
}

func (r *ClientRenderer) renderErrorUnionType(methodName string, errorsMap map[string]errorInfo) *tsg.Statement {

	stmt := tsg.NewStatement()

	errorTypes := make([]string, 0, len(errorsMap))
	for _, errInfo := range common.SortedPairs(errorsMap) {
		pkgParts := strings.Split(errInfo.pkgPath, "/")
		pkgName := pkgParts[len(pkgParts)-1]
		typeName := fmt.Sprintf("%s%s", pkgName, errInfo.typeName)
		errorTypes = append(errorTypes, typeName)
	}

	// Если есть ошибки, создаём union тип
	if len(errorTypes) > 0 {
		for i := 0; i < len(errorTypes)-1; i++ {
			for j := i + 1; j < len(errorTypes); j++ {
				if errorTypes[i] > errorTypes[j] {
					errorTypes[i], errorTypes[j] = errorTypes[j], errorTypes[i]
				}
			}
		}

		unionTypeName := fmt.Sprintf("%sError", methodName)
		stmt.Comment(fmt.Sprintf("Union type for possible errors of %s method", methodName))
		stmt.Export().TypeAlias(unionTypeName)

		unionTypes := make([]*tsg.Statement, len(errorTypes))
		for i, errorType := range errorTypes {
			unionTypes[i] = tsg.NewStatement().Id(errorType)
		}
		stmt.Union(unionTypes...)

		stmt.Semicolon()
	}

	return stmt
}

func (r *ClientRenderer) getHTTPStatusText(code int) string {

	statusTexts := map[int]string{
		400: "Bad Request",
		401: "Unauthorized",
		402: "Payment Required",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		406: "Not Acceptable",
		407: "Proxy Authentication Required",
		408: "Request Timeout",
		409: "Conflict",
		410: "Gone",
		411: "Length Required",
		412: "Precondition Failed",
		413: "Request Entity Too Large",
		414: "Request URI Too Long",
		415: "Unsupported Media Type",
		416: "Requested Range Not Satisfiable",
		417: "Expectation Failed",
		418: "I'm a teapot",
		422: "Unprocessable Entity",
		423: "Locked",
		424: "Failed Dependency",
		425: "Upgrade Required",
		426: "Precondition Required",
		429: "Too Many Requests",
		431: "Request Header Fields Too Large",
		451: "Unavailable For Legal Reasons",
		500: "Internal Server Error",
		501: "Not Implemented",
		502: "Bad Gateway",
		503: "Service Unavailable",
		504: "Gateway Timeout",
		505: "HTTP Version Not Supported",
		506: "Variant Also Negotiates",
		507: "Insufficient Storage",
		508: "Loop Detected",
		510: "Not Extended",
		511: "Network Authentication Required",
	}

	if text, found := statusTexts[code]; found {
		return text
	}
	return fmt.Sprintf("HTTP Error %d", code)
}
