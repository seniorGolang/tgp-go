package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"tgp/core/i18n"
)

func ensureEmptyOut(outDir string) (err error) {

	var empty bool
	if empty, err = isEmptyDir(outDir); err != nil || !empty {
		return errors.New(i18n.Msg("directory is not empty"))
	}
	return
}

func isEmptyDir(path string) (empty bool, err error) {

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return
	}
	if !info.IsDir() {
		return false, errors.New(i18n.Msg("path is not a directory"))
	}
	var hasGo bool
	err = filepath.WalkDir(path, func(entryPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			hasGo = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return !hasGo, nil
}
