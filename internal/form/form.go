package form

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

func DecodeToStruct(values url.Values, dst any) (err error) {

	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return nil
	}
	dstVal = dstVal.Elem()
	if dstVal.Kind() != reflect.Struct {
		return nil
	}
	typ := dstVal.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		key := strings.TrimSpace(strings.Split(tag, ",")[0])
		s := values.Get(key)
		if s == "" {
			continue
		}
		fv := dstVal.Field(i)
		if !fv.CanSet() {
			continue
		}
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(s)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, e := strconv.ParseInt(s, 10, 64)
			if e != nil {
				continue
			}
			fv.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, e := strconv.ParseUint(s, 10, 64)
			if e != nil {
				continue
			}
			fv.SetUint(n)
		case reflect.Bool:
			b, _ := strconv.ParseBool(s)
			fv.SetBool(b)
		case reflect.Float32, reflect.Float64:
			x, e := strconv.ParseFloat(s, 64)
			if e != nil {
				continue
			}
			fv.SetFloat(x)
		case reflect.Ptr:
			if fv.Type().Elem().Kind() == reflect.String {
				fv.Set(reflect.ValueOf(&s))
			}
		default:
			fv.Set(reflect.ValueOf(s).Convert(fv.Type()))
		}
	}
	return nil
}

func EncodeFromStruct(src any) (values url.Values) {

	values = make(url.Values)
	srcVal := reflect.ValueOf(src)
	for srcVal.Kind() == reflect.Ptr && !srcVal.IsNil() {
		srcVal = srcVal.Elem()
	}
	if srcVal.Kind() != reflect.Struct {
		return values
	}
	typ := srcVal.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		key := strings.TrimSpace(strings.Split(tag, ",")[0])
		fv := srcVal.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}
		var s string
		switch fv.Kind() {
		case reflect.String:
			s = fv.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			s = strconv.FormatInt(fv.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			s = strconv.FormatUint(fv.Uint(), 10)
		case reflect.Bool:
			s = strconv.FormatBool(fv.Bool())
		case reflect.Float32, reflect.Float64:
			s = strconv.FormatFloat(fv.Float(), 'f', -1, 64)
		default:
			s = fmtString(fv)
		}
		if s != "" {
			values.Set(key, s)
		}
	}
	return values
}

func fmtString(v reflect.Value) string {

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Interface:
		if !v.IsNil() {
			return fmt.Sprint(v.Interface())
		}
	}
	return fmt.Sprint(v.Interface())
}
