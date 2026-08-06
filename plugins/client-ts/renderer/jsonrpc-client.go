// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"tgp/plugins/client-ts/tsg"

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/model"
)

func (r *ClientRenderer) RenderJSONRPCClientClass(contract *model.Contract) (err error) {

	if contract == nil {
		slog.Error(i18n.Msg("RenderJSONRPCClientClass: contract is nil"))
		return errors.New("contract is nil")
	}
	if contract.PkgPath == "" {
		slog.Error(i18n.Msg("RenderJSONRPCClientClass: contract.PkgPath is empty"), slog.String("contract", contract.Name))
		return fmt.Errorf("contract.PkgPath is empty for contract %s", contract.Name)
	}
	hasUnary := r.contractHasUnaryJSONRPC(contract)

	outDir := r.outDir
	r.contract = contract
	r.knownTypes = make(map[string]int)
	r.typeDefTS = make(map[string]typeDefTS)

	file := tsg.NewFile()
	file.ImportNamed("./client", "Client")
	if hasUnary {
		file.ImportNamed("./jsonrpc/client", "JsonRpcClient")
		file.ImportType("./jsonrpc/utils/jsonrpc", "JsonRpcCall", "JsonRpcParams", "ResponseRPC")
		file.ImportType("./jsonrpc/utils/ts", "MapBatchResult")
		file.ImportType("./batch", "BatchRequest")
	}

	for _, method := range contract.Methods {
		var args []*model.Variable
		if r.methodIsJSONRPC(contract, method) {
			args = r.argsForExchangeRequest(contract, method)
		} else {
			args = r.argsForClient(contract, method)
		}
		for _, arg := range args {
			r.walkStreamAwareVariable(contract, method, arg, true)
		}
		results := r.resultsWithoutError(method)
		for _, ret := range results {
			r.walkStreamAwareVariable(contract, method, ret, false)
		}
	}

	exchangePath := fmt.Sprintf("./%s-exchange", r.tsFileName(contract))
	exchangeTypes := make([]string, 0, len(contract.Methods)*2)
	seenTypes := make(map[string]bool)
	for _, method := range contract.Methods {
		if r.methodIsJSONRPC(contract, method) {
			args := r.argsForExchangeRequest(contract, method)
			if len(args) > 0 {
				requestType := r.requestTypeName(contract, method)
				if !seenTypes[requestType] {
					exchangeTypes = append(exchangeTypes, requestType)
					seenTypes[requestType] = true
				}
			}
			results := r.resultsWithoutError(method)
			if len(results) > 0 {
				responseType := r.responseTypeName(contract, method)
				if !seenTypes[responseType] {
					exchangeTypes = append(exchangeTypes, responseType)
					seenTypes[responseType] = true
				}
			}
		}
		r.appendStreamExchangeTypes(contract, method, &exchangeTypes, seenTypes)
	}
	if len(exchangeTypes) > 0 {
		file.ImportType(exchangePath, exchangeTypes...)
	}
	hasDtoTypes := false
	for _, def := range common.SortedPairs(r.typeDefTS) {
		if def.importPkg == pkgDTO {
			hasDtoTypes = true
			break
		}
	}
	if hasDtoTypes {
		file.ImportAll(exchangePath, pkgDTO)
	}
	usedPkgs := r.collectUsedNamespacePackages(contract)
	for _, pkg := range common.SortedKeys(usedPkgs) {
		file.ImportAll(exchangePath, pkg)
	}

	file.GenerateImports()
	file.Line()

	importedTypes := make(map[string]bool)
	for _, typeName := range exchangeTypes {
		importedTypes[typeName] = true
	}
	for _, def := range common.SortedPairs(r.typeDefTS) {
		typeName := def.importName
		if typeName == "" {
			typeName = def.name
		}
		if importedTypes[typeName] {
			continue
		}
		if def.importPkg != "" {
			continue
		}
		file.Add(r.renderTypeDef(def))
		file.Line()
	}

	allErrorsMap := make(map[string]errorInfo)
	for _, method := range contract.Methods {
		methodErrors := r.collectMethodErrors(method, contract)
		for key, errInfo := range methodErrors {
			if _, exists := allErrorsMap[key]; !exists {
				allErrorsMap[key] = errInfo
			}
		}
	}

	for _, errInfo := range common.SortedPairs(allErrorsMap) {
		file.Add(r.renderErrorType(errInfo))
		file.Line()
	}

	for _, method := range contract.Methods {
		methodErrors := r.collectMethodErrors(method, contract)
		if len(methodErrors) > 0 {
			unionType := r.renderErrorUnionType(method.Name, methodErrors)
			if unionType != nil {
				file.Add(unionType)
				file.Line()
			}
		}
	}

	for _, method := range contract.Methods {
		if !r.methodIsJSONRPC(contract, method) {
			continue
		}
		results := r.resultsWithoutError(method)
		callbackTypeStmt := tsg.NewStatement()
		callbackTypeStmt.Export().Type("ret" + contract.Name + method.Name)
		callbackTypeStmt.Op("=")
		callbackFnType := tsg.NewStatement()
		callbackFnType.Params(func(fg *tsg.Group) {
			for _, ret := range results {
				typeStr := r.walkVariable(ret.Name, contract.PkgPath, ret, method.Annotations, false).typeLink()
				nullableType := tsg.NewStatement()
				nullableType.Add(tsg.TypeFromString(typeStr)).Op("|").ID("null")
				fg.Add(tsg.NewStatement().ID(ret.Name).Colon().Add(nullableType))
			}
			fg.Add(tsg.NewStatement().ID(typeError).Colon().ID("Error").Op("|").ID("null"))
		}).Op("=>").ID("void")
		callbackTypeStmt.Add(callbackFnType).Semicolon()
		file.Add(callbackTypeStmt)
		file.Line()
	}

	clientClass := r.renderJSONRPCClientClass(contract)
	file.Add(clientClass)
	file.Line()

	outFilename := path.Join(outDir, r.tsFileName(contract)+".ts")
	return file.Save(outFilename)
}

func (r *ClientRenderer) walkStreamAwareVariable(contract *model.Contract, method *model.Method, variable *model.Variable, input bool) {

	if variable == nil {
		return
	}
	if element, ok := model.TypeRefChanElement(r.project, &variable.TypeRef); ok {
		_ = r.walkVariable(variable.Name, contract.PkgPath, &model.Variable{Name: variable.Name, TypeRef: *element}, method.Annotations, input)
		return
	}
	_ = r.walkVariable(variable.Name, contract.PkgPath, variable, method.Annotations, input)
}

func (r *ClientRenderer) collectUsedNamespacePackages(contract *model.Contract) (pkgs map[string]bool) {

	pkgs = make(map[string]bool)
	for _, method := range contract.Methods {
		streamMethod := model.MethodIsStream(r.project, contract, method)
		if !r.methodIsJSONRPC(contract, method) && !streamMethod {
			continue
		}
		for _, arg := range r.argsForExchangeRequest(contract, method) {
			r.collectNamespaceFromVariable(contract, method, arg, true, pkgs)
		}
		if in, element, ok := model.MethodStreamInChan(r.project, method); ok {
			r.collectNamespaceFromVariable(contract, method, &model.Variable{Name: in.Name, TypeRef: *element}, true, pkgs)
		}
		for _, ret := range r.resultsWithoutError(method) {
			if model.TypeRefIsChan(r.project, &ret.TypeRef) {
				continue
			}
			r.collectNamespaceFromVariable(contract, method, ret, false, pkgs)
		}
		if out, element, ok := model.MethodStreamOutChan(r.project, method); ok {
			r.collectNamespaceFromVariable(contract, method, &model.Variable{Name: out.Name, TypeRef: *element}, false, pkgs)
		}
	}
	delete(pkgs, pkgDTO)
	delete(pkgs, pkgTime)
	delete(pkgs, "uuid")
	return pkgs
}

func (r *ClientRenderer) collectNamespaceFromVariable(contract *model.Contract, method *model.Method, variable *model.Variable, input bool, pkgs map[string]bool) {

	if variable == nil {
		return
	}
	link := r.walkVariable(variable.Name, contract.PkgPath, variable, method.Annotations, input).typeLink()
	if pkg := namespaceFromTypeLink(link); pkg != "" {
		pkgs[pkg] = true
	}
}

func namespaceFromTypeLink(link string) (pkg string) {

	if link == "" {
		return ""
	}
	base := link
	if idx := strings.Index(link, "|"); idx > 0 {
		base = strings.TrimSpace(link[:idx])
	}
	if base == "" || !strings.Contains(base, ".") {
		return ""
	}
	before, _, _ := strings.Cut(base, ".")
	return strings.TrimSpace(before)
}

func (r *ClientRenderer) renderJSONRPCClientClass(contract *model.Contract) (st *tsg.Statement) {
	// Захват contract в замыкании: r.contract может меняться при обходе контрактов.
	currentContract := contract

	stmt := tsg.NewStatement()
	stmt.Comment(fmt.Sprintf("JSON-RPC client for %s service", contract.Name))
	filteredSvcDocs := r.filterDocsComments(contract.Docs)
	if len(filteredSvcDocs) > 0 {
		stmt.Comment(strings.Join(filteredSvcDocs, "\n"))
	}
	stmt.Export().Class(contract.Name+"Client", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().Private().ID("baseClient").Colon().ID("Client").Semicolon())
		if r.contractHasUnaryJSONRPC(contract) {
			grp.Add(tsg.NewStatement().Private().ID("client").Colon().ID("JsonRpcClient").Semicolon())
		}
		grp.Line()

		constructor := tsg.NewStatement()
		constructor.Comment(fmt.Sprintf("Creates a new JSON-RPC client for %s service", contract.Name))
		constructorStmt := tsg.NewStatement()
		constructorStmt.ID("constructor")
		constructorStmt.Params(func(cg *tsg.Group) {
			cg.Add(tsg.NewStatement().ID("_baseClient").Colon().ID("Client"))
		})
		constructorStmt.Block(func(cg *tsg.Group) {
			cg.Add(tsg.NewStatement().This().Dot("baseClient").Op("=").ID("_baseClient").Semicolon())
			if r.contractHasUnaryJSONRPC(contract) {
				cg.Add(tsg.NewStatement().This().Dot("client").Op("=").ID("_baseClient").Dot("getRpcClient").Call().Semicolon())
			}
		})
		grp.Add(constructorStmt)
		grp.Line()

		if r.contract == nil {
			slog.Error(i18n.Msg("renderJSONRPCClientClass: r.contract is nil in closure"))
			return
		}

		for _, method := range currentContract.Methods {
			if r.methodIsJSONRPC(currentContract, method) {
				r.renderJSONRPCMethod(grp, currentContract, method)
				r.renderJSONRPCRequestMethod(grp, currentContract, method)
			}
		}
		hasStream := false
		for _, method := range currentContract.Methods {
			if model.MethodIsWS(r.project, currentContract, method) || model.MethodIsSSE(r.project, currentContract, method) {
				hasStream = true
				break
			}
		}
		if hasStream {
			r.renderStreamSupport(grp, currentContract)
			for _, method := range currentContract.Methods {
				r.renderStreamMethods(grp, currentContract, method)
			}
		}

		if r.contractHasUnaryJSONRPC(contract) {
			r.renderJSONRPCBatchMethod(grp, contract)
		}
	})
	stmt.Export()
	return stmt
}

func (r *ClientRenderer) contractHasUnaryJSONRPC(contract *model.Contract) (ok bool) {

	for _, method := range contract.Methods {
		if r.methodIsJSONRPC(contract, method) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) renderJSONRPCMethod(grp *tsg.Group, contract *model.Contract, method *model.Method) {

	filteredDocs := r.filterDocsComments(method.Docs)
	if len(filteredDocs) > 0 {
		grp.Comment(strings.Join(filteredDocs, "\n"))
	} else {
		grp.Comment(fmt.Sprintf("Calls %s.%s method", contract.Name, method.Name))
	}

	args := r.argsForExchangeRequest(contract, method)
	results := r.resultsWithoutError(method)

	methodParams := tsg.NewStatement()
	methodParams.Params(func(pg *tsg.Group) {
		if len(args) > 0 {
			for _, arg := range args {
				typeStr := r.walkVariable(arg.Name, contract.PkgPath, arg, method.Annotations, true).typeLink()
				paramStmt := tsg.NewStatement()
				paramStmt.ID(tsSafeName(arg.Name))
				if model.IsAnnotationSet(r.project, contract, method, nil, "nullable") {
					paramStmt.Optional()
				}
				paramStmt.Colon()
				paramStmt.Add(tsg.TypeFromString(typeStr))
				pg.Add(paramStmt)
			}
		}
	})

	returnType := r.resultToTypeStatement(method, results)
	requestTypeName := r.requestTypeName(contract, method)
	responseTypeName := r.responseTypeName(contract, method)

	methodStmt := tsg.NewStatement()
	methodStmt.Public()
	argsForBody := r.argsForExchangeRequest(contract, method)
	methodStmt.AsyncMethodWithParams(r.lcName(method.Name), methodParams, returnType, func(mg *tsg.Group) {
		if len(argsForBody) > 0 {
			paramsObj := tsg.NewStatement()
			paramsObj.Const("params").Colon().ID(requestTypeName).Op("=")
			paramsObj.Values(func(vg *tsg.Group) {
				for _, arg := range argsForBody {
					vg.Add(tsg.NewStatement().ID(arg.Name).Colon().ID(tsSafeName(arg.Name)))
				}
			})
			mg.Add(paramsObj.Semicolon())
		} else {
			mg.Add(tsg.NewStatement().Const("params").Colon().ID("Record").Generic(typeString, "never").Op("=").Values(nil).Semicolon())
		}
		methodName := model.JSONRPCWireMethod(contract.Name, method.Name)
		execCall := tsg.NewStatement()
		execCall.This().Dot("client").Dot("exec")
		execCall.Call(
			tsg.NewStatement().Lit(methodName),
			tsg.NewStatement().ID("params"),
		)

		mg.Add(
			tsg.NewStatement().
				Const("execResult").
				Op("=").
				Await(execCall).
				Semicolon(),
		)

		mg.If(
			tsg.NewStatement().
				ID("execResult").
				Dot("type").
				Op("!==").
				Lit("success"),
			func(ig *tsg.Group) {
				ig.Throw(tsg.NewStatement().ID("this").Dot("baseClient").Dot("decodeRPCError").Call(
					tsg.NewStatement().ID("execResult").Dot(typeError),
				))
			},
		)

		if len(results) == 0 {
			mg.Return()
		} else {
			mg.Add(tsg.NewStatement().Const(nameResult).Colon().ID(responseTypeName).Op("=").ID("execResult").Dot(nameResult).Op("as").ID(responseTypeName).Semicolon())

			if len(results) == 1 {
				mg.Return(r.tsRPCSingleResultExpr(contract, method, results[0]))
			} else {
				returnObj := tsg.NewStatement()
				returnObj.Values(func(rg *tsg.Group) {
					for _, ret := range results {
						rg.Add(tsg.NewStatement().ID(ret.Name).Colon().ID(nameResult).Dot(ret.Name))
					}
				})
				mg.Return(returnObj)
			}
		}
	})
	grp.Add(methodStmt)
	grp.Line()
}

func (r *ClientRenderer) renderJSONRPCRequestMethod(grp *tsg.Group, contract *model.Contract, method *model.Method) {

	filteredDocs := r.filterDocsComments(method.Docs)
	if len(filteredDocs) > 0 {
		grp.Comment(strings.Join(filteredDocs, "\n"))
	} else {
		grp.Comment(fmt.Sprintf("Creates a RequestRPC for %s.%s method", contract.Name, method.Name))
	}

	args := r.argsForExchangeRequest(contract, method)

	methodParams := tsg.NewStatement()
	methodParams.Params(func(pg *tsg.Group) {
		callbackTypeName := "ret" + contract.Name + method.Name
		pg.Add(tsg.NewStatement().ID("callback").Colon().ID(callbackTypeName))
		if len(args) > 0 {
			for _, arg := range args {
				typeStr := r.walkVariable(arg.Name, contract.PkgPath, arg, method.Annotations, true).typeLink()
				paramStmt := tsg.NewStatement()
				paramStmt.ID(tsSafeName(arg.Name))
				if model.IsAnnotationSet(r.project, contract, method, nil, "nullable") {
					paramStmt.Optional()
				}
				paramStmt.Colon()
				paramStmt.Add(tsg.TypeFromString(typeStr))
				pg.Add(paramStmt)
			}
		}
	})

	returnType := tsg.NewStatement()
	returnType.ID("BatchRequest")

	requestTypeName := r.requestTypeName(contract, method)
	argsForBody := r.argsForExchangeRequest(contract, method)

	methodStmt := tsg.NewStatement()
	methodStmt.ID("req" + method.Name)
	if methodParams != nil {
		methodStmt.Add(methodParams)
	} else {
		methodStmt.ID("()")
	}
	if returnType != nil {
		methodStmt.Colon().Add(returnType)
	}
	methodStmt.Block(func(bg *tsg.Group) {
		if len(argsForBody) > 0 {
			paramsObj := tsg.NewStatement()
			paramsObj.Const("params").Colon().ID(requestTypeName).Op("=")
			paramsObj.Values(func(vg *tsg.Group) {
				for _, arg := range argsForBody {
					vg.Add(tsg.NewStatement().ID(arg.Name).Colon().ID(tsSafeName(arg.Name)))
				}
			})
			bg.Add(paramsObj.Semicolon())
		} else {
			bg.Add(tsg.NewStatement().Const("params").Colon().ID("Record").Generic(typeString, "never").Op("=").Values(nil).Semicolon())
		}

		requestStmt := tsg.NewStatement()
		requestStmt.Const("_request").Colon().ID("BatchRequest").Op("=")
		requestStmt.Values(func(vg *tsg.Group) {
			rpcRequestStmt := tsg.NewStatement()
			rpcRequestStmt.ID("rpcRequest").Colon()
			rpcRequestStmt.Values(func(rg *tsg.Group) {
				rg.Add(tsg.NewStatement().ObjectField("jsonrpc", tsg.NewStatement().Lit("2.0")))
				rg.Add(tsg.NewStatement().ObjectField("method", tsg.NewStatement().Lit(model.JSONRPCWireMethod(contract.Name, method.Name))))
				rg.Add(tsg.NewStatement().ObjectField("params", tsg.NewStatement().ID("params")))
				idGen := tsg.NewStatement()
				idGen.ID("id").Colon().ID("this.baseClient.getRpcClient").Call().Dot("generateId").Call()
				rg.Add(idGen)
			})
			vg.Add(rpcRequestStmt)
			vg.Add(tsg.NewStatement().ObjectField("retHandler", tsg.NewStatement().ID(typeUndefined)))
		})
		bg.Add(requestStmt.Semicolon())

		bg.If(
			tsg.NewStatement().ID("callback"),
			func(ig *tsg.Group) {
				results := r.resultsWithoutError(method)
				responseTypeName := r.responseTypeName(contract, method)

				retHandlerStmt := tsg.NewStatement()
				retHandlerStmt.ID("_request.retHandler").Op("=")
				retHandlerFn := tsg.NewStatement()
				arrowParams := tsg.NewStatement()
				arrowParams.Params(func(fg *tsg.Group) {
					fg.Add(tsg.NewStatement().ID(typeError).Colon().ID("Error").Op("|").ID("null"))
					fg.Add(tsg.NewStatement().ID("rpcResponse").Colon().ID("ResponseRPC").Op("|").ID("null"))
				})
				retHandlerFn.Add(arrowParams).Op("=>")
				retHandlerFn.Block(func(hg *tsg.Group) {
					hg.If(
						tsg.NewStatement().ID(typeError).Op("===").ID("null").Op("&&").ID("rpcResponse"),
						func(ig *tsg.Group) {
							ig.If(
								tsg.NewStatement().ID("rpcResponse.error"),
								func(eg *tsg.Group) {
									eg.Add(tsg.NewStatement().Const("rpcErr").Op("=").ID("this").Dot("baseClient").Dot("decodeRPCError").Call(
										tsg.NewStatement().ID("rpcResponse.error"),
									).Semicolon())

									callbackArgs := make([]*tsg.Statement, 0, len(results)+1)
									for range results {
										callbackArgs = append(callbackArgs, tsg.NewStatement().ID("null"))
									}
									callbackArgs = append(callbackArgs, tsg.NewStatement().ID("rpcErr"))
									eg.Add(tsg.NewStatement().ID("callback").Call(callbackArgs...).Semicolon())
								},
							)
							ig.If(
								tsg.NewStatement().ID("rpcResponse").Op("&&").Op("!").ID("rpcResponse.error"),
								func(sg *tsg.Group) {
									if len(results) == 0 {
										sg.Add(tsg.NewStatement().ID("callback").Call(
											tsg.NewStatement().ID("null"),
										).Semicolon())
									} else {
										sg.Add(tsg.NewStatement().Const(nameResult).Colon().ID(responseTypeName).Op("=").ID("rpcResponse.result").Op("as").ID(responseTypeName).Semicolon())

										callbackArgs := []*tsg.Statement{}
										if len(results) == 1 {
											callbackArgs = append(callbackArgs, r.tsRPCSingleResultExpr(contract, method, results[0]))
										} else {
											for _, ret := range results {
												callbackArgs = append(callbackArgs, tsg.NewStatement().ID(nameResult).Dot(ret.Name))
											}
										}
										callbackArgs = append(callbackArgs, tsg.NewStatement().ID("null"))
										sg.Add(tsg.NewStatement().ID("callback").Call(callbackArgs...).Semicolon())
									}
								},
							)
						},
					)
					hg.If(
						tsg.NewStatement().ID(typeError).Op("||").Op("!").ID("rpcResponse"),
						func(eg *tsg.Group) {
							callbackArgs := make([]*tsg.Statement, 0, len(results)+1)
							for range results {
								callbackArgs = append(callbackArgs, tsg.NewStatement().ID("null"))
							}
							callbackArgs = append(callbackArgs, tsg.NewStatement().ID(typeError).Op("||").New("Error").Call(tsg.NewStatement().Lit("unknown error")))
							eg.Add(tsg.NewStatement().ID("callback").Call(callbackArgs...).Semicolon())
						},
					)
				})
				retHandlerStmt.Add(retHandlerFn).Semicolon()
				ig.Add(retHandlerStmt)
			},
		)

		bg.Return(tsg.NewStatement().ID("_request"))
	})
	grp.Add(methodStmt)
	grp.Line()
}

func (r *ClientRenderer) renderJSONRPCBatchMethod(grp *tsg.Group, _contract *model.Contract) {

	grp.Comment("Executes a batch of JSON-RPC requests")
	grp.Comment("@param calls - Array of JSON-RPC calls to execute")

	genericParams := tsg.NewStatement()
	genericParams.Generic("Calls extends readonly JsonRpcCall<JsonRpcParams>[]")

	methodParams := tsg.NewStatement()
	methodParams.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("calls").Colon().ID("Calls"))
	})

	returnType := tsg.NewStatement()
	returnType.ID("MapBatchResult").Generic("Calls")

	methodStmt := tsg.NewStatement()
	methodStmt.AsyncMethodWithGeneric("execBatch", genericParams, methodParams, returnType, func(mg *tsg.Group) {
		mg.Return(
			tsg.NewStatement().
				This().
				Dot("client").
				Dot("callBatch").
				Call(tsg.NewStatement().ID("calls")).
				Op("as").
				ID("MapBatchResult").
				Generic("Calls"),
		)
	})
	grp.Add(methodStmt)
	grp.Line()
}

func (r *ClientRenderer) resultToTypeStatement(method *model.Method, vars []*model.Variable) (st *tsg.Statement) {

	if len(vars) == 0 {
		return tsg.NewStatement().ID("void")
	}

	// r.contract может быть nil при обходе контрактов; обращение к полям вызовет панику.
	if r.contract == nil {
		slog.Error(i18n.Msg("resultToTypeStatement: r.contract is nil"), slog.String("method", method.Name))
		return tsg.NewStatement().ID(typeAny)
	}

	var pkgPath string
	if r.contract != nil {
		pkgPath = r.contract.PkgPath
		if pkgPath == "" {
			slog.Error(i18n.Msg("resultToTypeStatement: contract.PkgPath is empty"), slog.String("contract", r.contract.Name), slog.String("method", method.Name))
			return tsg.NewStatement().ID(typeAny)
		}
	} else {
		slog.Error(i18n.Msg("resultToTypeStatement: r.contract is nil after check"), slog.String("method", method.Name))
		return tsg.NewStatement().ID(typeAny)
	}

	if len(vars) == 1 {
		schema := r.walkVariable(vars[0].Name, pkgPath, vars[0], method.Annotations, false)
		typeStr := schema.typeLink()
		return tsg.TypeFromString(typeStr)
	}

	stmt := tsg.NewStatement()
	stmt.Values(func(grp *tsg.Group) {
		for _, ret := range vars {
			schema := r.walkVariable(ret.Name, pkgPath, ret, method.Annotations, false)
			typeStr := schema.typeLink()
			field := tsg.NewStatement()
			field.ID(ret.Name)
			if model.IsAnnotationSet(r.project, r.contract, method, nil, "nullable") {
				field.Optional()
			}
			field.Colon()
			field.Add(tsg.TypeFromString(typeStr))
			field.Semicolon()
			grp.Add(field)
		}
	})
	return stmt
}
