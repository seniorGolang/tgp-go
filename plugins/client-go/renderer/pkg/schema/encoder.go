// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package schema

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

type encoderFunc func(reflect.Value) string

// Encoder encodes a struct into map[string][]string.
type Encoder struct {
	cache  *cache
	regenc map[reflect.Type]encoderFunc
}

// NewEncoder returns a new Encoder.
func NewEncoder() *Encoder {

	return &Encoder{cache: newCache(), regenc: make(map[reflect.Type]encoderFunc)}
}

// Encode encodes src into dst.
func (e *Encoder) Encode(src any, dst map[string][]string) error {

	return e.encode(reflect.ValueOf(src), dst)
}

// RegisterEncoder registers an encoder for a custom type.
func (e *Encoder) RegisterEncoder(value any, encoder func(reflect.Value) string) {

	e.regenc[reflect.TypeOf(value)] = encoder
}

// SetAliasTag sets the struct tag used for field names.
func (e *Encoder) SetAliasTag(tag string) {

	e.cache.tag = tag
}

func isValidStructPointer(v reflect.Value) bool {

	return v.Type().Kind() == reflect.Ptr && v.Elem().IsValid() && v.Elem().Type().Kind() == reflect.Struct
}

func isZero(v reflect.Value) bool {

	switch v.Kind() {
	case reflect.Func:
		return true
	case reflect.Map, reflect.Slice:
		return v.IsNil() || v.Len() == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		type zero interface {
			IsZero() bool
		}
		if v.Type().Implements(reflect.TypeOf((*zero)(nil)).Elem()) {
			return v.MethodByName("IsZero").Call(nil)[0].Interface().(bool)
		}
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}

func (e *Encoder) encode(v reflect.Value, dst map[string][]string) error {

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return errors.New("schema: interface must be a struct")
	}
	t := v.Type()
	encodeErrors := MultiError{}
	for i := 0; i < v.NumField(); i++ {
		name, opts := fieldAlias(t.Field(i), e.cache.tag)
		if name == "-" {
			continue
		}
		if isValidStructPointer(v.Field(i)) {
			_ = e.encode(v.Field(i).Elem(), dst)
			continue
		}
		encFunc := typeEncoder(v.Field(i).Type(), e.regenc)
		if encFunc != nil {
			value := encFunc(v.Field(i))
			if opts.Contains("omitempty") && isZero(v.Field(i)) {
				continue
			}
			dst[name] = append(dst[name], value)
			continue
		}
		if v.Field(i).Type().Kind() == reflect.Struct {
			_ = e.encode(v.Field(i), dst)
			continue
		}
		if v.Field(i).Type().Kind() == reflect.Slice {
			encFunc = typeEncoder(v.Field(i).Type().Elem(), e.regenc)
		}
		if encFunc == nil {
			encodeErrors[v.Field(i).Type().String()] = fmt.Errorf("schema: encoder not found for %v", v.Field(i))
			continue
		}
		if v.Field(i).Len() == 0 && opts.Contains("omitempty") {
			continue
		}
		dst[name] = nil
		for j := 0; j < v.Field(i).Len(); j++ {
			dst[name] = append(dst[name], encFunc(v.Field(i).Index(j)))
		}
	}
	if len(encodeErrors) > 0 {
		return encodeErrors
	}
	return nil
}

func typeEncoder(t reflect.Type, reg map[reflect.Type]encoderFunc) encoderFunc {

	if f, ok := reg[t]; ok {
		return f
	}
	switch t.Kind() {
	case reflect.Bool:
		return encodeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encodeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return encodeUint
	case reflect.Float32:
		return encodeFloat32
	case reflect.Float64:
		return encodeFloat64
	case reflect.Ptr:
		f := typeEncoder(t.Elem(), reg)
		return func(v reflect.Value) string {
			if v.IsNil() {
				return "null"
			}
			return f(v.Elem())
		}
	case reflect.String:
		return encodeString
	default:
		return nil
	}
}

func encodeBool(v reflect.Value) string {

	return strconv.FormatBool(v.Bool())
}

func encodeInt(v reflect.Value) string {

	return strconv.FormatInt(v.Int(), 10)
}

func encodeUint(v reflect.Value) string {

	return strconv.FormatUint(v.Uint(), 10)
}

func encodeFloat(v reflect.Value, bits int) string {

	return strconv.FormatFloat(v.Float(), 'f', 6, bits)
}

func encodeFloat32(v reflect.Value) string {

	return encodeFloat(v, 32)
}

func encodeFloat64(v reflect.Value) string {

	return encodeFloat(v, 64)
}

func encodeString(v reflect.Value) string {

	return v.String()
}
