// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"regexp"
	"strings"
)

var numberSequence = regexp.MustCompile(`([a-zA-Z])(\d+)([a-zA-Z]?)`)
var numberReplacement = []byte(`$1 $2 $3`)

// Converts a string to CamelCase
func toCamelInitCase(s string, initCase bool) (result string) {

	s = addWordBoundariesToNumbers(s)
	s = strings.Trim(s, " ")
	n := ""
	capNext := initCase
	for _, v := range s {
		if v >= 'A' && v <= 'Z' {
			n += string(v)
		}
		if v >= '0' && v <= '9' {
			n += string(v)
		}
		if v >= 'a' && v <= 'z' {
			if capNext {
				n += strings.ToUpper(string(v))
			} else {
				n += string(v)
			}
		}
		if v == '_' || v == ' ' || v == '-' {
			capNext = true
		} else {
			capNext = false
		}
	}
	result = n
	return
}

// Converts a string to CamelCase
func ToCamel(s string) (result string) {

	result = toCamelInitCase(s, true)
	return
}

func isAllUpper(s string) (isUpper bool) {

	for _, v := range s {
		if v >= 'a' && v <= 'z' {
			return
		}
	}
	isUpper = true
	return
}

// Converts a string to lowerCamelCase
func ToLowerCamel(s string) (result string) {

	if isAllUpper(s) {
		result = s
		return
	}

	if s == "" {
		result = s
		return
	}
	var r rune
	if r = rune(s[0]); r >= 'A' && r <= 'Z' {
		s = strings.ToLower(string(r)) + s[1:]
	}
	result = toCamelInitCase(s, false)
	return
}

func addWordBoundariesToNumbers(s string) (result string) {

	b := []byte(s)
	b = numberSequence.ReplaceAll(b, numberReplacement)
	result = string(b)
	return
}
