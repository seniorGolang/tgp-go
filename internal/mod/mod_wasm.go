// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

//go:build wasip1

package mod

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tgp/core/exec"
)

func goModPath(root string) (string, error) {

	var goModPath string
	var err error
	currentDir := root

	for {
		cmd := exec.Command("go", "env", "GOMOD")
		cmd = cmd.Dir(currentDir)

		stdoutPipe, cmdErr := cmd.StdoutPipe()
		if cmdErr == nil {
			stdoutBytes, readErr := io.ReadAll(stdoutPipe)
			stdoutPipe.Close()

			waitErr := cmd.Wait()
			if waitErr == nil && readErr == nil {
				goModPath = strings.TrimSpace(string(stdoutBytes))
				if goModPath != "" {
					return goModPath, nil
				}
			}
			if readErr != nil {
				err = readErr
			}
			if waitErr != nil {
				err = waitErr
			}
		} else {
			err = cmdErr
		}

		goModPath := filepath.Join(currentDir, "go.mod")
		if _, statErr := os.Stat(goModPath); statErr == nil {
			return goModPath, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir { // когда мы в корневой директории, прекращаем попытки
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("go.mod not found starting from %s", root)
		}
		currentDir = parentDir
	}
}
