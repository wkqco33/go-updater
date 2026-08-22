package privatecache

import "testing"

func FuzzParseModuleSpecNeverPanics(f *testing.F) {
	for _, seed := range []string{"github.com/acme/lib", "github.com/acme/lib@v1.2.3", "", "@@"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, spec string) {
		module, version, err := ParseModuleSpec(spec)
		if err == nil && module == "" {
			t.Fatal("successful parse returned an empty module")
		}
		if err == nil && module+version == "" {
			t.Fatal("successful parse returned no data")
		}
	})
}
