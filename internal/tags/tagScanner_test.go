// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tags

import "testing"

func TestTagScanner_splitsUnquotedMultiWordSummary(t *testing.T) {

	got, err := TagScanner("handler=pkg/path:PingHandler summary=Custom fiber handler")
	if err != nil {
		t.Fatalf("TagScanner: %v", err)
	}

	if got["handler"] != "pkg/path:PingHandler" {
		t.Fatalf("handler = %q", got["handler"])
	}
	if got["summary"] != "Custom" {
		t.Fatalf("summary = %q, want Custom (unquoted multi-word is truncated)", got["summary"])
	}
}

func TestTagScanner_keepsEmptyTagWithoutTrailingComma(t *testing.T) {

	got, err := TagScanner("handler=pkg/path:PingHandler handler")
	if err != nil {
		t.Fatalf("TagScanner: %v", err)
	}

	want := "pkg/path:PingHandler"
	if got["handler"] != want {
		t.Fatalf("handler = %q, want %q", got["handler"], want)
	}
}
