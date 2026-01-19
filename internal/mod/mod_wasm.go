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

// goModPath находит путь к go.mod файлу в WASM окружении через хост.
func goModPath(root string) (string, error) {

	var goModPath string
	var err error
	currentDir := root

	for {
		// Пытаемся выполнить команду через хост
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

		// Если команда не сработала, пытаемся найти go.mod вручную
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, statErr := os.Stat(goModPath); statErr == nil {
			return goModPath, nil
		}

		// Пытаемся подняться на уровень выше
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
