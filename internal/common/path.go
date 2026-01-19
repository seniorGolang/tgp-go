//go:build !wasip1

package common

import (
	"path/filepath"
)

// NormalizeWASMPath нормализует путь для файловой системы.
// В нативном окружении просто очищает путь.
func NormalizeWASMPath(path string) string {

	return filepath.Clean(path)
}
