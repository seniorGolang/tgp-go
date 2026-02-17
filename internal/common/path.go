// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package common

import (
	"path/filepath"
	"strings"
)

// NormalizeWASMPath: в WASM ФС монтируется в "/", относительные пути приводятся к абсолютным.
func NormalizeWASMPath(path string) (s string) {

	path = filepath.Clean(path)
	if !strings.HasPrefix(path, "/") {
		return "/" + filepath.ToSlash(path)
	}
	return path
}
