// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"reflect"
	"testing"

	"tgp/internal/tags"
)

func TestNormalizeHTTPPrefix(t *testing.T) {

	cases := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"api/v1", "/api/v1"},
		{"/api/v1/", "/api/v1"},
		{"  api/v2  ", "/api/v2"},
	}
	for _, tc := range cases {
		if got := NormalizeHTTPPrefix(tc.in); got != tc.want {
			t.Fatalf("NormalizeHTTPPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHTTPPrefixAncestors(t *testing.T) {

	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"/", nil},
		{"api", []string{"/api"}},
		{"api/v1", []string{"/api/v1", "/api"}},
		{"/api/v2/admin/", []string{"/api/v2/admin", "/api/v2", "/api"}},
	}
	for _, tc := range cases {
		if got := HTTPPrefixAncestors(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("HTTPPrefixAncestors(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

func TestContractInJSONRPCBatchScope(t *testing.T) {

	cases := []struct {
		contractPrefix string
		batchPath      string
		want           bool
	}{
		{"api/v1", "/", true},
		{"", "/", true},
		{"", "/api", false},
		{"api/v1", "/api", true},
		{"api/v1", "/api/v1", true},
		{"api/v1", "/api/v2", false},
		{"api/v2/admin", "/api", true},
		{"api/v2/admin", "/api/v2", true},
		{"api/v2/admin", "/api/v2/admin", true},
		{"api/v2", "/api/v2/admin", false},
		{"api", "/api/v1", false},
	}
	for _, tc := range cases {
		if got := ContractInJSONRPCBatchScope(tc.contractPrefix, tc.batchPath); got != tc.want {
			t.Fatalf("ContractInJSONRPCBatchScope(%q, %q) = %v, want %v", tc.contractPrefix, tc.batchPath, got, tc.want)
		}
	}
}

func TestJSONRPCBatchMounts(t *testing.T) {

	project := &Project{
		Contracts: []*Contract{
			{
				Name: "Users",
				Annotations: tags.DocTags{
					TagServerJsonRPC: "",
					TagHttpPrefix:    "api/v1",
				},
			},
			{
				Name: "Orders",
				Annotations: tags.DocTags{
					TagServerJsonRPC: "",
					TagHttpPrefix:    "api/v1",
				},
			},
			{
				Name: "Catalog",
				Annotations: tags.DocTags{
					TagServerJsonRPC: "",
					TagHttpPrefix:    "api/v2",
				},
			},
			{
				Name: "Admin",
				Annotations: tags.DocTags{
					TagServerJsonRPC: "",
					TagHttpPrefix:    "api/v2/admin",
				},
			},
			{
				Name: "HTTPOnly",
				Annotations: tags.DocTags{
					TagServerHTTP: "",
					TagHttpPrefix: "api/v9",
				},
			},
			{
				Name: "RootRPC",
				Annotations: tags.DocTags{
					TagServerJsonRPC: "",
				},
			},
		},
	}

	want := []string{"/", "/api", "/api/v1", "/api/v2", "/api/v2/admin"}
	if got := JSONRPCBatchMounts(project); !reflect.DeepEqual(got, want) {
		t.Fatalf("JSONRPCBatchMounts = %#v, want %#v", got, want)
	}
}

func TestJSONRPCBatchMounts_NilProject(t *testing.T) {

	if got := JSONRPCBatchMounts(nil); !reflect.DeepEqual(got, []string{"/"}) {
		t.Fatalf("JSONRPCBatchMounts(nil) = %#v, want [/]", got)
	}
}

func TestJSONRPCContractPrefix(t *testing.T) {

	contract := &Contract{
		Annotations: tags.DocTags{TagHttpPrefix: "api/v1/"},
	}
	if got := JSONRPCContractPrefix(nil, contract); got != "/api/v1" {
		t.Fatalf("JSONRPCContractPrefix = %q, want /api/v1", got)
	}
	if got := JSONRPCContractPrefix(nil, nil); got != "/" {
		t.Fatalf("JSONRPCContractPrefix(nil) = %q, want /", got)
	}
}
