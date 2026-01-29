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
	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderExchange() error {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	typeGen := types.NewGenerator(r.project, &srcFile)

	for _, method := range r.contract.Methods {
		requestFields := r.fieldsArgument(method)
		responseFields := r.fieldsResult(method)

		srcFile.Line().Add(r.exchangeStruct(typeGen, requestStructName(r.contract.Name, method.Name), requestFields))
		srcFile.Line().Add(r.exchangeStruct(typeGen, responseStructName(r.contract.Name, method.Name), responseFields))
	}

	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-exchange.go"))
}

func (r *contractRenderer) exchangeStruct(typeGen *types.Generator, name string, fields []exchangeField) Code {

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

type exchangeField struct {
	name             string
	typeID           string
	numberOfPointers int
	isSlice          bool
	arrayLen         int
	isEllipsis       bool
	elementPointers  int
	mapKeyID         string
	mapValueID       string
	mapKeyPointers   int
	tags             map[string]string
}

func (r *contractRenderer) fieldsArgument(method *model.Method) []exchangeField {

	vars := argsWithoutContext(method)
	return r.varsToFields(vars, method.Annotations)
}

func (r *contractRenderer) fieldsResult(method *model.Method) []exchangeField {

	vars := resultsWithoutError(method)
	return r.varsToFields(vars, method.Annotations)
}

func (r *contractRenderer) varsToFields(vars []*model.Variable, methodTags tags.DocTags) []exchangeField {

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
			mapKeyID:         v.MapKeyID,
			mapValueID:       v.MapValueID,
			mapKeyPointers:   v.MapKeyPointers,
			tags:             make(map[string]string),
		}

		for key, value := range common.SortedPairs(methodTags.Sub(v.Name)) {
			if key == "tag" {
				// Формат: tag:json:fieldName,omitempty|tag:xml:fieldName
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

func (r *contractRenderer) structField(typeGen *types.Generator, field exchangeField, template string) *Statement {

	var isInlined bool
	tags := map[string]string{"json": fmt.Sprintf(template, field.name)}
	for tag, value := range common.SortedPairs(field.tags) {
		if tag == "json" {
			if strings.Contains(value, "inline") {
				isInlined = true
			}
			continue
		}
		tags[tag] = value
	}

	var s *Statement
	if isInlined {
		// Для inline используем базовую версию fieldType
		s = typeGen.FieldType(field.typeID, field.numberOfPointers, false)
		s.Tag(map[string]string{"json": ",inline"})
	} else {
		s = Id(toCamel(field.name))
		if field.isSlice || field.arrayLen > 0 || field.mapKeyID != "" {
			// Создаем временный Variable для передачи в FieldTypeFromVariable
			v := &model.Variable{
				TypeID:           field.typeID,
				NumberOfPointers: field.numberOfPointers,
				IsSlice:          field.isSlice,
				ArrayLen:         field.arrayLen,
				IsEllipsis:       field.isEllipsis,
				ElementPointers:  field.elementPointers,
				MapKeyID:         field.mapKeyID,
				MapValueID:       field.mapValueID,
				MapKeyPointers:   field.mapKeyPointers,
			}
			s.Add(typeGen.FieldTypeFromVariable(v, false))
		} else {
			s.Add(typeGen.FieldType(field.typeID, field.numberOfPointers, false))
		}
		s.Tag(tags)
	}
	return s
}
