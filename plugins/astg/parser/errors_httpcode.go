// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"strings"

	"tgp/internal/model"
)

func enrichErrorInfoHTTPCode(errInfo *model.ErrorInfo, loader *AutonomousPackageLoader) {

	if errInfo == nil || errInfo.HTTPCode != 0 {
		return
	}

	code, ok := resolveHTTPCodeFromErrorRef(errInfo.PkgPath, errInfo.TypeName, loader)
	if !ok {
		return
	}

	errInfo.HTTPCode = code
	errInfo.HTTPCodeText = getHTTPStatusText(code)
}

func resolveHTTPCodeFromErrorRef(pkgPath string, name string, loader *AutonomousPackageLoader) (code int, ok bool) {

	if pkgPath == "" || name == "" {
		return 0, false
	}

	var pkgInfo *PackageInfo
	var found bool
	if pkgInfo, found = loader.GetPackage(pkgPath); !found || pkgInfo == nil || pkgInfo.Types == nil {
		var err error
		if pkgInfo, err = loader.LoadPackageForErrorType(pkgPath, name); err != nil {
			return 0, false
		}
	}

	if pkgInfo == nil || pkgInfo.Types == nil {
		return 0, false
	}

	obj := pkgInfo.Types.Scope().Lookup(name)
	if obj == nil {
		return 0, false
	}

	switch o := obj.(type) {
	case *types.Const:
		code, ok = constantIntValue(o.Val())
	case *types.Var:
		var initExpr ast.Expr
		if initExpr, ok = findInitExprForName(pkgInfo, name); ok {
			code, ok = httpCodeFromInitExpr(pkgInfo, initExpr)
		}
	}

	return code, ok
}

func findInitExprForName(pkgInfo *PackageInfo, name string) (expr ast.Expr, ok bool) {

	for _, file := range pkgInfo.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			valueSpec, isSpec := node.(*ast.ValueSpec)
			if !isSpec {
				return true
			}
			for index, ident := range valueSpec.Names {
				if ident == nil || ident.Name != name {
					continue
				}
				if len(valueSpec.Values) == 0 {
					continue
				}
				if len(valueSpec.Values) == 1 {
					expr = valueSpec.Values[0]
					ok = true
					return false
				}
				if index < len(valueSpec.Values) {
					expr = valueSpec.Values[index]
					ok = true
					return false
				}
			}
			return true
		})
		if ok {
			return expr, true
		}
	}
	return nil, false
}

func httpCodeFromInitExpr(pkgInfo *PackageInfo, initExpr ast.Expr) (code int, ok bool) {

	call, isCall := initExpr.(*ast.CallExpr)
	if !isCall || !isMakeErrorCall(call.Fun) || len(call.Args) < 3 {
		return 0, false
	}

	return evalConstIntExpr(pkgInfo, call.Args[2])
}

func isMakeErrorCall(fun ast.Expr) (ok bool) {

	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name == "makeError"
	case *ast.SelectorExpr:
		return f.Sel.Name == "makeError"
	default:
		return false
	}
}

func evalConstIntExpr(pkgInfo *PackageInfo, expr ast.Expr) (value int, ok bool) {

	if expr == nil || pkgInfo == nil || pkgInfo.Types == nil {
		return 0, false
	}

	if pkgInfo.TypeInfo != nil {
		if typeAndValue, exists := pkgInfo.TypeInfo.Types[expr]; exists && typeAndValue.IsValue() && typeAndValue.Value != nil {
			return constantIntValue(typeAndValue.Value)
		}
	}

	var source strings.Builder
	if err := printer.Fprint(&source, pkgInfo.Fset, expr); err != nil {
		return 0, false
	}

	typeAndValue, err := types.Eval(pkgInfo.Fset, pkgInfo.Types, token.NoPos, source.String())
	if err != nil || !typeAndValue.IsValue() || typeAndValue.Value == nil {
		return 0, false
	}

	return constantIntValue(typeAndValue.Value)
}

func constantIntValue(constValue constant.Value) (value int, ok bool) {

	if constValue == nil || constValue.Kind() != constant.Int {
		return 0, false
	}

	int64Value, ok := constant.Int64Val(constValue)
	if !ok {
		return 0, false
	}

	return int(int64Value), true
}
