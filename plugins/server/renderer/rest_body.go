package renderer

import (
	"fmt"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/content"
	"tgp/internal/model"
)

func (r *contractRenderer) httpServeRequestBodyDecode(jsonPkg string, method *model.Method, reqKind string) Code {

	writeBadRequest := func(ig *Group) {
		ig.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok").Op("&&").Id("server").Dot("metrics").Op("!=").Nil()).Block(
			Id("server").Dot("metrics").Dot("ErrorResponsesTotal").Dot("WithLabelValues").Call(Lit("rest"), Lit("400"), Id("clientID")).Dot("Inc").Call(),
		)
		ig.Id(VarNameFtx).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(PackageFiber, "StatusBadRequest"))
		ig.List(Id("_"), Err()).Op("=").Id(VarNameFtx).Dot("WriteString").Call(Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
		ig.Return()
	}

	switch reqKind {
	case content.KindForm:
		return BlockFunc(func(bg *Group) {
			bg.Id("bodyStream").Op(":=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			bg.Var().Id("bodyBytes").Index().Byte()
			bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual("io", "ReadAll").Call(Id("bodyStream")).Op(";").Err().Op("!=").Nil()).BlockFunc(writeBadRequest)
			bg.Id(VarNameFtx).Dot("Context").Call().Dot("Request").Dot("SetBodyRaw").Call(Id("bodyBytes"))
			bg.If(Err().Op("=").Id(VarNameFtx).Dot("BodyParser").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(writeBadRequest)
		})
	case content.KindXML:
		return BlockFunc(func(bg *Group) {
			bg.Id("bodyStream").Op(":=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			bg.If(Err().Op("=").Qual(PackageXML, "NewDecoder").Call(Id("bodyStream")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				writeBadRequest(ig)
			})
		})
	case content.KindMsgpack:
		return BlockFunc(func(bg *Group) {
			bg.Id("bodyStream").Op(":=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			bg.If(Err().Op("=").Qual(PackageMsgpack, "NewDecoder").Call(Id("bodyStream")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				writeBadRequest(ig)
			})
		})
	case content.KindCBOR:
		return BlockFunc(func(bg *Group) {
			bg.Id("bodyStream").Op(":=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			bg.If(Err().Op("=").Qual(PackageCBOR, "NewDecoder").Call(Id("bodyStream")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				writeBadRequest(ig)
			})
		})
	case content.KindYAML:
		return BlockFunc(func(bg *Group) {
			bg.Id("bodyStream").Op(":=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			bg.If(Err().Op("=").Qual(PackageYAML, "NewDecoder").Call(Id("bodyStream")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				writeBadRequest(ig)
			})
		})
	default:
		return BlockFunc(func(bg *Group) {
			bg.Id("bodyStream").Op(":=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			bg.If(Err().Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("bodyStream")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				writeBadRequest(ig)
			})
		})
	}
}

func (r *contractRenderer) httpServeResponseEncode(method *model.Method, resKind string, responseVar string, inlineValue bool) Code {

	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
	writeErr := func() *Statement {
		return If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
			ig.If(Id("logger").Op(":=").Qual(srvctxPkgPath, "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameFtx).Dot("UserContext").Call()).Op(";").Id("logger").Op("!=").Nil()).Block(
				Id("logger").Dot("Error").Call(Lit("response encode error"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
			)
			ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusInternalServerError"))
			ig.Return(Err())
		})
	}

	bodyWriter := Id(VarNameFtx).Dot("Response").Call().Dot("BodyWriter").Call()
	var respArg Code
	if inlineValue {
		respArg = Id(responseVar).Dot(r.responseStructFieldName(method, resultsWithoutError(method)[0]))
	} else {
		respArg = Id(responseVar)
	}

	switch resKind {
	case content.KindForm:
		return r.httpServeResponseEncodeForm(method, bodyWriter, responseVar, respArg, inlineValue, writeErr)
	case content.KindXML:
		return BlockFunc(func(bg *Group) {
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit(content.CanonicalMIME(content.KindXML)))
			bg.If(Err().Op("=").Qual(PackageXML, "NewEncoder").Call(bodyWriter).Dot("Encode").Call(respArg).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Add(writeErr())
			})
			bg.Return(Nil())
		})
	case content.KindMsgpack:
		return BlockFunc(func(bg *Group) {
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit(content.CanonicalMIME(content.KindMsgpack)))
			bg.If(Err().Op("=").Qual(PackageMsgpack, "NewEncoder").Call(bodyWriter).Dot("Encode").Call(respArg).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Add(writeErr())
			})
			bg.Return(Nil())
		})
	case content.KindCBOR:
		return BlockFunc(func(bg *Group) {
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit(content.CanonicalMIME(content.KindCBOR)))
			bg.If(Err().Op("=").Qual(PackageCBOR, "NewEncoder").Call(bodyWriter).Dot("Encode").Call(respArg).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Add(writeErr())
			})
			bg.Return(Nil())
		})
	case content.KindYAML:
		return BlockFunc(func(bg *Group) {
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit(content.CanonicalMIME(content.KindYAML)))
			bg.If(Err().Op("=").Qual(PackageYAML, "NewEncoder").Call(bodyWriter).Dot("Encode").Call(respArg).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Add(writeErr())
			})
			bg.Return(Nil())
		})
	default:
		return Return(Id("sendResponse").Call(Id(VarNameFtx), respArg))
	}
}

func (r *contractRenderer) httpServeResponseEncodeForm(method *model.Method, bodyWriter Code, responseVar string, respArg Code, inlineValue bool, writeErr func() *Statement) Code {

	fields := r.fieldsResult(method)
	formKey := func(f exchangeField) string {
		if k := f.tags["form"]; k != "" {
			return k
		}
		return toLowerCamel(f.name)
	}

	return BlockFunc(func(bg *Group) {
		bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit(content.CanonicalMIME(content.KindForm)))
		bg.Id("v").Op(":=").Qual(PackageURL, "Values").Call(Dict{})
		for _, f := range fields {
			key := formKey(f)
			var value Code
			if inlineValue && len(fields) == 1 {
				value = respArg
			} else {
				value = Id(responseVar).Dot(toCamel(f.name))
			}
			bg.Id("v").Dot("Set").Call(Lit(key), Qual(PackageFmt, "Sprint").Call(value))
		}
		bg.List(Id("_"), Err()).Op("=").Qual("io", "WriteString").Call(bodyWriter, Id("v").Dot("Encode").Call())
		bg.Add(writeErr())
		bg.Return(Nil())
	})
}
