// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package internal

import "runtime"

func LineFeed() (s string) {

	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
