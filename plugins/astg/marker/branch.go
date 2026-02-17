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

func NormalizeBranchName(branch string) (s string) {

	if branch == "" {
		return "default"
	}

	// Недопустимые в имени файла: / \ : * ? " < > | и пробел
	invalidChars := regexp.MustCompile(`[/\\:*?"<>|\s]+`)
	normalized := invalidChars.ReplaceAllString(branch, "-")

	multipleDashes := regexp.MustCompile(`-+`)
	normalized = multipleDashes.ReplaceAllString(normalized, "-")

	normalized = strings.Trim(normalized, "-.")

	if normalized == "" {
		return "default"
	}

	if len(normalized) > maxBranchNameLength {
		normalized = normalized[:maxBranchNameLength]
		normalized = strings.TrimSuffix(normalized, "-")
	}

	return normalized
}
