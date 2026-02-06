// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

//go:build wasip1

package common

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"tgp/core/exec"
)

var (
	goModPathCache = make(map[string]string)
)

func GetPkgPath(fName string, isDir bool) (string, error) {
	goModPath, err := GoModPath(fName, isDir)
	if err != nil {
		// В WASM при отсутствии go.mod используем GOPATH как fallback.
		return GetPkgPathFromGOPATH(fName, isDir)
	}

	if strings.Contains(goModPath, "go.mod") {
		pkgPath, err := GetPkgPathFromGoMod(fName, isDir, goModPath)
		if err != nil {
			return "", err
		}
		return pkgPath, nil
	}
	return GetPkgPathFromGOPATH(fName, isDir)
}

func GoModPath(fName string, isDir bool) (string, error) {
	root := fName

	if !isDir {
		root = filepath.Dir(fName)
	}

	goModPath, ok := goModPathCache[root]

	if ok {
		return goModPath, nil
	}

	defer func() {
		goModPathCache[root] = goModPath
	}()

	// В WASM используем core.ExecuteCommandInDir вместо exec.Command
	// В WASM файловая система монтируется в корень "/", поэтому используем "." как рабочую директорию
	// Если root абсолютный, просто используем "." так как корень проекта уже является текущей директорией
	workDir := "."
	if !filepath.IsAbs(root) {
		// Если root уже относительный, используем его
		workDir = root
	}

	var lastErr error
	for {
		cmd := exec.Command("go", "env", "GOMOD")
		cmd = cmd.Dir(workDir)

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			lastErr = err
		} else {
			stdoutBytes, readErr := io.ReadAll(stdoutPipe)
			stdoutPipe.Close()

			waitErr := cmd.Wait()
			if waitErr == nil && readErr == nil {
				goModPath = strings.TrimSpace(string(stdoutBytes))
				if goModPath != "" && goModPath != "/dev/null" {
					return goModPath, nil
				}
			}
			if readErr != nil {
				lastErr = readErr
			}
			if waitErr != nil {
				lastErr = waitErr
			}
		}

		if err != nil {
			lastErr = err
		}

		// Поднимаемся на уровень выше
		if workDir == "." {
			// Если уже в корне, возвращаем ошибку
			if lastErr != nil {
				return "", lastErr
			}
			return "", &os.PathError{Op: "go env GOMOD", Path: workDir, Err: os.ErrNotExist}
		}

		// Поднимаемся на уровень выше
		parent := filepath.Join(workDir, "..")
		if parent == workDir {
			break
		}
		workDir = parent
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", &os.PathError{Op: "go env GOMOD", Path: workDir, Err: os.ErrNotExist}
}

func GetPkgPathFromGoMod(fName string, isDir bool, goModPath string) (string, error) {
	modulePath := GetModulePath(goModPath)

	if modulePath == "" {
		return "", &os.PathError{Op: "GetModulePath", Path: goModPath, Err: os.ErrInvalid}
	}

	rel := path.Join(modulePath, filePathToPackagePath(strings.TrimPrefix(fName, filepath.Dir(goModPath))))

	if !isDir {
		return path.Dir(rel), nil
	}
	return path.Clean(rel), nil
}

var (
	gopathCache           = ""
	pkgPathFromGoModCache = make(map[string]string)
)

func GetModulePath(goModPath string) string {
	pkgPath, ok := pkgPathFromGoModCache[goModPath]

	if ok {
		return pkgPath
	}

	defer func() {
		pkgPathFromGoModCache[goModPath] = pkgPath
	}()

	// В WASM файловая система монтируется в корень "/", поэтому абсолютные пути не работают
	// Преобразуем абсолютный путь в относительный
	actualPath := goModPath
	if filepath.IsAbs(goModPath) {
		// Если путь абсолютный, используем только имя файла "go.mod"
		// так как корень проекта уже является текущей директорией в WASM
		actualPath = "go.mod"
	}

	data, err := os.ReadFile(actualPath)

	if err != nil {
		return ""
	}

	// Ищем строку с "module "
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			pkgPath = strings.TrimSpace(line[7:]) // Пропускаем "module "
			// Убираем кавычки если есть
			if len(pkgPath) > 0 && pkgPath[0] == '"' {
				if len(pkgPath) > 1 && pkgPath[len(pkgPath)-1] == '"' {
					pkgPath = pkgPath[1 : len(pkgPath)-1]
				}
			}
			// Убираем комментарии если есть
			if idx := strings.Index(pkgPath, "//"); idx >= 0 {
				pkgPath = strings.TrimSpace(pkgPath[:idx])
			}
			return pkgPath
		}
	}

	return ""
}

func GetPkgPathFromGOPATH(fName string, isDir bool) (string, error) {
	if gopathCache == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			// В WASM не можем использовать exec.Command, возвращаем ошибку
			return "", &os.PathError{Op: "GetDefaultGoPath", Path: "", Err: os.ErrNotExist}
		}
		gopathCache = gopath
	}

	for _, p := range filepath.SplitList(gopathCache) {
		prefix := filepath.Join(p, "src") + string(filepath.Separator)
		if rel := strings.TrimPrefix(fName, prefix); rel != fName {
			if !isDir {
				return path.Dir(filePathToPackagePath(rel)), nil
			} else {
				return path.Clean(filePathToPackagePath(rel)), nil
			}
		}
	}

	return "", &os.PathError{Op: "GetPkgPathFromGOPATH", Path: fName, Err: os.ErrNotExist}
}

func filePathToPackagePath(path string) string {
	return filepath.ToSlash(path)
}
