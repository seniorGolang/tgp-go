// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/common"
	"tgp/internal/model"
)

func (r *transportRenderer) sendResponseJSONRPCFunc() (c Code) {

	jsonPkg := r.getPackageJSON()
	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	return Func().Id("sendResponse").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("resp").Any()).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.If(List(Id("responses"), Id("ok")).Op(":=").Id("resp").Op(".").Call(Index().Op("*").Id("baseJSONRPC")).Op(";").Id("ok").Op("&&").Len(Id("responses")).Op("==").Lit(0)).Block(
				Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusNoContent")),
				Return(Nil()),
			)
			bg.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
			bg.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok")).BlockFunc(func(sg *Group) {
				sg.If(List(Id("response"), Id("okResp")).Op(":=").Id("resp").Assert(Op("*").Id("baseJSONRPC")).Op(";").Id("okResp").Op("&&").Id("response").Dot("Error").Op("!=").Nil()).BlockFunc(func(eg *Group) {
					eg.If(Id("server").Dot("metrics").Op("!=").Nil()).Block(
						Id("server").Dot("metrics").Dot("ErrorResponsesTotal").Dot("WithLabelValues").Call(
							Lit("json-rpc"),
							Qual(PackageStrconv, "Itoa").Call(Id("response").Dot("Error").Dot("Code")),
							Id("clientID"),
						).Dot("Inc").Call(),
					)
				})
				sg.If(List(Id("responses"), Id("okBatch")).Op(":=").Id("resp").Assert(Index().Op("*").Id("baseJSONRPC")).Op(";").Id("okBatch")).BlockFunc(func(bg2 *Group) {
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

func (r *transportRenderer) sendJSONRPCResponseFunc() (c Code) {

	return Func().Id("sendJSONRPCResponse").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("requestID").Id("idJSONRPC"), Id("resp").Op("*").Id("baseJSONRPC")).
		Params(Err().Error()).
		Block(
			If(Id("requestID").Op("==").Nil()).Block(
				Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusNoContent")),
				Return(Nil()),
			),
			Return(Id("sendResponse").Call(Id(VarNameFtx), Id("resp"))),
		)
}

func (r *transportRenderer) initJSONRPCMethodMaps() (c Code) {

	mounts := model.JSONRPCBatchMounts(r.project)
	return Func().Id("initJSONRPCMethodMaps").
		Params(Id("srv").Op("*").Id("Server")).
		Params().
		BlockFunc(func(bg *Group) {
			bg.Id("srv").Dot("jsonRPCMethodMaps").Op("=").Make(Map(String()).Map(String()).Id("methodJsonRPC"))
			for _, mount := range mounts {
				bg.Id("srv").Dot("jsonRPCMethodMaps").Index(Lit(mount)).Op("=").Make(Map(String()).Id("methodJsonRPC"))
			}
			for _, contract := range r.contractsSorted() {
				if !model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) {
					continue
				}
				servicePath := model.JSONRPCServiceBatchPath(r.project, contract)
				bg.If(Id("srv").Dot("jsonRPCMethodMaps").Index(Lit(servicePath)).Op("==").Nil()).Block(
					Id("srv").Dot("jsonRPCMethodMaps").Index(Lit(servicePath)).Op("=").Make(Map(String()).Id("methodJsonRPC")),
				)
			}
			for _, contract := range r.contractsSorted() {
				if !model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) {
					continue
				}
				contractPrefix := model.JSONRPCContractPrefix(r.project, contract)
				servicePath := model.JSONRPCServiceBatchPath(r.project, contract)
				bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).BlockFunc(func(ib *Group) {
					for _, method := range methodsSorted(contract.Methods) {
						if !r.methodIsJSONRPCForContract(contract, method) {
							continue
						}
						contractLCName := strings.ToLower(contract.Name)
						methodLCName := strings.ToLower(method.Name)
						methodKey := contractLCName + "." + methodLCName
						ib.BlockFunc(func(mb *Group) {
							mb.Id("handler").Op(":=").Func().
								Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("requestBase").Id("baseJSONRPC")).
								Params(Id("responseBase").Op("*").Id("baseJSONRPC")).
								Block(
									Id(VarNameCtx).Op("=").Id("withMethodLogger").Call(Id(VarNameCtx), Lit(toLowerCamel(contract.Name)), Lit(toLowerCamel(method.Name))),
									Return(Id("srv").Dot("http"+contract.Name).Dot(toLowerCamel(method.Name)+"WithContext").Call(Id(VarNameCtx), Id("requestBase"))),
								)
							for _, mount := range mounts {
								if !model.ContractInJSONRPCBatchScope(contractPrefix, mount) {
									continue
								}
								mb.Id("srv").Dot("jsonRPCMethodMaps").Index(Lit(mount)).Index(Lit(methodKey)).Op("=").Id("handler")
							}
							mb.Id("srv").Dot("jsonRPCMethodMaps").Index(Lit(servicePath)).Index(Lit(methodLCName)).Op("=").Id("handler")
						})
					}
				})
			}
		})
}

func (r *transportRenderer) batchHandlerFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("batchHandler").
		Params(Id("batchPath").String()).
		Params(Qual(PackageFiber, "Handler")).
		Block(
			Return(Func().
				Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
				Params(Id("err").Error()).
				Block(
					Return(Id("srv").Dot("serveBatch").Call(Id(VarNameFtx), Id("batchPath"))),
				),
			),
		)
}

func (r *transportRenderer) singleBatchFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("doSingleBatch").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("request").Id("baseJSONRPC"), Id("methods").Map(String()).Id("methodJsonRPC")).
		Params(Id("response").Op("*").Id("baseJSONRPC")).
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
			bg.Id("handler").Op(",").Id("ok").Op(":=").Id("methods").Index(Id("method"))
			bg.If(Op("!").Id("ok")).Block(
				Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil())),
			)
			bg.Line()
			bg.Return(Id("handler").Call(Id(VarNameCtx), Id("request")))
		})
}

func (r *transportRenderer) batchFunc() (c Code) {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("doBatch").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("requests").Op("[]").Id("baseJSONRPC"), Id("methods").Map(String()).Id("methodJsonRPC")).
		Params(Id("responses").Op("[]").Op("*").Id("baseJSONRPC")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("userCtx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.Id("batchTimeout").Op(":=").Id(VarNameFtx).Dot("App").Call().Dot("Config").Call().Dot("WriteTimeout")
			bg.Var().Id("batchCtx").Qual(PackageContext, "Context")
			bg.Var().Id("cancel").Qual(PackageContext, "CancelFunc")
			bg.If(Id("batchTimeout").Op(">").Lit(0)).Block(
				List(Id("batchCtx"), Id("cancel")).Op("=").Qual(PackageContext, "WithTimeout").Call(Id("userCtx"), Id("batchTimeout")),
				Defer().Id("cancel").Call(),
			).Else().Block(
				Id("batchCtx").Op("=").Id("userCtx"),
			)
			bg.If(Qual(PackageStrings, "EqualFold").Call(Id(VarNameFtx).Dot("Get").Call(Id("syncHeader")), Lit("true"))).Block(
				Id("syncResponses").Op(":=").Make(Index().Op("*").Id("baseJSONRPC"), Lit(0), Len(Id("requests"))),
				For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(
					Id("response").Op(":=").Id("srv").Dot("doSingleBatch").Call(Id("batchCtx"), Id("request"), Id("methods")),
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
			bg.Id("results").Op(":=").Make(Index().Op("*").Id("baseJSONRPC"), Len(Id("requests")))
			bg.Id("resultsCh").Op(":=").Make(Chan().Id("batchResult"), Len(Id("requests")))
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
						fg.Func().Params(Id("idx").Int()).BlockFunc(func(ig *Group) {
							ig.Defer().Func().Params().Block(
								If(Id("r").Op(":=").Recover().Op(";").Id("r").Op("!=").Nil()).Block(
									If(Id("logger").Op(":=").Qual(srvctxPkgPath, "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id("batchCtx")).Op(";").Id("logger").Op("!=").Nil()).Block(
										Id("logger").Dot("Error").Call(Lit("JSON-RPC batch panic"), Qual(PackageSlog, "Any").Call(Lit("panic"), Id("r"))),
									),
									If(Id("requests").Index(Id("idx")).Dot("ID").Op("!=").Nil()).Block(
										Id("resultsCh").Op("<-").Id("batchResult").Values(Dict{
											Id("idx"): Id("idx"),
											Id("response"): Id("makeErrorResponseJsonRPC").Call(
												Id("requests").Index(Id("idx")).Dot("ID"),
												Id("internalError"),
												Lit("internal error"),
												Nil(),
											),
										}),
									),
								),
							).Call()
							ig.If(Id("batchCtx").Dot("Err").Call().Op("!=").Nil()).Block(Return())
							ig.Id("req").Op(":=").Id("requests").Index(Id("idx"))
							ig.Id("resp").Op(":=").Id("srv").Dot("doSingleBatch").Call(Id("batchCtx"), Id("req"), Id("methods"))
							ig.If(Id("req").Dot("ID").Op("!=").Nil()).Block(
								Id("resultsCh").Op("<-").Id("batchResult").Values(Dict{
									Id("idx"):      Id("idx"),
									Id("response"): Id("resp"),
								}),
							)
						}).Call(Id("idx"))
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
			bg.Go().Func().Params().Block(
				Id("wg").Dot("Wait").Call(),
				Close(Id("resultsCh")),
			).Call()
			bg.For(List(Id("result")).Op(":=").Range().Id("resultsCh")).Block(
				Id("results").Index(Id("result").Dot("idx")).Op("=").Id("result").Dot("response"),
			)
			bg.Line()
			bg.Id("responses").Op("=").Make(Index().Op("*").Id("baseJSONRPC"), Lit(0), Id("expectedCount"))
			bg.For(Id("i").Op(":=").Range().Id("results")).Block(
				If(Id("results").Index(Id("i")).Op("!=").Nil()).Block(
					Id("responses").Op("=").Append(Id("responses"), Id("results").Index(Id("i"))),
				).Else().If(Id("requests").Index(Id("i")).Dot("ID").Op("!=").Nil()).Block(
					Id("responses").Op("=").Append(Id("responses"), Id("makeErrorResponseJsonRPC").Call(Id("requests").Index(Id("i")).Dot("ID"), Id("invalidRequestError"), Lit("request context cancelled"), Nil())),
				),
			)
			bg.Return()
		})
}

func (r *transportRenderer) serveBatchFunc() (c Code) {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	streamPkg := r.pkgPath(r.outDir) + "/stream"
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("serveBatch").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("batchPath").String()).
		Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			if r.jsonRPCNeedsOverlay(nil) {
				bg.Add(setRequestOverlayContext())
			}
			bg.Var().Id("single").Bool()
			bg.Var().Id("requests").Op("[]").Id("baseJSONRPC")
			bg.Id("methods").Op(":=").Id("srv").Dot("jsonRPCMethodMaps").Index(Id("batchPath"))
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
			writeJSONRPCEnvelopeDecode(bg, streamPkg, Id("srv").Dot("maxBatchSize"), func(g *Group, stmt Code) {
				g.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(stmt)
			}, func(reason string) Code {
				return Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit(reason), Id("clientID")).Dot("Inc").Call()
			})
			bg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
				Id("srv").Dot("metrics").Dot("BatchSize").Dot("WithLabelValues").Call(Lit("json-rpc"), Id("batchPath"), Id("clientID")).Dot("Observe").Call(Id("float64").Call(Len(Id("requests")))),
			)
			bg.If(Id("single")).BlockFunc(func(ig *Group) {
				ig.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("requests").Op("[").Lit(0).Op("]")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(vg *Group) {
					vg.If(Id("requests").Op("[").Lit(0).Op("]").Dot("ID").Op("==").Nil()).Block(
						Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusNoContent")),
						Return(Nil()),
					)
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
				ig.Return(Id("sendJSONRPCResponse").Call(Id(VarNameFtx), Id("requests").Op("[").Lit(0).Op("]").Dot("ID"), Id("srv").Dot("doSingleBatch").Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("requests").Op("[").Lit(0).Op("]"), Id("methods"))))
			})
			bg.Defer().Func().Params().Block(
				If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("ok"), Id("clientID")).Dot("Inc").Call(),
				),
			).Call()
			bg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("srv").Dot("doBatch").Call(Id(VarNameFtx), Id("requests"), Id("methods"))))
		})
}

func (r *transportRenderer) toLowercaseMethodFunc() (c Code) {

	return Func().Id("toLowercaseMethod").
		Params(Id("s").String()).
		Params(String()).
		Block(
			Return(Qual(PackageStrings, "ToLower").Call(Id("s"))),
		)
}

func (r *transportRenderer) validateJSONRPCRequestFunc() (c Code) {

	return Func().Id("validateJsonRPCRequest").
		Params(Id("requestBase").Id("baseJSONRPC")).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("requestBase").Dot("Version").Op("!=").Id("Version")).Block(
				Return(Qual(PackageFmt, "Errorf").Call(Lit("incorrect protocol version: %s"), Id("requestBase").Dot("Version"))),
			)
			bg.Return()
		})
}

func (r *transportRenderer) makeErrorResponseJSONRPCFunc() (c Code) {

	jsonPkg := r.getPackageJSON()
	return Func().Id("makeErrorResponseJsonRPC").
		Params(Id("id").Id("idJSONRPC"), Id("code").Int(), Id("msg").String(), Id("data").Any()).
		Params(Op("*").Id("baseJSONRPC")).
		BlockFunc(func(bg *Group) {
			bg.Id("responseID").Op(":=").Id("id")
			bg.If(Id("code").Op("==").Id("internalError")).Block(
				Id("msg").Op("=").Lit("internal error"),
			)
			bg.If(Id("responseID").Op("==").Nil()).Block(
				Id("responseID").Op("=").Qual(jsonPkg, "RawMessage").Call(Lit("null")),
			)
			bg.Return(Op("&").Id("baseJSONRPC").Values(Dict{
				Id("ID"):      Id("responseID"),
				Id("Version"): Id("Version"),
				Id("Error"): Op("&").Id("errorJSONRPC").Values(Dict{
					Id("Code"):    Id("code"),
					Id("Message"): Id("msg"),
					Id("Data"):    Id("data"),
				}),
			}))
		})
}

func (r *transportRenderer) methodIsJSONRPCForContract(contract *model.Contract, method *model.Method) (ok bool) {

	return model.MethodIsJSONRPC(r.project, contract, method)
}

// jsonRPCOverlayKeys — header/cookie keys для requestOverlay.
// contract == nil → все JSON-RPC контракты проекта; иначе только указанный.
func (r *baseRenderer) jsonRPCOverlayKeys(contract *model.Contract) (headerNames []string, cookieNames []string) {

	headers := make(map[string]struct{})
	cookies := make(map[string]struct{})
	contracts := r.contractsSorted()
	if contract != nil {
		contracts = []*model.Contract{contract}
	}
	for _, c := range contracts {
		if !model.IsAnnotationSet(r.project, c, nil, nil, model.TagServerJSONRPC) {
			continue
		}
		for _, method := range methodsSorted(c.Methods) {
			if !model.MethodIsJSONRPC(r.project, c, method) {
				continue
			}
			for _, h := range usedHeaderNamesForRequestOverlay(r.project, c, method) {
				headers[h] = struct{}{}
			}
			for _, name := range usedCookieNamesForRequestOverlay(r.project, c, method) {
				cookies[name] = struct{}{}
			}
		}
	}
	return common.SortedKeys(headers), common.SortedKeys(cookies)
}

func (r *baseRenderer) jsonRPCNeedsOverlay(contract *model.Contract) (ok bool) {

	headerNames, cookieNames := r.jsonRPCOverlayKeys(contract)
	return len(headerNames)+len(cookieNames) > 0
}

func (r *transportRenderer) jsonRPCUsedOverlayKeys() (headerNames []string, cookieNames []string) {

	return r.jsonRPCOverlayKeys(nil)
}

func overlayKeyToFieldName(key string) (s string) {

	parts := strings.Split(key, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "")
}

func writeJSONRPCEnvelopeDecode(bg *Group, streamPkg string, maxBatch Code, ifMetrics func(g *Group, stmt Code), metricInc func(reason string) Code) {

	bg.Id("bodyStream").Op(":=").Id("decodeBodyReader").Call(Id(VarNameFtx))
	bg.If(List(Id("requests"), Id("single"), Err()).Op("=").Qual(streamPkg, "DecodeEnvelope").Types(Id("baseJSONRPC")).Call(Id("bodyStream"), maxBatch).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
		ig.Switch().Block(
			Case(Qual(PackageErrors, "Is").Call(Err(), Id("errBodyTooLarge"))).BlockFunc(func(cg *Group) {
				ifMetrics(cg, metricInc("body_too_large"))
				cg.Return(Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusRequestEntityTooLarge"), Lit("request body too large")))
			}),
			Case(Qual(PackageErrors, "Is").Call(Err(), Qual(streamPkg, "ErrBatchTooLarge"))).BlockFunc(func(cg *Group) {
				ifMetrics(cg, metricInc("batch_size_exceeded"))
				cg.Return(Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("batch size exceeded")))
			}),
			Case(Qual(PackageErrors, "Is").Call(Err(), Qual(streamPkg, "ErrEmptyBatch"))).BlockFunc(func(cg *Group) {
				ifMetrics(cg, metricInc("invalid_request"))
				cg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("invalidRequestError"), Lit("empty batch request"), Nil())))
			}),
			Case(Qual(PackageErrors, "Is").Call(Err(), Qual(streamPkg, "ErrEmptyBody"))).BlockFunc(func(cg *Group) {
				ifMetrics(cg, metricInc("empty_body"))
				cg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: empty body"), Nil())))
			}),
			Case(Qual(PackageErrors, "Is").Call(Err(), Qual(streamPkg, "ErrBadEnvelope"))).BlockFunc(func(cg *Group) {
				ifMetrics(cg, metricInc("parse_error"))
				cg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: expected { or ["), Nil())))
			}),
			Default().BlockFunc(func(cg *Group) {
				ifMetrics(cg, metricInc("parse_error"))
				cg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())))
			}),
		)
	})
}
