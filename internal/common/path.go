// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

//go:build !wasip1

package common

import (
	"path/filepath"
)

func NormalizeWASMPath(path string) string {

	return filepath.Clean(path)
}
