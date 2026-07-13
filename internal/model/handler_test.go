// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"
	"testing"
)

func TestParseHandlerRef_valid(t *testing.T) {

	pkgPath, funcName, err := ParseHandlerRef("example/internal/fiberhooks:PingHandler")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkgPath != "example/internal/fiberhooks" || funcName != "PingHandler" {
		t.Fatalf("got %q:%q", pkgPath, funcName)
	}
}

func TestParseHandlerRef_rejectsInvalid(t *testing.T) {

	cases := []string{"invalid", "pkg/path:PingHandler,", "pkg/path:", ":Func"}
	for _, value := range cases {
		if _, _, err := ParseHandlerRef(value); err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}

func TestHandlerInfoFromAnnotations_bestEffort(t *testing.T) {

	info := HandlerInfoFromAnnotations(map[string]string{
		TagHandler: "example/pkg:Handler",
	})
	if info == nil || info.Name != "Handler" {
		t.Fatalf("unexpected info: %#v", info)
	}

	if HandlerInfoFromAnnotations(map[string]string{TagHandler: "broken"}) != nil {
		t.Fatal("expected nil for invalid handler")
	}
}

func TestParseArgMapEntriesStrict_valid(t *testing.T) {

	items, err := ParseArgMapEntriesStrict("token|X-Token|explicit,ref|SessionRef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].Mode != ArgModeExplicit {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestParseArgMapEntriesStrict_rejectsInvalid(t *testing.T) {

	_, err := ParseArgMapEntriesStrict("broken")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "arg|key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
