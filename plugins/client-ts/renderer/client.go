// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/internal/model"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderClient() (err error) {

	outDir := r.outDir
	file := tsg.NewFile()
	// Импорты
	file.ImportType("./options", "ClientOptions")
	if r.clientIdentity {
		file.ImportNamed("./identity", "resolveDefaultClientName")
	}
	file.ImportNamed("./headers", "buildClientHeaders")
	if r.HasJSONRPC() {
		file.ImportNamed("./error", "defaultErrorDecoder")
	}
	if r.HasHTTP() {
		file.ImportNamed("./error", "defaultHTTPErrorDecoder")
	}
	if r.HasJSONRPC() {
		file.ImportNamed("./jsonrpc/client", "JsonRpcClient")
		file.ImportType("./batch", "BatchRequest", "RpcCallback")
		file.ImportType("./jsonrpc/utils/jsonrpc", "RequestRPC", "ResponseRPC", "ID")
	}

	for _, contractName := range r.ContractKeys() {
		contract := r.FindContract(contractName)
		if contract == nil {
			continue
		}
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) ||
			model.ContractHasWS(r.project, contract) ||
			model.ContractHasSSE(r.project, contract) {
			clientClassName := contract.Name + "Client"
			file.ImportNamed("./"+r.tsFileName(contract), clientClassName)
		}
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			httpClientClassName := contract.Name + "HTTPClient"
			file.ImportNamed("./"+r.tsFileName(contract)+"-http", httpClientClassName)
		}
	}

	file.GenerateImports()
	file.Line()

	file.Add(r.renderBaseClientClass())
	file.Line()

	file.Add(r.renderNewClientFunction())
	file.Line()

	err = file.Save(path.Join(outDir, "client.ts"))
	return
}

func (r *ClientRenderer) renderBaseClientClass() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Comment("Base client for all services")
	stmt.Export().Class("Client", func(grp *tsg.Group) {
		// Приватные поля
		if r.HasJSONRPC() {
			grp.Add(tsg.NewStatement().Private().ID("rpcClient").Colon().ID("JsonRpcClient").Semicolon())
		}
		grp.Add(tsg.NewStatement().Private().ID("endpoint").Colon().ID(typeString).Semicolon())
		grp.Add(tsg.NewStatement().Private().ID("options").Colon().ID("ClientOptions").Semicolon())
		grp.Line()

		// Конструктор
		constructor := tsg.NewStatement()
		constructor.ID("constructor")
		constructor.Params(func(cg *tsg.Group) {
			cg.Add(tsg.NewStatement().ID("endpoint").Colon().ID(typeString))
			cg.Add(tsg.NewStatement().ID("opts").Optional().Colon().ID("Partial").Generic("ClientOptions"))
		})
		constructor.Block(func(cg *tsg.Group) {
			cg.Add(tsg.NewStatement().This().Dot("endpoint").Op("=").ID("endpoint").Semicolon())

			// Объединяем опции
			cg.Add(tsg.NewStatement().Const("defaultOptions").Colon().ID("ClientOptions").Op("=").Values(func(og *tsg.Group) {
				og.Add(tsg.NewStatement().ID("url").Colon().ID("endpoint"))
				if r.clientIdentity {
					og.Add(tsg.NewStatement().ID("clientName").Colon().ID("resolveDefaultClientName").Call())
				}
				defaultFn := tsg.NewStatement()
				defaultFn.ArrowFunc().Block(func(bg *tsg.Group) {
					bg.Return(tsg.NewStatement().ID("crypto").Dot("randomUUID").Call())
				})
				og.Add(tsg.NewStatement().ID("idGeneratorFn").Colon().Add(defaultFn))
				if r.HasJSONRPC() {
					og.Add(tsg.NewStatement().ID("errorDecoder").Colon().ID("defaultErrorDecoder"))
				}
				if r.HasHTTP() {
					og.Add(tsg.NewStatement().ID("httpErrorDecoder").Colon().ID("defaultHTTPErrorDecoder"))
				}
			}).Semicolon())

			cg.Add(tsg.NewStatement().This().Dot("options").Op("=").ID("Object").Dot("assign").Call(
				tsg.NewStatement().Values(func(_vg *tsg.Group) {}),
				tsg.NewStatement().ID("defaultOptions"),
				tsg.NewStatement().ID("opts").Op("||").Values(func(_vg *tsg.Group) {}),
			).Semicolon())

			if r.HasJSONRPC() {
				cg.Add(tsg.NewStatement().This().Dot("rpcClient").Op("=").New("JsonRpcClient").Call(
					tsg.NewStatement().This().Dot("options"),
				).Semicolon())
			}
		})
		grp.Add(constructor)
		grp.Line()

		// Публичные геттеры для доступа к внутренним ресурсам
		if r.HasJSONRPC() {
			grp.Add(tsg.NewStatement().ID("getRpcClient").Call().Colon().ID("JsonRpcClient").Block(func(mg *tsg.Group) {
				mg.Return(tsg.NewStatement().This().Dot("rpcClient"))
			}))
			grp.Line()
		}

		grp.Add(tsg.NewStatement().ID("getEndpoint").Call().Colon().ID(typeString).Block(func(mg *tsg.Group) {
			mg.Return(tsg.NewStatement().This().Dot("endpoint"))
		}))
		grp.Line()

		grp.Add(r.renderResolveURLMethod())
		grp.Line()
		grp.Add(r.renderJoinEndpointPathMethod())
		grp.Line()
		grp.Add(r.renderSplitEndpointPathMethod())
		grp.Line()
		grp.Add(r.renderEndpointPathOverlapMethod())
		grp.Line()
		grp.Add(r.renderEndpointPathSegmentsEqualMethod())
		grp.Line()

		if r.HasJSONRPC() {
			grp.Add(r.renderDecodeRPCErrorMethod())
			grp.Line()
		}

		if r.HasHTTP() {
			grp.Add(r.renderDecodeHTTPErrorMethod())
			grp.Line()
		}

		// Вспомогательный метод для получения заголовков (поддерживает статичные и динамические)
		grp.Add(tsg.NewStatement().
			Comment("Gets headers for the request, supporting both static headers and dynamic header functions").
			AsyncMethodWithParams("getHeaders", nil, tsg.NewStatement().ID("Record").Generic(typeString, typeString), func(mg *tsg.Group) {
				mg.Return(tsg.NewStatement().ID("buildClientHeaders").Call(tsg.NewStatement().This().Dot("options")))
			}))
		grp.Line()

		// Метод Batch для выполнения batch запросов
		if r.HasJSONRPC() {
			grp.Add(r.renderBatchMethod())
			grp.Line()
		}

		for _, contractName := range r.ContractKeys() {
			contract := r.FindContract(contractName)
			if contract == nil {
				continue
			}
			if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) ||
				model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) ||
				model.ContractHasWS(r.project, contract) ||
				model.ContractHasSSE(r.project, contract) {
				grp.Add(r.renderContractClientMethod(contract))
				grp.Line()
			}
		}
	})
	return stmt
}

func (r *ClientRenderer) renderNewClientFunction() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Comment("Creates a new base client")
	stmt.Comment("@param endpoint - Server endpoint URL")
	stmt.Comment("@param opts - Optional client configuration")
	stmt.Export().Func("newClient")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("endpoint").Colon().ID(typeString))
		pg.Add(tsg.NewStatement().ID("opts").Optional().Colon().ID("Partial").Generic("ClientOptions"))
	})
	stmt.Colon()
	stmt.ID("Client")
	stmt.Block(func(bg *tsg.Group) {
		bg.Return(tsg.NewStatement().New("Client").Call(
			tsg.NewStatement().ID("endpoint"),
			tsg.NewStatement().ID("opts"),
		))
	})
	return stmt
}

func (r *ClientRenderer) renderContractClientMethod(contract *model.Contract) *tsg.Statement {

	stmt := tsg.NewStatement()
	fileName := r.tsFileName(contract)
	methodName := r.fileNameToMethodName(fileName)
	clientClassName := contract.Name + "Client"

	// Для JSON-RPC контрактов
	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) ||
		model.ContractHasWS(r.project, contract) ||
		model.ContractHasSSE(r.project, contract) {
		stmt.ID(methodName)
		stmt.Params(func(_pg *tsg.Group) {
			// Пустые параметры
		})
		stmt.Colon()
		stmt.ID(clientClassName)
		stmt.Block(func(bg *tsg.Group) {
			bg.Return(tsg.NewStatement().New(clientClassName).Call(tsg.NewStatement().This()))
		})
	}

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
		httpMethodName := methodName + suffixHTTP
		httpClientClassName := contract.Name + "HTTPClient"
		stmt.Line()
		stmt.ID(httpMethodName)
		stmt.Params(func(_pg *tsg.Group) {
			// Пустые параметры
		})
		stmt.Colon()
		stmt.ID(httpClientClassName)
		stmt.Block(func(bg *tsg.Group) {
			bg.Return(tsg.NewStatement().New(httpClientClassName).Call(tsg.NewStatement().This()))
		})
	}

	return stmt
}

func (r *ClientRenderer) renderBatchMethod() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Comment("Executes a batch of JSON-RPC requests")
	methodParams := tsg.NewStatement()
	methodParams.Params(func(pg *tsg.Group) {
		// Параметр: requests: BatchRequest[]
		pg.Add(tsg.NewStatement().ID("requests").Colon().ID("BatchRequest").Array(nil))
	})
	returnType := tsg.NewStatement()
	returnType.Void()
	stmt.AsyncMethodWithParams("batch", methodParams, returnType, func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().Const("rpcRequests").Colon().ID("RequestRPC").Array(nil).Op("=").Index(nil).Semicolon())
		bg.Add(tsg.NewStatement().Const("callbacks").Colon().ID("Map").Generic("ID", "RpcCallback").Op("=").New("Map").Call().Semicolon())
		bg.Add(tsg.NewStatement().Const("requestIDs").Colon().ID("Set").Generic("ID").Op("=").New("Set").Call().Semicolon())
		bg.Add(tsg.NewStatement().ForOf("request", "requests", func(fbg *tsg.Group) {
			fbg.If(tsg.NewStatement().ID("requestIDs").Dot("has").Call(tsg.NewStatement().ID("request").Dot("rpcRequest").Dot("id")), func(ig *tsg.Group) {
				ig.Throw(tsg.NewStatement().New("Error").Call(
					tsg.NewStatement().Lit("duplicate batch request id: ").Op("+").ID("String").Call(tsg.NewStatement().ID("request").Dot("rpcRequest").Dot("id")),
				))
			})
			fbg.Add(tsg.NewStatement().ID("requestIDs").Dot("add").Call(tsg.NewStatement().ID("request").Dot("rpcRequest").Dot("id")).Semicolon())
			fbg.Add(tsg.NewStatement().ID("rpcRequests").Dot("push").Call(tsg.NewStatement().ID("request").Dot("rpcRequest")).Semicolon())
			fbg.If(
				tsg.NewStatement().ID("request").Dot("retHandler").Op("!==").ID(typeUndefined),
				func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().ID("callbacks").Dot("set").Call(
						tsg.NewStatement().ID("request").Dot("rpcRequest").Dot("id"),
						tsg.NewStatement().ID("request").Dot("retHandler"),
					).Semicolon())
				},
			)
		}))

		responseType := tsg.NewStatement()
		responseType.Nullable(tsg.NewStatement().ID("Map").Generic("ID", "ResponseRPC"))
		bg.Add(tsg.NewStatement().Const("rpcResponses").Colon().Add(responseType).Op("=").Await(
			tsg.NewStatement().This().Dot("rpcClient").Dot("callBatch").Call(tsg.NewStatement().ID("rpcRequests")),
		).Semicolon())

		// Обработка ошибки или null
		bg.If(
			tsg.NewStatement().ID("rpcResponses").Op("===").ID("null"),
			func(ig *tsg.Group) {
				ig.Add(tsg.NewStatement().ForOf("callback", "callbacks.values()", func(fbg *tsg.Group) {
					fbg.Add(tsg.NewStatement().ID("callback").Call(
						tsg.NewStatement().New("Error").Call(tsg.NewStatement().Lit("batch request failed")),
						tsg.NewStatement().ID("null"),
					).Semicolon())
				}))
				ig.Return()
			},
		)

		forLoopStmt := tsg.NewStatement()
		forLoopStmt.ForOf("[id, response]", "rpcResponses", func(fbg *tsg.Group) {
			fbg.Add(tsg.NewStatement().Const("callback").Colon().ID("RpcCallback").Op("|").ID(typeUndefined).Op("=").ID("callbacks").Dot("get").Call(tsg.NewStatement().ID("id")).Semicolon())
			fbg.If(
				tsg.NewStatement().ID("callback"),
				func(cig *tsg.Group) {
					cig.If(tsg.NewStatement().ID("response").Op("!==").ID("null"), func(dg *tsg.Group) {
						dg.Add(tsg.NewStatement().ID("callbacks").Dot("delete").Call(tsg.NewStatement().ID("id")).Semicolon())
					})
					cig.If(
						tsg.NewStatement().ID("response").Op("!=").ID("null").Op("&&").ID("response").Dot(typeError),
						func(eg *tsg.Group) {
							eg.Add(tsg.NewStatement().Const("rpcErr").Op("=").ID("this").Dot("decodeRPCError").Call(
								tsg.NewStatement().ID("response.error"),
							).Semicolon())
							eg.Add(tsg.NewStatement().ID("callback").Call(
								tsg.NewStatement().ID("rpcErr"),
								tsg.NewStatement().ID("response"),
							).Semicolon())
						},
					)
					cig.If(
						tsg.NewStatement().ID("response").Op("!=").ID("null").Op("&&").Op("!").ID("response").Dot(typeError),
						func(eg *tsg.Group) {
							eg.Add(tsg.NewStatement().ID("callback").Call(
								tsg.NewStatement().ID("null"),
								tsg.NewStatement().ID("response"),
							).Semicolon())
						},
					)
				},
			)
		})
		bg.Add(forLoopStmt)
		bg.Add(tsg.NewStatement().ForOf("callback", "callbacks.values()", func(fbg *tsg.Group) {
			fbg.Add(tsg.NewStatement().ID("callback").Call(
				tsg.NewStatement().New("Error").Call(tsg.NewStatement().Lit("batch response missing")),
				tsg.NewStatement().ID("null"),
			).Semicolon())
		}))
	})
	return stmt
}

func (r *ClientRenderer) renderResolveURLMethod() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Comment("Resolves a declared route against the endpoint path with overlapping segment collapse")
	stmt.ID("resolveURL")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("path").Colon().ID(typeString))
		pg.Add(tsg.NewStatement().ID("query").Optional().Colon().ID("Record").Generic(typeString, typeString))
	})
	stmt.Colon().ID(typeString)
	stmt.Block(func(mg *tsg.Group) {
		mg.Add(tsg.NewStatement().Const("url").Op("=").New("URL").Call(tsg.NewStatement().This().Dot("endpoint")).Semicolon())
		mg.Add(tsg.NewStatement().Const("route").Op("=").ID("path").Dot("startsWith").Call(tsg.NewStatement().Lit("/")).Op("?").ID("path").Op(":").Paren(tsg.NewStatement().Lit("/").Op("+").ID("path")).Semicolon())
		mg.Add(tsg.NewStatement().ID("url").Dot("pathname").Op("=").ID("this").Dot("joinEndpointPath").Call(tsg.NewStatement().ID("url").Dot("pathname"), tsg.NewStatement().ID("route")).Semicolon())
		mg.Add(tsg.NewStatement().ID("url").Dot("search").Op("=").Lit("").Semicolon())
		mg.If(tsg.NewStatement().ID("query"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().ForOf("[key, value]", "Object.entries(query)", func(fg *tsg.Group) {
				fg.Add(tsg.NewStatement().ID("url").Dot("searchParams").Dot("set").Call(tsg.NewStatement().ID("key"), tsg.NewStatement().ID("value")).Semicolon())
			}))
		})
		mg.Return(tsg.NewStatement().ID("url").Dot("toString").Call())
	})
	return stmt
}

func (r *ClientRenderer) renderJoinEndpointPathMethod() *tsg.Statement {

	params := tsg.NewStatement().Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("basePath").Colon().ID(typeString))
		pg.Add(tsg.NewStatement().ID("routePath").Colon().ID(typeString))
	})
	stmt := tsg.NewStatement()
	stmt.Private()
	stmt.MethodTyped("joinEndpointPath", params, tsg.NewStatement().ID(typeString), func(mg *tsg.Group) {
		mg.Add(tsg.NewStatement().Const("base").Op("=").ID("this").Dot("splitEndpointPath").Call(tsg.NewStatement().ID("basePath")).Semicolon())
		mg.Add(tsg.NewStatement().Const("route").Op("=").ID("this").Dot("splitEndpointPath").Call(tsg.NewStatement().ID("routePath")).Semicolon())
		mg.Add(tsg.NewStatement().Const("overlap").Op("=").ID("this").Dot("endpointPathOverlap").Call(tsg.NewStatement().ID("base"), tsg.NewStatement().ID("route")).Semicolon())
		mg.Add(tsg.NewStatement().Const("out").Op("=").ID("base").Dot("slice").Call(tsg.NewStatement().Lit(0), tsg.NewStatement().ID("base").Dot("length").Op("-").ID("overlap")).Dot("concat").Call(tsg.NewStatement().ID("route")).Semicolon())
		mg.If(tsg.NewStatement().ID("out").Dot("length").Op("===").Lit(0), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().Lit("/"))
		})
		mg.Return(tsg.NewStatement().Lit("/").Op("+").ID("out").Dot("join").Call(tsg.NewStatement().Lit("/")))
	})
	return stmt
}

func (r *ClientRenderer) renderSplitEndpointPathMethod() *tsg.Statement {

	params := tsg.NewStatement().Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("raw").Colon().ID(typeString))
	})
	stmt := tsg.NewStatement()
	stmt.Private()
	stmt.MethodTyped("splitEndpointPath", params, tsg.NewStatement().ID(typeString).Array(nil), func(mg *tsg.Group) {
		mg.Add(tsg.NewStatement().Const("segments").Colon().ID(typeString).Array(nil).Op("=").ArrayLit().Semicolon())
		mg.Add(tsg.NewStatement().ForOf("part", "raw.split(\"/\")", func(fg *tsg.Group) {
			fg.If(tsg.NewStatement().ID("part").Op("===").Lit(""), func(ig *tsg.Group) {
				ig.Add(tsg.NewStatement().Continue())
			})
			fg.Add(tsg.NewStatement().ID("segments").Dot("push").Call(tsg.NewStatement().ID("part")).Semicolon())
		}))
		mg.Return(tsg.NewStatement().ID("segments"))
	})
	return stmt
}

func (r *ClientRenderer) renderEndpointPathOverlapMethod() *tsg.Statement {

	params := tsg.NewStatement().Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("base").Colon().ID(typeString).Array(nil))
		pg.Add(tsg.NewStatement().ID("route").Colon().ID(typeString).Array(nil))
	})
	stmt := tsg.NewStatement()
	stmt.Private()
	stmt.MethodTyped("endpointPathOverlap", params, tsg.NewStatement().ID(typeNumber), func(mg *tsg.Group) {
		mg.Add(tsg.NewStatement().ID("let max = base.length").Semicolon())
		mg.If(tsg.NewStatement().ID("route").Dot("length").Op("<").ID("max"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().ID("max = route.length").Semicolon())
		})
		mg.Add(tsg.NewStatement().For(
			tsg.NewStatement().ID("let n = max"),
			tsg.NewStatement().ID("n >= 1"),
			tsg.NewStatement().ID("n--"),
			func(fg *tsg.Group) {
				fg.If(tsg.NewStatement().ID("this").Dot("endpointPathSegmentsEqual").Call(
					tsg.NewStatement().ID("base").Dot("slice").Call(tsg.NewStatement().ID("base").Dot("length").Op("-").ID("n")),
					tsg.NewStatement().ID("route").Dot("slice").Call(tsg.NewStatement().Lit(0), tsg.NewStatement().ID("n")),
				), func(ig *tsg.Group) {
					ig.Return(tsg.NewStatement().ID("n"))
				})
			},
		))
		mg.Return(tsg.NewStatement().Lit(0))
	})
	return stmt
}

func (r *ClientRenderer) renderEndpointPathSegmentsEqualMethod() *tsg.Statement {

	params := tsg.NewStatement().Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("left").Colon().ID(typeString).Array(nil))
		pg.Add(tsg.NewStatement().ID("right").Colon().ID(typeString).Array(nil))
	})
	stmt := tsg.NewStatement()
	stmt.Private()
	stmt.MethodTyped("endpointPathSegmentsEqual", params, tsg.NewStatement().ID("boolean"), func(mg *tsg.Group) {
		mg.If(tsg.NewStatement().ID("left").Dot("length").Op("!==").ID("right").Dot("length"), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().ID("false"))
		})
		mg.Add(tsg.NewStatement().For(
			tsg.NewStatement().ID("let i = 0"),
			tsg.NewStatement().ID("i < left.length"),
			tsg.NewStatement().ID("i++"),
			func(fg *tsg.Group) {
				fg.If(tsg.NewStatement().ID("left").Index(tsg.NewStatement().ID("i")).Op("!==").ID("right").Index(tsg.NewStatement().ID("i")), func(ig *tsg.Group) {
					ig.Return(tsg.NewStatement().ID("false"))
				})
			},
		))
		mg.Return(tsg.NewStatement().ID("true"))
	})
	return stmt
}

func (r *ClientRenderer) renderDecodeRPCErrorMethod() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Comment("Decodes JSON-RPC error object from response")
	stmt.ID("decodeRPCError")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("errData").Colon().ID("unknown"))
	})
	stmt.Colon().ID("Error")
	stmt.Block(func(mg *tsg.Group) {
		mg.Return(tsg.NewStatement().This().Dot("options").Dot("errorDecoder").Bang().Call(tsg.NewStatement().ID("errData")))
	})
	return stmt
}

func (r *ClientRenderer) renderDecodeHTTPErrorMethod() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Comment("Decodes REST HTTP error response body")
	stmt.ID("decodeHTTPError")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("statusCode").Colon().ID(typeNumber))
		pg.Add(tsg.NewStatement().ID("contentType").Colon().ID(typeString))
		pg.Add(tsg.NewStatement().ID("body").Colon().ID(typeString))
	})
	stmt.Colon().ID("Error")
	stmt.Block(func(mg *tsg.Group) {
		mg.Return(tsg.NewStatement().This().Dot("options").Dot("httpErrorDecoder").Bang().Call(
			tsg.NewStatement().ID("statusCode"),
			tsg.NewStatement().ID("contentType"),
			tsg.NewStatement().ID("body"),
		))
	})
	return stmt
}
