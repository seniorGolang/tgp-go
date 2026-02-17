// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/generated"
	"tgp/internal/model"
)

func (r *ClientRenderer) RenderExchange(contract *model.Contract) (err error) {

	outDir := r.outDir
	pkgName := filepath.Base(outDir)
	srcFile := NewSrcFile(pkgName)
	srcFile.PackageComment(generated.ByToolGateway)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint
	ctx = context.WithValue(ctx, keyPackage, pkgName)                // nolint

	for _, method := range contract.Methods {
		isHTTP := r.methodIsHTTP(contract, method)
		isJSONRPC := r.methodIsJsonRPC(contract, method)

		if isJSONRPC {
			srcFile.Add(r.exchange(ctx, contract, r.requestStructName(contract, method), r.fieldsArgumentForClient(contract, method))).Line()
		}
		if isHTTP && len(r.argsForRequestBody(contract, method)) > 0 {
			srcFile.Add(r.exchange(ctx, contract, r.requestBodyStructName(contract, method), r.fieldsRequestForBody(contract, method))).Line()
		}

		responseStreamResult := r.methodResponseBodyStreamResult(method)
		responseMultipart := r.methodResponseMultipart(contract, method)
		if isHTTP && (responseStreamResult != nil || responseMultipart) {
			continue
		}

		exclude := r.resultNamesExcludeFromBody(contract, method)
		if len(exclude) > 0 && isHTTP {
			srcFile.Add(r.exchange(ctx, contract, r.responseBodyStructName(contract, method), r.fieldsResultBody(contract, method))).Line()
		} else {
			srcFile.Add(r.exchange(ctx, contract, r.responseStructName(contract, method), r.fieldsResult(method))).Line()
		}
	}
	return srcFile.Save(path.Join(outDir, strings.ToLower(contract.Name)+"-exchange.go"))
}

func (r *ClientRenderer) exchange(ctx context.Context, contract *model.Contract, name string, fields []exchangeField) Code {

	if len(fields) == 0 {
		return Comment("Formal exchange type, please do not delete.").Line().Type().Id(name).Struct()
	}

	sortedFields := slices.Clone(fields)
	slices.SortFunc(sortedFields, func(a, b exchangeField) int {
		if a.name < b.name {
			return -1
		}
		if a.name > b.name {
			return 1
		}
		return 0
	})

	template := "%s"
	if model.IsAnnotationSet(r.project, contract, nil, nil, "tagOmitemptyAll") {
		template = "%s,omitempty"
	}
	return Type().Id(name).StructFunc(func(gr *Group) {
		for _, field := range sortedFields {
			fieldCode := r.structField(ctx, field, template)
			gr.Add(fieldCode)
		}
	})
}

func (r *ClientRenderer) structField(ctx context.Context, field exchangeField, template string) *Statement {

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
		// Для inline используем версию fieldType, которая использует локальные типы
		s = r.fieldType(ctx, field.typeID, field.numberOfPointers, false)
		s.Tag(map[string]string{"json": ",inline"})
	} else {
		s = Id(ToCamel(field.name))
		if field.isSlice || field.arrayLen > 0 || field.mapKey != nil {
			typeRef := &model.TypeRef{
				TypeID:           field.typeID,
				NumberOfPointers: field.numberOfPointers,
				IsSlice:          field.isSlice,
				ArrayLen:         field.arrayLen,
				IsEllipsis:       field.isEllipsis,
				ElementPointers:  field.elementPointers,
				MapKey:           field.mapKey,
				MapValue:         field.mapValue,
			}
			s.Add(r.fieldTypeFromTypeRef(ctx, typeRef, false))
		} else {
			s.Add(r.fieldType(ctx, field.typeID, field.numberOfPointers, false))
		}
		s.Tag(tags)
	}
	if field.isEllipsis {
		s.Comment("This field was defined with ellipsis (...).")
	}
	return s
}
