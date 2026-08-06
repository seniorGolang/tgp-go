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

func TestRenderREST_FormUsesBodyParserOnly(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Annotations",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "FormBody",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "name", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "displayName", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:         "POST",
							model.TagHTTPPath:           "/annotations/form",
							model.TagRequestContentType: "application/x-www-form-urlencoded",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "annotations-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "BodyParser(&request)") {
		t.Fatalf("form must use BodyParser:\n%s", source)
	}
	if strings.Contains(source, "SetBodyRaw") || strings.Contains(source, "ReadAll(bodyStream)") {
		t.Fatalf("form must not ReadAll+SetBodyRaw:\n%s", source)
	}
}

func TestRenderREST_FormResponseUsesMakeURLValues(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Annotations",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "FormRoundTrip",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "name", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "displayName", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:          "POST",
							model.TagHTTPPath:            "/annotations/form-roundtrip",
							model.TagRequestContentType:  "application/x-www-form-urlencoded",
							model.TagResponseContentType: "application/x-www-form-urlencoded",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "annotations-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "make(url.Values)") {
		t.Fatalf("form response must use make(url.Values), got:\n%s", source)
	}
	if strings.Contains(source, "url.Values()") {
		t.Fatalf("form response must not call url.Values() as function:\n%s", source)
	}
	if !strings.Contains(source, `SetContentType("application/x-www-form-urlencoded")`) {
		t.Fatalf("form response must set form content-type:\n%s", source)
	}
}

func TestRenderREST_HeaderBindingUsesPeek(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Modes",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "BodyHeader",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "token", TypeID: "string"},
							{Name: "body", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeID: "bool"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "POST",
							model.TagHTTPPath:   "/modes/body-header",
							model.TagHTTPHeader: "token|X-Token",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "modes-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, `Header.Peek("X-Token")`) {
		t.Fatalf("expected Peek for header presence:\n%s", source)
	}
	if strings.Contains(source, `ftx.Get("X-Token")`) {
		t.Fatalf("must not use Get+empty check for header overlay:\n%s", source)
	}
}

func TestRenderREST_MultiMultipartUsesOrderedStream(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Http",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "MultipartUploadMulti",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "partA", TypeID: "io:Reader"},
							{Name: "partB", TypeID: "io:Reader"},
						},
						Results: []*model.Variable{
							{Name: "id", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:      "POST",
							model.TagHTTPPath:        "/upload-multi",
							model.TagHTTPPartName:    "partA|partA,partB|partB",
							model.TagHTTPPartContent: "partA|text/plain,partB|application/octet-stream",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	tr := NewTransportRenderer(project, dir)
	if err := tr.RenderTransportContext(); err != nil {
		t.Fatalf("RenderTransportContext: %v", err)
	}
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "http-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "stream.NewParts(") {
		t.Fatalf("multi multipart must use stream.NewParts:\n%s", source)
	}
	if !strings.Contains(source, "parts.Close()") {
		t.Fatalf("multi multipart must defer parts.Close:\n%s", source)
	}
	if !strings.Contains(source, "parts.Reader(") {
		t.Fatalf("multi multipart must assign via parts.Reader:\n%s", source)
	}
	if strings.Contains(source, "partBodies") || strings.Contains(source, "ReadAll(p)") {
		t.Fatalf("multi must not buffer part bodies:\n%s", source)
	}
	if _, err = os.Stat(filepath.Join(dir, "stream", "parts.go")); err != nil {
		t.Fatalf("stream package must be rendered: %v", err)
	}
}

func TestRenderREST_SingleMultipartClosesPart(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Http",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "MultipartUpload",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "file", TypeID: "io:Reader"},
						},
						Results: []*model.Variable{
							{Name: "id", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:      "POST",
							model.TagHTTPPath:        "/upload",
							model.TagHTTPMultipart:   "",
							model.TagHTTPPartName:    "file|file",
							model.TagHTTPPartContent: "file|application/octet-stream",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	tr := NewTransportRenderer(project, dir)
	if err := tr.RenderTransportContext(); err != nil {
		t.Fatalf("RenderTransportContext: %v", err)
	}
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "http-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "stream.NewParts(") {
		t.Fatalf("single multipart must keep direct Part path:\n%s", source)
	}
	if strings.Contains(source, "io.Copy(io.Discard, p)") {
		t.Fatalf("single multipart skip must use p.Close, not Copy:\n%s", source)
	}
	if !strings.Contains(source, "_ = p.Close()") {
		t.Fatalf("single multipart must Close part:\n%s", source)
	}
	if !strings.Contains(source, "if found") {
		t.Fatalf("single multipart must defer Close when found:\n%s", source)
	}
}

func TestRenderREST_HandlerSkipsServeMethod(t *testing.T) {

	project, contract := overridesTestContract()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, contract, dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "overrides-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "func (http *httpOverrides) serveCustomPing(") {
		t.Fatalf("handler= method must not generate serve*:\n%s", source)
	}
}

func TestRenderREST_QuerySliceParsesElemStr(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Animals",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "All",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "animalID", TypeID: "int64", IsSlice: true},
						},
						Results: []*model.Variable{
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "GET",
							model.TagHTTPPath:   "/animals",
							model.TagHTTPArg:    "animalID|animalId",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "animals-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, `strconv.ParseInt(elemStr, 10, 64)`) {
		t.Fatalf("query slice must ParseInt(elemStr):\n%s", source)
	}
	if strings.Contains(source, "ParseInt(_animalID_") {
		t.Fatalf("query slice must not ParseInt full raw string:\n%s", source)
	}
}

func TestRenderREST_ResultHeaderSliceUsesCSVJoin(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Animals",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Codes",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "codes", TypeID: "int64", IsSlice: true},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "GET",
							model.TagHTTPPath:   "/codes",
							model.TagHTTPHeader: "codes|X-Codes",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "animals-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, `strings.Join(_parts, ",")`) {
		t.Fatalf("result header slice must CSV-join elements:\n%s", source)
	}
	if strings.Contains(source, `fmt.Sprint(response.Codes)`) || strings.Contains(source, "fmt.Sprint(response.codes)") {
		t.Fatalf("must not Sprint whole slice for header:\n%s", source)
	}
}
