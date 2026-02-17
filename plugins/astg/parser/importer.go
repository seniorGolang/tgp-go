// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/types"
	"strings"
	"sync"
)

type FileSystemImporter struct {
	loader          *AutonomousPackageLoader
	cache           map[string]*types.Package
	mu              sync.RWMutex
	lazyMode        bool
	requiredImports map[string]bool
}

func (i *FileSystemImporter) Import(path string) (pkg *types.Package, err error) {

	i.mu.RLock()
	var ok bool
	if pkg, ok = i.cache[path]; ok {
		i.mu.RUnlock()
		return
	}
	i.mu.RUnlock()

	if path == "unsafe" {
		pkg = types.Unsafe
		i.mu.Lock()
		i.cache[path] = pkg
		i.mu.Unlock()
		return
	}

	if i.lazyMode {
		i.mu.RLock()
		isRequired := i.requiredImports[path]
		i.mu.RUnlock()

		if !isRequired {
			stubPkg := types.NewPackage(path, path)
			i.mu.Lock()
			i.cache[path] = stubPkg
			i.mu.Unlock()
			pkg = stubPkg
			return
		}
	}

	// Внешние пакеты загружаем из export data
	if !i.loader.isLocalPackage(path) {
		if pkg, err = i.loader.gcImporter.Import(path); err != nil {
			name := path
			if idx := strings.LastIndex(path, "/"); idx >= 0 {
				name = path[idx+1:]
			}
			pkg = types.NewPackage(path, name)
			err = nil
		}
		i.mu.Lock()
		i.cache[path] = pkg
		i.mu.Unlock()
		return
	}

	var info *PackageInfo
	info, ok = i.loader.GetPackage(path)

	if !ok {
		if info, err = i.loader.LoadPackageMinimal(path, i.requiredImports); err != nil {
			return
		}
	}

	i.mu.Lock()
	i.cache[path] = info.Types
	i.mu.Unlock()

	pkg = info.Types
	return
}
