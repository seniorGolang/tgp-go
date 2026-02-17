package renderer

import (
	"fmt"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/content"
	"tgp/internal/model"
)

func (r *ClientRenderer) httpRequestBodyEncode(bg *Group, contract *model.Contract, method *model.Method, requestStructName string, requestVar string, jsonPkg string, reqKind string) {

	schemaPkg := fmt.Sprintf("%s/schema", r.pkgPath(r.outDir))
	switch reqKind {
	case content.KindForm:
		bg.Id("formData").Op(":=").Make(Map(String()).Index().String())
		bg.Id("formEncoder").Op(":=").Qual(schemaPkg, "NewEncoder").Call()
		bg.Id("formEncoder").Dot("SetAliasTag").Call(Lit("form"))
		bg.If(Err().Op("=").Id("formEncoder").Dot("Encode").Call(Id(requestVar), Id("formData")).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.Id("bodyStr").Op(":=").Qual(PackageURL, "Values").Call(Id("formData")).Dot("Encode").Call()
		bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(model.GetHTTPMethod(r.project, contract, method)), Id("baseURL").Dot("String").Call(), Qual(PackageBytes, "NewBufferString").Call(Id("bodyStr"))).Op(";").Err().Op("!=").Nil()).Block(Return())
		return
	case content.KindXML:
		bg.Var().Id("bodyBytes").Index().Byte()
		bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(PackageXML, "Marshal").Call(Id(requestVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(model.GetHTTPMethod(r.project, contract, method)), Id("baseURL").Dot("String").Call(), Qual(PackageBytes, "NewReader").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
		return
	case content.KindMsgpack:
		bg.Var().Id("bodyBytes").Index().Byte()
		bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(PackageMsgpack, "Marshal").Call(Id(requestVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(model.GetHTTPMethod(r.project, contract, method)), Id("baseURL").Dot("String").Call(), Qual(PackageBytes, "NewReader").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
		return
	case content.KindCBOR:
		bg.Var().Id("bodyBytes").Index().Byte()
		bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(PackageCBOR, "Marshal").Call(Id(requestVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(model.GetHTTPMethod(r.project, contract, method)), Id("baseURL").Dot("String").Call(), Qual(PackageBytes, "NewReader").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
		return
	case content.KindYAML:
		bg.Var().Id("bodyBytes").Index().Byte()
		bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(PackageYAML, "Marshal").Call(Id(requestVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(model.GetHTTPMethod(r.project, contract, method)), Id("baseURL").Dot("String").Call(), Qual(PackageBytes, "NewReader").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
		return
	default:
		bg.Var().Id("bodyBytes").Index().Byte()
		bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id(requestVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(model.GetHTTPMethod(r.project, contract, method)), Id("baseURL").Dot("String").Call(), Qual(PackageBytes, "NewReader").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
	}
}

func (r *ClientRenderer) httpResponseDecode(bg *Group, contract *model.Contract, method *model.Method, jsonPkg string, resKind string, responseVar string) {

	schemaPkg := fmt.Sprintf("%s/schema", r.pkgPath(r.outDir))
	bodyReader := Id("httpResp").Dot("Body")
	switch resKind {
	case content.KindForm:
		bg.Var().Id("bodyBytes").Index().Byte()
		bg.Var().Id("formValues").Qual(PackageURL, "Values")
		bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(PackageIO, "ReadAll").Call(bodyReader).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.If(List(Id("formValues"), Err()).Op("=").Qual(PackageURL, "ParseQuery").Call(Id("string").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.Id("formDecoder").Op(":=").Qual(schemaPkg, "NewDecoder").Call()
		bg.Id("formDecoder").Dot("SetAliasTag").Call(Lit("form"))
		bg.Id("formDecoder").Dot("IgnoreUnknownKeys").Call(True())
		bg.If(Err().Op("=").Id("formDecoder").Dot("Decode").Call(Op("&").Id(responseVar), Id("formValues")).Op(";").Err().Op("!=").Nil()).Block(Return())
	case content.KindXML:
		bg.If(Err().Op("=").Qual(PackageXML, "NewDecoder").Call(bodyReader).Dot("Decode").Call(Op("&").Id(responseVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
	case content.KindMsgpack:
		bg.If(Err().Op("=").Qual(PackageMsgpack, "NewDecoder").Call(bodyReader).Dot("Decode").Call(Op("&").Id(responseVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
	case content.KindCBOR:
		bg.If(Err().Op("=").Qual(PackageCBOR, "NewDecoder").Call(bodyReader).Dot("Decode").Call(Op("&").Id(responseVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
	case content.KindYAML:
		bg.If(Err().Op("=").Qual(PackageYAML, "NewDecoder").Call(bodyReader).Dot("Decode").Call(Op("&").Id(responseVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
	default:
		bg.If(Err().Op("=").Qual(jsonPkg, "NewDecoder").Call(bodyReader).Dot("Decode").Call(Op("&").Id(responseVar)).Op(";").Err().Op("!=").Nil()).Block(Return())
	}
}
