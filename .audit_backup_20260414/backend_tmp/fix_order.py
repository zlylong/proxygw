
with open('main.go', 'r') as f:
    s = f.read()

old_cfg = '''%s%s  - tag: forward_local
    type: forward
    args: { upstreams: %s }
  - tag: forward_remote
    type: forward
    args: { upstreams: %s }
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)'''

new_cfg = '''%s  - tag: forward_local
    type: forward
    args: { upstreams: %s }
  - tag: forward_remote
    type: forward
    args: { upstreams: %s }
%s  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, formatUpstreams(local, false), formatUpstreams(remote, true), smartPlugins, seqStr)'''

s = s.replace(old_cfg, new_cfg)
with open('main.go', 'w') as f:
    f.write(s)
