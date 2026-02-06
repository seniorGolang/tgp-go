// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package common

func SliceStringToMap(slice []string) (m map[string]int) {

	m = make(map[string]int)

	for i, v := range slice {
		m[v] = i
	}
	return
}
