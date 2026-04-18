import re
import os

# --- Patch nodes_routes.go ---
with open('backend/nodes_routes.go', 'r') as f:
    routes_content = f.read()

old_get_nodes = '''	api.GET("/nodes", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, COALESCE(grp, ''), type, address, port, uuid, active, ping, COALESCE(params, '{}') FROM nodes")'''

new_get_nodes = '''	api.GET("/nodes", func(c *gin.Context) {
		var defNodeStr string
		db.QueryRow("SELECT value FROM settings WHERE key='default_node_id'").Scan(&defNodeStr)
		defNodeId, _ := strconv.Atoi(defNodeStr)

		rows, err := db.Query("SELECT id, name, COALESCE(grp, ''), type, address, port, uuid, active, ping, COALESCE(params, '{}') FROM nodes")'''

routes_content = routes_content.replace(old_get_nodes, new_get_nodes)

old_nodes_response = '''		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db rows error"})
			return
		}
		c.JSON(http.StatusOK, nodes)
	})'''

new_nodes_response = '''		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db rows error"})
			return
		}
		
		var activeCount int
		for _, n := range nodes {
			if n["active"] == true || n["active"] == int64(1) || n["active"] == 1 {
				activeCount++
			}
		}
		
		if activeCount == 1 {
			for _, n := range nodes {
				if n["active"] == true || n["active"] == int64(1) || n["active"] == 1 {
					n["is_default"] = true
				} else {
					n["is_default"] = false
				}
			}
		} else {
			for _, n := range nodes {
				if idVal, ok := n["id"].(int); ok && idVal == defNodeId {
					n["is_default"] = true
				} else {
					n["is_default"] = false
				}
			}
		}

		c.JSON(http.StatusOK, nodes)
	})

	api.PUT("/nodes/:id/default", func(c *gin.Context) {
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('default_node_id', ?)", c.Param("id"))
		scheduleApply()
		c.JSON(http.StatusOK, gin.H{"success": true})
	})'''

routes_content = routes_content.replace(old_nodes_response, new_nodes_response)

with open('backend/nodes_routes.go', 'w') as f:
    f.write(routes_content)


# --- Patch main.go ---
with open('backend/main.go', 'r') as f:
    main_content = f.read()

old_proxy_tags = '''	var proxyTags []string
	for rows.Next() {
		var name, ntype, address, uuid, paramsStr string
		var port, id int'''

new_proxy_tags = '''	var defNodeStr string
	db.QueryRow("SELECT value FROM settings WHERE key='default_node_id'").Scan(&defNodeStr)
	defaultNodeId, _ := strconv.Atoi(defNodeStr)
	
	var activeIds []int
	var proxyTags []string
	for rows.Next() {
		var name, ntype, address, uuid, paramsStr string
		var port, id int'''

main_content = main_content.replace(old_proxy_tags, new_proxy_tags)

main_content = main_content.replace('ntypeLow := strings.ToLower(ntype)', 'activeIds = append(activeIds, id)\n\n\t\tntypeLow := strings.ToLower(ntype)')

old_rule_rewrite = '''		for _, r := range rules {
			if r["outboundTag"] == "proxy" {
				delete(r, "outboundTag")
				r["balancerTag"] = "proxy-balancer"
			}
			if bTag, ok := r["balancerTag"].(string); ok && strings.HasPrefix(bTag, "bal-ha-") {'''

new_rule_rewrite = '''		actualDefault := 0
		if len(activeIds) == 1 {
			actualDefault = activeIds[0]
		} else {
			for _, aid := range activeIds {
				if aid == defaultNodeId {
					actualDefault = aid
					break
				}
			}
		}

		for _, r := range rules {
			if r["outboundTag"] == "proxy" {
				if actualDefault > 0 {
					r["outboundTag"] = fmt.Sprintf("proxy-%d-out", actualDefault)
				} else {
					delete(r, "outboundTag")
					r["balancerTag"] = "proxy-balancer"
				}
			}
			if bTag, ok := r["balancerTag"].(string); ok && strings.HasPrefix(bTag, "bal-ha-") {'''

main_content = main_content.replace(old_rule_rewrite, new_rule_rewrite)

with open('backend/main.go', 'w') as f:
    f.write(main_content)


# --- Patch index.html ---
with open('frontend/dist/index.html', 'r') as f:
    html = f.read()

# Add set as default button logic
html = html.replace('const loadNodeHistory', 'const setDefaultNode = async (id) => { await apiFetch(`/api/nodes/${id}/default`, {method: "PUT"}); showToast("已设为默认节点"); loadData(); };\n                const loadNodeHistory')

html = html.replace('regenerateNode, loadNodeHistory', 'setDefaultNode, regenerateNode, loadNodeHistory')

old_table_td = '''                                        <td class="px-6 py-4 whitespace-nowrap">
                                            <div class="flex items-center">
                                                <span class="font-medium text-gray-900">{{ node.name }}</span>
                                            </div>
                                        </td>'''

new_table_td = '''                                        <td class="px-6 py-4 whitespace-nowrap">
                                            <div class="flex items-center">
                                                <span class="font-medium text-gray-900">{{ node.name }}</span>
                                                <span v-if="node.is_default" class="ml-2 px-2 py-0.5 bg-green-100 text-green-800 text-xs rounded-full border border-green-200">默认</span>
                                            </div>
                                        </td>'''
html = html.replace(old_table_td, new_table_td)

old_table_actions = '''                                        <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <button @click="editNode(node)" class="text-indigo-600 hover:text-indigo-900 mr-3">编辑</button>
                                            <button @click="deleteNode(node.id)" class="text-red-600 hover:text-red-900">删除</button>
                                        </td>'''

new_table_actions = '''                                        <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <button v-if="!node.is_default" @click="setDefaultNode(node.id)" class="text-gray-500 hover:text-blue-600 mr-3">设为默认</button>
                                            <button @click="editNode(node)" class="text-indigo-600 hover:text-indigo-900 mr-3">编辑</button>
                                            <button @click="deleteNode(node.id)" class="text-red-600 hover:text-red-900">删除</button>
                                        </td>'''
html = html.replace(old_table_actions, new_table_actions)

with open('frontend/dist/index.html', 'w') as f:
    f.write(html)
