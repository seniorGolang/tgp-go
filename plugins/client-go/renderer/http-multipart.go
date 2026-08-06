// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
)

func (r *ClientRenderer) StreamingMultipartHelperTypes() *Statement {

	return Line().Type().Id("sharedMultipartCloser").Struct(
		Id("body").Qual(PackageIO, "Closer"),
		Id("once").Qual(PackageSync, "Once"),
	).Line().
		Func().Params(Id("s").Op("*").Id("sharedMultipartCloser")).Id("Close").Params().Params(Err().Error()).Block(
		Id("s").Dot("once").Dot("Do").Call(Func().Params().Block(
			Err().Op("=").Id("s").Dot("body").Dot("Close").Call(),
		)),
		Return(),
	).Line().
		Type().Id("bufferedPartReader").Struct(
		Op("*").Qual(PackageBytes, "Reader"),
		Id("shared").Op("*").Id("sharedMultipartCloser"),
		Id("once").Qual(PackageSync, "Once"),
	).Line().
		Func().Params(Id("r").Op("*").Id("bufferedPartReader")).Id("Close").Params().Params(Err().Error()).Block(
		Id("r").Dot("once").Dot("Do").Call(Func().Params().Block(
			Err().Op("=").Id("r").Dot("shared").Dot("Close").Call(),
		)),
		Return(),
	)
}

func (r *ClientRenderer) httpMultipartRequestBody(contract *model.Contract, method *model.Method) Code {

	streamArgs := r.methodRequestBodyStreamArgs(method)
	if len(streamArgs) == 0 {
		return nil
	}

	st := Line()
	st.Id("pipeReader").Op(",").Id("pipeWriter").Op(":=").Qual(PackageIO, "Pipe").Call()
	st.Line()
	st.Id("mw").Op(":=").Qual(PackageMimeMultipart, "NewWriter").Call(Id("pipeWriter"))
	st.Line()
	st.Id("multipartBoundary").Op(":=").Id("mw").Dot("Boundary").Call()
	st.Line()
	st.Go().Func().Params().BlockFunc(func(goBg *Group) {
		goBg.Var().Id("partHeader").Qual(PackageNetTextproto, "MIMEHeader")
		goBg.Var().Id("partWriter").Qual(PackageIO, "Writer")
		goBg.Var().Id("writeErr").Id(typeError)
		goBg.Line()
		for i, arg := range streamArgs {
			partName := r.streamPartName(contract, method, arg)
			contentType := r.streamPartContent(contract, method, arg)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			paramName := ToLowerCamel(arg.Name)
			if i > 0 {
				goBg.Line()
			}
			goBg.Id("partHeader").Op("=").Make(Qual(PackageNetTextproto, "MIMEHeader"))
			goBg.Id("partHeader").Index(Lit("Content-Disposition")).Op("=").Index().String().Values(Lit("form-data; name=\"" + partName + "\""))
			goBg.Id("partHeader").Index(Lit("Content-Type")).Op("=").Index().String().Values(Lit(contentType))
			goBg.List(Id("partWriter"), Id("writeErr")).Op("=").Id("mw").Dot("CreatePart").Call(Id("partHeader"))
			goBg.If(Id("writeErr").Op("!=").Nil()).Block(
				Id("pipeWriter").Dot("CloseWithError").Call(Id("writeErr")),
				Return(),
			)
			goBg.List(Id("_"), Id("writeErr")).Op("=").Qual(PackageIO, "Copy").Call(Id("partWriter"), Id(paramName))
			goBg.If(Id("writeErr").Op("!=").Nil()).Block(
				Id("pipeWriter").Dot("CloseWithError").Call(Id("writeErr")),
				Return(),
			)
		}
		goBg.If(Id("writeErr").Op("=").Id("mw").Dot("Close").Call(), Id("writeErr").Op("!=").Nil()).Block(
			Id("_").Op("=").Id("pipeWriter").Dot("CloseWithError").Call(Id("writeErr")),
			Return(),
		)
		goBg.Id("_").Op("=").Id("pipeWriter").Dot("Close").Call()
	}).Call()
	st.Line()
	st.Var().Id("httpReq").Op("*").Qual(PackageHTTP, "Request")
	st.Line()
	st.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHTTP, "NewRequestWithContext").Call(Id(_ctx), Lit(r.httpMethodForContract(contract, method)), Id("baseURL").Dot("String").Call(), Id("pipeReader")).Op(";").Err().Op("!=").Nil()).Block(
		Id("_").Op("=").Id("pipeReader").Dot("Close").Call(),
		Return(),
	)
	st.Line()
	st.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit("multipart/form-data; boundary=").Op("+").Id("multipartBoundary"))
	st.Line()
	st.Id("httpReq").Dot("Close").Op("=").True()
	return st
}

func (r *ClientRenderer) httpMethodForContract(contract *model.Contract, method *model.Method) string {

	return model.GetHTTPMethod(r.project, contract, method)
}

func (r *ClientRenderer) httpMultipartResponseBody(contract *model.Contract, method *model.Method) Code {

	streamResults := r.methodResponseBodyStreamResults(method)
	if len(streamResults) == 0 {
		return nil
	}

	st := Line()
	st.List(Id("_"), Id("params"), Err()).Op(":=").Qual(PackageMime, "ParseMediaType").Call(Id("httpResp").Dot("Header").Dot("Get").Call(Lit("Content-Type")))
	st.Line()
	st.If(Err().Op("!=").Nil()).Block(
		Id("_").Op("=").Id("httpResp").Dot("Body").Dot("Close").Call(),
		Return(),
	)
	st.Line()
	st.Id("boundary").Op(",").Id("ok").Op(":=").Id("params").Index(Lit("boundary"))
	st.Line()
	st.If(Op("!").Id("ok").Op("||").Id("boundary").Op("==").Lit("")).Block(
		Id("_").Op("=").Id("httpResp").Dot("Body").Dot("Close").Call(),
		Return(),
	)
	st.Line()
	st.Id("mr").Op(":=").Qual(PackageMimeMultipart, "NewReader").Call(Id("httpResp").Dot("Body"), Id("boundary"))
	st.Line()
	st.Id("sharedCloser").Op(":=").Op("&").Id("sharedMultipartCloser").Values(Dict{
		Id("body"): Id("httpResp").Dot("Body"),
	})
	st.Line()
	st.Id("parts").Op(":=").Map(String()).Op("*").Qual(PackageBytes, "Buffer").Values()
	st.Line()
	for _, res := range streamResults {
		partName := r.streamPartName(contract, method, res)
		st.Id("parts").Index(Lit(partName)).Op("=").Op("&").Qual(PackageBytes, "Buffer").Values()
		st.Line()
	}
	st.For().Block(
		List(Id("part"), Id("partErr")).Op(":=").Id("mr").Dot("NextPart").Call(),
		If(Id("partErr").Op("==").Qual(PackageIO, "EOF")).Block(Break()),
		If(Id("partErr").Op("!=").Nil()).Block(
			Err().Op("=").Id("partErr"),
			Id("_").Op("=").Id("sharedCloser").Dot("Close").Call(),
			Return(),
		),
		If(Id("partBuffer").Op(":=").Id("parts").Index(Id("part").Dot("FormName").Call()).Op(";").Id("partBuffer").Op("!=").Nil()).Block(
			If(List(Id("_"), Id("copyErr")).Op(":=").Qual(PackageIO, "Copy").Call(Id("partBuffer"), Id("part")).Op(";").Id("copyErr").Op("!=").Nil()).Block(
				Err().Op("=").Id("copyErr"),
				Id("_").Op("=").Id("part").Dot("Close").Call(),
				Id("_").Op("=").Id("sharedCloser").Dot("Close").Call(),
				Return(),
			),
		),
		Id("_").Op("=").Id("part").Dot("Close").Call(),
	)
	st.Line()
	for _, res := range streamResults {
		partName := r.streamPartName(contract, method, res)
		st.Id(ToLowerCamel(res.Name)).Op("=").Op("&").Id("bufferedPartReader").Values(Dict{
			Id("Reader"): Qual(PackageBytes, "NewReader").Call(Id("parts").Index(Lit(partName)).Dot("Bytes").Call()),
			Id("shared"): Id("sharedCloser"),
		})
		st.Line()
	}
	st.Return()
	return st
}
