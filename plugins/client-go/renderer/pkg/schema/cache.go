// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package schema

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const maxParserIndex = 1000

var (
	errInvalidPath   = errors.New("schema: invalid path")
	errIndexTooLarge = errors.New("schema: index exceeds parser limit")
)

func newCache() *cache {

	c := cache{
		m:       make(map[reflect.Type]*structInfo),
		regconv: make(map[reflect.Type]Converter),
		tag:     "schema",
	}
	return &c
}

type cache struct {
	l       sync.RWMutex
	m       map[reflect.Type]*structInfo
	regconv map[reflect.Type]Converter
	tag     string
}

func (c *cache) registerConverter(value any, converterFunc Converter) {

	c.regconv[reflect.TypeOf(value)] = converterFunc
}

func (c *cache) parsePath(p string, t reflect.Type) ([]pathPart, error) {

	var struc *structInfo
	var field *fieldInfo
	var index64 int64
	var err error
	parts := make([]pathPart, 0)
	path := make([]string, 0)
	keys := strings.Split(p, ".")
	for i := 0; i < len(keys); i++ {
		if t.Kind() != reflect.Struct {
			return nil, errInvalidPath
		}
		if struc = c.get(t); struc == nil {
			return nil, errInvalidPath
		}
		if field = struc.get(keys[i]); field == nil {
			return nil, errInvalidPath
		}
		path = append(path, field.name)
		switch {
		case field.isSliceOfStructs && (!field.unmarshalerInfo.IsValid || (field.unmarshalerInfo.IsValid && field.unmarshalerInfo.IsSliceElement)):
			i++
			if i+1 > len(keys) {
				return nil, errInvalidPath
			}
			if index64, err = strconv.ParseInt(keys[i], 10, 0); err != nil {
				return nil, errInvalidPath
			}
			if index64 > maxParserIndex {
				return nil, errIndexTooLarge
			}
			parts = append(parts, pathPart{
				path:  path,
				field: field,
				index: int(index64),
			})
			path = make([]string, 0)
			if field.typ.Kind() == reflect.Ptr {
				t = field.typ.Elem()
			} else {
				t = field.typ
			}
			if t.Kind() == reflect.Slice {
				t = t.Elem()
				if t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
			}
		case field.typ.Kind() == reflect.Ptr:
			t = field.typ.Elem()
		default:
			t = field.typ
		}
	}
	parts = append(parts, pathPart{
		path:  path,
		field: field,
		index: -1,
	})
	return parts, nil
}

func (c *cache) get(t reflect.Type) *structInfo {

	c.l.RLock()
	info := c.m[t]
	c.l.RUnlock()
	if info == nil {
		info = c.create(t, "")
		c.l.Lock()
		c.m[t] = info
		c.l.Unlock()
	}
	return info
}

func (c *cache) create(t reflect.Type, parentAlias string) *structInfo {

	info := &structInfo{}
	var anonymousInfos []*structInfo
	for i := 0; i < t.NumField(); i++ {
		if f := c.createField(t.Field(i), parentAlias); f != nil {
			info.fields = append(info.fields, f)
			if ft := indirectType(f.typ); ft.Kind() == reflect.Struct && f.isAnonymous {
				anonymousInfos = append(anonymousInfos, c.create(ft, f.canonicalAlias))
			}
		}
	}
	for i, a := range anonymousInfos {
		others := make([]*structInfo, 0, len(anonymousInfos))
		others = append(others, info)
		others = append(others, anonymousInfos[:i]...)
		others = append(others, anonymousInfos[i+1:]...)
		for _, f := range a.fields {
			if !containsAlias(others, f.alias) {
				info.fields = append(info.fields, f)
			}
		}
	}
	return info
}

func (c *cache) createField(field reflect.StructField, parentAlias string) *fieldInfo {

	alias, options := fieldAlias(field, c.tag)
	if alias == "-" {
		return nil
	}
	canonicalAlias := alias
	if parentAlias != "" {
		canonicalAlias = parentAlias + "." + alias
	}
	var isSlice, isStruct bool
	ft := field.Type
	m := isTextUnmarshaler(reflect.Zero(ft))
	if ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	if isSlice = ft.Kind() == reflect.Slice; isSlice {
		ft = ft.Elem()
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
	}
	if ft.Kind() == reflect.Array {
		ft = ft.Elem()
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
	}
	if isStruct = ft.Kind() == reflect.Struct; !isStruct {
		if c.converter(ft) == nil && builtinConverters[ft.Kind()] == nil {
			return nil
		}
	}
	return &fieldInfo{
		typ:              field.Type,
		name:             field.Name,
		alias:            alias,
		canonicalAlias:   canonicalAlias,
		unmarshalerInfo:  m,
		isSliceOfStructs: isSlice && isStruct,
		isAnonymous:      field.Anonymous,
		isRequired:       options.Contains("required"),
	}
}

func (c *cache) converter(t reflect.Type) Converter {

	return c.regconv[t]
}

type structInfo struct {
	fields []*fieldInfo
}

func (i *structInfo) get(alias string) *fieldInfo {

	for _, field := range i.fields {
		if strings.EqualFold(field.alias, alias) {
			return field
		}
	}
	return nil
}

func containsAlias(infos []*structInfo, alias string) bool {

	for _, info := range infos {
		if info.get(alias) != nil {
			return true
		}
	}
	return false
}

type fieldInfo struct {
	typ              reflect.Type
	name             string
	alias            string
	canonicalAlias   string
	unmarshalerInfo  unmarshaler
	isSliceOfStructs bool
	isAnonymous      bool
	isRequired       bool
}

func (f *fieldInfo) paths(prefix string) []string {

	if f.alias == f.canonicalAlias {
		return []string{prefix + f.alias}
	}
	return []string{prefix + f.alias, prefix + f.canonicalAlias}
}

type pathPart struct {
	field *fieldInfo
	path  []string
	index int
}

func indirectType(typ reflect.Type) reflect.Type {

	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}
	return typ
}

func fieldAlias(field reflect.StructField, tagName string) (alias string, options tagOptions) {

	if tag := field.Tag.Get(tagName); tag != "" {
		alias, options = parseTag(tag)
	}
	if alias == "" {
		alias = field.Name
	}
	return alias, options
}

type tagOptions []string

func parseTag(tag string) (string, tagOptions) {

	s := strings.Split(tag, ",")
	return s[0], s[1:]
}

func (o tagOptions) Contains(option string) bool {

	for _, s := range o {
		if s == option {
			return true
		}
	}
	return false
}
