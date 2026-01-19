// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package helper

import (
	"errors"
	"strings"

	"tgp/core/data"
)

// ParseStringList парсит строку из request по указанному ключу в список строк.
// Поддерживает разделители: запятая, пробел, табуляция.
// Возвращает пустой слайс, если ключ не найден или значение пустое.
func ParseStringList(request data.Storage, key string) (result []string, err error) {

	var value string
	if value, err = data.Get[string](request, key); err != nil || value == "" {
		// Если не найдено или пустое - возвращаем пустой слайс, это не ошибка
		if errors.Is(err, data.ErrNotFound) {
			return []string{}, nil
		}
		// Другие ошибки возвращаем как есть
		return nil, err
	}

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})

	result = make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return
}
