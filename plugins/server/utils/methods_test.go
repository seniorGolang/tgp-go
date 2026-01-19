// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package utils

import (
	"testing"

	"tgp/internal/model"
)

func TestIsContextFirst(t *testing.T) {

	tests := []struct {
		name string
		vars []*model.Variable
		want bool
	}{
		{
			name: "empty vars",
			vars: []*model.Variable{},
			want: false,
		},
		{
			name: "context first",
			vars: []*model.Variable{
				{TypeID: "context:Context"},
				{TypeID: "string"},
			},
			want: true,
		},
		{
			name: "context not first",
			vars: []*model.Variable{
				{TypeID: "string"},
				{TypeID: "context:Context"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContextFirst(tt.vars); got != tt.want {
				t.Errorf("IsContextFirst() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsErrorLast(t *testing.T) {

	tests := []struct {
		name string
		vars []*model.Variable
		want bool
	}{
		{
			name: "empty vars",
			vars: []*model.Variable{},
			want: false,
		},
		{
			name: "error last",
			vars: []*model.Variable{
				{TypeID: "string"},
				{TypeID: "error"},
			},
			want: true,
		},
		{
			name: "error not last",
			vars: []*model.Variable{
				{TypeID: "error"},
				{TypeID: "string"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsErrorLast(tt.vars); got != tt.want {
				t.Errorf("IsErrorLast() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArgsWithoutContext(t *testing.T) {

	tests := []struct {
		name   string
		method *model.Method
		want   int
	}{
		{
			name: "with context",
			method: &model.Method{
				Args: []*model.Variable{
					{TypeID: "context:Context"},
					{TypeID: "string"},
				},
			},
			want: 1,
		},
		{
			name: "without context",
			method: &model.Method{
				Args: []*model.Variable{
					{TypeID: "string"},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArgsWithoutContext(tt.method)
			if len(got) != tt.want {
				t.Errorf("ArgsWithoutContext() len = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestResultsWithoutError(t *testing.T) {

	tests := []struct {
		name   string
		method *model.Method
		want   int
	}{
		{
			name: "with error",
			method: &model.Method{
				Results: []*model.Variable{
					{TypeID: "string"},
					{TypeID: "error"},
				},
			},
			want: 1,
		},
		{
			name: "without error",
			method: &model.Method{
				Results: []*model.Variable{
					{TypeID: "string"},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResultsWithoutError(tt.method)
			if len(got) != tt.want {
				t.Errorf("ResultsWithoutError() len = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestRequestStructName(t *testing.T) {

	if got := RequestStructName("Contract", "Method"); got != "requestContractMethod" {
		t.Errorf("RequestStructName() = %v, want requestContractMethod", got)
	}
}

func TestResponseStructName(t *testing.T) {

	if got := ResponseStructName("Contract", "Method"); got != "responseContractMethod" {
		t.Errorf("ResponseStructName() = %v, want responseContractMethod", got)
	}
}
