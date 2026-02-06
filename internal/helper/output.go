// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/internal/common"
)

func GetOutput(request data.Storage) (output string, err error) {

	var outputRaw string
	if outputRaw, err = data.Get[string](request, "out"); err != nil || outputRaw == "" {
		// Если не найдено - возвращаем пустую строку, это не ошибка
		if errors.Is(err, data.ErrNotFound) {
			return "", nil
		}
		// Другие ошибки возвращаем как есть
		return "", fmt.Errorf("%s: %w", i18n.Msg("failed to get output"), err)
	}

	// Нормализуем путь для WASM файловой системы
	// В WASM все пути должны быть абсолютными (начинаться с "/")
	output = common.NormalizeWASMPath(outputRaw)

	// Автоматически определяем, файл это или директория по наличию расширения
	if filepath.Ext(output) != "" {
		// Для файла создаем родительскую директорию
		dir := filepath.Dir(output)
		if dir == "" || dir == "." {
			dir = "/"
		}
		if err = os.MkdirAll(dir, 0700); err != nil {
			return "", fmt.Errorf("%s: %w", i18n.Msg("failed to create output directory"), err)
		}
	} else {
		// Для директории создаем саму директорию
		if err = os.MkdirAll(output, 0700); err != nil {
			return "", fmt.Errorf("%s: %w", i18n.Msg("failed to create output directory"), err)
		}
	}

	return
}
