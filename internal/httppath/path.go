// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package httppath

import "strings"

// Join склеивает path endpoint и заявленный route со сворачиванием совпадающего хвоста/головы.
func Join(basePath, routePath string) (joined string) {

	base := split(basePath)
	route := split(routePath)
	overlap := overlapLen(base, route)
	out := make([]string, 0, len(base)-overlap+len(route))
	out = append(out, base[:len(base)-overlap]...)
	out = append(out, route...)
	if len(out) == 0 {
		return "/"
	}
	joined = "/" + strings.Join(out, "/")
	return
}

func split(raw string) (segments []string) {

	for part := range strings.SplitSeq(raw, "/") {
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	return
}

func overlapLen(base, route []string) (overlap int) {

	limit := min(len(route), len(base))
	for n := limit; n >= 1; n-- {
		if segmentsEqual(base[len(base)-n:], route[:n]) {
			return n
		}
	}
	return 0
}

func segmentsEqual(left, right []string) (equal bool) {

	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
