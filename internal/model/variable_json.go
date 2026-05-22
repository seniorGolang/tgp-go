// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	stdjson "encoding/json"

	"tgp/internal/tags"
)

// variableWire — плоское представление Variable для JSON.
// goccy/go-json паникует при json:",inline" у TypeRef с пустым typeID и заполненными mapKey/mapValue.
type variableWire struct {
	TypeID           string       `json:"typeID,omitempty"`
	NumberOfPointers int          `json:"numberOfPointers,omitempty"`
	IsSlice          bool         `json:"isSlice,omitempty"`
	ArrayLen         int          `json:"arrayLen,omitempty"`
	IsEllipsis       bool         `json:"isEllipsis,omitempty"`
	ElementPointers  int          `json:"elementPointers,omitempty"`
	MapKey           *TypeRef     `json:"mapKey,omitempty"`
	MapValue         *TypeRef     `json:"mapValue,omitempty"`
	Name             string       `json:"name"`
	Docs             []string     `json:"docs,omitempty"`
	Directives       []string     `json:"directives,omitempty"`
	Annotations      tags.DocTags `json:"annotations,omitempty"`
}

func variableToWire(variable Variable) (wire variableWire) {

	wire.TypeID = variable.TypeID
	wire.NumberOfPointers = variable.NumberOfPointers
	wire.IsSlice = variable.IsSlice
	wire.ArrayLen = variable.ArrayLen
	wire.IsEllipsis = variable.IsEllipsis
	wire.ElementPointers = variable.ElementPointers
	wire.MapKey = variable.MapKey
	wire.MapValue = variable.MapValue
	wire.Name = variable.Name
	wire.Docs = variable.Docs
	wire.Directives = variable.Directives
	wire.Annotations = variable.Annotations

	return
}

func variableFromWire(wire variableWire) (variable Variable) {

	variable.TypeRef = TypeRef{
		TypeID:           wire.TypeID,
		NumberOfPointers: wire.NumberOfPointers,
		IsSlice:          wire.IsSlice,
		ArrayLen:         wire.ArrayLen,
		IsEllipsis:       wire.IsEllipsis,
		ElementPointers:  wire.ElementPointers,
		MapKey:           wire.MapKey,
		MapValue:         wire.MapValue,
	}
	variable.Name = wire.Name
	variable.Docs = wire.Docs
	variable.Directives = wire.Directives
	variable.Annotations = wire.Annotations

	return
}

// MarshalJSON сериализует Variable в плоский JSON без встраивания TypeRef.
func (variable Variable) MarshalJSON() (data []byte, err error) {

	return stdjson.Marshal(variableToWire(variable))
}

// UnmarshalJSON восстанавливает Variable из плоского JSON.
func (variable *Variable) UnmarshalJSON(data []byte) (err error) {

	var wire variableWire

	if err = stdjson.Unmarshal(data, &wire); err != nil {
		return
	}

	*variable = variableFromWire(wire)

	return
}
