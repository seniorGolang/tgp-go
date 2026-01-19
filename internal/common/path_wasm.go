//go:build wasip1

package common

import (
	"path/filepath"
	"strings"
)

// NormalizeWASMPath нормализует путь для WASM файловой системы.
// В WASM файловая система монтируется в корень "/", поэтому все пути должны быть абсолютными.
// Преобразует относительный путь в абсолютный, добавляя "/" в начало.
func NormalizeWASMPath(path string) string {

	// Если путь уже абсолютный, возвращаем очищенный
	if strings.HasPrefix(path, "/") {
		return filepath.ToSlash(filepath.Clean(path))
	}

	// Убираем ведущие "./" если есть
	path = strings.TrimPrefix(path, "./")

	// Очищаем путь
	path = filepath.Clean(path)

	// Если путь пустой или ".", возвращаем корень
	if path == "" || path == "." {
		return "/"
	}

	// Добавляем "/" в начало для WASM файловой системы
	return "/" + filepath.ToSlash(path)
}
