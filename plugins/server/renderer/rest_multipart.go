// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *contractRenderer) httpServeMultipartRequest(method *model.Method) Code {

	streamArgs := r.methodRequestBodyStreamArgs(method)
	if len(streamArgs) == 0 {
		return nil
	}

	st := Line()
	st.List(Id("_"), Id("params"), Err()).Op(":=").Qual(PackageMime, "ParseMediaType").Call(Id(VarNameFtx).Dot("Get").Call(Lit("Content-Type")))
	st.Line().If(Err().Op("!=").Nil()).Block(
		Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
		Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("invalid or missing Content-Type"))),
	)
	st.Line().Id("boundary").Op(",").Id("ok").Op(":=").Id("params").Index(Lit("boundary"))
	st.Line().If(Op("!").Id("ok").Op("||").Id("boundary").Op("==").Lit("")).Block(
		Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
		Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("missing boundary"))),
	)
	st.Line().Id("bodyStream").Op(":=").Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call()
	st.Line().If(Id("bodyStream").Op("==").Nil()).Block(
		Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
		Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("failed to read request body"))),
	)
	st.Line().Id("mr").Op(":=").Qual(PackageMimeMultipart, "NewReader").Call(Id("bodyStream"), Id("boundary"))
	if len(streamArgs) > 1 {
		st.Line().Id("partBodies").Op(":=").Make(Map(String()).Index().Byte())
		st.Line().Id("partContentTypes").Op(":=").Make(Map(String()).String())
	}
	if len(streamArgs) == 1 {
		st.Line().Id("request").Dot(toCamel(streamArgs[0].Name)).Op("=").Qual(PackageBytes, "NewReader").Call(Nil())
		st.Line().Var().Id("found").Bool()
	}
	st.Line().Var().Id("p").Op("*").Qual(PackageMimeMultipart, "Part")
	st.Line().For().BlockFunc(func(fg *Group) {
		fg.List(Id("p"), Err()).Op("=").Id("mr").Dot("NextPart").Call()
		fg.If(Err().Op("==").Qual("io", "EOF")).Block(Break())
		fg.If(Err().Op("!=").Nil()).Block(
			Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest")),
			Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("multipart read error"))),
		)
		fg.Id("partName").Op(":=").Id("p").Dot("FormName").Call()
		fg.Id("partContentType").Op(":=").Id("p").Dot("Header").Dot("Get").Call(Lit("Content-Type"))
		cases := make([]Code, 0, len(streamArgs)+1)
		for _, arg := range streamArgs {
			partName := r.streamPartName(method, arg)
			expectedContent := r.streamPartContent(method, arg)
			fieldName := toCamel(arg.Name)
			var caseBlock []Code
			if expectedContent != "" {
				caseBlock = append(caseBlock, If(Id("partContentType").Op("!=").Lit(expectedContent)).BlockFunc(func(mg *Group) {
					for _, a := range streamArgs {
						mg.Id("request").Dot(toCamel(a.Name)).Op("=").Qual(PackageBytes, "NewReader").Call(Nil())
					}
					mg.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest"))
					mg.Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("part ").Op("+").Lit(partName).Op("+").Lit(": invalid content-type")))
				}))
			}
			if len(streamArgs) == 1 {
				caseBlock = append(caseBlock, Id("request").Dot(fieldName).Op("=").Id("p"))
				caseBlock = append(caseBlock, Id("found").Op("=").True())
				caseBlock = append(caseBlock, Break())
			} else {
				caseBlock = append(caseBlock, List(Id("body"), Id("_")).Op(":=").Qual("io", "ReadAll").Call(Id("p")))
				caseBlock = append(caseBlock, Id("partBodies").Index(Id("partName")).Op("=").Id("body"))
				caseBlock = append(caseBlock, Id("partContentTypes").Index(Id("partName")).Op("=").Id("partContentType"))
			}
			cases = append(cases, Case(Lit(partName)).Block(caseBlock...))
		}
		fg.Switch(Id("partName")).Block(append(cases, Default().Block(Qual("io", "Copy").Call(Qual("io", "Discard"), Id("p"))))...)
		if len(streamArgs) == 1 {
			fg.If(Id("found")).Block(Break())
		}
	})
	if len(streamArgs) > 1 {
		st.Line().Var().Id("body").Index().Byte()
		for _, arg := range streamArgs {
			partName := r.streamPartName(method, arg)
			expectedContent := r.streamPartContent(method, arg)
			fieldName := toCamel(arg.Name)
			st.Line().Id("body").Op(",").Id("ok").Op("=").Id("partBodies").Index(Lit(partName))
			st.Line().If(Op("!").Id("ok")).Block(
				Id("request").Dot(fieldName).Op("=").Qual(PackageBytes, "NewReader").Call(Nil()),
			).Else().BlockFunc(func(eg *Group) {
				if expectedContent != "" {
					eg.If(Id("partContentTypes").Index(Lit(partName)).Op("!=").Lit(expectedContent)).BlockFunc(func(mg *Group) {
						for _, a := range streamArgs {
							mg.Id("request").Dot(toCamel(a.Name)).Op("=").Qual(PackageBytes, "NewReader").Call(Nil())
						}
						mg.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest"))
						mg.Return().Id("sendResponse").Call(Id(VarNameFtx), Id("errBadRequestData").Call(Lit("part ").Op("+").Lit(partName).Op("+").Lit(": invalid content-type")))
					})
				}
				eg.Id("request").Dot(fieldName).Op("=").Qual(PackageBytes, "NewReader").Call(Id("body"))
			})
		}
	}
	return st
}

// httpServeMultipartResponseDefers возвращает отложенное закрытие всех stream-частей ответа.
// Вызывается в начале блока err == nil до проверки редиректа, чтобы при return по редиректу ресурсы освобождались.
func (r *contractRenderer) httpServeMultipartResponseDefers(method *model.Method) Code {

	streamResults := r.methodResponseBodyStreamResults(method)
	if len(streamResults) == 0 {
		return nil
	}

	st := Line()
	for _, res := range streamResults {
		fieldName := toCamel(res.Name)
		st.Line().Defer().Id("response").Dot(fieldName).Dot("Close").Call()
	}
	return st
}

func (r *contractRenderer) httpServeMultipartResponse(method *model.Method) Code {

	streamResults := r.methodResponseBodyStreamResults(method)
	if len(streamResults) == 0 {
		return nil
	}

	st := Line()
	st.Line().Id("mw").Op(":=").Qual(PackageMimeMultipart, "NewWriter").Call(Id(VarNameFtx).Dot("Response").Call().Dot("BodyWriter").Call())
	st.Line().Id("boundary").Op(":=").Id("mw").Dot("Boundary").Call()
	st.Line().Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(
		Lit("multipart/form-data; boundary=").Op("+").Id("boundary"),
	)
	st.Line().Defer().Id("mw").Dot("Close").Call()
	st.Line().Var().Id("partHeader").Qual(PackageNetTextproto, "MIMEHeader")
	st.Line().Var().Id("partWriter").Qual("io", "Writer")
	for _, res := range streamResults {
		partName := r.streamPartName(method, res)
		contentType := r.streamPartContent(method, res)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		fieldName := toCamel(res.Name)
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
