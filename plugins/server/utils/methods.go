// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"slices"

	"tgp/internal/model"
)

func IsContextFirst(vars []*model.Variable) bool {

	if len(vars) == 0 {
		return false
	}
	return vars[0].TypeID == "context:Context"
}

func IsErrorLast(vars []*model.Variable) bool {

	if len(vars) == 0 {
		return false
	}
	return vars[len(vars)-1].TypeID == "error"
}

func ArgsWithoutContext(method *model.Method) []*model.Variable {

	if IsContextFirst(method.Args) {
		return method.Args[1:]
	}
	return method.Args
}

func ArgsFieldsWithoutContext(method *model.Method) []*model.Variable {

	argVars := ArgsWithoutContext(method)
	// Inline поля обрабатываются в renderer.ArgsFieldsWithoutContext()
	return argVars
}

func ResultsWithoutError(method *model.Method) []*model.Variable {

	if IsErrorLast(method.Results) {
		return method.Results[:len(method.Results)-1]
	}
	return method.Results
}

func ResultFieldsWithoutError(method *model.Method) []*model.Variable {

	resultVars := ResultsWithoutError(method)
	// Inline поля обрабатываются в renderer.ResultFieldsWithoutError()
	return resultVars
}

func RequestStructName(contractName string, methodName string) string {

	return "request" + contractName + methodName
}

func ResponseStructName(contractName string, methodName string) string {

	return "response" + contractName + methodName
}

func RemoveSkippedFields(vars []*model.Variable, skipFields []string) []*model.Variable {

	var result []*model.Variable
	for _, v := range vars {
		if !slices.Contains(skipFields, v.Name) {
			result = append(result, v)
		}
	}
	return result
}

func Arguments(method *model.Method) []*model.Variable {

	// Эта функция не может быть полностью реализована без доступа к аннотациям метода
	// Полная реализация находится в renderer.arguments()
	return ArgsWithoutContext(method)
}
