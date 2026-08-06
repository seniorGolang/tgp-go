// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"sort"
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
)

func (r *contractRenderer) httpServeMultipartRequest(method *model.Method) (c Code) {

	streamArgs := r.methodRequestBodyStreamArgs(method)
	if len(streamArgs) == 0 {
		return nil
	}
	r.validatePartContentMapping(method, streamArgs, "request")

	st := Line()
	st.Var().Id("params").Map(String()).String()
	st.Line().If(List(Id("_"), Id("params"), Err()).Op("=").Qual(PackageMime, "ParseMediaType").Call(Id(VarNameFtx).Dot("Get").Call(Lit("Content-Type"))).Op(";").Err().Op("!=").Nil()).Block(
		Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
		Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("invalid or missing Content-Type"))),
	)
	st.Line().Id("boundary").Op(",").Id("ok").Op(":=").Id("params").Index(Lit("boundary"))
	st.Line().If(Op("!").Id("ok").Op("||").Id("boundary").Op("==").Lit("")).Block(
		Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
		Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("missing boundary"))),
	)
	st.Line().Id("bodyStream").Op(":=").Id("streamBodyReader").Call(Id(VarNameFtx))
	st.Line().Id("mr").Op(":=").Qual(PackageMimeMultipart, "NewReader").Call(Id("bodyStream"), Id("boundary"))

	if len(streamArgs) == 1 {
		st.Line().Id("request").Dot(r.requestStructFieldName(method, streamArgs[0])).Op("=").Qual(PackageBytes, "NewReader").Call(Nil())
		st.Line().Var().Id("found").Bool()
		st.Line().Var().Id("p").Op("*").Qual(PackageMimeMultipart, "Part")
		st.Line().For().BlockFunc(func(fg *Group) {
			fg.List(Id("p"), Err()).Op("=").Id("mr").Dot("NextPart").Call()
			fg.If(Qual("errors", "Is").Call(Err(), Qual("io", "EOF"))).Block(Break())
			fg.If(Err().Op("!=").Nil()).Block(
				Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
				Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("multipart read error"))),
			)
			fg.Id("partName").Op(":=").Id("p").Dot("FormName").Call()
			fg.Id("partContentType").Op(":=").Id("p").Dot("Header").Dot("Get").Call(Lit("Content-Type"))
			arg := streamArgs[0]
			partName := r.streamPartName(method, arg)
			expectedContent := r.streamPartContent(method, arg)
			fieldName := r.requestStructFieldName(method, arg)
			fg.Switch(Id("partName")).Block(
				Case(Lit(partName)).BlockFunc(func(cg *Group) {
					if expectedContent != "" {
						cg.If(Id("partContentType").Op("!=").Lit(expectedContent)).Block(
							Id("_").Op("=").Id("p").Dot("Close").Call(),
							Id("request").Dot(fieldName).Op("=").Qual(PackageBytes, "NewReader").Call(Nil()),
							Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
							Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("part ").Op("+").Lit(partName).Op("+").Lit(": invalid content-type"))),
						)
					}
					cg.Id("request").Dot(fieldName).Op("=").Id("p")
					cg.Id("found").Op("=").True()
				}),
				Default().Block(Id("_").Op("=").Id("p").Dot("Close").Call()),
			)
			fg.If(Id("found")).Block(Break())
		})
		st.Line().If(Id("found")).Block(
			Defer().Func().Params().Block(
				Id("_").Op("=").Id("p").Dot("Close").Call(),
			).Call(),
		)
		return st
	}

	streamPkg := r.pkgPath(r.outDir) + "/stream"
	st.Line().Id("partNames").Op(":=").Index().String().ValuesFunc(func(vg *Group) {
		for _, arg := range streamArgs {
			vg.Lit(r.streamPartName(method, arg))
		}
	})
	st.Line().Id("partContentTypes").Op(":=").Index().String().ValuesFunc(func(vg *Group) {
		for _, arg := range streamArgs {
			vg.Lit(r.streamPartContent(method, arg))
		}
	})
	st.Line().Id("parts").Op(":=").Qual(streamPkg, "NewParts").Call(Id("mr"), Id("partNames"), Id("partContentTypes"))
	st.Line().Defer().Func().Params().Block(
		Id("_").Op("=").Id("parts").Dot("Close").Call(),
	).Call()
	for i, arg := range streamArgs {
		fieldName := r.requestStructFieldName(method, arg)
		st.Line().Id("request").Dot(fieldName).Op("=").Id("parts").Dot("Reader").Call(Lit(i))
	}
	return st
}

// Вызывается сразу после сервисного вызова, чтобы ресурсы освобождались и при ошибке, и при редиректе.
func (r *contractRenderer) httpServeMultipartResponseDefers(method *model.Method) (c Code) {

	streamResults := r.methodResponseBodyStreamResults(method)
	if len(streamResults) == 0 {
		return nil
	}

	st := Line()
	for _, res := range streamResults {
		fieldName := r.responseStructFieldName(method, res)
		st.Line().If(Id("response").Dot(fieldName).Op("!=").Nil()).Block(
			Defer().Func().Params().Block(
				Id("_").Op("=").Id("response").Dot(fieldName).Dot("Close").Call(),
			).Call(),
		)
	}
	return st
}

func (r *contractRenderer) httpServeMultipartResponse(method *model.Method) (c Code) {

	streamResults := r.methodResponseBodyStreamResults(method)
	if len(streamResults) == 0 {
		return nil
	}
	r.validatePartContentMapping(method, streamResults, "response")

	st := Line()
	st.Line().Id("mw").Op(":=").Qual(PackageMimeMultipart, "NewWriter").Call(Id(VarNameFtx).Dot("Response").Call().Dot("BodyWriter").Call())
	st.Line().Id("boundary").Op(":=").Id("mw").Dot("Boundary").Call()
	st.Line().Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(
		Lit("multipart/form-data; boundary=").Op("+").Id("boundary"),
	)
	st.Line().Defer().Func().Params().Block(
		Id("_").Op("=").Id("mw").Dot("Close").Call(),
	).Call()
	st.Line().Var().Id("partHeader").Qual(PackageNetTextproto, "MIMEHeader")
	st.Line().Var().Id("partWriter").Qual("io", "Writer")
	for _, res := range streamResults {
		partName := r.streamPartName(method, res)
		contentType := r.streamPartContent(method, res)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		fieldName := r.responseStructFieldName(method, res)
		st.Line().Id("partHeader").Op("=").Make(Qual(PackageNetTextproto, "MIMEHeader"))
		st.Line().Id("partHeader").Index(Lit("Content-Disposition")).Op("=").Index().String().Values(Lit("form-data; name=\"" + partName + "\""))
		st.Line().Id("partHeader").Index(Lit("Content-Type")).Op("=").Index().String().Values(Lit(contentType))
		st.Line().Id("partWriter").Op(",").Err().Op("=").Id("mw").Dot("CreatePart").Call(Id("partHeader"))
		st.Line().If(Err().Op("!=").Nil()).Block(Return())
		st.Line().List(Id("_"), Err()).Op("=").Qual("io", "Copy").Call(Id("partWriter"), Id("response").Dot(fieldName))
		st.Line().If(Err().Op("!=").Nil()).Block(Return())
	}
	return st
}

func (r *contractRenderer) validatePartContentMapping(method *model.Method, vars []*model.Variable, direction string) {

	if method == nil || method.Annotations == nil {
		return
	}
	annotationValue, found := method.Annotations[model.TagHTTPPartContent]
	if !found || strings.TrimSpace(annotationValue) == "" {
		return
	}
	allowedByName := make(map[string]struct{}, len(vars))
	for _, v := range vars {
		if v != nil {
			allowedByName[v.Name] = struct{}{}
		}
	}
	allowedNames := make([]string, 0, len(allowedByName))
	for name := range allowedByName {
		allowedNames = append(allowedNames, name)
	}
	sort.Strings(allowedNames)

	pairs := strings.SplitSeq(annotationValue, ",")
	for rawPair := range pairs {
		pair := strings.TrimSpace(rawPair)
		if pair == "" {
			continue
		}
		tokens := strings.Split(pair, "|")
		if len(tokens) != 2 {
			panic(fmt.Sprintf("invalid @tg %s for %s.%s: expected '<arg>|<mime>', got %q", model.TagHTTPPartContent, r.contract.Name, method.Name, pair))
		}
		argName := strings.TrimSpace(tokens[0])
		if _, ok := allowedByName[argName]; !ok {
			panic(fmt.Sprintf(
				"invalid @tg %s for %s.%s (%s): unknown argument %q; allowed: %s",
				model.TagHTTPPartContent,
				r.contract.Name,
				method.Name,
				direction,
				argName,
				strings.Join(allowedNames, ", "),
			))
		}
	}
}
