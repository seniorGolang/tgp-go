// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package converter

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/dave/jennifer/jen" //nolint:revive

	"tgp/internal/model"
)

func renderCode(t *testing.T, code *Statement) (source string) {

	t.Helper()
	f := NewFile("test")
	f.Func().Id("generated").Params().Block(code)
	var buf bytes.Buffer
	if err := f.Render(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	source = buf.String()
	return
}

func renderExpr(t *testing.T, code *Statement) (source string) {

	t.Helper()
	f := NewFile("test")
	f.Func().Id("generated").Params().String().Block(Return(code))
	var buf bytes.Buffer
	if err := f.Render(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	source = buf.String()
	return
}

func testFieldType(typeID string, numberOfPointers int, _ bool) (st *Statement) {

	st = &Statement{}
	for range numberOfPointers {
		st.Op("*")
	}
	switch typeID {
	case "int64":
		return st.Int64()
	case "string":
		return st.String()
	case "bool":
		return st.Bool()
	default:
		return st.Id(typeID)
	}
}

func TestBuildStringToType_SliceParsesElemStr(t *testing.T) {

	st := BuildStringToType(StringToTypeConfig{
		Project: &model.Project{Types: map[string]*model.Type{}},
		From:    Id("_animalID_"),
		Arg: &model.Variable{
			Name:    "animalID",
			TypeID:  "int64",
			IsSlice: true,
		},
		ID: Id("animalID"),
		ErrBody: []Code{
			Return(Err()),
		},
		FieldType: testFieldType,
		AddImport: func(_ string, _ string) {},
		JSONPkg:   "encoding/json",
	})
	source := renderCode(t, st)
	if !strings.Contains(source, `strconv.ParseInt(elemStr, 10, 64)`) {
		t.Fatalf("expected ParseInt(elemStr, ...), got:\n%s", source)
	}
	if strings.Contains(source, `strconv.ParseInt(_animalID_`) {
		t.Fatalf("must not ParseInt full query string:\n%s", source)
	}
	if !strings.Contains(source, `strings.Split(_animalID_, ",")`) {
		t.Fatalf("expected Split of full string:\n%s", source)
	}
}

func TestBuildStringToType_SliceStringUsesElemStr(t *testing.T) {

	st := BuildStringToType(StringToTypeConfig{
		Project: &model.Project{Types: map[string]*model.Type{}},
		From:    Id("_tags_"),
		Arg: &model.Variable{
			Name:    "tags",
			TypeID:  "string",
			IsSlice: true,
		},
		ID:        Id("tags"),
		FieldType: testFieldType,
		AddImport: func(_ string, _ string) {},
		JSONPkg:   "encoding/json",
	})
	source := renderCode(t, st)
	if !strings.Contains(source, "elem = elemStr") {
		t.Fatalf("expected assign from elemStr, got:\n%s", source)
	}
	if strings.Contains(source, "elem = _tags_") {
		t.Fatalf("must not assign full string to each element:\n%s", source)
	}
}

func TestBuildStringToType_SliceOptionalAssignKeepsParseInsideLoop(t *testing.T) {

	st := BuildStringToType(StringToTypeConfig{
		Project: &model.Project{Types: map[string]*model.Type{}},
		From:    Id("headerValue"),
		Arg: &model.Variable{
			Name:    "ids",
			TypeID:  "int64",
			IsSlice: true,
		},
		ID:             Id("ids"),
		OptionalAssign: true,
		AddTo:          &Group{},
		ErrVar:         Err(),
		ValVarName:     "_parseIds_",
		FieldType:      testFieldType,
		AddImport:      func(_ string, _ string) {},
		JSONPkg:        "encoding/json",
	})
	if st == nil {
		t.Fatal("slice with OptionalAssign+AddTo must still return in-loop block, not nil")
	}
	source := renderCode(t, st)
	if !strings.Contains(source, `strconv.ParseInt(elemStr, 10, 64)`) {
		t.Fatalf("expected element parse inside generated block:\n%s", source)
	}
	if strings.Contains(source, `strconv.ParseInt(headerValue`) {
		t.Fatalf("must not parse full header value:\n%s", source)
	}
}

func TestBuildStringToType_SliceSkipsEmptyParts(t *testing.T) {

	st := BuildStringToType(StringToTypeConfig{
		Project: &model.Project{Types: map[string]*model.Type{}},
		From:    Id("raw"),
		Arg: &model.Variable{
			Name:    "ids",
			TypeID:  "int64",
			IsSlice: true,
		},
		ID:        Id("ids"),
		FieldType: testFieldType,
		AddImport: func(_ string, _ string) {},
		JSONPkg:   "encoding/json",
	})
	source := renderCode(t, st)
	if !strings.Contains(source, `elemStr == ""`) {
		t.Fatalf("expected empty part skip:\n%s", source)
	}
	if !strings.Contains(source, "continue") {
		t.Fatalf("expected continue on empty part:\n%s", source)
	}
}

func TestBuildSliceCSVExpr_StringElements(t *testing.T) {

	st := BuildSliceCSVExpr(Id("tags"), true, func(_ string, _ string) {})
	source := renderExpr(t, st)
	if !strings.Contains(source, "strings.Join(tags,") {
		t.Fatalf("unexpected CSV expr:\n%s", source)
	}
}

func TestBuildSliceCSVExpr_NonStringElements(t *testing.T) {

	st := BuildSliceCSVExpr(Id("animalID"), false, func(_ string, _ string) {})
	source := renderExpr(t, st)
	if !strings.Contains(source, `fmt.Sprint(_v)`) {
		t.Fatalf("expected Sprint per element:\n%s", source)
	}
	if !strings.Contains(source, `strings.Join(_parts, ",")`) {
		t.Fatalf("expected Join parts:\n%s", source)
	}
	if strings.Contains(source, "fmt.Sprint(animalID)") {
		t.Fatalf("must not Sprint whole slice:\n%s", source)
	}
}
