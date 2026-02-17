// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package merkle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoot_emptyPaths(t *testing.T) {

	dir := t.TempDir()

	marker, err := Root(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if marker == "" {
		t.Error("empty paths root must be non-empty (SHA256 of empty)")
	}

	marker2, err := Root(dir, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if marker != marker2 {
		t.Errorf("nil and empty slice must give same root: %q != %q", marker, marker2)
	}
}

func TestRoot_deterministic(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("b"), 0600)

	m1, err := Root(dir, []string{"a.go", "b.go"})
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Root(dir, []string{"b.go", "a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 {
		t.Errorf("order of paths must not change root: %q != %q", m1, m2)
	}
}

func TestRoot_contentChange(t *testing.T) {

	dir := t.TempDir()
	p := filepath.Join(dir, "x.go")

	_ = os.WriteFile(p, []byte("v1"), 0600)
	m1, err := Root(dir, []string{"x.go"})
	if err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(p, []byte("v2"), 0600)
	m2, err := Root(dir, []string{"x.go"})
	if err != nil {
		t.Fatal(err)
	}
	if m1 == m2 {
		t.Error("content change must change root")
	}
}

func TestRoot_pathNormalization(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("b"), 0600)

	base, err := Root(dir, []string{"a.go", "b.go"})
	if err != nil {
		t.Fatal(err)
	}

	for _, paths := range [][]string{
		{"./a.go", "./b.go"},
		{"a.go", "b.go"},
		{"a.go", "x/../b.go"},
	} {
		m, err := Root(dir, paths)
		if err != nil {
			t.Fatalf("paths %v: %v", paths, err)
		}
		if m != base {
			t.Errorf("normalized paths %v: got %q want %q", paths, m, base)
		}
	}
}

func TestRoot_duplicatePaths(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)

	m1, err := Root(dir, []string{"a.go", "a.go", "a.go"})
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Root(dir, []string{"a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 {
		t.Errorf("duplicate paths must dedupe: %q != %q", m1, m2)
	}
}

func TestRoot_missingFile(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)

	_, err := Root(dir, []string{"a.go", "missing.go"})
	if err == nil {
		t.Fatal("missing file must return error")
	}
}

func TestRoot_pathEscapesRoot(t *testing.T) {

	dir := t.TempDir()
	parent := filepath.Dir(dir)
	_ = os.WriteFile(filepath.Join(parent, "outside"), []byte("x"), 0600)

	escape := filepath.Join("..", "outside")
	_, err := Root(dir, []string{escape})
	if err == nil {
		t.Fatal("path escaping root must return error")
	}
}

func TestRoot_excludePrefixes(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.MkdirAll(filepath.Join(dir, "vendor", "x"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "vendor", "x", "y.go"), []byte("y"), 0600)

	withVendor, err := Root(dir, []string{"a.go", "vendor/x/y.go"})
	if err != nil {
		t.Fatal(err)
	}
	withoutVendor, err := Root(dir, []string{"a.go", "vendor/x/y.go"}, ExcludePrefixes("vendor/"))
	if err != nil {
		t.Fatal(err)
	}
	if withVendor == withoutVendor {
		t.Error("excluded path must change root")
	}
	onlyA, err := Root(dir, []string{"a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if withoutVendor != onlyA {
		t.Errorf("excluded path must be ignored: %q != %q", withoutVendor, onlyA)
	}
}

func TestRoot_excludePrefix_exactMatch(t *testing.T) {

	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "astg"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "astg", "gen.go"), []byte("gen"), 0600)

	included, _ := Root(dir, []string{"astg/gen.go"})
	excluded, _ := Root(dir, []string{"astg/gen.go"}, ExcludePrefixes("astg/"))
	if included == excluded {
		t.Error("excluded path must not contribute to root")
	}
	empty, _ := Root(dir, []string{}, ExcludePrefixes("astg/"))
	if excluded != empty {
		t.Errorf("only excluded path must equal empty root: %q != %q", excluded, empty)
	}
}

func TestRoot_excludePrefix_withTrailingSlash(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "vendors"), []byte("file"), 0600)
	_ = os.MkdirAll(filepath.Join(dir, "vendor", "x"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "vendor", "x", "y.go"), []byte("y"), 0600)

	all, _ := Root(dir, []string{"vendors", "vendor/x/y.go"})
	excl, _ := Root(dir, []string{"vendors", "vendor/x/y.go"}, ExcludePrefixes("vendor/"))
	if all == excl {
		t.Error("exclude prefix vendor/ must drop vendor/x/y.go")
	}
	onlyVendors, _ := Root(dir, []string{"vendors"})
	if excl != onlyVendors {
		t.Errorf("with vendor/ exclude only vendors file must remain: %q != %q", excl, onlyVendors)
	}
}

func TestRoot_multipleExcludePrefixes(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.MkdirAll(filepath.Join(dir, "vendor", "x"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "vendor", "x", "v.go"), []byte("v"), 0600)
	_ = os.MkdirAll(filepath.Join(dir, "astg"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "astg", "g.go"), []byte("g"), 0600)

	onlyA, _ := Root(dir, []string{"a.go"})
	allThree, _ := Root(dir, []string{"a.go", "vendor/x/v.go", "astg/g.go"}, ExcludePrefixes("vendor/", "astg/"))
	if onlyA != allThree {
		t.Errorf("multiple excludes must leave only a.go: %q != %q", onlyA, allThree)
	}
}

func TestRoot_emptyFile(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "empty.go"), nil, 0600)

	m, err := Root(dir, []string{"empty.go"})
	if err != nil {
		t.Fatal(err)
	}
	if m == "" {
		t.Error("empty file root must be non-empty")
	}
}

func TestRoot_rootDirCleaned(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "f.go"), []byte("x"), 0600)

	clean := filepath.Clean(dir)
	m1, err := Root(dir, []string{"f.go"})
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Root(clean+string(filepath.Separator)+".", []string{"f.go"})
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 {
		t.Errorf("root dir normalization: %q != %q", m1, m2)
	}
}

func TestBuilder_emptyRoot(t *testing.T) {

	dir := t.TempDir()
	b := NewBuilder(dir)
	marker, err := b.Root()
	if err != nil {
		t.Fatal(err)
	}
	funcRoot, _ := Root(dir, []string{})
	if marker != funcRoot {
		t.Errorf("Builder empty vs Root empty: %q != %q", marker, funcRoot)
	}
}

func TestBuilder_sameAsRoot(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("b"), 0600)
	paths := []string{"a.go", "b.go"}

	funcMarker, err := Root(dir, paths)
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuilder(dir)
	b.AddPaths(paths)
	builderMarker, err := b.Root()
	if err != nil {
		t.Fatal(err)
	}
	if funcMarker != builderMarker {
		t.Errorf("Builder.Root() must match Root(): %q != %q", builderMarker, funcMarker)
	}
}

func TestBuilder_addPathIncremental(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("b"), 0600)

	b := NewBuilder(dir)
	b.AddPath("a.go")
	b.AddPath("b.go")
	m1, _ := b.Root()

	b2 := NewBuilder(dir)
	b2.AddPaths([]string{"a.go", "b.go"})
	m2, _ := b2.Root()
	if m1 != m2 {
		t.Errorf("incremental AddPath vs AddPaths: %q != %q", m1, m2)
	}
}

func TestBuilder_dedupe(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "x.go"), []byte("x"), 0600)

	b := NewBuilder(dir)
	b.AddPath("x.go")
	b.AddPath("x.go")
	b.AddPath("./x.go")
	m, err := b.Root()
	if err != nil {
		t.Fatal(err)
	}
	single, _ := Root(dir, []string{"x.go"})
	if m != single {
		t.Errorf("Builder must dedupe paths: %q != %q", m, single)
	}
}

func TestBuilder_excludeViaAddPath(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.MkdirAll(filepath.Join(dir, "vendor", "x"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "vendor", "v.go"), []byte("v"), 0600)

	b := NewBuilder(dir, ExcludePrefixes("vendor/"))
	b.AddPath("a.go")
	b.AddPath("vendor/v.go")
	m, err := b.Root()
	if err != nil {
		t.Fatal(err)
	}
	onlyA, _ := Root(dir, []string{"a.go"})
	if m != onlyA {
		t.Errorf("Builder must exclude via option: %q != %q", m, onlyA)
	}
}

func TestBuilder_missingFile(t *testing.T) {

	dir := t.TempDir()
	b := NewBuilder(dir)
	b.AddPath("nonexistent.go")
	_, err := b.Root()
	if err == nil {
		t.Fatal("missing file must return error")
	}
}

func TestBuilder_pathEscapesRoot(t *testing.T) {

	dir := t.TempDir()
	parent := filepath.Dir(dir)
	_ = os.WriteFile(filepath.Join(parent, "out"), []byte("x"), 0600)

	b := NewBuilder(dir)
	b.AddPath(filepath.Join("..", "out"))
	_, err := b.Root()
	if err == nil {
		t.Fatal("path escaping root must return error")
	}
}

func TestRoot_hexFormat(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "f.go"), []byte("x"), 0600)
	m, err := Root(dir, []string{"f.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 64 {
		t.Errorf("marker must be 64 hex chars (SHA256), got len %d", len(m))
	}
	for _, c := range m {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		t.Errorf("marker must be hex lowercase: invalid byte %q", c)
		break
	}
}

func TestFileHashes(t *testing.T) {

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte("a"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "b.go"), []byte("b"), 0600)
	paths := []string{"a.go", "b.go"}

	files, err := FileHashes(dir, paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("FileHashes: want 2 entries, got %d", len(files))
	}
	for path, h := range files {
		if len(h) != 64 {
			t.Errorf("FileHashes[%q]: want 64 hex chars, got len %d", path, len(h))
		}
	}
}
