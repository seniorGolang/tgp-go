// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

//go:build !wasip1

package mod

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

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
