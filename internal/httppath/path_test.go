// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package httppath

import "testing"

func TestJoin(t *testing.T) {

	tests := []struct {
		name  string
		base  string
		route string
		want  string
	}{
		{name: "empty_base", base: "", route: "/api/v1/x", want: "/api/v1/x"},
		{name: "root_base", base: "/", route: "/api/v1/x", want: "/api/v1/x"},
		{name: "mount_cli", base: "/cli", route: "/api/v1/x", want: "/cli/api/v1/x"},
		{name: "mount_auto", base: "/auto", route: "/api/v1/policies/42/archive", want: "/auto/api/v1/policies/42/archive"},
		{name: "partial_prefix", base: "/api", route: "/api/v1/x", want: "/api/v1/x"},
		{name: "full_prefix", base: "/api/v1", route: "/api/v1/x", want: "/api/v1/x"},
		{name: "mount_and_partial", base: "/cli/api", route: "/api/v1/x", want: "/cli/api/v1/x"},
		{name: "mount_and_prefix", base: "/cli/api/v1", route: "/api/v1/x", want: "/cli/api/v1/x"},
		{name: "non_prefix_mount", base: "/v1", route: "/api/v1/x", want: "/v1/api/v1/x"},
		{name: "full_route_in_base", base: "/api/v1/x", route: "/api/v1/x", want: "/api/v1/x"},
		{name: "base_longer_prefix", base: "/api/v1/policies/42", route: "/api/v1/policies/42/archive", want: "/api/v1/policies/42/archive"},
		{name: "double_slash_base", base: "//cli//", route: "/api/v1/x", want: "/cli/api/v1/x"},
		{name: "route_without_leading_slash", base: "/cli", route: "api/v1/x", want: "/cli/api/v1/x"},
		{name: "empty_route", base: "/cli", route: "", want: "/cli"},
		{name: "both_empty", base: "", route: "", want: "/"},
		{name: "no_false_suffix", base: "/gateway/api", route: "/api/v1/x", want: "/gateway/api/v1/x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Join(tt.base, tt.route); got != tt.want {
				t.Fatalf("Join(%q, %q) = %q, want %q", tt.base, tt.route, got, tt.want)
			}
		})
	}
}
