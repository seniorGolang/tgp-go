// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package utils

// convert slice of strings to map[string]int
func SliceStringToMap(slice []string) (m map[string]int) {

	m = make(map[string]int)

	for i, v := range slice {
		m[v] = i
	}
	return
}
