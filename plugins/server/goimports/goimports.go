// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package goimports

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	Name string
	In   io.Reader
	Out  io.Writer
}

type Runner struct {
	files []File
}

func NewFromFile(path string) (runner Runner, err error) {

	runner.files, err = buildFile(path)
	return
}

func (r Runner) Run(modulePath string) (err error) {

	for _, file := range r.files {
		if err = r.processFile(file, modulePath); err != nil {
			return
		}
	}
	return
}

func (r Runner) processFile(file File, modulePath string) (err error) {

	var src []byte
	if file.In == nil {
		if src, err = os.ReadFile(file.Name); err != nil {
			return
		}
	} else {
		if src, err = io.ReadAll(file.In); err != nil {
			return
		}
	}

	res, err := formatImports(src, file.Name, modulePath)
	if err != nil {
		err = nil
		return
	}

	if len(res) == 0 {
		return
	}

	if bytes.Equal(src, res) {
		if s, ok := file.In.(io.Seeker); ok {
			_, err = s.Seek(0, 0)
		}
		return
	}

	if file.Out == nil {
		err = os.WriteFile(file.Name, res, 0)
		return
	}

	_, err = file.Out.Write(res)
	if c, ok := file.Out.(io.Closer); ok {
		_ = c.Close()
	}
	return
}

func isGoFile(f os.FileInfo) (ok bool) {

	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func buildFile(path string) (files []File, err error) {

	info, _ := os.Stat(path)
	if info == nil {
		return files, nil
	}
	if info.IsDir() {
		return files, nil
	}
	if !isGoFile(info) {
		return files, nil
	}
	var b []byte
	if b, err = os.ReadFile(path); err != nil {
		return
	}
	files = append(files, File{
		Name: path,
		In:   bytes.NewReader(b),
	})
	return
}

func GetModulePath(filePath string) (s string) {

	dir := filepath.Dir(filePath)
	for {
		if dir == "" || dir == "/" {
			return ""
		}
		goModPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			// Простой парсинг module path из go.mod
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
					return strings.Trim(modulePath, `"`)
				}
			}
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break
		}
		dir = parentDir
	}
	return ""
}
