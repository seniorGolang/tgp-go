package tags

import "testing"

func TestExchangeXMLTag(t *testing.T) {

	tests := []struct {
		json string
		want string
	}{
		{json: "payload", want: "payload"},
		{json: "out,omitempty", want: "out,omitempty"},
		{json: "-", want: "-"},
		{json: "", want: ""},
		{json: ",inline", want: ""},
		{json: "body,inline", want: ""},
	}
	for _, tt := range tests {
		if got := ExchangeXMLTag(tt.json); got != tt.want {
			t.Fatalf("ExchangeXMLTag(%q)=%q, want %q", tt.json, got, tt.want)
		}
	}
}
