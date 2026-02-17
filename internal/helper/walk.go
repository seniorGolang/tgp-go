// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package helper

import (
	"path/filepath"
	"strings"
)

var defaultExcludeDirNames = []string{
	".tg", ".git", "vendor",
}

func IsRelevantGoFile(fileName string) (ok bool) {

	return strings.HasSuffix(fileName, ".go") && !strings.HasSuffix(fileName, "_test.go")
}

func IsDirNameExcluded(name string) (yes bool) {

	for _, n := range defaultExcludeDirNames {
		if name == n {
			return true
		}
	}
	return false
}

func IsRelPathExcluded(relPath string, extraDirs []string) (yes bool) {

	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimPrefix(relPath, "./")
	relPath = strings.TrimPrefix(relPath, ".\\")

	for _, prefix := range defaultExcludeDirNames {
		p := prefix + "/"
		if relPath == prefix || strings.HasPrefix(relPath, p) {
			return true
		}
	}
	for _, ex := range extraDirs {
		ex = filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(ex, "./"), ".\\"))
		if relPath == ex || strings.HasPrefix(relPath, ex+"/") {
			return true
		}
	}
	return false
}
