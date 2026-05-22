// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
)

func (l *AutonomousPackageLoader) LoadPackageForErrorType(pkgPath string, typeName string) (info *PackageInfo, err error) {

	if info, ok := l.cachedPackage(pkgPath); ok {
		return info, nil
	}

	return l.loadPackageOnce(pkgPath, func() (*PackageInfo, error) {
		return l.loadPackageForErrorTypeBody(pkgPath, typeName)
	})
}

func (l *AutonomousPackageLoader) loadPackageForErrorTypeBody(pkgPath string, typeName string) (info *PackageInfo, err error) {

	var pkgDir string
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		return nil, fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		return nil, fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no Go files found in package %s", pkgPath)
	}

	return l.typeCheckPackage(pkgPath, pkgDir, l.fset, files)
}

func (l *AutonomousPackageLoader) LoadPackageForType(pkgPath string, typeName string) (info *PackageInfo, err error) {

	l.mu.RLock()
	if info, ok := l.cache[pkgPath]; ok && info != nil && info.Types != nil {
		if info.Types.Scope().Lookup(typeName) != nil {
			l.mu.RUnlock()
			return info, nil
		}
	}
	l.mu.RUnlock()

	return l.loadPackageOnce(pkgPath, func() (*PackageInfo, error) {
		return l.loadPackageForTypeBody(pkgPath, typeName)
	})
}

func (l *AutonomousPackageLoader) loadPackageForTypeBody(pkgPath string, typeName string) (info *PackageInfo, err error) {

	var pkgDir string
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		return nil, fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		return nil, fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no Go files found in package %s", pkgPath)
	}

	return l.typeCheckPackage(pkgPath, pkgDir, l.fset, files)
}

func (l *AutonomousPackageLoader) LoadPackageMinimal(pkgPath string) (info *PackageInfo, err error) {

	if info, ok := l.cachedPackage(pkgPath); ok {
		return info, nil
	}

	return l.loadPackageOnce(pkgPath, func() (*PackageInfo, error) {
		return l.loadPackageMinimalBody(pkgPath)
	})
}

func (l *AutonomousPackageLoader) loadPackageMinimalBody(pkgPath string) (info *PackageInfo, err error) {

	var pkgDir string
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		return nil, fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		return nil, fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no Go files found in package %s", pkgPath)
	}

	return l.typeCheckPackage(pkgPath, pkgDir, l.fset, files)
}
