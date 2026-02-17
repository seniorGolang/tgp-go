// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package common

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

func SortedKeys[M ~map[K]V, K cmp.Ordered, V any](m M) (out []K) {

	out = slices.Collect(maps.Keys(m))
	slices.Sort(out)
	return
}

func SortedPairs[M ~map[K]V, K cmp.Ordered, V any](m M) (seq iter.Seq2[K, V]) {

	return func(yield func(K, V) bool) {
		keys := SortedKeys(m)
		for _, key := range keys {
			if !yield(key, m[key]) {
				return
			}
		}
	}
}
