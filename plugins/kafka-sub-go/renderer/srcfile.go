// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"tgp/plugins/kafka-sub-go/goimports"
)

type GoFile struct {
	*jen.File
	filepath string
}

func NewSrcFile(pkgName string) (file GoFile) {

	return GoFile{File: jen.NewFile(pkgName)}
}

func (file *GoFile) Save(filePath string) (err error) {

	file.filepath = filePath
	if err = os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		return
	}
	if err = file.File.Save(file.filepath); err != nil {
		return
	}

	var runner goimports.Runner
	if runner, err = goimports.NewFromFile(filePath); err != nil {
		return
	}
	return runner.Run(goimports.GetModulePath(filePath))
}
