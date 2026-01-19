// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package common

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// SortedKeys возвращает отсортированные ключи из map'а.
// Универсальная функция для работы с любыми map'ами через дженерики.
// Используется для гарантии детерминированного порядка генерации кода во всех плагинах.
func SortedKeys[M ~map[K]V, K cmp.Ordered, V any](m M) []K {
	keys := slices.Collect(maps.Keys(m))
	slices.Sort(keys)
	return keys
}

// SortedPairs возвращает итератор по отсортированным парам ключ-значение из map'а.
// Универсальная функция для работы с любыми map'ами через дженерики.
// Позволяет сразу получить и ключ, и значение без дополнительного обращения к map'у.
// Используется для гарантии детерминированного порядка генерации кода во всех плагинах.
func SortedPairs[M ~map[K]V, K cmp.Ordered, V any](m M) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		keys := SortedKeys(m)
		for _, key := range keys {
			if !yield(key, m[key]) {
				return
			}
		}
	}
}
