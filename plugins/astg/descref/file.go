// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package descref

import (
	"os"
	"path/filepath"
	"strings"
)

func readFileContent(rootDir string, relPath string, section string) (out string, err error) {

	cleanRel := filepath.Clean(filepath.ToSlash(relPath))
	if strings.HasPrefix(cleanRel, "..") {
		return "", os.ErrNotExist
	}
	absPath := filepath.Join(rootDir, cleanRel)

	var data []byte
	if data, err = os.ReadFile(absPath); err != nil {
		return "", err
	}
	content := string(data)
	if section == "" {
		return strings.TrimSpace(content), nil
	}
	return strings.TrimSpace(extractSectionFromMarkdown(content, section)), nil
}
