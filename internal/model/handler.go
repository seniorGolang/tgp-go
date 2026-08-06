// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"errors"
	"fmt"
	"strings"
)

const (
	TagHandler      = "handler"
	TagHTTPResponse = "http-response"
)

// ParseHandlerRef разбирает значение аннотаций handler и http-response: module/path:FuncName.
func ParseHandlerRef(value string) (pkgPath string, funcName string, err error) {

	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", errors.New("handler annotation must be in format module/path:FuncName")
	}

	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("handler annotation %q must be in format module/path:FuncName", value)
	}

	pkgPath = strings.TrimSpace(parts[0])
	funcName = strings.TrimSpace(parts[1])
	if pkgPath == "" || funcName == "" || strings.ContainsAny(funcName, ", ") {
		return "", "", fmt.Errorf("handler annotation %q must be in format module/path:FuncName", value)
	}
	return pkgPath, funcName, nil
}

// HandlerInfoFromAnnotations возвращает HandlerInfo best-effort без ошибки (для astg).
func HandlerInfoFromAnnotations(annotations map[string]string) (handlerInfo *HandlerInfo) {

	if annotations == nil {
		return nil
	}

	value := annotations[TagHandler]
	if value == "" {
		value = annotations[TagHTTPResponse]
	}
	if value == "" {
		return nil
	}

	var pkgPath string
	var funcName string
	var err error
	if pkgPath, funcName, err = ParseHandlerRef(value); err != nil {
		return nil
	}
	return &HandlerInfo{PkgPath: pkgPath, Name: funcName}
}
