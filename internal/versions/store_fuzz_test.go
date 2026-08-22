package versions

import "testing"

func FuzzVersionHelpersNeverPanic(f *testing.F) {
	for _, seed := range []string{"go1.2.3", "go1.20", "goabc", "", "../go1.2.3"} {
		f.Add(seed, seed+"-other")
	}
	f.Fuzz(func(t *testing.T, a, b string) {
		_ = validVersionName(a)
		_ = compare(a, b)
		_ = versionParts(a)
	})
}
