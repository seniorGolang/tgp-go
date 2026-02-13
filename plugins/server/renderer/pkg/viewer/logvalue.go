package viewer

import (
	"encoding/json"
	"log/slog"
	"reflect"
)

type logValuer struct {
	v any
}

func (l logValuer) LogValue() slog.Value {
	tree, err := toJSONTree(reflect.ValueOf(l.v), 0, make(map[uintptr]int), &Config, nil)
	if err != nil {
		return slog.AnyValue(nil)
	}
	return slog.AnyValue(tree)
}

func Any(key string, v any) slog.Attr {
	return slog.Any(key, logValuer{v: v})
}

func String(v any) string {
	if v == nil {
		return "null"
	}
	tree, err := toJSONTree(reflect.ValueOf(v), 0, make(map[uintptr]int), &Config, nil)
	if err != nil {
		return jsonPlaceholderInvalid
	}
	out, err := json.Marshal(tree)
	if err != nil {
		return jsonPlaceholderInvalid
	}
	return string(out)
}
