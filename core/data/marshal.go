// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package data

import (
	stdjson "encoding/json"
	"fmt"
	"log/slog"

	"github.com/goccy/go-json"

	"tgp/core/i18n"
)

// MarshalValue сериализует значение для Storage. При панике или ошибке goccy использует encoding/json.
func MarshalValue(value any) (jsonData []byte, err error) {

	jsonData, err = marshalValueGoccy(value)
	if err == nil {
		return
	}

	slog.Debug(i18n.Msg("goccy marshal failed, using encoding/json"), slog.Any("error", err))
	return stdjson.Marshal(value)
}

// UnmarshalValue десериализует JSON. При панике или ошибке goccy использует encoding/json.
func UnmarshalValue(jsonData []byte, value any) (err error) {

	err = unmarshalValueGoccy(jsonData, value)
	if err == nil {
		return
	}

	slog.Debug(i18n.Msg("goccy unmarshal failed, using encoding/json"), slog.Any("error", err))
	return stdjson.Unmarshal(jsonData, value)
}

func marshalValueGoccy(value any) (jsonData []byte, err error) {

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("goccy marshal panic: %v", recovered)
			jsonData = nil
		}
	}()

	return json.Marshal(value)
}

func unmarshalValueGoccy(jsonData []byte, value any) (err error) {

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("goccy unmarshal panic: %v", recovered)
		}
	}()

	return json.Unmarshal(jsonData, value)
}
