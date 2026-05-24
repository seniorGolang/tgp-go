// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import "testing"

func TestJsonRPCWireMethod(t *testing.T) {

	tests := []struct {
		contract string
		method   string
		want     string
	}{
		{contract: "Rpc", method: "InlineSingle", want: "rpc.inlineSingle"},
		{contract: "Rpc", method: "ReturnsError", want: "rpc.returnsError"},
		{contract: "Rpc", method: "Do", want: "rpc.do"},
		{contract: "RPC", method: "Echo", want: "RPC.echo"},
	}

	for _, tt := range tests {
		t.Run(tt.contract+"."+tt.method, func(t *testing.T) {
			got := JsonRPCWireMethod(tt.contract, tt.method)
			if got != tt.want {
				t.Fatalf("JsonRPCWireMethod(%q, %q) = %q, want %q", tt.contract, tt.method, got, tt.want)
			}
		})
	}
}
