// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/internal/generated"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderClientOptions() (err error) {

	outDir := r.outDir
	file := tsg.NewFile()
	file.Comment(generated.ByToolGatewayComment)

	stmt := tsg.NewStatement()
	if r.HasJsonRPC() {
		file.ImportType("./error", "ErrorDecoder")
	}
	if r.HasHTTP() {
		file.ImportType("./error", "HTTPErrorDecoder")
	}
	stmt.Export().Type("ClientOptions")
	stmt.Op("=")
	stmt.Block(func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().Id("url").Colon().Id("string").Semicolon())
		if r.HasJsonRPC() {
			grp.Add(tsg.NewStatement().Id("errorDecoder").Optional().Colon().Id("ErrorDecoder").Semicolon())
		}
		if r.HasHTTP() {
			grp.Add(tsg.NewStatement().Id("httpErrorDecoder").Optional().Colon().Id("HTTPErrorDecoder").Semicolon())
		}
		// Поддержка статичных заголовков и функции для динамических заголовков
		recordType := tsg.NewStatement().Id("Record").Generic("string", "string")
		headersType := tsg.NewStatement()
		headersType.Add(recordType)
		headersType.Op("|")
		// Функция должна быть в скобках в union типе
		headersFnType := tsg.NewStatement()
		promiseRecordType := tsg.NewStatement().Id("Promise").Generic("Record<string, string>")
		// Создаем функцию в скобках: (() => Record<string, string> | Promise<Record<string, string>>)
		headersFnType.Op("(").Params(func(fg *tsg.Group) {}).Op("=>").Add(recordType).Op("|").Add(promiseRecordType).Op(")")
		headersType.Add(headersFnType)
		grp.Add(tsg.NewStatement().Id("headers").Optional().Colon().Add(headersType).Semicolon())
		fnType := tsg.NewStatement()
		fnType.Params(func(fg *tsg.Group) {}).Op("=>").Id("string")
		grp.Add(tsg.NewStatement().Id("idGeneratorFn").Optional().Colon().Add(fnType).Semicolon())
	})
	file.Add(stmt)
	file.Line()

	file.GenerateImports()

	return file.Save(path.Join(outDir, "options.ts"))
}
