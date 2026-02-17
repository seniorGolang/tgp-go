// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/generated"
	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderExchange() (err error) {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(generated.ByToolGateway)

	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(fmt.Sprintf("%s/viewer", r.pkgPath(r.outDir)), "viewer")

	typeGen := types.NewGenerator(r.project, &srcFile)

	for _, method := range r.contract.Methods {
		requestFields := r.fieldsArgument(method)
		responseFields := r.fieldsResult(method)
		if len(responseFields) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpEnableInlineSingle) {
			responseFields[0].tags["json"] = ",inline"
		}

		reqName := requestStructName(r.contract.Name, method.Name)
		respName := responseStructName(r.contract.Name, method.Name)
		srcFile.Line().Add(r.exchangeStruct(typeGen, reqName, requestFields))
		srcFile.Line().Add(r.exchangeLogValueMethod(reqName, requestFields))
		srcFile.Line().Add(r.exchangeStruct(typeGen, respName, responseFields))
		srcFile.Line().Add(r.exchangeLogValueMethod(respName, responseFields))

		if len(r.resultNamesExcludeFromBody(method)) > 0 {
			bodyFields := r.fieldsResultBody(method)
			if len(bodyFields) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpEnableInlineSingle) {
				bodyFields[0].tags["json"] = ",inline"
			}
			bodyName := responseBodyStructName(r.contract.Name, method.Name)
			srcFile.Line().Add(r.exchangeStruct(typeGen, bodyName, bodyFields))
			srcFile.Line().Add(r.exchangeLogValueMethod(bodyName, bodyFields))
			resultFields := r.fieldsResultForMarshal(method)
			resultName := responseResultStructName(r.contract.Name, method.Name)
			srcFile.Line().Add(r.exchangeStruct(typeGen, resultName, resultFields))
			srcFile.Line().Add(r.exchangeLogValueMethod(resultName, resultFields))
		}
	}

	err = srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-exchange.go"))
	return
}

func (r *contractRenderer) exchangeStruct(typeGen *types.Generator, name string, fields []exchangeField) (c Code) {

	if len(fields) == 0 {
		return Type().Id(name).Struct()
	}

	template := "%s"
	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagOmitemptyAll) {
		template = "%s,omitempty"
	}

	return Type().Id(name).StructFunc(func(gr *Group) {
		for _, field := range fields {
			fieldCode := r.structField(typeGen, field, template)
			gr.Add(fieldCode)
		}
	})
}

func (r *contractRenderer) exchangeLogValueMethod(typeName string, fields []exchangeField) (c Code) {

	// Viewer по полям типа может паниковать на неэкспортируемых полях (например io.Reader); используем placeholder.
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
	mappings := model.BuildHTTPArgMappings(r.project, r.contract, method)
	excludeSet := model.HTTPExcludeFromExchangeRequestSet(mappings)
	return r.varsToFields(vars, method.Annotations, excludeSet)
}

func (r *contractRenderer) fieldsResult(method *model.Method) []exchangeField {

	vars := resultsWithoutError(method)
	exclude := r.resultNamesExcludeFromBody(method)
	return r.varsToFields(vars, method.Annotations, exclude)
}

func (r *contractRenderer) fieldsResultForMarshal(method *model.Method) []exchangeField {

	vars := resultsWithoutError(method)
	return r.varsToFields(vars, method.Annotations, nil)
}

func (r *contractRenderer) fieldsResultBody(method *model.Method) []exchangeField {

	vars := r.resultsForBody(method)
	return r.varsToFields(vars, method.Annotations, nil)
}

func (r *contractRenderer) varsToFields(vars []*model.Variable, methodTags tags.DocTags, requestJsonOmitNames map[string]struct{}) []exchangeField {

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

		if requestJsonOmitNames != nil {
			if _, omit := requestJsonOmitNames[v.Name]; omit {
				field.tags["json"] = "-"
			}
		}
		for key, value := range common.SortedPairs(methodTags.Sub(v.Name)) {
			if key == model.TagParamTags {
				if list := strings.Split(value, "|"); len(list) > 0 {
					for _, item := range list {
						if tokens := strings.Split(item, ":"); len(tokens) >= 2 {
							tagName := tokens[0]
							tagValue := strings.Join(tokens[1:], ":")
							if tagValue == "inline" {
								tagValue = ",inline"
							}
							field.tags[tagName] = tagValue
						}
					}
				}
			}
		}
		fields = append(fields, field)
	}
	return fields
}

func (r *contractRenderer) structField(typeGen *types.Generator, field exchangeField, template string) (st *Statement) {

	jsonTag := field.tags["json"]
	if jsonTag == "" {
		jsonTag = fmt.Sprintf(template, field.name)
	}
	tags := map[string]string{"json": jsonTag}
	for tag, value := range common.SortedPairs(field.tags) {
		if tag == "json" {
			continue
		}
		tags[tag] = value
	}

	isEmbedded := tags["json"] == ",inline" && typeGen.TypeIsEmbeddable(field.typeID)

	var s *Statement
	switch {
	case field.isSlice || field.arrayLen > 0 || field.mapKey != nil:
		v := &model.Variable{
			TypeRef: model.TypeRef{
				TypeID:           field.typeID,
				NumberOfPointers: field.numberOfPointers,
				IsSlice:          field.isSlice,
				ArrayLen:         field.arrayLen,
				IsEllipsis:       field.isEllipsis,
				ElementPointers:  field.elementPointers,
				MapKey:           field.mapKey,
				MapValue:         field.mapValue,
			},
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
	s.Tag(tags)
	return s
}
