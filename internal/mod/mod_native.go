//go:build !wasip1

package mod

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

// goModPath находит путь к go.mod файлу в нативном окружении через exec.Command.
func goModPath(root string) (string, error) {

	var stdout []byte
	var err error
	for {
		cmd := exec.Command("go", "env", "GOMOD")
		cmd.Dir = root
		stdout, err = cmd.Output()
		if err == nil {
			break
		}
		if _, ok := err.(*os.PathError); ok {
			// try to find go.mod on level higher
			r := filepath.Join(root, "..")
			if r == root { // when we in root directory stop trying
				return "", err
			}
			root = r
			continue
		}
		return "", err
	}
	goModPath := string(bytes.TrimSpace(stdout))
	return goModPath, nil
}
