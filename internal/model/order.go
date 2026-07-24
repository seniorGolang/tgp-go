// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"cmp"
	"slices"
)

// ContractsSorted возвращает копию контрактов, отсортированную по Name, затем по ID.
func ContractsSorted(contracts []*Contract) (out []*Contract) {

	out = slices.Clone(contracts)
	slices.SortFunc(out, func(left *Contract, right *Contract) int {
		if left == nil && right == nil {
			return 0
		}
		if left == nil {
			return -1
		}
		if right == nil {
			return 1
		}
		if result := cmp.Compare(left.Name, right.Name); result != 0 {
			return result
		}
		return cmp.Compare(left.ID, right.ID)
	})
	return out
}
