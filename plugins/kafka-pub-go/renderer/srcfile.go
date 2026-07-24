// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"tgp/internal/generated"
	"tgp/plugins/kafka-pub-go/goimports"
)

// GoFile представляет генерируемый Go-исходник.
type GoFile struct {
	*jen.File
}

func newSrcFile(packageName string) (source *GoFile) {

	file := jen.NewFile(packageName)
	file.PackageComment(generated.ByToolGateway)
	return &GoFile{File: file}
}

func (source *GoFile) Save(path string) (err error) {

	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err = source.File.Save(path); err != nil {
		return fmt.Errorf("save %s: %w", filepath.Base(path), err)
	}
	var runner goimports.Runner
	if runner, err = goimports.NewFromFile(path); err != nil {
		return fmt.Errorf("prepare imports %s: %w", filepath.Base(path), err)
	}
	if err = runner.Run(goimports.GetModulePath(path)); err != nil {
		return fmt.Errorf("format imports %s: %w", filepath.Base(path), err)
	}
	return nil
}
