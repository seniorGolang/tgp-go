// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/types"
	"log/slog"
	"sync"
)

type FileSystemImporter struct {
	loader *AutonomousPackageLoader
	cache  map[string]*types.Package
	mu     sync.RWMutex
}

func (i *FileSystemImporter) Import(path string) (pkg *types.Package, err error) {

	defer traceRecover("Import:" + path)

	i.mu.RLock()
	var ok bool
	if pkg, ok = i.cache[path]; ok {
		i.mu.RUnlock()
		traceStep("Import cache hit", slog.String("path", path))
		return
	}
	i.mu.RUnlock()

	local := i.loader.isLocalPackage(path)
	traceStep("Import", slog.String("path", path), slog.Bool("local", local))

	if path == "unsafe" {
		pkg = types.Unsafe
		i.mu.Lock()
		i.cache[path] = pkg
		i.mu.Unlock()
		return
	}

	if !local {
		if pkg, err = i.loader.gcImporter.Import(path); err != nil {
			traceStep("Import gc failed", slog.String("path", path), slog.String("error", err.Error()))
			err = fmt.Errorf("import %s: %w", path, err)
			return
		}
		traceStep("Import gc ok", slog.String("path", path))
		i.mu.Lock()
		i.cache[path] = pkg
		i.mu.Unlock()
		return
	}

	var info *PackageInfo
	info, ok = i.loader.GetPackage(path)

	if !ok || info == nil || info.Types == nil {
		traceStep("Import load minimal", slog.String("path", path))
		if info, err = i.loader.LoadPackageMinimal(path); err != nil {
			traceStep("Import minimal failed", slog.String("path", path), slog.String("error", err.Error()))
			return
		}
	}

	if info == nil || info.Types == nil {
		err = fmt.Errorf("package %s not loaded", path)
		return
	}

	i.mu.Lock()
	i.cache[path] = info.Types
	i.mu.Unlock()

	pkg = info.Types
	return
}
