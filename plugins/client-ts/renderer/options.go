// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderClientOptions() (err error) {

	outDir := r.outDir
	file := tsg.NewFile()
	stmt := tsg.NewStatement()
	if r.HasJSONRPC() {
		file.ImportType("./error", "ErrorDecoder")
	}
	if r.HasHTTP() {
		file.ImportType("./error", "HTTPErrorDecoder")
	}
	stmt.Export().Type("ClientOptions")
	stmt.Op("=")
	stmt.Block(func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().ID("url").Colon().ID(typeString).Semicolon())
		if r.clientIdentity {
			grp.Add(tsg.NewStatement().ID("clientName").Colon().ID(typeString).Semicolon())
		}
		if r.HasJSONRPC() {
			grp.Add(tsg.NewStatement().ID("errorDecoder").Optional().Colon().ID("ErrorDecoder").Semicolon())
		}
		if r.HasHTTP() {
			grp.Add(tsg.NewStatement().ID("httpErrorDecoder").Optional().Colon().ID("HTTPErrorDecoder").Semicolon())
		}
		// Поддержка статичных заголовков и функции для динамических заголовков
		recordType := tsg.NewStatement().ID("Record").Generic(typeString, typeString)
		headersType := tsg.NewStatement()
		headersType.Add(recordType)
		headersType.Op("|")
		// Функция должна быть в скобках в union типе
		headersFnType := tsg.NewStatement()
		promiseRecordType := tsg.NewStatement().ID("Promise").Generic("Record<string, string>")
		// Создаем функцию в скобках: (() => Record<string, string> | Promise<Record<string, string>>)
		headersFnType.Op("(").Params(func(_fg *tsg.Group) {}).Op("=>").Add(recordType).Op("|").Add(promiseRecordType).Op(")")
		headersType.Add(headersFnType)
		grp.Add(tsg.NewStatement().ID("headers").Optional().Colon().Add(headersType).Semicolon())
		fnType := tsg.NewStatement()
		fnType.Params(func(_fg *tsg.Group) {}).Op("=>").ID(typeString)
		grp.Add(tsg.NewStatement().ID("idGeneratorFn").Optional().Colon().Add(fnType).Semicolon())
	})
	file.Add(stmt)
	file.Line()

	file.GenerateImports()

	return file.Save(path.Join(outDir, "options.ts"))
}
