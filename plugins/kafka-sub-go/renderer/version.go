// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/internal"
	"tgp/internal/generated"
)

func (r *Renderer) renderVersion() (err error) {

	file := NewSrcFile(r.pkgName)
	file.PackageComment(generated.ByToolGateway)
	file.Const().Id("VersionASTg").Op("=").Lit(internal.Version)
	return file.Save(path.Join(r.outDir, "version.go"))
}
