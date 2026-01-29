// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package marker

import (
	"regexp"
	"strings"
)

const (
	// maxBranchNameLength максимальная длина имени ветки для использования в имени файла.
	maxBranchNameLength = 255
)

func NormalizeBranchName(branch string) string {

	// Если ветка пустая, возвращаем "default"
	if branch == "" {
		return "default"
	}

	// Недопустимые: / \ : * ? " < > | и пробел
	invalidChars := regexp.MustCompile(`[/\\:*?"<>|\s]+`)
	normalized := invalidChars.ReplaceAllString(branch, "-")

	multipleDashes := regexp.MustCompile(`-+`)
	normalized = multipleDashes.ReplaceAllString(normalized, "-")

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
