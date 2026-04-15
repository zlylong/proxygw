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
		lazyExec = "      - exec: $lazy_cache\n"
	}

	return fmt.Sprintf(`log:
  level: info
  file: "/root/proxygw/core/mosdns/mosdns.log"
plugins:
  - tag: proxy_domain
    type: domain_set
    args:
      files:
        - "/root/proxygw/core/mosdns/proxy_domains.txt"
  - tag: geosite_cn
    type: domain_set
    args:
      files:
        - "/root/proxygw/core/mosdns/geosite_cn.txt"
%s  - tag: forward_local
    type: forward
    args: { upstreams: %s }
  - tag: forward_remote
    type: forward
    args: { upstreams: %s }
  - tag: main_sequence
    type: sequence
    args:
%s      - matches: [ qname $proxy_domain ]
        exec: $forward_remote
      - matches: [ qname $geosite_cn ]
        exec: $forward_local
      - exec: $forward_remote
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, formatUpstreams(local, false), formatUpstreams(remote, true), lazyExec)
}
