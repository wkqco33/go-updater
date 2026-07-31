package privatecache

import "strings"

func BuildEnv(cfg Config, offline bool) map[string]string {
	env := map[string]string{
		"GOMODCACHE": cfg.CacheDir,
	}
	if len(cfg.PrivatePatterns) > 0 {
		env["GOPRIVATE"] = strings.Join(cfg.PrivatePatterns, ",")
	}
	if len(cfg.NoSumDBPatterns) > 0 {
		env["GONOSUMDB"] = strings.Join(cfg.NoSumDBPatterns, ",")
	}
	if len(cfg.NoProxyPatterns) > 0 {
		env["GONOPROXY"] = strings.Join(cfg.NoProxyPatterns, ",")
	}
	if offline {
		env["GOPROXY"] = "off"
	}
	return env
}
