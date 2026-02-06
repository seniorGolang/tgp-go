// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"tgp/plugins/client-go/goimports"
)

type GoFile struct {
	*jen.File
	filepath string
}

func NewSrcFile(pkgName string) GoFile {
	return GoFile{
		File: jen.NewFile(pkgName),
	}
}

func (src *GoFile) Save(filePath string) (err error) {

	src.filepath = filePath

	// Создаем директорию, если она не существует
	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return
	}

	if err = src.File.Save(src.filepath); err != nil {
		return
	}

	var runner goimports.Runner
	if runner, err = goimports.NewFromFile(filePath); err != nil {
		return
	}

	if err = runner.Run(goimports.GetModulePath(filePath)); err != nil {
		return
	}

	return
}
