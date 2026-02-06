// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"bytes"
	"fmt"
	"go/build"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"tgp/core/i18n"
)

var (
	log = slog.Default().With("module", "server")
)

func GetPkgPath(fName string, isDir bool) (pkgPath string, err error) {

	var goModPath string
	if goModPath, err = GoModPath(fName, isDir); err != nil {
		log.Error(i18n.Msg("cannot find go.mod"), slog.Any("error", errors.Wrap(err, "cannot find go.mod because of")))
	}

	if strings.Contains(goModPath, "go.mod") {
		if pkgPath, err = GetPkgPathFromGoMod(fName, isDir, goModPath); err != nil {
			return
		}
		return
	}
	pkgPath, err = GetPkgPathFromGOPATH(fName, isDir)
	return
}

var (
	goModPathCache = make(map[string]string)
)

func GoModPath(fName string, isDir bool) (goModPath string, err error) {

	root := fName

	if !isDir {
		root = filepath.Dir(fName)
	}

	var ok bool
	if goModPath, ok = goModPathCache[root]; ok {
		return
	}

	defer func() {
		goModPathCache[root] = goModPath
	}()

	// @go монтируется так, что go.mod находится в корне: "/go.mod"
	candidatePath := "/go.mod"
	if _, err = os.Stat(candidatePath); err != nil {
		err = errors.New(i18n.Msg("go.mod not found: @go resolution not provided or go.mod is missing in /go.mod"))
		return
	}
	goModPath = candidatePath
	return
}

func GetPkgPathFromGoMod(fName string, isDir bool, goModPath string) (pkgPath string, err error) {

	modulePath := GetModulePath(goModPath)

	if modulePath == "" {
		err = errors.Errorf("cannot determine module path from %s", goModPath)
		return
	}

	rel := path.Join(modulePath, filePathToPackagePath(strings.TrimPrefix(fName, filepath.Dir(goModPath))))

	if !isDir {
		pkgPath = path.Dir(rel)
		return
	}
	pkgPath = path.Clean(rel)
	return
}

var (
	gopathCache           = ""
	modulePrefix          = []byte("\nmodule ")
	pkgPathFromGoModCache = make(map[string]string)
)

func GetModulePath(goModPath string) (pkgPath string) {

	var ok bool
	if pkgPath, ok = pkgPathFromGoModCache[goModPath]; ok {
		return
	}

	defer func() {
		pkgPathFromGoModCache[goModPath] = pkgPath
	}()

	var data []byte
	var err error
	if data, err = os.ReadFile(goModPath); err != nil {
		return
	}

	var i int

	if bytes.HasPrefix(data, modulePrefix[1:]) {
		i = 0
	} else {
		i = bytes.Index(data, modulePrefix)
		if i < 0 {
			return
		}
		i++
	}

	line := data[i:]

	var j int
	if j = bytes.IndexByte(line, '\n'); j >= 0 {
		line = line[:j]
	}

	if line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	line = line[len("module "):]

	// If quoted, unquote.
	pkgPath = strings.TrimSpace(string(line))

	if pkgPath != "" && pkgPath[0] == '"' {
		var s string
		if s, err = strconv.Unquote(pkgPath); err != nil {
			return
		}
		pkgPath = s
	}
	return
}

func GetPkgPathFromGOPATH(fName string, isDir bool) (pkgPath string, err error) {

	if gopathCache == "" {
		var gopath string
		if gopath = os.Getenv("GOPATH"); gopath == "" {
			if gopath, err = GetDefaultGoPath(); err != nil {
				err = errors.Wrap(err, "cannot determine GOPATH")
				return
			}
		}
		gopathCache = gopath
	}

	for _, p := range filepath.SplitList(gopathCache) {
		prefix := filepath.Join(p, "src") + string(filepath.Separator)
		if rel := strings.TrimPrefix(fName, prefix); rel != fName {
			if !isDir {
				pkgPath = path.Dir(filePathToPackagePath(rel))
				return
			}
			pkgPath = path.Clean(filePathToPackagePath(rel))
			return
		}
	}

	err = errors.Errorf("file '%s' is not in GOPATH. Checked paths:\n%s", fName, strings.Join(filepath.SplitList(gopathCache), "\n"))
	return
}

func filePathToPackagePath(path string) (pkgPath string) {

	pkgPath = filepath.ToSlash(path)
	return
}

func GetDefaultGoPath() (gopath string, err error) {

	if gopath = os.Getenv("GOPATH"); gopath != "" {
		return
	}

	// Fallback на build.Default.GOPATH
	if build.Default.GOPATH != "" {
		gopath = build.Default.GOPATH
		return
	}

	err = fmt.Errorf("GOPATH not found")
	return
}
