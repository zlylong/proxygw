with open('main.go', 'r') as f:
    code = f.read()

old_str = '''  - tag: forward_remote
    type: forward
    args: { upstreams: %s }
  - tag: udp_server
    type: udp_server
    args:
      entry: main_sequence
      listen: 0.0.0.0:53
, lazyCache, smartPlugins, formatUpstreams(local, false), formatUpstreams(remote, true), seqStr)'''

if old_str in code:
    code = code.replace(old_str, new_str)
    with open('main.go', 'w') as f:
        f.write(code)
    print("Fixed main.go Mosdns config format.")
else:
    print("Could not find the target string in main.go.")
