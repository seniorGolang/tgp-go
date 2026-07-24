package cdb

import (
	"testing"
)

func TestParseRef(t *testing.T) {

	t.Parallel()

	tests := []struct {
		name      string
		ref       string
		wantKey   string
		wantVer   string
		wantNames []string
		wantErr   bool
	}{
		{
			name:    "empty",
			ref:     "",
			wantErr: true,
		},
		{
			name:    "project and version",
			ref:     "proj@main",
			wantKey: "proj",
			wantVer: "main",
		},
		{
			name:      "contracts in head",
			ref:       "proj:C1,C2@main",
			wantKey:   "proj",
			wantVer:   "main",
			wantNames: []string{"C1", "C2"},
		},
		{
			name:      "contracts in tail",
			ref:       "proj@main:C1,C2",
			wantKey:   "proj",
			wantVer:   "main",
			wantNames: []string{"C1", "C2"},
		},
		{
			name:      "contracts without version",
			ref:       "proj:C1",
			wantKey:   "proj",
			wantNames: []string{"C1"},
		},
		{
			name:    "project only",
			ref:     "proj",
			wantKey: "proj",
		},
		{
			name:      "spaces around contracts",
			ref:       "proj: C1 , C2 @main",
			wantKey:   "proj",
			wantVer:   "main",
			wantNames: []string{"C1", "C2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := ParseRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRef: %v", err)
			}
			if parsed.ProjectKey != tt.wantKey {
				t.Fatalf("ProjectKey: got %q want %q", parsed.ProjectKey, tt.wantKey)
			}
			if parsed.Version != tt.wantVer {
				t.Fatalf("Version: got %q want %q", parsed.Version, tt.wantVer)
			}
			if len(parsed.Contracts) != len(tt.wantNames) {
				t.Fatalf("Contracts: got %v want %v", parsed.Contracts, tt.wantNames)
			}
			for i, name := range tt.wantNames {
				if parsed.Contracts[i] != name {
					t.Fatalf("Contracts[%d]: got %q want %q", i, parsed.Contracts[i], name)
				}
			}
		})
	}
}
