// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"
)

func (r *ClientRenderer) RenderVersion() error {

	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.Const().Id("VersionASTg").Op("=").Lit(r.project.Version)

	return srcFile.Save(path.Join(outDir, "version.go"))
}
