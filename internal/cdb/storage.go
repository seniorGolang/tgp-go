package cdb

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-json"

	"tgp/internal/model"
)

// Корень — путь монтирования @tg/astg/db.
func Root() (root string, err error) {
	return "/tg/astg/db", nil
}

// Сжатый JSON, путь относительный от root.
func ReadProject(root string, relPath string) (project *model.Project, err error) {

	if strings.Contains(relPath, "..") || filepath.IsAbs(relPath) {
		return nil, fmt.Errorf("invalid project path: %s", relPath)
	}
	path := filepath.Join(root, relPath)

	var f *os.File
	if f, err = os.Open(path); err != nil {
		return nil, fmt.Errorf("open project file: %w", err)
	}
	defer f.Close()

	var gz *gzip.Reader
	if gz, err = gzip.NewReader(f); err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	var data []byte
	if data, err = io.ReadAll(gz); err != nil {
		return nil, fmt.Errorf("read project: %w", err)
	}

	project = new(model.Project)
	if err = json.Unmarshal(data, project); err != nil {
		return nil, fmt.Errorf("unmarshal project: %w", err)
	}
	return
}

// Сжатый JSON, путь относительный от root.
func WriteProject(root string, relPath string, project *model.Project) (err error) {

	if strings.Contains(relPath, "..") || filepath.IsAbs(relPath) {
		return fmt.Errorf("invalid project path: %s", relPath)
	}
	path := filepath.Join(root, relPath)

	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var data []byte
	if data, err = json.Marshal(project); err != nil {
		return fmt.Errorf("marshal project: %w", err)
	}

	var f *os.File
	if f, err = os.Create(path); err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	if _, err = gz.Write(data); err != nil {
		_ = gz.Close()
		return fmt.Errorf("write gzip: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	return
}
