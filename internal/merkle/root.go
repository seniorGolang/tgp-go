// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package merkle

func Root(rootDir string, paths []string, opts ...Option) (marker string, err error) {

	b := NewBuilder(rootDir, opts...)
	b.AddPaths(paths)
	return b.Root()
}

func FileHashes(rootDir string, paths []string, opts ...Option) (files map[string]string, err error) {

	b := NewBuilder(rootDir, opts...)
	b.AddPaths(paths)
	return b.FileHashes()
}
