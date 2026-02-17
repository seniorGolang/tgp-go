// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (b *Builder) FileHashes() (files map[string]string, err error) {

	leaves, err := b.readLeaves()

	if err != nil {
		return nil, err
	}
	files = make(map[string]string, len(leaves))
	for _, l := range leaves {
		files[l.path] = hex.EncodeToString(l.hash)
	}
	return files, nil
}

func (b *Builder) Root() (marker string, err error) {

	leaves, err := b.readLeaves()

	if err != nil {
		return "", err
	}
	if len(leaves) == 0 {
		h := sha256.Sum256(nil)
		return hex.EncodeToString(h[:]), nil
	}
	sort.Slice(leaves, func(i, j int) bool { return leaves[i].path < leaves[j].path })
	rootHash := buildTree(leaves)
	return hex.EncodeToString(rootHash), nil
}

func buildTree(leaves []leaf) (root []byte) {

	level := make([][]byte, 0, len(leaves))

	for _, l := range leaves {
		level = append(level, l.hash)
	}
	for len(level) > 1 {
		next := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				next = append(next, hashPair(level[i], level[i+1]))
			} else {
				next = append(next, hashPair(level[i], level[i]))
			}
		}
		level = next
	}
	return level[0]
}

func hashPair(left, right []byte) (out []byte) {

	h := sha256.New()

	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

type leaf struct {
	path string
	hash []byte
}

func (b *Builder) readLeaves() (leaves []leaf, err error) {

	leaves = make([]leaf, 0, len(b.paths))
	hasher := sha256.New()

	for _, rel := range b.paths {
		absPath := filepath.Join(b.rootDir, filepath.FromSlash(rel))
		absPath = filepath.Clean(absPath)
		if !underRoot(b.rootDir, absPath) {
			return nil, fmt.Errorf("path escapes root: %s", rel)
		}
		var f *os.File
		if f, err = os.Open(absPath); err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		hasher.Reset()
		if _, err = io.Copy(hasher, f); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		_ = f.Close()
		leaves = append(leaves, leaf{path: rel, hash: append([]byte(nil), hasher.Sum(nil)...)})
	}
	return leaves, nil
}

func underRoot(rootDir string, absPath string) (ok bool) {

	rel, err := filepath.Rel(rootDir, absPath)

	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
