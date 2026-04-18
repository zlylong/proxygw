import re

# 1. Update main.go
with open('backend/main.go', 'r') as f:
    main_content = f.read()

# Update tag prefix to prevent collision in balancers (proxy-1 matches proxy-10)
main_content = main_content.replace('outbound["tag"] = fmt.Sprintf("proxy-%d", id)', 'outbound["tag"] = fmt.Sprintf("proxy-%d-out", id)')
main_content = main_content.replace('proxyTags = append(proxyTags, fmt.Sprintf("proxy-%d", id))', 'proxyTags = append(proxyTags, fmt.Sprintf("proxy-%d-out", id))')

# Update rule creation logic
old_rule_logic = '''		rule := map[string]interface{}{"type": "field", "outboundTag": policy}

		if rtype == "geosite" || rtype == "domain" {'''

new_rule_logic = '''		rule := map[string]interface{}{"type": "field"}

		if policy == "direct" || policy == "block" {
			rule["outboundTag"] = policy
		} else if policy == "proxy" {
			rule["balancerTag"] = "proxy-balancer"
		} else if strings.HasPrefix(policy, "proxy-") {
			// Single node binding (e.g. proxy-1)
			rule["outboundTag"] = policy + "-out"
		} else if strings.HasPrefix(policy, "ha-") {
			// HA Mode (e.g. ha-1-2)
			parts := strings.Split(strings.TrimPrefix(policy, "ha-"), "-")
			if len(parts) == 2 {
				rule["balancerTag"] = "bal-" + policy
			} else {
				rule["outboundTag"] = "proxy"
			}
		} else {
			rule["outboundTag"] = policy
		}

		if rtype == "geosite" || rtype == "domain" {'''

main_content = main_content.replace(old_rule_logic, new_rule_logic)

# Update balancer creation
old_balancer_logic = '''		for _, r := range rules {
			if r["outboundTag"] == "proxy" {
				delete(r, "outboundTag")
				r["balancerTag"] = "proxy-balancer"
			}
		}
	}
	config["routing"].(map[string]interface{})["rules"] = rules'''

new_balancer_logic = '''		customBalancers := make(map[string]map[string]interface{})
		for _, r := range rules {
			if r["outboundTag"] == "proxy" {
				delete(r, "outboundTag")
				r["balancerTag"] = "proxy-balancer"
			}
			if bTag, ok := r["balancerTag"].(string); ok && strings.HasPrefix(bTag, "bal-ha-") {
				parts := strings.Split(strings.TrimPrefix(bTag, "bal-ha-"), "-")
				if len(parts) == 2 {
					customBalancers[bTag] = map[string]interface{}{
						"tag": bTag,
						"selector": []string{"proxy-" + parts[0] + "-out"},
						"fallbackTag": "proxy-" + parts[1] + "-out",
					}
				}
			}
		}
		
		balancers := routing["balancers"].([]map[string]interface{})
		for _, cb := range customBalancers {
			balancers = append(balancers, cb)
		}
		routing["balancers"] = balancers
	}
	config["routing"].(map[string]interface{})["rules"] = rules'''

main_content = main_content.replace(old_balancer_logic, new_balancer_logic)

with open('backend/main.go', 'w') as f:
    f.write(main_content)
print('main.go patched')

# 2. Update index.html
with open('frontend/dist/index.html', 'r') as f:
    html = f.read()

# Replace the select inputs for rules
old_select = '''                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">策略绑定 (对应的节点/出口)</label>
                                <select v-model="newRule.policy" class="border border-gray-300 rounded px-3 py-2 w-48">
                                    <option value="proxy">默认代理 (Proxy)</option>
                                    <option value="direct">直连 (Direct)</option>
                                    <option value="block">拦截 (Block)</option>
                                    <optgroup label="指定节点">
                                        <option v-for="node in nodes" :key="node.id" :value="'proxy-' + node.id">
                                            {{ node.name }}
                                        </option>
                                    </optgroup>
                                </select>
                            </div>'''

new_select = '''                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">路由策略</label>
                                <select v-model="newRule.action" class="border border-gray-300 rounded px-3 py-2 w-40">
                                    <option value="proxy">全局代理 (Proxy)</option>
                                    <option value="direct">直连 (Direct)</option>
                                    <option value="block">拦截 (Block)</option>
                                    <option value="node">指定单节点</option>
                                    <option value="ha">主备容灾 (HA)</option>
                                </select>
                            </div>
                            <div v-if="newRule.action === 'node' || newRule.action === 'ha'">
                                <label class="block text-sm font-medium text-gray-700 mb-1">主用节点</label>
                                <select v-model="newRule.primary" class="border border-gray-300 rounded px-3 py-2 w-40">
                                    <option v-for="node in nodes" :key="node.id" :value="node.id">{{ node.name }}</option>
                                </select>
                            </div>
                            <div v-if="newRule.action === 'ha'">
                                <label class="block text-sm font-medium text-gray-700 mb-1">备用节点 (Fallback)</label>
                                <select v-model="newRule.standby" class="border border-gray-300 rounded px-3 py-2 w-40">
                                    <option v-for="node in nodes" :key="node.id" :value="node.id">{{ node.name }}</option>
                                </select>
                            </div>'''

html = html.replace(old_select, new_select)

# Replace table display
old_table_span = '''                                            <span class="px-2 py-1 rounded text-xs font-bold"
                                                  :class="{'bg-green-100 text-green-800': rule.policy==='direct', 'bg-blue-100 text-blue-800': rule.policy.startsWith('proxy'), 'bg-red-100 text-red-800': rule.policy==='block'}">
                                                {{ rule.policy.startsWith('proxy-') ? (nodes.find(n => 'proxy-'+n.id === rule.policy)?.name || rule.policy) : rule.policy }}
                                            </span>'''

new_table_span = '''                                            <span class="px-2 py-1 rounded text-xs font-bold"
                                                  :class="{'bg-green-100 text-green-800': rule.policy==='direct', 'bg-blue-100 text-blue-800': rule.policy==='proxy' || rule.policy.startsWith('proxy-'), 'bg-red-100 text-red-800': rule.policy==='block', 'bg-purple-100 text-purple-800': rule.policy.startsWith('ha-')}">
                                                {{ rule.policy === 'proxy' ? 'Proxy (全局)' : 
                                                   rule.policy === 'direct' ? 'Direct (直连)' : 
                                                   rule.policy === 'block' ? 'Block (拦截)' : 
                                                   rule.policy.startsWith('proxy-') ? '节点: ' + (nodes.find(n => n.id == rule.policy.split('-')[1])?.name || rule.policy) : 
                                                   rule.policy.startsWith('ha-') ? '主备: ' + (nodes.find(n => n.id == rule.policy.split('-')[1])?.name || '?') + ' -> ' + (nodes.find(n => n.id == rule.policy.split('-')[2])?.name || '?') : rule.policy }}
                                            </span>'''

html = html.replace(old_table_span, new_table_span)

# Add properties to newRule and handle addRule logic
old_newRule = "const newRule = ref({ type: 'domain', value: '', policy: 'proxy' });"
new_newRule = "const newRule = ref({ type: 'domain', value: '', action: 'proxy', primary: '', standby: '', policy: 'proxy' });"
html = html.replace(old_newRule, new_newRule)

old_addRule = '''                const addRule = async () => {
                    if(!newRule.value.value) return showToast("匹配值不能为空");
                    await apiFetch('/api/rules', { method: 'POST', body: JSON.stringify({ Type: newRule.value.type, Value: newRule.value.value, Policy: newRule.value.policy }) });
                    newRule.value.value = '';
                    loadData();
                };'''

new_addRule = '''                const addRule = async () => {
                    if(!newRule.value.value) return showToast("匹配值不能为空");
                    let finalPolicy = newRule.value.action;
                    if(finalPolicy === 'node') {
                        if(!newRule.value.primary) return showToast("请选择指定节点");
                        finalPolicy = 'proxy-' + newRule.value.primary;
                    } else if(finalPolicy === 'ha') {
                        if(!newRule.value.primary || !newRule.value.standby) return showToast("请选择主备节点");
                        if(newRule.value.primary === newRule.value.standby) return showToast("主备节点不能相同");
                        finalPolicy = 'ha-' + newRule.value.primary + '-' + newRule.value.standby;
                    }
                    await apiFetch('/api/rules', { method: 'POST', body: JSON.stringify({ Type: newRule.value.type, Value: newRule.value.value, Policy: finalPolicy }) });
                    newRule.value.value = '';
                    loadData();
                };'''

html = html.replace(old_addRule, new_addRule)

with open('frontend/dist/index.html', 'w') as f:
    f.write(html)
print('index.html patched')
