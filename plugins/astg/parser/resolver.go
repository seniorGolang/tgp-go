// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"tgp/internal"
)

type PackageResolver struct {
	modulePath        string
	modFile           *modfile.File
	resolveCache      map[string]string // Кэш результатов Resolve: pkgPath -> dir
	resolveCacheMu    sync.RWMutex
	modulePathCache   map[string]string // Кэш результатов findModuleByPackagePath: pkgPath -> modDir
	modulePathCacheMu sync.RWMutex
}

func NewPackageResolver(modFile *modfile.File) (resolver *PackageResolver, err error) {

	var modulePath string
	if modFile != nil && modFile.Module != nil {
		modulePath = modFile.Module.Mod.Path
	}

	resolver = &PackageResolver{
		modulePath:      modulePath,
		modFile:         modFile,
		resolveCache:    make(map[string]string),
		modulePathCache: make(map[string]string),
	}
	return
}

func (r *PackageResolver) Resolve(pkgPath string) (result string, err error) {

	r.resolveCacheMu.RLock()
	var ok bool
	var cached string
	if cached, ok = r.resolveCache[pkgPath]; ok {
		r.resolveCacheMu.RUnlock()
		return cached, nil
	}
	r.resolveCacheMu.RUnlock()

	// 1. Модульный пакет
	if r.modulePath != "" && strings.HasPrefix(pkgPath, r.modulePath) {
		relPath := strings.TrimPrefix(pkgPath, r.modulePath)
		relPath = strings.TrimPrefix(relPath, "/")
		var dir string
		if dir, err = resolveProjectPath(relPath); err != nil {
			return
		}
		var info os.FileInfo
		//nolint:gosec // dir ограничен корнем проекта через resolveProjectPath
		if info, err = os.Stat(dir); err == nil && info.IsDir() {
			result = dir
		} else if err != nil {
			_ = fmt.Errorf("failed to stat directory %s: %w (pkgPath=%s, modulePath=%s, relPath=%s)",
				dir, err, pkgPath, r.modulePath, relPath)
		} else if relPath == "" {
			if info, err = os.Stat(internal.ProjectRoot); err == nil && info.IsDir() {
				result = internal.ProjectRoot
			}
		}
		if err == nil {
			r.resolveCacheMu.Lock()
			r.resolveCache[pkgPath] = result
			r.resolveCacheMu.Unlock()
			return
		}
	}

	var goroot string
	if goroot = os.Getenv("GOROOT"); goroot != "" {
		stdPath := filepath.Join(goroot, "src", filepath.FromSlash(pkgPath))
		//nolint:gosec // G703 — путь из GOROOT и pkgPath (импорт), не пользовательский ввод
		if _, statErr := os.Stat(stdPath); statErr == nil {
			result = stdPath
			r.resolveCacheMu.Lock()
			r.resolveCache[pkgPath] = result
			r.resolveCacheMu.Unlock()
			return
		}
	}

	// 3. Внешняя зависимость через go.mod (прямые и транзитивные)
	if r.modFile != nil {
		for _, req := range r.modFile.Require {
			if strings.HasPrefix(pkgPath, req.Mod.Path) {
				var modDir string
				if modDir, err = r.findModuleDir(req.Mod.Path, req.Mod.Version); err == nil {
					relPath := strings.TrimPrefix(pkgPath, req.Mod.Path)
					relPath = strings.TrimPrefix(relPath, "/")
					var dir string
					if dir, err = resolvePathWithinBase(modDir, relPath); err != nil {
						continue
					}
					//nolint:gosec // dir ограничен корнем модуля через resolvePathWithinBase
					if _, statErr := os.Stat(dir); statErr == nil {
						return dir, err
					}
				}
			}
		}

		var modDir string
		if modDir, err = r.findModuleByPackagePath(pkgPath); err == nil {
			result = modDir
			r.resolveCacheMu.Lock()
			r.resolveCache[pkgPath] = result
			r.resolveCacheMu.Unlock()
			return
		}
	}

	return "", fmt.Errorf("package not found: %s", pkgPath)
}

func (r *PackageResolver) findModuleDir(modulePath string, version string) (modDir string, err error) {

	var gomodcache string
	if gomodcache = os.Getenv("GOMODCACHE"); gomodcache != "" {
		// Экранируем путь модуля для файловой системы (например, KimMachineGun -> !kim!machine!gun)
		escapedPath, escapeErr := module.EscapePath(modulePath)
		if escapeErr != nil {
			escapedPath = modulePath // Fallback на оригинальный путь
		}
		modDir = filepath.Join(gomodcache, fmt.Sprintf("%s@%s", escapedPath, version))
		var info os.FileInfo
		//nolint:gosec // G703 — modDir из GOMODCACHE и module.EscapePath, не пользовательский ввод
		if info, err = os.Stat(modDir); err == nil && info.IsDir() {
			return
		}
	}

	var gopath string
	if gopath = os.Getenv("GOPATH"); gopath != "" {
		// Экранируем путь модуля для файловой системы (например, KimMachineGun -> !kim!machine!gun)
		escapedPath, escapeErr := module.EscapePath(modulePath)
		if escapeErr != nil {
			escapedPath = modulePath // Fallback на оригинальный путь
		}
		modDir = filepath.Join(gopath, "pkg", "mod", fmt.Sprintf("%s@%s", escapedPath, version))
		var info os.FileInfo
		//nolint:gosec // G703 — modDir из GOPATH и module.EscapePath
		if info, err = os.Stat(modDir); err == nil && info.IsDir() {
			return
		}
	}

	return "", fmt.Errorf("module directory not found: %s@%s", modulePath, version)
}

func (r *PackageResolver) findModuleByPackagePath(pkgPath string) (result string, err error) {

	r.modulePathCacheMu.RLock()
	var ok bool
	var cached string
	if cached, ok = r.modulePathCache[pkgPath]; ok {
		r.modulePathCacheMu.RUnlock()
		return cached, nil
	}
	r.modulePathCacheMu.RUnlock()

	// Для golang.org/x/crypto/cryptobyte пробуем:
	// - golang.org/x/crypto
	// - golang.org/x
	// - golang.org
	parts := strings.Split(pkgPath, "/")

	modCacheDirs := []string{}

	if gomodcache := os.Getenv("GOMODCACHE"); gomodcache != "" {
		modCacheDirs = append(modCacheDirs, gomodcache)
	}

	if gopath := os.Getenv("GOPATH"); gopath != "" {
		modCacheDirs = append(modCacheDirs, filepath.Join(gopath, "pkg", "mod"))
	}

	if len(modCacheDirs) == 0 {
		return "", fmt.Errorf("GOMODCACHE and GOPATH not set")
	}

	// Пробуем найти модуль, начиная с полного пути
	for i := len(parts); i > 0; i-- {
		modulePath := strings.Join(parts[:i], "/")
		relPath := strings.Join(parts[i:], "/")

		// Экранируем путь модуля для файловой системы (например, KimMachineGun -> !kim!machine!gun)
		escapedPath, escapeErr := module.EscapePath(modulePath)
		if escapeErr != nil {
			escapedPath = modulePath // Fallback на оригинальный путь
		}

		for _, modCacheDir := range modCacheDirs {
			pattern := filepath.Join(modCacheDir, escapedPath+"@*")
			var matches []string
			if matches, err = filepath.Glob(pattern); err != nil || len(matches) == 0 {
				continue
			}

			var bestModDir string
			for _, match := range matches {
				var info os.FileInfo
				//nolint:gosec // G703 — match из filepath.Glob по go.mod, не пользовательский ввод
				if info, err = os.Stat(match); err == nil && info.IsDir() {
					if bestModDir == "" || match > bestModDir {
						bestModDir = match
					}
				}
			}

			if bestModDir != "" {
				var resultDir string
				if relPath != "" {
					pkgDir := filepath.Join(bestModDir, filepath.FromSlash(relPath))
					var info os.FileInfo
					//nolint:gosec // G703 — pkgDir из go.mod и relPath (импорт)
					if info, err = os.Stat(pkgDir); err == nil && info.IsDir() {
						resultDir = pkgDir
					}
				} else {
					var info os.FileInfo
					//nolint:gosec // G703 — bestModDir из filepath.Glob
					if info, err = os.Stat(bestModDir); err == nil && info.IsDir() {
						resultDir = bestModDir
					}
				}
				if resultDir != "" {
					r.modulePathCacheMu.Lock()
					r.modulePathCache[pkgPath] = resultDir
					r.modulePathCacheMu.Unlock()
					return resultDir, nil
				}
			}
		}
	}

	return "", fmt.Errorf("module not found for package: %s", pkgPath)
}

func resolveProjectPath(relPath string) (dir string, err error) {

	var basePath string
	if basePath, err = filepath.Abs(filepath.Clean(internal.ProjectRoot)); err != nil {
		return
	}
	if dir, err = resolvePathWithinBase(basePath, relPath); err != nil {
		return
	}
	return
}

func resolvePathWithinBase(basePath string, relPath string) (path string, err error) {

	targetPath := filepath.Join(basePath, filepath.FromSlash(relPath))
	var targetAbsPath string
	if targetAbsPath, err = filepath.Abs(filepath.Clean(targetPath)); err != nil {
		return
	}
	var baseAbsPath string
	if baseAbsPath, err = filepath.Abs(filepath.Clean(basePath)); err != nil {
		return
	}
	basePrefix := baseAbsPath + string(os.PathSeparator)
	if targetAbsPath != baseAbsPath && !strings.HasPrefix(targetAbsPath, basePrefix) {
		return "", fmt.Errorf("unsafe package path: %s", relPath)
	}
	return
}
