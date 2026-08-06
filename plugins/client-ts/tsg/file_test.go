// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tsg_test

import (
	"strings"
	"testing"

	"tgp/internal/generated"
	"tgp/plugins/client-ts/tsg"
)

func TestNewFileHasGeneratedComment(t *testing.T) {

	file := tsg.NewFile()
	if !strings.HasPrefix(file.String(), generated.ByToolGatewayComment) {
		t.Fatalf("NewFile() missing generated comment, got %q", file.String())
	}
}

func TestCommentOverridesGeneratedMarker(t *testing.T) {

	file := tsg.NewFile().Comment("// custom\n")
	if !strings.HasPrefix(file.String(), "// custom\n") {
		t.Fatalf("Comment() must override default marker, got %q", file.String())
	}
}
