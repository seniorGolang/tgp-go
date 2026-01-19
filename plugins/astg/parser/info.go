// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/token"
	"go/types"
	"time"
)

// PackageInfo содержит информацию о загруженном пакете.
type PackageInfo struct {
	PkgPath     string
	PackageName string
	Dir         string
	Files       []*ast.File
	Types       *types.Package
	TypeInfo    *types.Info
	Fset        *token.FileSet
	Imports     map[string]string
}

// loadPackageStat содержит статистику загрузки пакета.
type loadPackageStat struct {
	count     int
	totalTime time.Duration
	maxTime   time.Duration
}
