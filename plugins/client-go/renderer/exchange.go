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
	"tgp/internal/content"
	"tgp/internal/generated"
	"tgp/internal/model"
	"tgp/internal/tags"
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
		isStream := model.MethodIsStream(r.project, contract, method)
		reqXML := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json")) == content.KindXML
		respXML := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json")) == content.KindXML

		if isJSONRPC || isStream {
			srcFile.Add(r.exchange(ctx, contract, r.requestStructName(contract, method), r.fieldsArgumentForClient(contract, method), false)).Line()
		}
		if isHTTP && len(r.argsForRequestBody(contract, method)) > 0 {
			srcFile.Add(r.exchange(ctx, contract, r.requestBodyStructName(contract, method), r.fieldsRequestForBody(contract, method), reqXML)).Line()
		}

		responseStreamResult := r.methodResponseBodyStreamResult(method)
		responseMultipart := r.methodResponseMultipart(contract, method)
		if isHTTP && (responseStreamResult != nil || responseMultipart) {
			continue
		}
		if !r.shouldGenerateResponseExchange(contract, method, isHTTP, isJSONRPC || isStream) {
			continue
		}

		exclude := r.resultNamesExcludeFromBody(contract, method)
		if len(exclude) > 0 && isHTTP {
			bodyFields := r.fieldsResultBody(contract, method)
			if len(bodyFields) == 0 {
				continue
			}
			srcFile.Add(r.exchange(ctx, contract, r.responseBodyStructName(contract, method), bodyFields, respXML)).Line()
			continue
		}
		fields := r.fieldsResult(method)
		if isStream {
			// response нужен только client-stream (streamFinalResult в wsClientInputMethod);
			// server/bidi отбрасывают sync non-chan results (streamClientResults).
			if model.MethodStreamMode(r.project, contract, method) != model.StreamModeClient {
				continue
			}
			fields = r.fieldsStreamResult(method)
		}
		// JSON-RPC всегда Unmarshal в response-тип, даже без полей результата.
		if len(fields) == 0 && !isJSONRPC {
			continue
		}
		withXML := respXML && isHTTP && !isJSONRPC
		srcFile.Add(r.exchange(ctx, contract, r.responseStructName(contract, method), fields, withXML)).Line()
	}
	return srcFile.Save(path.Join(outDir, strings.ToLower(contract.Name)+"-exchange.go"))
}

func (r *ClientRenderer) shouldGenerateResponseExchange(contract *model.Contract, method *model.Method, isHTTP bool, isJSONRPC bool) (ok bool) {

	if isHTTP && !isJSONRPC && len(r.resultsWithoutError(method)) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) {
		return false
	}

	return true
}

func (r *ClientRenderer) exchange(ctx context.Context, contract *model.Contract, name string, fields []exchangeField, withXML bool) Code {

	if len(fields) == 0 {
		return Type().Id(name).Struct()
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
			fieldCode := r.structField(ctx, field, template, withXML)
			gr.Add(fieldCode)
		}
	})
}

func (r *ClientRenderer) structField(ctx context.Context, field exchangeField, template string, withXML bool) *Statement {

	var isInlined bool
	fieldTags := map[string]string{"json": fmt.Sprintf(template, field.name)}
	for tag, value := range common.SortedPairs(field.tags) {
		if tag == "json" {
			fieldTags["json"] = value
			if strings.Contains(value, "inline") {
				isInlined = true
			}
			continue
		}
		fieldTags[tag] = value
	}
	var s *Statement
	if isInlined {
		// Для inline используем версию fieldType, которая использует локальные типы
		s = r.fieldType(ctx, field.typeID, field.numberOfPointers, false)
		s.Tag(map[string]string{"json": ",inline"})
	} else {
		if withXML {
			if _, hasXML := fieldTags["xml"]; !hasXML {
				if xmlTag := tags.ExchangeXMLTag(fieldTags["json"]); xmlTag != "" {
					fieldTags["xml"] = xmlTag
				}
			}
		}
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
		s.Tag(fieldTags)
	}
	if field.isEllipsis {
		s.Comment("This field was defined with ellipsis (...).")
	}
	return s
}
