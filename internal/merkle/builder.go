// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package merkle

import (
	"path/filepath"
	"strings"
)

type Builder struct {
	rootDir string
	paths   []string
	exclude []string
	pathSet map[string]struct{}
}

func NewBuilder(rootDir string, opts ...Option) (b *Builder) {

	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	b = &Builder{
		rootDir: filepath.Clean(rootDir),
		paths:   make([]string, 0),
		pathSet: make(map[string]struct{}),
	}
	if len(cfg.excludePrefixes) > 0 {
		b.exclude = make([]string, len(cfg.excludePrefixes))
		for i, p := range cfg.excludePrefixes {
			b.exclude[i] = normalizeExcludePrefix(p)
		}
	}
	return
}

func (b *Builder) AddPath(relPath string) {

	norm := normalizeRelPath(relPath)
	if norm == "" {
		return
	}
	if b.isExcluded(norm) {
		return
	}
	if _, ok := b.pathSet[norm]; ok {
		return
	}
	b.pathSet[norm] = struct{}{}
	b.paths = append(b.paths, norm)
}

func (b *Builder) AddPaths(relPaths []string) {

	for _, p := range relPaths {
		b.AddPath(p)
	}
}

func normalizeRelPath(relPath string) (norm string) {

	norm = filepath.ToSlash(filepath.Clean(relPath))
	norm = strings.TrimPrefix(norm, "./")
	norm = strings.TrimPrefix(norm, ".\\")
	return
}

func normalizeExcludePrefix(p string) (norm string) {

	norm = filepath.ToSlash(p)
	norm = strings.TrimPrefix(norm, "./")
	norm = strings.TrimPrefix(norm, ".\\")
	return
}

func (b *Builder) isExcluded(normPath string) (yes bool) {

	for _, prefix := range b.exclude {
		if normPath == prefix {
			return true
		}
		if !strings.HasPrefix(normPath, prefix) {
			continue
		}
		if strings.HasSuffix(prefix, "/") {
			return true
		}
		if len(normPath) == len(prefix) || normPath[len(prefix)] == '/' {
			return true
		}
	}
	return false
}
