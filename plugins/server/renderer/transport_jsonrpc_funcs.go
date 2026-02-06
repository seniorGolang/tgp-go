// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/model"
)

func (r *transportRenderer) sendResponseJsonRPCFunc() Code {

	jsonPkg := r.getPackageJSON()
	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
	return Func().Id("sendResponse").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("resp").Any()).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.If(List(Id("responses"), Id("ok")).Op(":=").Id("resp").Op(".").Call(Index().Op("*").Id("baseJsonRPC")).Op(";").Id("ok").Op("&&").Len(Id("responses")).Op("==").Lit(0)).Block(
				Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusNoContent")),
				Return(Nil()),
			)
			bg.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
			bg.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok")).BlockFunc(func(sg *Group) {
				sg.If(List(Id("response"), Id("okResp")).Op(":=").Id("resp").Assert(Op("*").Id("baseJsonRPC")).Op(";").Id("okResp").Op("&&").Id("response").Dot("Error").Op("!=").Nil()).BlockFunc(func(eg *Group) {
					eg.If(Id("server").Dot("metrics").Op("!=").Nil()).Block(
						Id("server").Dot("metrics").Dot("ErrorResponsesTotal").Dot("WithLabelValues").Call(
							Lit("json-rpc"),
							Qual(PackageStrconv, "Itoa").Call(Id("response").Dot("Error").Dot("Code")),
							Id("clientID"),
						).Dot("Inc").Call(),
					)
				})
				sg.If(List(Id("responses"), Id("okBatch")).Op(":=").Id("resp").Assert(Index().Op("*").Id("baseJsonRPC")).Op(";").Id("okBatch")).BlockFunc(func(bg2 *Group) {
					bg2.For(List(Id("_"), Id("response")).Op(":=").Range().Id("responses")).BlockFunc(func(fg *Group) {
						fg.If(Id("response").Op("!=").Nil().Op("&&").Id("response").Dot("Error").Op("!=").Nil()).BlockFunc(func(eg *Group) {
							eg.If(Id("server").Dot("metrics").Op("!=").Nil()).Block(
								Id("server").Dot("metrics").Dot("ErrorResponsesTotal").Dot("WithLabelValues").Call(
									Lit("json-rpc"),
									Qual(PackageStrconv, "Itoa").Call(Id("response").Dot("Error").Dot("Code")),
									Id("clientID"),
								).Dot("Inc").Call(),
							)
						})
					})
				})
			})
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Id("contentTypeJson"))
			bg.If(Err().Op("=").Qual(jsonPkg, "NewEncoder").Call(Id(VarNameFtx).Dot("Response").Call().Dot("BodyWriter").Call()).Dot("Encode").Call(Id("resp")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Id("logger").Op(":=").Qual(srvctxPkgPath, "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameFtx).Dot("UserContext").Call()).Op(";").Id("logger").Op("!=").Nil()).Block(
					Id("logger").Dot("Error").Call(Lit("response marshal error"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
				)
				ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusInternalServerError"))
				ig.Return(Err())
			})
			bg.Return(Err())
		})
}

func (r *transportRenderer) initJsonRPCMethodMap() Code {

	return Func().Id("initJsonRPCMethodMap").
		Params(Id("srv").Op("*").Id("Server")).
		Params().
		BlockFunc(func(bg *Group) {
			bg.Id("srv").Dot("jsonRPCMethodMap").Op("=").Map(String()).Id("methodJsonRPC").Values(DictFunc(func(dict Dict) {
				for _, contract := range r.project.Contracts {
					if !model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
						continue
					}
					for _, method := range contract.Methods {
						if !r.methodIsJsonRPCForContract(contract, method) {
							continue
						}
						contractLCName := strings.ToLower(contract.Name)
						methodLCName := strings.ToLower(method.Name)
						methodKey := contractLCName + "." + methodLCName
						dict[Lit(methodKey)] = Func().
							Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("requestBase").Id("baseJsonRPC")).
							Params(Id("responseBase").Op("*").Id("baseJsonRPC")).
							Block(
								Return(Id("srv").Dot("http"+contract.Name).Dot(toLowerCamel(method.Name)+"WithContext").Call(Id(VarNameCtx), Id("requestBase"))),
							)
					}
				}
			}))
		})
}

func (r *transportRenderer) singleBatchFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("doSingleBatch").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("request").Id("baseJsonRPC")).
		Params(Id("response").Op("*").Id("baseJsonRPC")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Var().Err().Error()
			bg.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("request")).Op(";").Err().Op("!=").Nil()).Block(
				Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("invalidRequestError"), Lit("invalid JSON-RPC request: ").Op("+").Err().Dot("Error").Call(), Nil())),
			)
			bg.If(Id(VarNameCtx).Dot("Err").Call().Op("!=").Nil()).Block(
				Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("invalidRequestError"), Lit("request context cancelled"), Nil())),
			)
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Id("toLowercaseMethod").Call(Id("methodNameOrigin"))
			bg.Id("handler").Op(",").Id("ok").Op(":=").Id("srv").Dot("jsonRPCMethodMap").Index(Id("method"))
			bg.If(Op("!").Id("ok")).Block(
				Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil())),
			)
			bg.Id(VarNameCtx).Op("=").Id("withMethodLogger").Call(Id(VarNameCtx), Lit("rpc"), Id("method"))
			bg.Line()
			bg.Return(Id("handler").Call(Id(VarNameCtx), Id("request")))
		})
}

func (r *transportRenderer) batchFunc() Code {

	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("doBatch").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("requests").Op("[]").Id("baseJsonRPC")).
		Params(Id("responses").Op("[]").Op("*").Id("baseJsonRPC")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("userCtx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.Id("batchTimeout").Op(":=").Id(VarNameFtx).Dot("App").Call().Dot("Config").Call().Dot("WriteTimeout")
			bg.Var().Id("batchCtx").Qual(PackageContext, "Context")
			bg.Var().Id("cancel").Qual(PackageContext, "CancelFunc")
			bg.If(Id("batchTimeout").Op(">").Lit(0)).Block(
				List(Id("batchCtx"), Id("cancel")).Op("=").Qual(srvctxPkgPath, "WithTimeout").Call(Id("userCtx"), Id("batchTimeout")),
				Defer().Id("cancel").Call(),
			).Else().Block(
				Id("batchCtx").Op("=").Id("userCtx"),
			)
			bg.If(Qual(PackageStrings, "EqualFold").Call(Id(VarNameFtx).Dot("Get").Call(Id("syncHeader")), Lit("true"))).Block(
				Id("syncResponses").Op(":=").Make(Index().Op("*").Id("baseJsonRPC"), Lit(0), Len(Id("requests"))),
				For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(
					Id("response").Op(":=").Id("srv").Dot("doSingleBatch").Call(Id("batchCtx"), Id("request")),
					If(Id("request").Dot("ID").Op("!=").Nil()).Block(
						Id("syncResponses").Op("=").Append(Id("syncResponses"), Id("response")),
					),
				),
				Return(Id("syncResponses")),
			)
			bg.Id("expectedCount").Op(":=").Lit(0)
			bg.For(List(Id("_"), Id("req")).Op(":=").Range().Id("requests")).Block(
				If(Id("req").Dot("ID").Op("!=").Nil()).Block(
					Id("expectedCount").Op("++"),
				),
			)
			bg.Id("results").Op(":=").Make(Index().Op("*").Id("baseJsonRPC"), Len(Id("requests")))
			bg.Id("workerCount").Op(":=").Id("srv").Dot("maxParallelBatch")
			bg.If(Len(Id("requests")).Op("<").Id("workerCount")).Block(
				Id("workerCount").Op("=").Len(Id("requests")),
			)
			bg.Id("workCh").Op(":=").Make(Chan().Int(), Id("workerCount"))
			bg.Line()
			bg.Var().Id("wg").Qual(PackageSync, "WaitGroup")
			bg.For(Id("i").Op(":=").Lit(0).Op(";").Id("i").Op("<").Id("workerCount").Op(";").Id("i").Op("++")).Block(
				Id("wg").Dot("Add").Call(Lit(1)),
				Go().Func().Params().Block(
					Defer().Id("wg").Dot("Done").Call(),
					For(Id("idx").Op(":=").Range().Id("workCh")).BlockFunc(func(fg *Group) {
						fg.If(Id("batchCtx").Dot("Err").Call().Op("!=").Nil()).Block(Continue())
						fg.Id("req").Op(":=").Id("requests").Index(Id("idx"))
						fg.Id("resp").Op(":=").Id("srv").Dot("doSingleBatch").Call(Id("batchCtx"), Id("req"))
						fg.If(Id("req").Dot("ID").Op("!=").Nil()).Block(
							Id("results").Index(Id("idx")).Op("=").Id("resp"),
						)
					}),
				).Call(),
			)
			bg.Line()
			bg.Id("sendLoop").Op(":")
			bg.For(Id("i").Op(":=").Lit(0).Op(";").Id("i").Op("<").Len(Id("requests")).Op(";").Id("i").Op("++")).Block(
				Select().Block(
					Case(Id("workCh").Op("<-").Id("i")).Block(),
					Case(Op("<-").Id("batchCtx").Dot("Done").Call()).Block(Break().Id("sendLoop")),
				),
			)
			bg.Close(Id("workCh"))
			bg.Id("wg").Dot("Wait").Call()
			bg.Line()
			bg.Id("responses").Op("=").Make(Index().Op("*").Id("baseJsonRPC"), Lit(0), Id("expectedCount"))
			bg.For(Id("i").Op(":=").Range().Id("results")).Block(
				If(Id("results").Index(Id("i")).Op("!=").Nil()).Block(
					Id("responses").Op("=").Append(Id("responses"), Id("results").Index(Id("i"))),
				),
			)
			bg.Return()
		})
}

func (r *transportRenderer) serveBatchFunc() Code {

	jsonPkg := r.getPackageJSON()
	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("serveBatch").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Var().Id("single").Bool()
			bg.Var().Id("requests").Op("[]").Id("baseJsonRPC")
			bg.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
			bg.Id("methodHTTP").Op(":=").Id(VarNameFtx).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(PackageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				ig.Id(VarNameFtx).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(PackageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(VarNameFtx).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				ig.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("method_not_allowed"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return()
			})
			bg.Id("bodyStream").Op(":=").Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call()
			bg.List(Id("firstByte"), Err()).Op(":=").Id("readUntilFirstNonWhitespace").Call(Id("bodyStream"))
			bg.If(Err().Op("!=").Nil().Op("&&").Err().Op("!=").Qual("io", "EOF")).BlockFunc(func(ig *Group) {
				ig.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())))
			})
			bg.If(Id("firstByte").Op("==").Lit(0)).BlockFunc(func(ig *Group) {
				ig.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("empty_body"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: empty body"), Nil())))
			})
			bg.Id("r").Op(":=").Qual("io", "MultiReader").Call(Qual(PackageBytes, "NewReader").Call(Index().Byte().Values(Id("firstByte"))), Id("bodyStream"))
			bg.Switch(Id("firstByte")).BlockFunc(func(sg *Group) {
				sg.Case(Lit(123)).BlockFunc(func(cg *Group) {
					cg.Var().Id("request").Id("baseJsonRPC")
					cg.If(Err().Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("r")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(dg *Group) {
						dg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
							Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
						)
						dg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())))
					})
					cg.Id("single").Op("=").True()
					cg.Id("requests").Op("=").Append(Id("requests"), Id("request"))
				})
				sg.Case(Lit(91)).BlockFunc(func(cg *Group) {
					cg.If(Err().Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("r")).Dot("Decode").Call(Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(dg *Group) {
						dg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
							Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
						)
						dg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())))
					})
					cg.If(Len(Id("requests")).Op("==").Lit(0)).BlockFunc(func(eg *Group) {
						eg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
							Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("invalid_request"), Id("clientID")).Dot("Inc").Call(),
						)
						eg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("invalidRequestError"), Lit("empty batch request"), Nil())))
					})
				})
				sg.Default().BlockFunc(func(dg *Group) {
					dg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
						Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
					)
					dg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: expected { or ["), Nil())))
				})
			})
			bg.If(Len(Id("requests")).Op(">").Id("srv").Dot("maxBatchSize")).BlockFunc(func(ig *Group) {
				ig.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("batch_size_exceeded"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return(Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("batch size exceeded")))
			})
			bg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
				Id("srv").Dot("metrics").Dot("BatchSize").Dot("Observe").Call(Id("float64").Call(Len(Id("requests")))),
			)
			bg.If(Id("single")).BlockFunc(func(ig *Group) {
				ig.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("requests").Op("[").Lit(0).Op("]")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(vg *Group) {
					vg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
						Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("invalid_request"), Id("clientID")).Dot("Inc").Call(),
					)
					vg.Return(Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("invalid JSON-RPC request: ").Op("+").Err().Dot("Error").Call()))
				})
				ig.Defer().Func().Params().Block(
					If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
						Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("ok"), Id("clientID")).Dot("Inc").Call(),
					),
				).Call()
				ig.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("srv").Dot("doSingleBatch").Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("requests").Op("[").Lit(0).Op("]"))))
			})
			bg.Defer().Func().Params().Block(
				If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("ok"), Id("clientID")).Dot("Inc").Call(),
				),
			).Call()
			bg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("srv").Dot("doBatch").Call(Id(VarNameFtx), Id("requests"))))
		})
}

func (r *transportRenderer) toLowercaseMethodFunc() Code {

	return Func().Id("toLowercaseMethod").
		Params(Id("s").String()).
		Params(String()).
		Block(
			Return(Qual(PackageStrings, "ToLower").Call(Id("s"))),
		)
}

func (r *transportRenderer) sanitizeErrorMessageFunc() Code {

	return Func().Id("sanitizeErrorMessage").
		Params(Err().Error()).
		Params(String()).
		Block(
			If(Err().Op("==").Nil()).Block(
				Return(Lit("")),
			),
			Id("message").Op(":=").Err().Dot("Error").Call(),
			If(Id("idx").Op(":=").Qual(PackageStrings, "IndexByte").Call(Id("message"), Add(Id("'\\n'"))).Op(";").Id("idx").Op(">=").Lit(0)).Block(
				Return(Id("message").Index(Op(":").Id("idx"))),
			),
			Return(Id("message")),
		)
}

func (r *transportRenderer) validateJsonRPCRequestFunc() Code {

	return Func().Id("validateJsonRPCRequest").
		Params(Id("requestBase").Id("baseJsonRPC")).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("requestBase").Dot("Version").Op("!=").Id("Version")).Block(
				Return(Qual(PackageFmt, "Errorf").Call(Lit("incorrect protocol version: %s"), Id("requestBase").Dot("Version"))),
			)
			bg.Return()
		})
}

func (r *transportRenderer) makeErrorResponseJsonRPCFunc() Code {

	jsonPkg := r.getPackageJSON()
	return Func().Id("makeErrorResponseJsonRPC").
		Params(Id("id").Id("idJsonRPC"), Id("code").Int(), Id("msg").String(), Id("data").Any()).
		Params(Op("*").Id("baseJsonRPC")).
		BlockFunc(func(bg *Group) {
			bg.Id("responseID").Op(":=").Id("id")
			bg.If(Id("responseID").Op("==").Nil()).Block(
				Id("responseID").Op("=").Qual(jsonPkg, "RawMessage").Call(Lit("null")),
			)
			bg.Return(Op("&").Id("baseJsonRPC").Values(Dict{
				Id("ID"):      Id("responseID"),
				Id("Version"): Id("Version"),
				Id("Error"): Op("&").Id("errorJsonRPC").Values(Dict{
					Id("Code"):    Id("code"),
					Id("Message"): Id("msg"),
					Id("Data"):    Id("data"),
				}),
			}))
		})
}

func (r *transportRenderer) methodIsJsonRPCForContract(contract *model.Contract, method *model.Method) bool {

	if method == nil {
		return false
	}
	return contract != nil && model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) && !model.IsAnnotationSet(r.project, contract, method, nil, TagMethodHTTP)
}

func (r *transportRenderer) jsonRPCUsedOverlayKeys() (headerNames []string, cookieNames []string) {

	headers := make(map[string]struct{})
	cookies := make(map[string]struct{})
	for _, contract := range r.project.Contracts {
		if !model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
			continue
		}
		for _, method := range contract.Methods {
			if !r.methodIsJsonRPCForContract(contract, method) {
				continue
			}
			for _, h := range usedHeaderNamesForMethod(r.project, contract, method) {
				headers[h] = struct{}{}
			}
			for _, c := range usedCookieNamesForMethod(r.project, contract, method) {
				cookies[c] = struct{}{}
			}
		}
	}
	return common.SortedKeys(headers), common.SortedKeys(cookies)
}

func overlayKeyToFieldName(key string) string {

	parts := strings.Split(key, "-")
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "")
}

func (r *transportRenderer) readUntilFirstNonWhitespaceFunc() Code {

	return Func().Id("readUntilFirstNonWhitespace").
		Params(Id("r").Qual("io", "Reader")).
		Params(Id("firstByte").Byte(), Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Var().Id("buf").Index(Lit(1)).Byte()
			bg.For(Id("i").Op(":=").Lit(0).Op(";").Id("i").Op("<").Id("maxPeekBytes").Op(";").Id("i").Op("++")).BlockFunc(func(fg *Group) {
				fg.Var().Id("n").Int()
				fg.List(Id("n"), Err()).Op("=").Id("r").Dot("Read").Call(Id("buf").Op("[:]"))
				fg.If(Id("n").Op("==").Lit(0)).Block(
					Return(Lit(0), Err()),
				)
				fg.Id("b").Op(":=").Id("buf").Index(Lit(0))
				fg.If(Id("b").Op("!=").Lit(32).Op("&&").Id("b").Op("!=").Lit(9).Op("&&").Id("b").Op("!=").Lit(10).Op("&&").Id("b").Op("!=").Lit(13)).Block(
					Return(Id("b"), Nil()),
				)
			})
			bg.Return(Lit(0), Qual(PackageErrors, "New").Call(Lit("leading whitespace exceeds limit")))
		})
}
