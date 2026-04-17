import re

with open('/root/proxygw/backend/main.go', 'r') as f:
    content = f.read()

new_tables = """
\t\t"CREATE TABLE IF NOT EXISTS remote_nodes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, type TEXT, ssh_host TEXT, ssh_port INTEGER, ssh_user TEXT, ssh_auth_type TEXT, ssh_credential TEXT, region TEXT, status TEXT, remark TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);",
\t\t"CREATE TABLE IF NOT EXISTS remote_node_wg (node_id INTEGER PRIMARY KEY, server_priv TEXT, server_pub TEXT, client_priv TEXT, client_pub TEXT, endpoint TEXT, port INTEGER, tunnel_addr TEXT, client_addr TEXT);",
\t\t"CREATE TABLE IF NOT EXISTS remote_node_vless (node_id INTEGER PRIMARY KEY, uuid TEXT, reality_priv TEXT, reality_pub TEXT, short_id TEXT, server_name TEXT, dest TEXT, port INTEGER, share_link TEXT);",
\t\t"CREATE TABLE IF NOT EXISTS remote_node_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, node_id INTEGER, action TEXT, status TEXT, log_text TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);",
\t\t"CREATE TABLE IF NOT EXISTS remote_node_templates (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, type TEXT, default_params TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);",
"""

if 'remote_nodes' not in content:
    content = content.replace('tables := []string{', 'tables := []string{\n' + new_tables)
    
    with open('/root/proxygw/backend/main.go', 'w') as f:
        f.write(content)
    print("Patched main.go with new tables.")
else:
    print("Tables already exist in main.go.")
