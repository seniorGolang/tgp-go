// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package path

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"tgp/core/data"
	"tgp/core/i18n"
)

const wasmRoot = "/"

func ResolveOutput(request data.Storage) (out string, err error) {

	var raw string
	if raw, err = data.Get[string](request, "out"); err != nil || raw == "" {
		return "", errors.New(i18n.Msg("out option is required and must be a string"))
	}
	return resolve(raw)
}

func ResolveOptional(request data.Storage, key string) (absPath string, ok bool, err error) {

	var raw string
	if raw, err = data.Get[string](request, key); err != nil {
		if errors.Is(err, data.ErrNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("%s: %w", i18n.Msg("failed to get option"), err)
	}
	if raw == "" {
		return "", false, nil
	}
	if absPath, err = resolve(raw); err != nil {
		return "", false, err
	}
	return absPath, true, nil
}

func ResolveRaw(raw string) (absPath string, err error) {

	return resolve(raw)
}

func resolve(raw string) (absPath string, err error) {

	if raw == "" {
		return "", errors.New("empty path")
	}
	cleaned := filepath.Clean(raw)
	if !filepath.IsAbs(cleaned) {
		if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path %q is outside project root", raw)
		}
		absPath = filepath.Join(wasmRoot, cleaned)
	} else {
		absPath = cleaned
	}
	absPath = filepath.Clean(absPath)
	var rel string
	if rel, err = filepath.Rel(wasmRoot, absPath); err != nil {
		return "", fmt.Errorf("resolve path %q: %w", raw, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside project root", raw)
	}
	return absPath, nil
}
