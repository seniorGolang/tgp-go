package cdb

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-json"

	"tgp/internal/model"
)

func LoadProject(rootDir string, projectFile string) (project *model.Project, err error) {

	path := filepath.Join(rootDir, projectFile)
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

	project = &model.Project{}
	if err = json.NewDecoder(gz).Decode(project); err != nil {
		return nil, fmt.Errorf("decode project: %w", err)
	}
	return
}

func SaveProject(rootDir string, projectFile string, project *model.Project) (err error) {

	path := filepath.Join(rootDir, projectFile)
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("mkdir project dir: %w", err)
	}

	var f *os.File
	if f, err = os.Create(path); err != nil {
		return fmt.Errorf("create project file: %w", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	if err = json.NewEncoder(gz).Encode(project); err != nil {
		return fmt.Errorf("encode project: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	return
}
