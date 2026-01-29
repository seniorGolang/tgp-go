// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func isBuiltinTypeName(typeName string) (isBuiltin bool) {

	builtinTypes := map[string]bool{
		"bool": true, "string": true, "int": true, "int8": true, "int16": true,
		"int32": true, "int64": true, "uint": true, "uint8": true, "uint16": true,
		"uint32": true, "uint64": true, "uintptr": true, "byte": true, "rune": true,
		"float32": true, "float64": true, "complex64": true, "complex128": true,
		"error": true, "any": true,
	}
	isBuiltin = builtinTypes[typeName]
	return
}

func GoProjectPath(from string) (projectPath string) {

	var modPath string
	modPath, _ = findGoModPath()
	if modPath == "" {
		return
	}
	projectPath = strings.TrimSuffix(modPath, "go.mod")
	return
}

func PkgModPath(pkgName string) (modPathResult string) {

	var modPath string
	modPath, _ = findGoModPath()
	if modPath == "" {
		return
	}
	var modBytes []byte
	var err error
	if modBytes, err = os.ReadFile(modPath); err != nil {
		return
	}
	_, err = modfile.Parse(modPath, modBytes, nil)
	if err != nil {
		return
	}
	modInfo := parseMod(modPath)
	pkgTokens := strings.Split(pkgName, "/")
	for i := 0; i < len(pkgTokens); i++ {
		pathTry := strings.Join(pkgTokens[:len(pkgTokens)-i], "/")
		for modPkg, modPathVal := range modInfo {
			if pathTry == modPkg {
				var esc string
				esc, _ = module.EscapePath(modPkg)
				modPathVal = strings.Replace(modPathVal, modPkg, esc, 1)
				if len(strings.Split(modPkg, "/")) == 1 {
					modPathResult = path.Join(modPathVal, strings.Join(pkgTokens, "/"))
					return
				}
				modPathResult = path.Join(modPathVal, strings.Join(pkgTokens[len(pkgTokens)-i:], "/"))
				return
			}
		}
	}
	return
}

func moduleCacheRoot() (root string) {

	if root = os.Getenv("GOMODCACHE"); root != "" {
		return
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		root = filepath.Join(gopath, "pkg", "mod")
	}
	return
}

func parseMod(modPath string) (pkgPath map[string]string) {

	var fileBytes []byte
	var err error
	if fileBytes, err = os.ReadFile(modPath); err != nil {
		return
	}
	var mod *modfile.File
	if mod, err = modfile.Parse(modPath, fileBytes, nil); err != nil {
		return
	}
	pkgPath = make(map[string]string)
	if mod.Module != nil {
		pkgPath[mod.Module.Mod.Path] = filepath.Dir(modPath)
	}
	modRoot := moduleCacheRoot()
	if modRoot == "" {
		return
	}
	for _, require := range mod.Require {
		escapedPath, escapeErr := module.EscapePath(require.Mod.Path)
		if escapeErr != nil {
			escapedPath = require.Mod.Path
		}
		pkgPath[require.Mod.Path] = filepath.Join(modRoot, fmt.Sprintf("%s@%s", escapedPath, require.Mod.Version))
	}
	return
}

func makeTypeID(pkgPath string, typeName string) (id string) {

	id = pkgPath + ":" + typeName
	return
}

func splitTypeID(typeID string) (parts []string) {

	idx := strings.LastIndex(typeID, ":")
	if idx == -1 {
		parts = []string{typeID}
		return
	}
	parts = []string{typeID[:idx], typeID[idx+1:]}
	return
}

func isReceiverForStruct(recvType ast.Expr, structName string) (isReceiver bool) {

	switch rt := recvType.(type) {
	case *ast.Ident:
		isReceiver = rt.Name == structName
		return
	case *ast.StarExpr:
		var ident *ast.Ident
		var ok bool
		if ident, ok = rt.X.(*ast.Ident); ok {
			isReceiver = ident.Name == structName
			return
		}
	}
	return
}
