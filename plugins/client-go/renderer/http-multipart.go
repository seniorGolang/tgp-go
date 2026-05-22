// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

// Потоковое чтение multipart-ответа без буферизации всего тела в памяти.
func (r *ClientRenderer) StreamingMultipartHelperTypes() *Statement {

	return Line().Type().Id("sharedMultipartCloser").Struct(
		Id("body").Qual(PackageIO, "Closer"),
		Id("count").Int(),
		Id("mu").Qual(PackageSync, "Mutex"),
		Id("readMu").Qual(PackageSync, "Mutex"),
	).Line().
		Func().Params(Id("s").Op("*").Id("sharedMultipartCloser")).Id("Close").Params().Params(Err().Error()).Block(
		Id("s").Dot("mu").Dot("Lock").Call(),
		Id("s").Dot("count").Op("--"),
		Id("c").Op(":=").Id("s").Dot("count"),
		Id("s").Dot("mu").Dot("Unlock").Call(),
		If(Id("c").Op("==").Lit(0)).Block(
			Return(Id("s").Dot("body").Dot("Close").Call()),
		),
		Return(Nil()),
	).Line().
		Type().Id("streamingPartReader").Struct(
		Id("mr").Op("*").Qual(PackageMimeMultipart, "Reader"),
		Id("wantPart").String(),
		Id("cur").Op("*").Qual(PackageMimeMultipart, "Part"),
		Id("shared").Op("*").Id("sharedMultipartCloser"),
	).Line().
		Func().Params(Id("r").Op("*").Id("streamingPartReader")).Id("Read").Params(Id("p").Index().Byte()).Params(Id("n").Int(), Err().Error()).BlockFunc(func(bg *Group) {
		bg.Id("r").Dot("shared").Dot("readMu").Dot("Lock").Call()
		bg.Defer().Id("r").Dot("shared").Dot("readMu").Dot("Unlock").Call()
		bg.Line()
		bg.If(Id("r").Dot("cur").Op("==").Nil()).BlockFunc(func(adv *Group) {
			adv.For().BlockFunc(func(loop *Group) {
				loop.List(Id("part"), Id("partErr")).Op(":=").Id("r").Dot("mr").Dot("NextPart").Call()
				loop.If(Id("partErr").Op("==").Qual(PackageIO, "EOF")).Block(Return(Lit(0), Id("partErr")))
				loop.If(Id("partErr").Op("!=").Nil()).Block(Return(Lit(0), Id("partErr")))
				loop.If(Id("part").Dot("FormName").Call().Op("==").Id("r").Dot("wantPart")).Block(
					Id("r").Dot("cur").Op("=").Id("part"),
					Break(),
				)
				loop.Id("part").Dot("Close").Call()
			})
		})
		bg.Return(Id("r").Dot("cur").Dot("Read").Call(Id("p")))
	}).Line().
		Func().Params(Id("r").Op("*").Id("streamingPartReader")).Id("Close").Params().Params(Err().Error()).Block(
		If(Id("r").Dot("cur").Op("!=").Nil()).Block(
			Id("r").Dot("cur").Dot("Close").Call(),
			Id("r").Dot("cur").Op("=").Nil(),
		),
		Return(Id("r").Dot("shared").Dot("Close").Call()),
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
		goBg.Var().Id("writeErr").Id("error")
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
		goBg.Id("mw").Dot("Close").Call()
		goBg.Id("pipeWriter").Dot("Close").Call()
	}).Call()
	st.Line()
	st.Var().Id("httpReq").Op("*").Qual(PackageHttp, "Request")
	st.Line()
	st.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(r.httpMethodForContract(contract, method)), Id("baseURL").Dot("String").Call(), Id("pipeReader")).Op(";").Err().Op("!=").Nil()).Block(Return())
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
		Id("httpResp").Dot("Body").Dot("Close").Call(),
		Return(),
	)
	st.Line()
	st.Id("boundary").Op(",").Id("ok").Op(":=").Id("params").Index(Lit("boundary"))
	st.Line()
	st.If(Op("!").Id("ok").Op("||").Id("boundary").Op("==").Lit("")).Block(
		Id("httpResp").Dot("Body").Dot("Close").Call(),
		Return(),
	)
	st.Line()
	st.Id("mr").Op(":=").Qual(PackageMimeMultipart, "NewReader").Call(Id("httpResp").Dot("Body"), Id("boundary"))
	st.Line()
	st.Id("sharedCloser").Op(":=").Op("&").Id("sharedMultipartCloser").Values(Dict{
		Id("body"):  Id("httpResp").Dot("Body"),
		Id("count"): Lit(len(streamResults)),
	})
	st.Line()
	for _, res := range streamResults {
		partName := r.streamPartName(contract, method, res)
		st.Id(ToLowerCamel(res.Name)).Op("=").Op("&").Id("streamingPartReader").Values(Dict{
			Id("mr"):       Id("mr"),
			Id("wantPart"): Lit(partName),
			Id("shared"):   Id("sharedCloser"),
		})
		st.Line()
	}
	st.Return()
	return st
}
