// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package marker

import (
	"os"
	"path/filepath"
	"strings"

	"tgp/internal/generated"
	"tgp/internal/helper"
)

const (
	goMod = "go.mod"
	goSum = "go.sum"
)

func DiscoverPaths(rootDir string, contractsDir string, excludeDirs []string) (paths []string, err error) {

	rootDir = filepath.Clean(rootDir)
	paths = make([]string, 0)

	if _, statErr := os.Stat(filepath.Join(rootDir, goMod)); statErr == nil {
		paths = append(paths, goMod)
	}
	if _, statErr := os.Stat(filepath.Join(rootDir, goSum)); statErr == nil {
		paths = append(paths, goSum)
	}

	err = filepath.Walk(rootDir, func(absPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			if helper.IsDirNameExcluded(info.Name()) {
				return filepath.SkipDir
			}
			if discoverShouldExcludeDir(absPath, rootDir, excludeDirs) {
				return filepath.SkipDir
			}
			return nil
		}
		if !helper.IsRelevantGoFile(info.Name()) {
			return nil
		}
		if isGeneratedByHeader(absPath) {
			return nil
		}
		rel, relErr := filepath.Rel(rootDir, absPath)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, "./")
		if rel == "" || rel == "." {
			return nil
		}
		if helper.IsRelPathExcluded(rel, excludeDirs) {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	return paths, err
}

func discoverShouldExcludeDir(absDir string, rootDir string, excludeDirs []string) (yes bool) {

	rel, err := filepath.Rel(rootDir, absDir)
	if err != nil {
		return true
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "./")
	return helper.IsRelPathExcluded(rel, excludeDirs)
}

func isGeneratedByHeader(filePath string) (ok bool) {

	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 200)
	var n int
	n, _ = f.Read(buf)
	content := string(buf[:n])
	return strings.Contains(content, generated.ByToolGateway)
}
