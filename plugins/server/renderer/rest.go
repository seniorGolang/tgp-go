// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/content"
	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderREST() error {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	jsonPkg := r.getPackageJSON()
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))
	srcFile.ImportName(jsonPkg, "json")
	srcFile.ImportName(PackageReflect, "reflect")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageStrconv, "strconv")
	srcFile.ImportName("io", "io")
	srcFile.ImportName(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "srvctx")
	for _, method := range r.contract.Methods {
		if r.methodIsHTTP(method) && (r.methodRequestMultipart(method) || r.methodResponseMultipart(method)) {
			srcFile.ImportName(PackageMime, "mime")
			srcFile.ImportName(PackageMimeMultipart, "multipart")
			srcFile.ImportName(PackageBytes, "bytes")
			srcFile.ImportName(PackageNetTextproto, "textproto")
			break
		}
	}
	kindsUsed := make(map[string]struct{})
	for _, method := range r.contract.Methods {
		if !r.methodIsHTTP(method) {
			continue
		}
		kindsUsed[content.Kind(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagRequestContentType, "application/json"))] = struct{}{}
		kindsUsed[content.Kind(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagResponseContentType, "application/json"))] = struct{}{}
	}
	for k := range kindsUsed {
		switch k {
		case content.KindForm:
			srcFile.ImportName(PackageURL, "url")
		case content.KindXML:
			srcFile.ImportName(PackageXML, "xml")
		case content.KindMsgpack:
			srcFile.ImportName(PackageMsgpack, "msgpack")
		case content.KindCBOR:
			srcFile.ImportName(PackageCBOR, "cbor")
		case content.KindYAML:
			srcFile.ImportName(PackageYAML, "yaml")
		}
	}
	typeGen := types.NewGenerator(r.project, &srcFile)

	for _, method := range r.contract.Methods {
		if !r.methodIsHTTP(method) {
			continue
		}
		srcFile.Add(r.httpMethodFunc(typeGen, method))
		srcFile.Add(r.httpServeMethodFunc(&srcFile, typeGen, method, jsonPkg))
	}

	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-rest.go"))
}

func (r *contractRenderer) httpMethodFunc(typeGen *types.Generator, method *model.Method) Code {

	return Func().Params(Id("http").Op("*").Id("http"+r.contract.Name)).
		Id(toLowerCamel(method.Name)).
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("request").Id(requestStructName(r.contract.Name, method.Name))).
		Params(Id("response").Id(responseStructName(r.contract.Name, method.Name)), Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id(VarNameCtx).Op("=").Id("withMethodLogger").Call(Id(VarNameCtx), Lit(toLowerCamel(r.contract.Name)), Lit(toLowerCamel(method.Name)))
			bg.Line()
			bg.If(ListFunc(func(lg *Group) {
				for _, ret := range r.ResultFieldsWithoutError(method) {
					lg.Id("response").Dot(r.responseStructFieldName(method, ret))
				}
				lg.Err()
			}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
				cg.Id(VarNameCtx)
				for _, arg := range argsWithoutContext(method) {
					argCode := Id("request").Dot(r.requestStructFieldName(method, arg))
					if arg.IsEllipsis {
						argCode.Op("...")
					}
					cg.Add(argCode)
				}
			}).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Id("http").Dot("errorHandler").Op("!=").Nil()).Block(
					Err().Op("=").Id("http").Dot("errorHandler").Call(Err()),
				)
			})
			bg.Return()
		})
}

func (r *contractRenderer) httpServeMethodFunc(srcFile *GoFile, typeGen *types.Generator, method *model.Method, jsonPkg string) Code {

	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("serve" + method.Name).
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
			bg.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok").Op("&&").Id("server").Dot("metrics").Op("!=").Nil()).BlockFunc(func(mg *Group) {
				mg.Defer().Func().Params().Block(
					If(Err().Op("==").Nil()).Block(
						Id("server").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("rest"), Lit("ok"), Id("clientID")).Dot("Inc").Call(),
					),
				).Call()
			})
			bg.Var().Id("request").Id(requestStructName(r.contract.Name, method.Name))
			if successCodeStr := model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpSuccess, ""); successCodeStr != "" {
				if successCode, err := strconv.Atoi(successCodeStr); err == nil && successCode != 0 {
					bg.Id(VarNameFtx).Dot("Response").Call().Dot("SetStatusCode").Call(Lit(successCode))
				}
			}
			requestMultipart := r.methodRequestMultipart(method)
			bodyStreamArg := r.methodRequestBodyStreamArg(method)
			if requestMultipart {
				bg.Add(r.httpServeMultipartRequest(method))
			} else if bodyStreamArg == nil && len(r.arguments(method)) != 0 {
				reqKind := content.Kind(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagRequestContentType, "application/json"))
				bg.Add(r.httpServeRequestBodyDecode(jsonPkg, method, reqKind))
			}
			if !requestMultipart && bodyStreamArg != nil {
				bg.Id("request").Dot(r.requestStructFieldName(method, bodyStreamArg)).Op("=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			}
			bg.Add(r.urlArgs(srcFile, typeGen, method, func(arg, header string) *Statement {
				return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest"))
					ig.Return().Id("sendResponse").Call(Id(VarNameFtx), Lit("path arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				})
			}))
			bg.Add(r.urlParams(srcFile, typeGen, method, func(arg, header string) *Statement {
				return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest"))
					ig.Return().Id("sendResponse").Call(Id(VarNameFtx), Lit("url arguments could not be decoded: ").Op("+").Err().Dot("Error").Call())
				})
			}))
			bg.Add(r.httpArgHeaders(srcFile, typeGen, method, func(arg, header string) *Statement {
				return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest"))
					ig.Return().Id("sendResponse").Call(Id(VarNameFtx), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				})
			}))
			bg.Add(r.httpCookies(srcFile, typeGen, method, func(arg, header string) *Statement {
				return Line().If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusBadRequest"))
					ig.Return().Id("sendResponse").Call(Id(VarNameFtx), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call())
				})
			}))
			if responseMethod := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHttpResponse, ""); responseMethod != "" {
				args := argsWithoutContext(method)
				callArgs := make([]Code, 0, len(args)+2)
				callArgs = append(callArgs, Id(VarNameFtx), Id("http").Dot("base"))
				for _, arg := range args {
					callArgs = append(callArgs, Id("request").Dot(r.requestStructFieldName(method, arg)))
				}
				bg.Return().Add(toIDWithImport(responseMethod, srcFile).Call(callArgs...))
			} else {
				responseStreamResult := r.methodResponseBodyStreamResult(method)
				bg.Var().Id("response").Id(responseStructName(r.contract.Name, method.Name))
				bg.If().List(Id("response"), Err()).Op("=").Id("http").Dot(toLowerCamel(method.Name)).Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("request")).Op(";").Err().Op("==").Nil().BlockFunc(func(bf *Group) {
					if r.methodResponseMultipart(method) {
						bf.Add(r.httpServeMultipartResponseDefers(method))
						bf.Line()
					}
					var ex Statement
					if len(r.retCookieMap(method)) > 0 {
						for retName := range common.SortedPairs(r.retCookieMap(method)) {
							if ret := r.resultByName(method, retName); ret != nil {
								ex.If(List(Id("rCookie"), Id("ok")).Op(":=").
									Qual(PackageReflect, "ValueOf").Call(Id("response").Dot(r.responseStructFieldName(method, ret))).Dot("Interface").Call().
									Op(".").Call(Id("cookieType"))).Op(";").Id("ok").Block(
									Id("cookie").Op(":=").Id("rCookie").Dot("Cookie").Call(),
									Id(VarNameFtx).Dot("Cookie").Call(Op("&").Id("cookie")),
								)
							}
						}
					}
					ex.Add(r.httpRetHeaders(method))
					bf.Var().Id("iResponse").Any().Op("=").Id("response")
					bf.If(List(Id("redirect"), Id("ok")).Op(":=").Id("iResponse").Op(".").Call(Id("withRedirect")).Op(";").Id("ok")).Block(
						Return().Id(VarNameFtx).Dot("Redirect").Call(Id("redirect").Dot("RedirectTo").Call()),
					)
					if len(ex) > 0 {
						bf.Add(&ex)
					}
					switch {
					case r.methodResponseMultipart(method):
						bf.Add(r.httpServeMultipartResponse(method))
						bf.Return()
					case responseStreamResult != nil:
						contentType := model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagResponseContentType, "application/octet-stream")
						bf.Defer().Id("response").Dot(r.responseStructFieldName(method, responseStreamResult)).Dot("Close").Call()
						bf.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit(contentType))
						bf.List(Id("_"), Err()).Op("=").Qual("io", "Copy").Call(Id(VarNameFtx).Dot("Response").Call().Dot("BodyWriter").Call(), Id("response").Dot(r.responseStructFieldName(method, responseStreamResult)))
						bf.Return()
					case len(resultsWithoutError(method)) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpEnableInlineSingle):
						bf.Add(r.httpServeResponseEncode(method, content.Kind(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagResponseContentType, "application/json")), "response", true))
					default:
						bf.Add(r.httpServeResponseEncode(method, content.Kind(model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagResponseContentType, "application/json")), "response", false))
					}
				})
				bg.Var().Id("statusCode").Int()
				bg.If(List(Id("errCoder"), Id("ok")).Op(":=").Err().Op(".").Call(Id("withErrorCode")).Op(";").Id("ok")).Block(
					Id("statusCode").Op("=").Id("errCoder").Dot("Code").Call(),
					Id(VarNameFtx).Dot("Status").Call(Id("statusCode")),
				).Else().Block(
					Id("statusCode").Op("=").Qual(PackageFiber, "StatusInternalServerError"),
					Id(VarNameFtx).Dot("Status").Call(Id("statusCode")),
				)
				bg.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok").Op("&&").Id("server").Dot("metrics").Op("!=").Nil()).Block(
					Id("server").Dot("metrics").Dot("ErrorResponsesTotal").Dot("WithLabelValues").Call(
						Lit("rest"),
						Qual(PackageStrconv, "Itoa").Call(Id("statusCode")),
						Id("clientID"),
					).Dot("Inc").Call(),
				)
				bg.Return().Id("sendResponse").Call(Id(VarNameFtx), Err())
			}
		})
}
