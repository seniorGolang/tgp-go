// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tgp/core/i18n"
	"tgp/plugins/swagger/types"
)

func SaveFile(swaggerDoc types.Object, outFilePath string) (err error) {

	dir := filepath.Dir(outFilePath)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("%s: %w", fmt.Sprintf(i18n.Msg("failed to create directory %s"), dir), err)
	}

	var docData []byte
	ext := strings.ToLower(filepath.Ext(outFilePath))
	switch ext {
	case ".json":
		if docData, err = swaggerDoc.ToJSON(); err != nil {
			return fmt.Errorf("%s: %w", i18n.Msg("JSON marshaling error"), err)
		}
	case ".yaml", ".yml":
		if docData, err = swaggerDoc.ToYAML(); err != nil {
			return fmt.Errorf("%s: %w", i18n.Msg("YAML marshaling error"), err)
		}
	default:
		return fmt.Errorf(i18n.Msg("unsupported file format: %s (supported: .json, .yaml, .yml)"), ext)
	}

	if err = os.WriteFile(outFilePath, docData, 0600); err != nil {
		return fmt.Errorf("%s: %w", fmt.Sprintf(i18n.Msg("failed to write file %s"), outFilePath), err)
	}

	return
}
