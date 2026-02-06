// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/token"
	"go/types"
)

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
