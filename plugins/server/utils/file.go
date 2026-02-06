// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"os"
)

func ShouldSkipFile(filePath string) (bool, error) {

	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil // Файл существует, пропускаем
	}
	if os.IsNotExist(err) {
		return false, nil // Файл не существует, нужно генерировать
	}
	return false, err // Ошибка при проверке
}
