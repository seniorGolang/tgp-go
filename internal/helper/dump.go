// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package helper

import (
	"github.com/goccy/go-json"
)

func Dump(v any) string {

	data, _ := json.Marshal(v)
	return string(data)
}
