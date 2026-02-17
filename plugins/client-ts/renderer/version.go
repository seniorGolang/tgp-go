// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/internal/generated"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderVersion() (err error) {

	outDir := r.outDir
	file := tsg.NewFile()
	file.Comment(generated.ByToolGatewayComment)

	stmt := tsg.NewStatement()
	stmt.Export().Const("VersionASTg").Op("=").Lit(r.project.Version).Semicolon()
	file.Add(stmt)
	file.Line()

	file.GenerateImports()

	return file.Save(path.Join(outDir, "version.ts"))
}
