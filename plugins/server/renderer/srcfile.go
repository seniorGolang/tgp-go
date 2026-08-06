// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"tgp/internal/generated"
	"tgp/plugins/server/goimports"
)

type GoFile struct {
	*jen.File
	filepath string
}

func NewSrcFile(pkgName string) (file GoFile) {

	jenFile := jen.NewFile(pkgName)
	jenFile.PackageComment(generated.ByToolGateway)
	return GoFile{File: jenFile}
}

func (src *GoFile) Save(filePath string) (err error) {

	src.filepath = filePath

	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0o700); err != nil {
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

	// Подсчитываем строки в сгенерированном файле для статистики
	if lines, countErr := countLinesInFile(filePath); countErr == nil {
		// Вызываем callback для добавления статистики, если он установлен
		if onFileSaved != nil {
			onFileSaved(filePath, lines)
		}
	}

	return
}

func countLinesInFile(filePath string) (lines int64, err error) {

	var content []byte
	if content, err = os.ReadFile(filePath); err != nil {
		return 0, err
	}

	lines = 1
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	return lines, nil
}

type onFileSavedCallback func(filepath string, lines int64)

var onFileSaved onFileSavedCallback

func SetOnFileSaved(callback onFileSavedCallback) {

	onFileSaved = callback
}
