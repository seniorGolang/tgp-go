// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"os"
	"path"

	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderJSONRPCLibrary() (err error) {

	outDir := r.outDir

	jsonrpcDir := path.Join(outDir, "jsonrpc")
	if err = os.MkdirAll(jsonrpcDir, 0o700); err != nil {
		return fmt.Errorf("failed to create jsonrpc directory: %w", err)
	}

	utilsDir := path.Join(jsonrpcDir, "utils")
	if err = os.MkdirAll(utilsDir, 0o700); err != nil {
		return fmt.Errorf("failed to create jsonrpc/utils directory: %w", err)
	}

	if err = r.renderJSONRPCClient(jsonrpcDir); err != nil {
		return
	}

	if err = r.renderJSONRPCUtils(utilsDir); err != nil {
		return
	}

	if err = r.renderJSONRPCTSUtils(utilsDir); err != nil {
		return
	}

	return
}

func (r *ClientRenderer) renderJSONRPCClient(jsonrpcDir string) (err error) {

	file := tsg.NewFile()
	file.ImportType("./utils/jsonrpc", "RequestRPC", "ResponseRPC", "ID")
	file.ImportType("../options", "ClientOptions")
	file.ImportNamed("../headers", "buildClientHeaders")

	classStmt := tsg.NewStatement().
		Comment("JSON-RPC 2.0 client").
		Export().Class("JsonRpcClient", func(grp *tsg.Group) {
		// Приватные поля
		grp.Add(tsg.NewStatement().Private().ID("options").Colon().ID("ClientOptions").Semicolon())
		grp.Line()

		// Конструктор
		constructor := tsg.NewStatement().
			ID("constructor").
			Params(func(cg *tsg.Group) {
				cg.Add(tsg.NewStatement().ID("options").Colon().ID("ClientOptions"))
			}).
			Block(func(cg *tsg.Group) {
				cg.Add(tsg.NewStatement().
					This().
					Dot("options").
					Op("=").
					ID("options").
					Semicolon())
			})
		grp.Add(constructor)
		grp.Line()

		// Публичный метод для генерации ID
		generateIDStmt := tsg.NewStatement().
			ID("generateId").
			Call().
			Colon().
			ID(typeString).
			Block(func(mg *tsg.Group) {
				mg.Add(tsg.NewStatement().
					Return().
					ID("this.options.idGeneratorFn").
					Op("?").
					ID("this.options.idGeneratorFn").
					Call().
					Colon().
					ID("crypto").
					Dot("randomUUID").
					Call().
					Semicolon())
			})
		grp.Add(generateIDStmt)
		grp.Line()

		// Метод call для одиночного запроса
		callMethodParams := tsg.NewStatement().
			Params(func(pg *tsg.Group) {
				pg.Add(tsg.NewStatement().ID("method").Colon().ID(typeString))
				pg.Add(tsg.NewStatement().ID("params").Optional().Colon().ID(typeAny))
			})
		callReturnType := tsg.NewStatement().
			ObjectLiteral(func(og *tsg.Group) {
				og.Add(tsg.NewStatement().ObjectField("type", tsg.NewStatement().Union(
					tsg.NewStatement().Lit("success"),
					tsg.NewStatement().Lit(typeError),
				)))
				og.Add(tsg.NewStatement().OptionalField(nameResult, tsg.NewStatement().Any()))
				og.Add(tsg.NewStatement().OptionalField(typeError, tsg.NewStatement().Any()))
			})
		callMethod := tsg.NewStatement().
			Comment("Calls a JSON-RPC method").
			AsyncMethodWithParams("exec", callMethodParams, callReturnType, func(bg *tsg.Group) {
				idGen := tsg.NewStatement().
					Const("id").
					Op("=").
					ID("this.options.idGeneratorFn").
					Op("?").
					ID("this.options.idGeneratorFn").
					Call().
					Colon().
					ID("crypto").
					Dot("randomUUID").
					Call().
					Semicolon()
				bg.Add(idGen)

				requestStmt := tsg.NewStatement().
					Const("request").
					Colon().
					ID(typeAny).
					Op("=").
					ObjectLiteral(func(og *tsg.Group) {
						og.Add(tsg.NewStatement().ObjectField("jsonrpc", tsg.NewStatement().Lit("2.0")))
						og.Add(tsg.NewStatement().ObjectField("method", tsg.NewStatement().ID("method")))
						og.Add(tsg.NewStatement().ObjectField("params", tsg.NewStatement().ID("params").Op("||").ObjectLiteral(nil)))
						og.Add(tsg.NewStatement().ObjectField("id", tsg.NewStatement().ID("id")))
					}).
					Semicolon()
				bg.Add(requestStmt)

				headersVar := tsg.NewStatement().
					Const("headers").
					Colon().
					ID("Record").
					Generic(typeString, typeString).
					Op("=").
					Await(tsg.NewStatement().ID("buildClientHeaders").Call(tsg.NewStatement().This().Dot("options"))).
					Semicolon()
				bg.Add(headersVar)

				// Выполняем запрос
				fetchCall := tsg.NewStatement().
					Const("response").
					Colon().
					ID("Response").
					Op("=").
					Await(
						tsg.NewStatement().
							ID("fetch").
							Call(
								tsg.NewStatement().ID("this.options.url"),
								tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
									og.Add(tsg.NewStatement().ObjectField("method", tsg.NewStatement().Lit(methodPOST)))
									og.Add(tsg.NewStatement().ObjectField("headers", tsg.NewStatement().ObjectLiteral(func(hg *tsg.Group) {
										hg.Add(tsg.NewStatement().ObjectField("Content-Type", tsg.NewStatement().Lit("application/json")))
										hg.Add(tsg.NewStatement().Spread(tsg.NewStatement().ID("headers")))
									})))
									og.Add(tsg.NewStatement().ObjectField("body", tsg.NewStatement().ID("JSON.stringify").Call(tsg.NewStatement().ID("request"))))
								}),
							),
					).
					Semicolon()
				bg.Add(fetchCall)

				bg.If(
					tsg.NewStatement().Op("!").ID("response").Dot("ok"),
					func(ig *tsg.Group) {
						ig.Throw(tsg.NewStatement().
							New("Error").
							Call(tsg.NewStatement().Lit("HTTP error: ").Op("+").ID("response.status")))
					},
				)

				parse := tsg.NewStatement().
					Const("data").
					Colon().
					ID(typeAny).
					Op("=").
					Await(tsg.NewStatement().ID("response").Dot("json").Call()).
					Semicolon()
				bg.Add(parse)

				bg.If(
					tsg.NewStatement().ID("data.error"),
					func(ig *tsg.Group) {
						ig.Return(tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
							og.Add(tsg.NewStatement().ObjectField("type", tsg.NewStatement().Lit(typeError)))
							og.Add(tsg.NewStatement().ObjectField(typeError, tsg.NewStatement().ID("data.error")))
						}))
					},
				)

				bg.Return(tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
					og.Add(tsg.NewStatement().ObjectField("type", tsg.NewStatement().Lit("success")))
					og.Add(tsg.NewStatement().ObjectField(nameResult, tsg.NewStatement().ID("data.result")))
				}))
			})
		grp.Add(callMethod)
		grp.Line()

		// Метод callBatch для batch запросов
		batchParams := tsg.NewStatement().
			Params(func(pg *tsg.Group) {
				pg.Add(tsg.NewStatement().ID("requests").Colon().ReadonlyArray(tsg.NewStatement().ID("RequestRPC")))
			})
		batchReturnType := tsg.NewStatement().
			Nullable(tsg.NewStatement().ID("Map").Generic("ID", "ResponseRPC"))
		batchMethod := tsg.NewStatement().
			Comment("Calls multiple JSON-RPC methods in a single request").
			Comment("@returns Map with response ID as key and ResponseRPC as value, or null on error").
			AsyncMethodWithParams("callBatch", batchParams, batchReturnType, func(bg *tsg.Group) {
				headersVar := tsg.NewStatement().
					Const("headers").
					Colon().
					ID("Record").
					Generic(typeString, typeString).
					Op("=").
					Await(tsg.NewStatement().ID("buildClientHeaders").Call(tsg.NewStatement().This().Dot("options"))).
					Semicolon()
				bg.Add(headersVar)

				// Выполняем запрос
				fetchCall := tsg.NewStatement().
					Const("response").
					Colon().
					ID("Response").
					Op("=").
					Await(
						tsg.NewStatement().
							ID("fetch").
							Call(
								tsg.NewStatement().ID("this.options.url"),
								tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
									og.Add(tsg.NewStatement().ObjectField("method", tsg.NewStatement().Lit(methodPOST)))
									og.Add(tsg.NewStatement().ObjectField("headers", tsg.NewStatement().ObjectLiteral(func(hg *tsg.Group) {
										hg.Add(tsg.NewStatement().ObjectField("Content-Type", tsg.NewStatement().Lit("application/json")))
										hg.Add(tsg.NewStatement().Spread(tsg.NewStatement().ID("headers")))
									})))
									og.Add(tsg.NewStatement().ObjectField("body", tsg.NewStatement().ID("JSON.stringify").Call(tsg.NewStatement().ID("requests"))))
								}),
							),
					).
					Semicolon()
				bg.Add(fetchCall)

				bg.If(
					tsg.NewStatement().Op("!").ID("response").Dot("ok"),
					func(ig *tsg.Group) {
						ig.Return(tsg.NewStatement().ID("null"))
					},
				)

				parse := tsg.NewStatement().
					Const("data").
					Colon().
					ID("ResponseRPC[]").
					Op("=").
					Await(tsg.NewStatement().ID("response").Dot("json").Call()).
					Semicolon()
				bg.Add(parse)

				mapStmt := tsg.NewStatement().
					Const("responseMap").
					Colon().
					ID("Map").
					Generic("ID", "ResponseRPC").
					Op("=").
					New("Map").
					Generic("ID", "ResponseRPC").
					Call().
					Semicolon()
				bg.Add(mapStmt)

				// Заполняем Map
				forLoop := tsg.NewStatement().
					ForOf("resp", "data", func(fg *tsg.Group) {
						fg.Add(tsg.NewStatement().
							ID("responseMap").
							Dot("set").
							Call(
								tsg.NewStatement().ID("resp.id"),
								tsg.NewStatement().ID("resp"),
							))
					})
				bg.Add(forLoop)

				bg.Return(tsg.NewStatement().ID("responseMap"))
			})
		grp.Add(batchMethod)
	})
	file.Add(classStmt)
	file.Line()

	file.GenerateImports()

	return file.Save(path.Join(jsonrpcDir, "client.ts"))
}

func (r *ClientRenderer) renderJSONRPCUtils(utilsDir string) (err error) {

	file := tsg.NewFile()
	// Тип ID
	stmt1 := tsg.NewStatement()
	stmt1.Export().Type("ID").Op("=").ID(typeString).Op("|").ID(typeNumber).Semicolon()
	file.Add(stmt1)
	file.Line()

	// Тип JsonRpcParams
	stmt2 := tsg.NewStatement()
	stmt2.Export().Type("JsonRpcParams").Op("=").ID(typeAny).Semicolon()
	file.Add(stmt2)
	file.Line()

	// Тип RequestRPC
	stmt3 := tsg.NewStatement()
	stmt3.Export().Interface("RequestRPC", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().ID("jsonrpc").Colon().Lit("2.0").Semicolon())
		grp.Add(tsg.NewStatement().ID("method").Colon().ID(typeString).Semicolon())
		grp.Add(tsg.NewStatement().ID("params").Optional().Colon().ID("JsonRpcParams").Semicolon())
		grp.Add(tsg.NewStatement().ID("id").Colon().ID("ID").Semicolon())
	})
	file.Add(stmt3)
	file.Line()

	// Тип ResponseRPC
	stmt4 := tsg.NewStatement()
	stmt4.Export().Interface("ResponseRPC", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().ID("jsonrpc").Colon().Lit("2.0").Semicolon())
		grp.Add(tsg.NewStatement().ID(nameResult).Optional().Colon().ID(typeAny).Semicolon())
		grp.Add(tsg.NewStatement().ID(typeError).Optional().Colon().ID(typeAny).Semicolon())
		grp.Add(tsg.NewStatement().ID("id").Colon().ID("ID").Semicolon())
	})
	file.Add(stmt4)
	file.Line()

	// Тип JsonRpcCall
	stmt5 := tsg.NewStatement()
	stmt5.Export().Type("JsonRpcCall")
	stmt5.Generic("Params extends JsonRpcParams")
	stmt5.Op("=")
	stmt5.Block(func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().ID("jsonrpc").Colon().Lit("2.0").Semicolon())
		grp.Add(tsg.NewStatement().ID("method").Colon().ID(typeString).Semicolon())
		grp.Add(tsg.NewStatement().ID("params").Optional().Colon().ID("Params").Semicolon())
		grp.Add(tsg.NewStatement().ID("id").Colon().ID("ID").Semicolon())
	})
	stmt5.Semicolon()
	file.Add(stmt5)
	file.Line()

	file.GenerateImports()

	return file.Save(path.Join(utilsDir, "jsonrpc.ts"))
}

func (r *ClientRenderer) renderJSONRPCTSUtils(utilsDir string) (err error) {

	file := tsg.NewFile()
	// Импорты
	file.ImportType("./jsonrpc", "JsonRpcCall", "JsonRpcParams")

	// Тип MapBatchResult
	stmt := tsg.NewStatement()
	stmt.Export().Type("MapBatchResult")
	stmt.Generic("Calls extends readonly JsonRpcCall<JsonRpcParams>[]")
	stmt.Op("=")
	// Mapped type для MapBatchResult - используем TypeFromString, так как это сложная конструкция типа
	mapResultStmt := tsg.TypeFromString("{ readonly [K in keyof Calls]: Calls[K] extends JsonRpcCall<infer Params> ? JsonRpcCall<Params> : never }")
	stmt.Add(mapResultStmt)
	stmt.Semicolon()
	file.Add(stmt)
	file.Line()

	file.GenerateImports()

	return file.Save(path.Join(utilsDir, "ts.ts"))
}
