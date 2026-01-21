// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"
)

// RenderTransportVersion генерирует транспортный version файл.
func (r *transportRenderer) RenderTransportVersion() error {

	versionPath := path.Join(r.outDir, "version.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.Const().Id("VersionASTg").Op("=").Lit(r.project.Version)

	return srcFile.Save(versionPath)
}
