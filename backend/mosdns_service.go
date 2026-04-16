package main

import "fmt"

func renderMosdnsConfig(local, remote string, lazy bool) string {
	lazyCache := ""
	lazyExec := ""
	if lazy {
		lazyCache = `  - tag: lazy_cache
    type: cache
    args: { size: 10240, lazy_cache_ttl: 86400 }
`
		lazyExec = `      - exec: $lazy_cache
      - matches: [ has_resp ]
        exec: return
`
	}

	return fmt.Sprintf(`log:
  level: info
  file: %s
plugins:
  - tag: proxy_domain
    type: domain_set
    args:
      files:
        - %s
  - tag: geosite_cn
    type: domain_set
    args:
      files:
        - %s
%s  - tag: forward_local
    type: forward
    args: { upstreams: %s }
  - tag: forward_remote
    type: forward
    args: { upstreams: %s }
  - tag: forward_fakeip
    type: forward
    args: { upstreams: [{ addr: "127.0.0.1:5353" }] }

  - tag: main_sequence
    type: sequence
    args:
%s      - matches: [ qname $proxy_domain ]
        exec: $forward_fakeip
      - matches: [ has_resp ]
        exec: return
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - matches: [ has_resp ]
        exec: return
      - exec: $forward_remote
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, getPath("core", "mosdns", "mosdns.log"), getPath("core", "mosdns", "proxy_domains.txt"), getPath("core", "mosdns", "geosite_cn.txt"), lazyCache, formatUpstreams(local, false), formatUpstreams(remote, true), lazyExec)
}
