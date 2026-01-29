package generator

import (
	"errors"
	"os"

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
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}
