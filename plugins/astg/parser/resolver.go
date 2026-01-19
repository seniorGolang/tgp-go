// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
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

// PackageResolver разрешает пути пакетов в файловые пути.
type PackageResolver struct {
	modulePath        string
	modFile           *modfile.File
	resolveCache      map[string]string // Кэш результатов Resolve: pkgPath -> dir
	resolveCacheMu    sync.RWMutex
	modulePathCache   map[string]string // Кэш результатов findModuleByPackagePath: pkgPath -> modDir
	modulePathCacheMu sync.RWMutex
}

// NewPackageResolver создает новый PackageResolver.
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

// Resolve разрешает путь пакета в файловый путь.
func (r *PackageResolver) Resolve(pkgPath string) (result string, err error) {

	// Проверяем кэш
	r.resolveCacheMu.RLock()
	var cached string
	var ok bool
	if cached, ok = r.resolveCache[pkgPath]; ok {
		r.resolveCacheMu.RUnlock()
		result = cached
		return
	}
	r.resolveCacheMu.RUnlock()

	// 1. Модульный пакет
	if r.modulePath != "" && strings.HasPrefix(pkgPath, r.modulePath) {
		relPath := strings.TrimPrefix(pkgPath, r.modulePath)
		relPath = strings.TrimPrefix(relPath, "/")
		dir := filepath.Join(internal.ProjectRoot, filepath.FromSlash(relPath))
		var info os.FileInfo
		if info, err = os.Stat(dir); err == nil && info.IsDir() {
			result = dir
		} else if err != nil {
			// Логируем ошибку для отладки
			_ = fmt.Errorf("failed to stat directory %s: %w (pkgPath=%s, modulePath=%s, relPath=%s)",
				dir, err, pkgPath, r.modulePath, relPath)
		} else if relPath == "" {
			// Если директория не найдена, пробуем без префикса модуля (для корневых пакетов)
			// Корневой пакет модуля
			if info, err = os.Stat(internal.ProjectRoot); err == nil && info.IsDir() {
				result = internal.ProjectRoot
			}
		}
		if err == nil {
			// Кэшируем успешный результат
			r.resolveCacheMu.Lock()
			r.resolveCache[pkgPath] = result
			r.resolveCacheMu.Unlock()
			return
		}
	}

	// 2. Стандартная библиотека (проверяем ПЕРЕД поиском в модулях, чтобы не тратить время)
	// Стандартные библиотеки не должны искаться в модулях
	var goroot string
	if goroot = os.Getenv("GOROOT"); goroot != "" {
		stdPath := filepath.Join(goroot, "src", filepath.FromSlash(pkgPath))
		if _, statErr := os.Stat(stdPath); statErr == nil {
			result = stdPath
			// Кэшируем успешный результат
			r.resolveCacheMu.Lock()
			r.resolveCache[pkgPath] = result
			r.resolveCacheMu.Unlock()
			return
		}
	}

	// 3. Внешняя зависимость через go.mod (прямые и транзитивные)
	if r.modFile != nil {
		// Сначала проверяем Require (прямые зависимости)
		for _, req := range r.modFile.Require {
			if strings.HasPrefix(pkgPath, req.Mod.Path) {
				var modDir string
				if modDir, err = r.findModuleDir(req.Mod.Path, req.Mod.Version); err == nil {
					relPath := strings.TrimPrefix(pkgPath, req.Mod.Path)
					relPath = strings.TrimPrefix(relPath, "/")
					dir := filepath.Join(modDir, filepath.FromSlash(relPath))
					if _, statErr := os.Stat(dir); statErr == nil {
						result = dir
						return
					}
				}
			}
		}

		// Если не нашли в Require, ищем модуль по пути пакета в GOPATH/pkg/mod
		// Это нужно для транзитивных зависимостей
		var modDir string
		if modDir, err = r.findModuleByPackagePath(pkgPath); err == nil {
			result = modDir
			// Кэшируем успешный результат
			r.resolveCacheMu.Lock()
			r.resolveCache[pkgPath] = result
			r.resolveCacheMu.Unlock()
			return
		}
	}

	err = fmt.Errorf("package not found: %s", pkgPath)
	return
}

// findModuleDir находит директорию модуля в GOPATH/pkg/mod или GOMODCACHE.
func (r *PackageResolver) findModuleDir(modulePath string, version string) (modDir string, err error) {

	// Пробуем GOMODCACHE
	var gomodcache string
	if gomodcache = os.Getenv("GOMODCACHE"); gomodcache != "" {
		// Экранируем путь модуля для файловой системы (например, KimMachineGun -> !kim!machine!gun)
		escapedPath, escapeErr := module.EscapePath(modulePath)
		if escapeErr != nil {
			escapedPath = modulePath // Fallback на оригинальный путь
		}
		modDir = filepath.Join(gomodcache, fmt.Sprintf("%s@%s", escapedPath, version))
		var info os.FileInfo
		if info, err = os.Stat(modDir); err == nil && info.IsDir() {
			return
		}
	}

	// Пробуем GOPATH/pkg/mod
	var gopath string
	if gopath = os.Getenv("GOPATH"); gopath != "" {
		// Экранируем путь модуля для файловой системы (например, KimMachineGun -> !kim!machine!gun)
		escapedPath, escapeErr := module.EscapePath(modulePath)
		if escapeErr != nil {
			escapedPath = modulePath // Fallback на оригинальный путь
		}
		modDir = filepath.Join(gopath, "pkg", "mod", fmt.Sprintf("%s@%s", escapedPath, version))
		var info os.FileInfo
		if info, err = os.Stat(modDir); err == nil && info.IsDir() {
			return
		}
	}

	err = fmt.Errorf("module directory not found: %s@%s", modulePath, version)
	return
}

// findModuleByPackagePath находит модуль по пути пакета, сканируя GOPATH/pkg/mod.
// Например, для golang.org/x/crypto/cryptobyte найдет модуль golang.org/x/crypto.
func (r *PackageResolver) findModuleByPackagePath(pkgPath string) (result string, err error) {

	// Проверяем кэш
	r.modulePathCacheMu.RLock()
	var cached string
	var ok bool
	if cached, ok = r.modulePathCache[pkgPath]; ok {
		r.modulePathCacheMu.RUnlock()
		result = cached
		return
	}
	r.modulePathCacheMu.RUnlock()

	// Определяем возможные пути модулей
	// Для golang.org/x/crypto/cryptobyte пробуем:
	// - golang.org/x/crypto
	// - golang.org/x
	// - golang.org
	parts := strings.Split(pkgPath, "/")

	// Получаем директории кэша модулей
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
			// Ищем директории модуля с паттерном escapedPath@*
			// Используем filepath.Glob для эффективного поиска
			pattern := filepath.Join(modCacheDir, escapedPath+"@*")
			var matches []string
			if matches, err = filepath.Glob(pattern); err != nil || len(matches) == 0 {
				continue
			}

			// Выбираем самую новую версию (последнюю по алфавиту)
			var bestModDir string
			for _, match := range matches {
				var info os.FileInfo
				if info, err = os.Stat(match); err == nil && info.IsDir() {
					if bestModDir == "" || match > bestModDir {
						bestModDir = match
					}
				}
			}

			if bestModDir != "" {
				// Проверяем, что подпакет существует
				var resultDir string
				if relPath != "" {
					pkgDir := filepath.Join(bestModDir, filepath.FromSlash(relPath))
					var info os.FileInfo
					if info, err = os.Stat(pkgDir); err == nil && info.IsDir() {
						resultDir = pkgDir
					}
				} else {
					// Это корневой пакет модуля
					var info os.FileInfo
					if info, err = os.Stat(bestModDir); err == nil && info.IsDir() {
						resultDir = bestModDir
					}
				}
				if resultDir != "" {
					// Кэшируем успешный результат
					r.modulePathCacheMu.Lock()
					r.modulePathCache[pkgPath] = resultDir
					r.modulePathCacheMu.Unlock()
					result = resultDir
					return
				}
			}
		}
	}

	err = fmt.Errorf("module not found for package: %s", pkgPath)
	return
}
