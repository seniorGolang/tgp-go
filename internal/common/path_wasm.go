// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

//go:build wasip1

package common

import (
	"path/filepath"
	"strings"
)

// NormalizeWASMPath: в WASM ФС монтируется в "/", относительные пути приводятся к абсолютным.
func NormalizeWASMPath(path string) string {

	if strings.HasPrefix(path, "/") {
		return filepath.ToSlash(filepath.Clean(path))
	}

	path = strings.TrimPrefix(path, "./")
	path = filepath.Clean(path)

	if path == "" || path == "." {
		return "/"
	}

	return "/" + filepath.ToSlash(path)
}
