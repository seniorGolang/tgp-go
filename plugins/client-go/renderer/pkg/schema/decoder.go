// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package schema

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// NewDecoder returns a new Decoder.
func NewDecoder() *Decoder {

	return &Decoder{cache: newCache()}
}

// Decoder decodes values from a map[string][]string to a struct.
type Decoder struct {
	cache             *cache
	zeroEmpty         bool
	ignoreUnknownKeys bool
}

// SetAliasTag changes the tag used to locate custom field aliases (e.g. "form").
func (d *Decoder) SetAliasTag(tag string) {

	d.cache.tag = tag
}

// ZeroEmpty sets whether empty string values set the field to zero.
func (d *Decoder) ZeroEmpty(z bool) {

	d.zeroEmpty = z
}

// IgnoreUnknownKeys sets whether unknown keys in the map are ignored.
func (d *Decoder) IgnoreUnknownKeys(i bool) {

	d.ignoreUnknownKeys = i
}

// RegisterConverter registers a converter function for a custom type.
func (d *Decoder) RegisterConverter(value any, converterFunc Converter) {

	d.cache.registerConverter(value, converterFunc)
}

// Decode decodes src into dst (pointer to struct). Keys may use dotted notation for nested fields.
func (d *Decoder) Decode(dst any, src map[string][]string) (err error) {

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("schema: interface must be a pointer to struct")
	}
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("schema: panic while decoding: %v", r)
			}
		}
	}()
	v = v.Elem()
	t := v.Type()
	multiError := MultiError{}
	for path, values := range src {
		parts, pathErr := d.cache.parsePath(path, t)
		if pathErr == nil {
			if decodeErr := d.decode(v, path, parts, values); decodeErr != nil {
				multiError[path] = decodeErr
			}
		} else {
			if errors.Is(pathErr, errIndexTooLarge) {
				multiError[path] = pathErr
			} else if !d.ignoreUnknownKeys {
				multiError[path] = UnknownKeyError{Key: path}
			}
		}
	}
	multiError.merge(d.checkRequired(t, src))
	if len(multiError) > 0 {
		return multiError
	}
	return nil
}

func (d *Decoder) checkRequired(t reflect.Type, src map[string][]string) MultiError {

	m, _ := d.findRequiredFields(t, "", "")
	errs := MultiError{}
	for key, fields := range m {
		if isEmptyFields(fields, src) {
			errs[key] = EmptyFieldError{Key: key}
		}
	}
	return errs
}

func (d *Decoder) findRequiredFields(t reflect.Type, canonicalPrefix, searchPrefix string) (map[string][]fieldWithPrefix, MultiError) {

	struc := d.cache.get(t)
	if struc == nil {
		return nil, MultiError{canonicalPrefix + "*": errors.New("cache fail")}
	}
	m := map[string][]fieldWithPrefix{}
	errs := MultiError{}
	for _, f := range struc.fields {
		if f.typ.Kind() == reflect.Struct {
			fcprefix := canonicalPrefix + f.canonicalAlias + "."
			for _, fspath := range f.paths(searchPrefix) {
				fm, ferrs := d.findRequiredFields(f.typ, fcprefix, fspath+".")
				for key, fields := range fm {
					m[key] = append(m[key], fields...)
				}
				errs.merge(ferrs)
			}
		}
		if f.isRequired {
			key := canonicalPrefix + f.canonicalAlias
			m[key] = append(m[key], fieldWithPrefix{fieldInfo: f, prefix: searchPrefix})
		}
	}
	return m, errs
}

type fieldWithPrefix struct {
	*fieldInfo
	prefix string
}

func isEmptyFields(fields []fieldWithPrefix, src map[string][]string) bool {

	for _, f := range fields {
		for _, path := range f.paths(f.prefix) {
			v, ok := src[path]
			if ok && !isEmpty(f.typ, v) {
				return false
			}
			for key := range src {
				nested := strings.IndexByte(key, '.') != -1
				c1 := strings.HasSuffix(f.prefix, ".") && key == path
				c2 := f.prefix == "" && nested && strings.HasPrefix(key, path)
				c3 := f.prefix == "" && !nested && key == path
				if !isEmpty(f.typ, src[key]) && (c1 || c2 || c3) {
					return false
				}
			}
		}
	}
	return true
}

func isEmpty(t reflect.Type, value []string) bool {

	if len(value) == 0 {
		return true
	}
	switch t.Kind() {
	case reflect.Bool, reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64, reflect.String, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return len(value[0]) == 0
	}
	return false
}

func (d *Decoder) decode(v reflect.Value, path string, parts []pathPart, values []string) error {

	for _, name := range parts[0].path {
		if v.Type().Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		if v.Type().Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				field := v.Field(i)
				if field.Type().Kind() == reflect.Ptr && field.IsNil() && v.Type().Field(i).Anonymous {
					field.Set(reflect.New(field.Type().Elem()))
				}
			}
		}
		v = v.FieldByName(name)
	}
	if !v.CanSet() {
		return nil
	}
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		if v.IsNil() {
			v.Set(reflect.New(t))
		}
		v = v.Elem()
	}
	if len(parts) > 1 {
		idx := parts[0].index
		if v.IsNil() || v.Len() < idx+1 {
			value := reflect.MakeSlice(t, idx+1, idx+1)
			if v.Len() < idx+1 {
				reflect.Copy(value, v)
			}
			v.Set(value)
		}
		return d.decode(v.Index(idx), path, parts[1:], values)
	}
	conv := d.cache.converter(t)
	m := isTextUnmarshaler(v)
	if conv == nil && t.Kind() == reflect.Slice && m.IsSliceElement {
		var items []reflect.Value
		elemT := t.Elem()
		isPtrElem := elemT.Kind() == reflect.Ptr
		if isPtrElem {
			elemT = elemT.Elem()
		}
		convElem := d.cache.converter(elemT)
		if convElem == nil {
			convElem = builtinConverters[elemT.Kind()]
			if convElem == nil {
				return fmt.Errorf("schema: converter not found for %v", elemT)
			}
		}
		for key, value := range values {
			if value == "" {
				if d.zeroEmpty {
					items = append(items, reflect.Zero(elemT))
				}
			} else if m.IsValid {
				u := reflect.New(elemT)
				if m.IsSliceElementPtr {
					u = reflect.New(reflect.PointerTo(elemT).Elem())
				}
				if err := u.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(value)); err != nil {
					return ConversionError{Key: path, Type: t, Index: key, Err: err}
				}
				switch {
				case m.IsSliceElementPtr:
					items = append(items, u.Elem().Addr())
				case u.Kind() == reflect.Ptr:
					items = append(items, u.Elem())
				default:
					items = append(items, u)
				}
			} else if item := convElem(value); item.IsValid() {
				if isPtrElem {
					ptr := reflect.New(elemT)
					ptr.Elem().Set(item)
					item = ptr
				}
				if item.Type() != elemT && !isPtrElem {
					item = item.Convert(elemT)
				}
				items = append(items, item)
			} else {
				if strings.Contains(value, ",") {
					vals := strings.Split(value, ",")
					for _, val := range vals {
						if val == "" {
							if d.zeroEmpty {
								items = append(items, reflect.Zero(elemT))
							}
						} else if item := convElem(val); item.IsValid() {
							if isPtrElem {
								ptr := reflect.New(elemT)
								ptr.Elem().Set(item)
								item = ptr
							}
							if item.Type() != elemT && !isPtrElem {
								item = item.Convert(elemT)
							}
							items = append(items, item)
						} else {
							return ConversionError{Key: path, Type: elemT, Index: key}
						}
					}
				} else {
					return ConversionError{Key: path, Type: elemT, Index: key}
				}
			}
		}
		v.Set(reflect.Append(reflect.MakeSlice(t, 0, 0), items...))
		return nil
	}
	val := ""
	if len(values) > 0 {
		val = values[len(values)-1]
	}
	if conv != nil {
		if value := conv(val); value.IsValid() {
			v.Set(value.Convert(t))
		} else {
			return ConversionError{Key: path, Type: t, Index: -1}
		}
	} else if m.IsValid {
		if m.IsPtr {
			u := reflect.New(v.Type())
			if err := u.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(val)); err != nil {
				return ConversionError{Key: path, Type: t, Index: -1, Err: err}
			}
			v.Set(reflect.Indirect(u))
		} else {
			if err := m.Unmarshaler.UnmarshalText([]byte(val)); err != nil {
				return ConversionError{Key: path, Type: t, Index: -1, Err: err}
			}
		}
	} else if val == "" {
		if d.zeroEmpty {
			v.Set(reflect.Zero(t))
		}
	} else if conv := builtinConverters[t.Kind()]; conv != nil {
		if value := conv(val); value.IsValid() {
			v.Set(value.Convert(t))
		} else {
			return ConversionError{Key: path, Type: t, Index: -1}
		}
	} else {
		return fmt.Errorf("schema: converter not found for %v", t)
	}
	return nil
}

func isTextUnmarshaler(v reflect.Value) unmarshaler {

	m := unmarshaler{}
	if m.Unmarshaler, m.IsValid = v.Interface().(encoding.TextUnmarshaler); m.IsValid {
		return m
	}
	if m.Unmarshaler, m.IsValid = reflect.New(v.Type()).Interface().(encoding.TextUnmarshaler); m.IsValid {
		m.IsPtr = true
		return m
	}
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		if m.Unmarshaler, m.IsValid = v.Interface().(encoding.TextUnmarshaler); m.IsValid {
			return m
		}
		m.IsSliceElement = true
		if t = t.Elem(); t.Kind() == reflect.Ptr {
			t = reflect.PointerTo(t.Elem())
			v = reflect.Zero(t)
			m.IsSliceElementPtr = true
			m.Unmarshaler, m.IsValid = v.Interface().(encoding.TextUnmarshaler)
			return m
		}
	}
	v = reflect.New(t)
	m.Unmarshaler, m.IsValid = v.Interface().(encoding.TextUnmarshaler)
	return m
}

type unmarshaler struct {
	Unmarshaler       encoding.TextUnmarshaler
	IsValid           bool
	IsPtr             bool
	IsSliceElement    bool
	IsSliceElementPtr bool
}

// ConversionError is returned when a value cannot be converted to the target type.
type ConversionError struct {
	Key   string
	Type  reflect.Type
	Index int
	Err   error
}

func (e ConversionError) Error() string {

	if e.Index < 0 {
		if e.Err != nil {
			return fmt.Sprintf("schema: error converting value for %q. Details: %s", e.Key, e.Err)
		}
		return fmt.Sprintf("schema: error converting value for %q", e.Key)
	}
	if e.Err != nil {
		return fmt.Sprintf("schema: error converting value for index %d of %q. Details: %s", e.Index, e.Key, e.Err)
	}
	return fmt.Sprintf("schema: error converting value for index %d of %q", e.Index, e.Key)
}

// UnknownKeyError is returned when a key in the map does not match a struct field.
type UnknownKeyError struct {
	Key string
}

func (e UnknownKeyError) Error() string {

	return fmt.Sprintf("schema: invalid path %q", e.Key)
}

// EmptyFieldError is returned when a required field is empty.
type EmptyFieldError struct {
	Key string
}

func (e EmptyFieldError) Error() string {

	return fmt.Sprintf("%v is empty", e.Key)
}

// MultiError holds multiple decoding errors.
type MultiError map[string]error

func (e MultiError) Error() string {

	var s string
	for _, err := range e {
		s = err.Error()
		break
	}
	switch len(e) {
	case 0:
		return "(0 errors)"
	case 1:
		return s
	case 2:
		return s + " (and 1 other error)"
	}
	return fmt.Sprintf("%s (and %d other errors)", s, len(e)-1)
}

func (e MultiError) merge(errors MultiError) {

	for key, err := range errors {
		if e[key] == nil {
			e[key] = err
		}
	}
}
