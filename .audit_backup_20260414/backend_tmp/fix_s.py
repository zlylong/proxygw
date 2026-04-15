
with open('main.go', 'r') as f:
    s = f.read()

old_s = '''  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)'''

new_s = '''%s  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
`, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)'''

s = s.replace(old_s, new_s)

with open('main.go', 'w') as f:
    f.write(s)
