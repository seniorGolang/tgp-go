// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tags

import "testing"

func TestParseTags_doesNotAppendCommaForEmptyDuplicate(t *testing.T) {

	got := ParseTags([]string{"// @tg handler=pkg/path:PingHandler handler"})

	if got["handler"] != "pkg/path:PingHandler" {
		t.Fatalf("handler = %q, want pkg/path:PingHandler", got["handler"])
	}
}

func TestParseTags_preservesHandlerWhenSummaryOnNextLine(t *testing.T) {

	got := ParseTags([]string{
		"// @tg handler=pkg/path:PingHandler",
		"// @tg summary=Custom fiber handler",
	})

	if got["handler"] != "pkg/path:PingHandler" {
		t.Fatalf("handler = %q", got["handler"])
	}
	if got["summary"] != "Custom" {
		t.Fatalf("summary = %q, want Custom (first word without backticks)", got["summary"])
	}
}
