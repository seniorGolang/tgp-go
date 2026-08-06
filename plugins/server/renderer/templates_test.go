// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"io/fs"
	"strings"
	"testing"

	"tgp/internal/generated"
)

func TestPkgTemplatesHaveDoNotEditComment(t *testing.T) {

	var count int
	if err := fs.WalkDir(pkgTmplFS, ".", func(path string, entry fs.DirEntry, walkErr error) (err error) {

		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go.tmpl") {
			return nil
		}
		count++
		var body []byte
		if body, err = fs.ReadFile(pkgTmplFS, path); err != nil {
			return err
		}
		if !strings.HasPrefix(string(body), "{{.DoNotEditComment}}") && !strings.HasPrefix(string(body), generated.ByToolGatewayComment) {
			t.Errorf("%s: missing DoNotEditComment / Code generated prefix", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected pkg-tmpl *.go.tmpl files")
	}
}
