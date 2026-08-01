package privatecache

import "testing"

func TestParseModuleSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		wantModule  string
		wantVersion string
		wantErr     bool
	}{
		{"with version", "github.com/acme/lib@v1.0.0", "github.com/acme/lib", "v1.0.0", false},
		{"without version", "github.com/acme/lib", "github.com/acme/lib", "", false},
		{"invalid", "@v1.0.0", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, version, err := ParseModuleSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseModuleSpec() err=%v wantErr=%v", err, tt.wantErr)
			}
			if module != tt.wantModule || version != tt.wantVersion {
				t.Fatalf("ParseModuleSpec() module=%q version=%q want module=%q version=%q", module, version, tt.wantModule, tt.wantVersion)
			}
		})
	}
}
