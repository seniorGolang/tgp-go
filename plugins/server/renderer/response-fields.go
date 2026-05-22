// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

// responseFieldEmbedded — результат сериализуется как встроенная struct (json:,inline), как в exchange.
func (r *contractRenderer) responseFieldEmbedded(method *model.Method, ret *model.Variable) (ok bool) {

	return model.ResultFieldEmbedded(r.project, r.contract, method, ret)
}

// loggerResultExpr — значение результата svc для composite literal exchange (типы совпадают с полем struct).
func (r *contractRenderer) loggerResultExpr(ret *model.Variable) (code Code) {

	return Id(toLowerCamel(ret.Name))
}

// responseAssignmentTargets — цели присваивания из svc (совпадают с полями response-структуры exchange).
func (r *contractRenderer) responseAssignmentTargets(method *model.Method, responseVar string) (out []Code) {

	results := resultsWithoutError(method)
	if len(results) == 0 {
		return nil
	}
	if len(results) == 1 && r.responseFieldEmbedded(method, results[0]) {
		embedName := model.TypeNameFromTypeID(r.project, results[0].TypeID)
		return []Code{Id(responseVar).Dot(embedName)}
	}

	out = make([]Code, 0, len(results))
	for _, ret := range results {
		out = append(out, Id(responseVar).Dot(r.responseStructFieldName(method, ret)))
	}

	return
}

// responseLogValues — composite literal для логирования response (модель полей как в exchange).
func (r *contractRenderer) responseLogValues(method *model.Method) (values Code) {

	results := resultsWithoutError(method)
	if len(results) == 0 {
		return nil
	}

	respType := Id(responseStructName(r.contract.Name, method.Name))
	if len(results) == 1 && r.responseFieldEmbedded(method, results[0]) {
		ret := results[0]
		embedField := model.TypeNameFromTypeID(r.project, ret.TypeID)
		if embedField == "" {
			return respType.Values(r.loggerResultExpr(ret))
		}

		return respType.Values(Dict{Id(embedField): r.loggerResultExpr(ret)})
	}

	return respType.Values(DictFunc(func(d Dict) {
		for _, ret := range results {
			d[Id(r.responseStructFieldName(method, ret))] = r.loggerResultExpr(ret)
		}
	}))
}

// responseMarshalArg — выражение для json.Marshal результата JSON-RPC.
func (r *contractRenderer) responseMarshalArg(method *model.Method, responseVar string) (arg Code) {

	results := resultsWithoutError(method)
	if len(results) == 1 && !r.responseFieldEmbedded(method, results[0]) &&
		model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpEnableInlineSingle) {

		return Id(responseVar).Dot(r.responseStructFieldName(method, results[0]))
	}

	return Id(responseVar)
}

// resultForMarshalValues — заполнение resultForMarshal из response после вызова svc.
func (r *contractRenderer) resultForMarshalValues(method *model.Method, responseVar string) (values Code) {

	resultName := responseResultStructName(r.contract.Name, method.Name)
	results := resultsWithoutError(method)
	if len(results) == 0 {
		return Id(resultName).Values()
	}
	if len(results) == 1 && r.responseFieldEmbedded(method, results[0]) {
		ret := results[0]
		embedField := model.TypeNameFromTypeID(r.project, ret.TypeID)
		if embedField == "" {
			return Id(resultName).Values(Id(responseVar))
		}

		return Id(resultName).Values(Dict{Id(embedField): Id(responseVar).Dot(embedField)})
	}

	return Id(resultName).Values(DictFunc(func(d Dict) {
		for _, ret := range results {
			fieldName := r.responseStructFieldName(method, ret)
			d[Id(fieldName)] = Id(responseVar).Dot(fieldName)
		}
	}))
}
