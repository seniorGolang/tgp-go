// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestRenderLogger_OmitsLogSkipArgument(t *testing.T) {

	project := logSkipTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := NewTransportRenderer(project, dir).RenderLogViews(); err != nil {
		t.Fatalf("RenderLogViews: %v", err)
	}
	if err := renderer.RenderLogger(); err != nil {
		t.Fatalf("RenderLogger: %v", err)
	}
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}

	loggerSrc := readFile(t, filepath.Join(dir, "rpc-logger.go"))
	logViewSrc := readFile(t, filepath.Join(dir, "logview.go"))

	if !strings.Contains(loggerSrc, "requestRpcSecretGreeting{") {
		t.Fatalf("top-level skip must keep exchange struct:\n%s", loggerSrc)
	}
	if strings.Contains(loggerSrc, "Secret:") {
		t.Fatalf("secret must not be assigned into log struct:\n%s", loggerSrc)
	}
	if !strings.Contains(loggerSrc, "Name: name") {
		t.Fatalf("name must remain in log struct:\n%s", loggerSrc)
	}

	if !strings.Contains(loggerSrc, `map[string]any{"Payload": logViewDtoAnnotated(payload)}`) &&
		!strings.Contains(loggerSrc, "logViewDtoAnnotated(payload)") {
		t.Fatalf("nested projection must use logView helper:\n%s", loggerSrc)
	}
	if !strings.Contains(loggerSrc, "map[string]any") {
		t.Fatalf("nested projection must use map payload:\n%s", loggerSrc)
	}
	if !strings.Contains(logViewSrc, "func logViewDtoAnnotated(") {
		t.Fatalf("DTO projection helper must be generated:\n%s", logViewSrc)
	}
	if strings.Contains(logViewSrc, `"Secret"`) {
		t.Fatalf("DTO Secret field must not appear in projection:\n%s", logViewSrc)
	}
	if strings.Contains(logViewSrc, `"Internal"`) {
		t.Fatalf("dumper:hide Internal must be omitted from projection:\n%s", logViewSrc)
	}

	exchangeSrc := readFile(t, filepath.Join(dir, "rpc-exchange.go"))
	if !strings.Contains(exchangeSrc, `dumper:"hide"`) {
		t.Fatalf("exchange must stamp dumper:hide on skipped arg:\n%s", exchangeSrc)
	}
}

func TestRenderLogger_PlainMethodKeepsExchangeStruct(t *testing.T) {

	project := &model.Project{
		ModulePath: "example.com/app",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example.com/app/contracts",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					TagLogger:              "",
				},
				Methods: []*model.Method{
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderLogger(); err != nil {
		t.Fatalf("RenderLogger: %v", err)
	}
	src := readFile(t, filepath.Join(dir, "rpc-logger.go"))
	if strings.Contains(src, "map[string]any") {
		t.Fatalf("plain method must not use map payload:\n%s", src)
	}
	if !strings.Contains(src, "requestRpcEcho{") || !strings.Contains(src, "Msg: msg") {
		t.Fatalf("plain method must use exchange request struct:\n%s", src)
	}
	if !strings.Contains(src, "responseRpcEcho{") || !strings.Contains(src, "Out: out") {
		t.Fatalf("plain method must use exchange response struct:\n%s", src)
	}
}

func TestRenderLogger_SkipRequestResponse(t *testing.T) {

	project := &model.Project{
		ModulePath: "example.com/app",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example.com/app/contracts",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					TagLogger:              "",
				},
				Methods: []*model.Method{
					{
						Name: "Quiet",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{TagLogSkip: "request,response"},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderLogger(); err != nil {
		t.Fatalf("RenderLogger: %v", err)
	}
	src := readFile(t, filepath.Join(dir, "rpc-logger.go"))
	if strings.Contains(src, "viewer.Any") {
		t.Fatalf("request/response viewer attrs must be absent:\n%s", src)
	}
	if !strings.Contains(src, `"took"`) {
		t.Fatalf("took attr must remain:\n%s", src)
	}
}

func logSkipTestProject() (project *model.Project) {

	annotatedID := "example.com/app/contracts/dto:Annotated"
	return &model.Project{
		ModulePath: "example.com/app",
		Types: map[string]*model.Type{
			annotatedID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "Annotated",
				ImportPkgPath: "example.com/app/contracts/dto",
				PkgName:       "dto",
				StructFields: []*model.StructField{
					{Name: "ID", TypeID: "string"},
					{Name: "Name", TypeID: "string"},
					{Name: "Secret", TypeID: "string", Annotations: tags.DocTags{model.TagLogSkip: ""}},
					{Name: "Internal", TypeID: "string", Annotations: tags.DocTags{model.TagParamTags: "dumper:hide"}},
				},
			},
		},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example.com/app/contracts",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					TagLogger:              "",
				},
				Methods: []*model.Method{
					{
						Name: "SecretGreeting",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "secret", TypeID: "string"},
							{Name: "name", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "greeting", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{TagLogSkip: "secret"},
					},
					{
						Name: "EchoAnnotated",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "payload", TypeID: annotatedID},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: annotatedID},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
}

func readFile(t *testing.T, path string) (content string) {

	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}
