// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/common"
	"tgp/internal/content"
	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderExchange() (err error) {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	typeGen := types.NewGenerator(r.project, &srcFile)

	for _, method := range r.contract.Methods {
		requestFields := r.fieldsArgument(method)
		responseFields := r.fieldsResult(method)
		if len(responseFields) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHTTPEnableInlineSingle) {
			responseFields[0].tags[keyJSON] = jsonTagInline
		}

		reqXML := content.Kind(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagRequestContentType, "application/json")) == content.KindXML
		respXML := content.Kind(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagResponseContentType, "application/json")) == content.KindXML

		reqName := requestStructName(r.contract.Name, method.Name)
		respName := responseStructName(r.contract.Name, method.Name)
		srcFile.Line().Add(r.exchangeStruct(typeGen, reqName, requestFields, reqXML))
		r.addExchangeLogValue(&srcFile, reqName, requestFields)
		if len(responseFields) > 0 || !model.MethodIsStream(r.project, r.contract, method) {
			srcFile.Line().Add(r.exchangeStruct(typeGen, respName, responseFields, respXML))
			r.addExchangeLogValue(&srcFile, respName, responseFields)
		}

		if len(r.resultNamesExcludeFromBody(method)) > 0 {
			bodyFields := r.fieldsResultBody(method)
			if len(bodyFields) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHTTPEnableInlineSingle) {
				bodyFields[0].tags[keyJSON] = jsonTagInline
			}
			bodyName := responseBodyStructName(r.contract.Name, method.Name)
			srcFile.Line().Add(r.exchangeStruct(typeGen, bodyName, bodyFields, respXML))
			r.addExchangeLogValue(&srcFile, bodyName, bodyFields)
			resultFields := r.fieldsResultForMarshal(method)
			resultName := responseResultStructName(r.contract.Name, method.Name)
			srcFile.Line().Add(r.exchangeStruct(typeGen, resultName, resultFields, respXML))
			r.addExchangeLogValue(&srcFile, resultName, resultFields)
		}
	}

	err = srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-exchange.go"))
	return
}

func (r *contractRenderer) exchangeStruct(typeGen *types.Generator, name string, fields []exchangeField, withXML bool) (c Code) {

	if len(fields) == 0 {
		return Type().Id(name).Struct()
	}

	template := "%s"
	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagOmitemptyAll) {
		template = "%s,omitempty"
	}

	return Type().Id(name).StructFunc(func(gr *Group) {
		for _, field := range fields {
			fieldCode := r.structField(typeGen, field, template, withXML)
			gr.Add(fieldCode)
		}
	})
}

func (r *contractRenderer) addExchangeLogValue(srcFile *GoFile, typeName string, fields []exchangeField) {

	if !exchangeNeedsLogValue(fields) {
		return
	}
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.Line().Add(r.exchangeLogValueMethod(typeName))
}

func exchangeNeedsLogValue(fields []exchangeField) (needed bool) {

	for _, field := range fields {
		switch field.typeID {
		case TypeIDIOReader, TypeIDIOReadCloser:
			return true
		}
	}
	return false
}

func (r *contractRenderer) exchangeLogValueMethod(typeName string) (c Code) {

	// Placeholder: viewer уважает slog.LogValuer и не обходит поля (io.Reader и т.п.).
	return Func().Params(Id("r").Id(typeName)).Id("LogValue").Params().
		Params(Qual(PackageSlog, "Value")).
		Block(
			Return(Qual(PackageSlog, "StringValue").Call(Lit("<" + typeName + ">"))),
		)
}

type exchangeField struct {
	name             string
	typeID           string
	numberOfPointers int
	isSlice          bool
	arrayLen         int
	isEllipsis       bool
	elementPointers  int
	mapKey           *model.TypeRef
	mapValue         *model.TypeRef
	tags             map[string]string
}

func (r *contractRenderer) fieldsArgument(method *model.Method) []exchangeField {

	vars := argsWithoutContext(method)
	if model.MethodIsStream(r.project, r.contract, method) {
		vars = streamVariables(r.project, vars, false)
	}
	omitSet := model.HTTPOmitFromRequestJSON(r.project, r.contract, method)
	return r.varsToFields(vars, method, omitSet)
}

func (r *contractRenderer) fieldsResult(method *model.Method) []exchangeField {

	vars := resultsWithoutError(method)
	if model.MethodIsStream(r.project, r.contract, method) {
		vars = streamVariables(r.project, vars, false)
	}
	exclude := r.resultNamesExcludeFromBody(method)
	return r.varsToFields(vars, method, exclude)
}

func (r *contractRenderer) fieldsResultForMarshal(method *model.Method) []exchangeField {

	vars := resultsWithoutError(method)
	if model.MethodIsStream(r.project, r.contract, method) {
		vars = streamVariables(r.project, vars, false)
	}
	return r.varsToFields(vars, method, nil)
}

func (r *contractRenderer) fieldsResultBody(method *model.Method) []exchangeField {

	vars := r.resultsForBody(method)
	return r.varsToFields(vars, method, nil)
}

func streamVariables(project *model.Project, vars []*model.Variable, includeChannels bool) (out []*model.Variable) {

	out = make([]*model.Variable, 0, len(vars))
	for _, variable := range vars {
		isChannel := model.TypeRefIsChan(project, &variable.TypeRef)
		if isChannel != includeChannels {
			continue
		}
		out = append(out, variable)
	}
	return
}

func (r *contractRenderer) varsToFields(vars []*model.Variable, method *model.Method, requestJSONOmitNames map[string]struct{}) []exchangeField {

	var methodTags tags.DocTags
	var skipNames map[string]struct{}
	if method != nil {
		methodTags = method.Annotations
		_, skipNames = model.MethodLogSkip(r.project, r.contract, method)
	}

	fields := make([]exchangeField, 0, len(vars))
	for _, v := range vars {
		field := exchangeField{
			name:             v.Name,
			typeID:           v.TypeID,
			numberOfPointers: v.NumberOfPointers,
			isSlice:          v.IsSlice,
			arrayLen:         v.ArrayLen,
			isEllipsis:       v.IsEllipsis,
			elementPointers:  v.ElementPointers,
			mapKey:           v.MapKey,
			mapValue:         v.MapValue,
			tags:             make(map[string]string),
		}

		if requestJSONOmitNames != nil {
			if _, omit := requestJSONOmitNames[v.Name]; omit {
				field.tags[keyJSON] = "-"
			}
		}
		maps.Insert(field.tags, common.SortedPairs(tags.ParseMethodVarTags(methodTags, v.Name)))
		if model.VariableLogSkipped(method, v, skipNames) {
			if _, hasDumper := field.tags[keyDumper]; !hasDumper {
				field.tags[keyDumper] = dumperHide
			}
		}
		fields = append(fields, field)
	}
	return fields
}

func (r *contractRenderer) structField(typeGen *types.Generator, field exchangeField, template string, withXML bool) (st *Statement) {

	jsonTag := field.tags[keyJSON]
	if jsonTag == "" {
		jsonTag = fmt.Sprintf(template, field.name)
	}
	fieldTags := map[string]string{keyJSON: jsonTag}
	for tag, value := range common.SortedPairs(field.tags) {
		if tag == keyJSON {
			continue
		}
		fieldTags[tag] = value
	}
	if withXML {
		if _, hasXML := fieldTags["xml"]; !hasXML {
			if xmlTag := tags.ExchangeXMLTag(jsonTag); xmlTag != "" {
				fieldTags["xml"] = xmlTag
			}
		}
	}

	isEmbedded := fieldTags[keyJSON] == jsonTagInline && typeGen.TypeIsEmbeddable(field.typeID)

	var s *Statement
	switch {
	case field.isSlice || field.arrayLen > 0 || field.mapKey != nil:
		v := &model.Variable{
			TypeID:           field.typeID,
			NumberOfPointers: field.numberOfPointers,
			IsSlice:          field.isSlice,
			ArrayLen:         field.arrayLen,
			IsEllipsis:       field.isEllipsis,
			ElementPointers:  field.elementPointers,
			MapKey:           field.mapKey,
			MapValue:         field.mapValue,
		}
		s = typeGen.FieldTypeFromTypeRef(&v.TypeRef, false)
	case isEmbedded:
		s = typeGen.FieldTypeNoCache(field.typeID, field.numberOfPointers, false)
	default:
		s = typeGen.FieldType(field.typeID, field.numberOfPointers, false)
	}
	if !isEmbedded {
		s = Id(toCamel(field.name)).Add(s)
	}
	s.Tag(fieldTags)
	return s
}
