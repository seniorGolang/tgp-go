// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestValidateServerMethod_rejectsMalformedHandler(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name: "Overrides",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
	}
	method := &model.Method{
		Name: "CustomPing",
		Annotations: tags.DocTags{
			model.TagHandler: "pkg/path:PingHandler,",
		},
	}

	err := validateServerMethod(project, contract, method)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "handler") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateServerMethod_acceptsValidHandler(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name: "Overrides",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
	}
	method := &model.Method{
		Name: "CustomPing",
		Annotations: tags.DocTags{
			model.TagHandler: "example/internal/fiberhooks:PingHandler",
		},
	}

	if err := validateServerMethod(project, contract, method); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
