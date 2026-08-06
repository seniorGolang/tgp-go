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

func TestRenderExchange_SkipsEmptyStreamResponse(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Live",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerWS:   "",
					model.TagServerSSE:  "",
					model.TagHTTPPrefix: "api/v1",
					model.TagWSPath:     "/ws/live",
				},
				Methods: []*model.Method{
					{
						Name: "Chat",
						Annotations: tags.DocTags{
							model.TagStream: model.StreamModeBidi,
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "room", TypeID: "string"},
							{Name: "in", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
						},
						Results: []*model.Variable{
							{Name: "out", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1},
							{Name: "err", TypeID: "error"},
						},
					},
					{
						Name: "Subscribe",
						Annotations: tags.DocTags{
							model.TagStream:  model.StreamModeServer,
							model.TagSSEPath: "/sse/live/subscribe",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "symbol", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ticks", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1},
							{Name: "count", TypeID: "int"},
							{Name: "err", TypeID: "error"},
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
	content, err := os.ReadFile(filepath.Join(dir, "live-exchange.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "type responseLiveChat ") {
		t.Fatalf("empty stream response must not be generated:\n%s", source)
	}
	if !strings.Contains(source, "type responseLiveSubscribe struct") {
		t.Fatalf("stream response with final fields must be generated:\n%s", source)
	}
	if strings.Contains(source, "func (r requestLiveSubscribe) LogValue(") {
		t.Fatalf("safe exchange must not get LogValue placeholder:\n%s", source)
	}
}

func TestRenderExchange_LogValueOnlyForStreamBody(t *testing.T) {

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
						Name: "Upload",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "file", TypeID: "io:Reader"},
						},
						Results: []*model.Variable{
							{Name: "id", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "POST",
							model.TagHTTPPath:   "/http/upload",
						},
					},
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
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "POST",
							model.TagHTTPPath:   "/http/echo",
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
	content, err := os.ReadFile(filepath.Join(dir, "http-exchange.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "func (r requestHttpUpload) LogValue()") {
		t.Fatalf("io.Reader request must get LogValue placeholder:\n%s", source)
	}
	if strings.Contains(source, "func (r requestHttpEcho) LogValue(") || strings.Contains(source, "func (r responseHttpEcho) LogValue(") {
		t.Fatalf("safe DTO exchange must not get LogValue:\n%s", source)
	}
	if !strings.Contains(source, "type responseHttpEcho struct") {
		t.Fatalf("echo response must still be generated:\n%s", source)
	}
	if strings.Contains(source, `xml:"`) {
		t.Fatalf("JSON method must not emit xml tags:\n%s", source)
	}
}

func TestRenderExchange_XMLTagsOnlyForXMLContentType(t *testing.T) {

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
						Name: "XmlEcho",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "payload", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:          "POST",
							model.TagHTTPPath:            "/xml/echo",
							model.TagRequestContentType:  "application/xml",
							model.TagResponseContentType: "application/xml",
						},
					},
					{
						Name: "JsonEcho",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "POST",
							model.TagHTTPPath:   "/json/echo",
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
	content, err := os.ReadFile(filepath.Join(dir, "http-exchange.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, `json:"payload" xml:"payload"`) {
		t.Fatalf("XML request must mirror json wire name in xml tag:\n%s", source)
	}
	if !strings.Contains(source, `json:"out" xml:"out"`) {
		t.Fatalf("XML response must mirror json wire name in xml tag:\n%s", source)
	}
	if !strings.Contains(source, "type responseHttpJsonEcho struct") {
		t.Fatalf("JSON echo response must be generated:\n%s", source)
	}
	jsonPart := source[strings.Index(source, "type responseHttpJsonEcho struct"):] //nolint:gocritic // Index guarded by Contains above
	if end := strings.Index(jsonPart, "\ntype "); end > 0 {
		jsonPart = jsonPart[:end]
	}
	if strings.Contains(jsonPart, `xml:"`) {
		t.Fatalf("JSON echo must not have xml tags:\n%s", jsonPart)
	}
}

func TestRenderTransportHeader_CookieTypeOnlyForResponseCookies(t *testing.T) {

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
						Name: "CookieIn",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "session", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeID: "bool"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:  "GET",
							model.TagHTTPPath:    "/modes/cookie-in",
							model.TagHTTPCookies: "session|SessionId|explicit",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	tr := NewTransportRenderer(project, dir)
	if err := tr.RenderTransportHeader(); err != nil {
		t.Fatalf("RenderTransportHeader: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "header.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "type cookieType interface") {
		t.Fatalf("request-only http-cookies must not generate cookieType:\n%s", content)
	}
}

func TestRenderREST_DownloadClosesWithBlankAssign(t *testing.T) {

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
						Name: "Download",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "file", TypeID: "io:ReadCloser"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "GET",
							model.TagHTTPPath:   "/http/download",
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
	content, err := os.ReadFile(filepath.Join(dir, "http-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "defer response.File.Close()") {
		t.Fatalf("Close must use blank assign:\n%s", source)
	}
	if !strings.Contains(source, "_ = response.File.Close()") {
		t.Fatalf("expected blank-assigned Close:\n%s", source)
	}
	if !strings.Contains(source, "if response.File != nil") {
		t.Fatalf("ReadCloser close must be nil-safe:\n%s", source)
	}
	callIndex := strings.Index(source, "response, err = http.download")
	if callIndex < 0 {
		t.Fatalf("service call assignment is missing:\n%s", source)
	}
	closeIndex := strings.Index(source[callIndex:], "if response.File != nil")
	errorIndex := strings.Index(source[callIndex:], "if err == nil")
	if closeIndex < 0 || errorIndex < 0 || closeIndex > errorIndex {
		t.Fatalf("ReadCloser ownership must be established before error handling:\n%s", source)
	}
}
