package helper

import (
	"github.com/goccy/go-json"
)

func Dump(v any) string {

	data, _ := json.Marshal(v)
	return string(data)
}
