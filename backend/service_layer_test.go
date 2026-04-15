package main

import (
	"strings"
	"testing"
)

func TestRenderMosdnsConfigIncludesProxyDomainAndLazyCache(t *testing.T) {
	cfg := renderMosdnsConfig("223.5.5.5", "8.8.8.8", true)
	if !strings.Contains(cfg, "proxy_domains.txt") {
		t.Fatal("expected proxy_domains.txt in config")
	}
	if !strings.Contains(cfg, "tag: lazy_cache") {
		t.Fatal("expected lazy_cache block when lazy=true")
	}
}

func TestRenderMosdnsConfigNoLazyCacheWhenDisabled(t *testing.T) {
	cfg := renderMosdnsConfig("223.5.5.5", "8.8.8.8", false)
	if strings.Contains(cfg, "tag: lazy_cache") {
		t.Fatal("did not expect lazy_cache block when lazy=false")
	}
}

func TestBuildBaseXrayConfigHasRequiredSections(t *testing.T) {
	cfg := buildBaseXrayConfig()
	if _, ok := cfg["inbounds"]; !ok {
		t.Fatal("missing inbounds")
	}
	if _, ok := cfg["outbounds"]; !ok {
		t.Fatal("missing outbounds")
	}
	if _, ok := cfg["routing"]; !ok {
		t.Fatal("missing routing")
	}
}
