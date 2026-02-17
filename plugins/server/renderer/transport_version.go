// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	"tgp/internal/generated"
)

func (r *transportRenderer) RenderTransportVersion() (err error) {

	versionPath := path.Join(r.outDir, "version.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(generated.ByToolGateway)

	srcFile.Const().Id("VersionASTg").Op("=").Lit(r.project.Version)

	return srcFile.Save(versionPath)
}
