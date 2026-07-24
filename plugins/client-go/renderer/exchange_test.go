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

func TestRenderExchange_SkipsEmptyHTTPResponse(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Http",
				PkgPath: "example/contracts",
				ID:      "Http",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHttpPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Delete",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "DELETE",
							model.TagHttpPath:   "/item/{id}",
						},
					},
					{
						Name: "Get",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "value", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "GET",
							model.TagHttpPath:   "/item/{id}/value",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderExchange(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "http-exchange.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "type responseHttpDelete ") || strings.Contains(source, "Formal exchange type") {
		t.Fatalf("empty HTTP response must not be generated:\n%s", source)
	}
	if !strings.Contains(source, "type responseHttpGet struct") {
		t.Fatalf("non-empty HTTP response must be generated:\n%s", source)
	}
	if strings.Contains(source, `xml:"`) {
		t.Fatalf("JSON HTTP exchange must not emit xml tags:\n%s", source)
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
				ID:      "Http",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHttpPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "XmlEcho",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "payload", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:          "POST",
							model.TagHttpPath:            "/xml/echo",
							model.TagRequestContentType:  "application/xml",
							model.TagResponseContentType: "application/xml",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderExchange(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "http-exchange.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, `xml:"payload"`) || !strings.Contains(source, `xml:"out"`) {
		t.Fatalf("XML method must emit xml tags:\n%s", source)
	}
}

func TestRenderExchange_KeepsEmptyJSONRPCResponse(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
				},
				Methods: []*model.Method{
					{
						Name: "Do",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "msg", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderExchange(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "rpc-exchange.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "type responseRpcDo struct{}") {
		t.Fatalf("JSON-RPC empty response must be generated for Unmarshal:\n%s", source)
	}
	if !strings.Contains(source, "type responseRpcEcho struct") {
		t.Fatalf("JSON-RPC non-empty response must be generated:\n%s", source)
	}
}

func TestRenderExchange_StreamResponseOnlyForClientMode(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Live",
				PkgPath: "example/live",
				ID:      "Live",
				Annotations: tags.DocTags{
					model.TagServerWS: "",
					model.TagWSPath:   "/ws/live",
				},
				Methods: []*model.Method{
					{
						Name: "Subscribe",
						Annotations: tags.DocTags{
							model.TagStream: model.StreamModeServer,
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "symbol", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
							{Name: "count", TypeRef: model.TypeRef{TypeID: "int"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
					{
						Name: "Ingest",
						Annotations: tags.DocTags{
							model.TagStream: model.StreamModeClient,
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "batchID", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "samples", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
						},
						Results: []*model.Variable{
							{Name: "accepted", TypeRef: model.TypeRef{TypeID: "int"}},
							{Name: "rejected", TypeRef: model.TypeRef{TypeID: "int"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
					{
						Name: "Chat",
						Annotations: tags.DocTags{
							model.TagStream: model.StreamModeBidi,
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "room", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "in", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
						},
						Results: []*model.Variable{
							{Name: "out", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderExchange(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-exchange.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "type responseLiveSubscribe ") {
		t.Fatalf("server-stream response must not be generated:\n%s", source)
	}
	if strings.Contains(source, "type responseLiveChat ") {
		t.Fatalf("bidi-stream response must not be generated:\n%s", source)
	}
	if !strings.Contains(source, "type responseLiveIngest struct") {
		t.Fatalf("client-stream response must be generated for final Unmarshal:\n%s", source)
	}
	if !strings.Contains(source, "type requestLiveSubscribe struct") {
		t.Fatalf("stream request types must still be generated:\n%s", source)
	}
}

func TestRenderHTTP_ErrorPathClosesBodyWithBlankAssign(t *testing.T) {

	project := httpClientTestProject()
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "http.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "\thttpResp.Body.Close()\n") {
		t.Fatalf("Body.Close must use blank assign:\n%s", source)
	}
	if !strings.Contains(source, "_ = httpResp.Body.Close()") {
		t.Fatalf("expected blank-assigned Body.Close:\n%s", source)
	}
}
