// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package marker

import (
	"regexp"
	"strings"
)

const (
	// maxBranchNameLength максимальная длина имени ветки для использования в имени файла.
	maxBranchNameLength = 255
)

// NormalizeBranchName нормализует имя ветки Git для использования в качестве имени файла.
// Заменяет недопустимые символы на дефисы и обрабатывает краевые случаи.
func NormalizeBranchName(branch string) string {

	// Если ветка пустая, возвращаем "default"
	if branch == "" {
		return "default"
	}

	// Заменяем недопустимые символы на дефис
	// Недопустимые: / \ : * ? " < > | и пробел
	invalidChars := regexp.MustCompile(`[/\\:*?"<>|\s]+`)
	normalized := invalidChars.ReplaceAllString(branch, "-")

	// Удаляем множественные дефисы подряд
	multipleDashes := regexp.MustCompile(`-+`)
	normalized = multipleDashes.ReplaceAllString(normalized, "-")

	// Удаляем ведущие и завершающие дефисы и точки
	normalized = strings.Trim(normalized, "-.")

	// Если после обработки пустая строка, возвращаем "default"
	if normalized == "" {
		return "default"
	}

	// Обрезаем до максимальной длины
	if len(normalized) > maxBranchNameLength {
		normalized = normalized[:maxBranchNameLength]
		// Убираем возможный дефис в конце после обрезки
		normalized = strings.TrimSuffix(normalized, "-")
	}

	return normalized
}
