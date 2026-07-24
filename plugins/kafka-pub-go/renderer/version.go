// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	"tgp/internal"
)

func (r *Renderer) renderVersion() (err error) {

	source := newSrcFile(filepath.Base(r.outDir))
	source.Const().Id("VersionASTg").Op("=").Lit(internal.Version)
	return source.Save(path.Join(r.outDir, "version.go"))
}
