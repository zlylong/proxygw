# ProxyGW 深度代码审计（前后端功能/接口 & 可清理项）

审计日期：2026-04-17  
审计范围：
- 前端：`frontend/dist/index.html`（当前唯一可见 UI 入口）
- 后端：`backend/*routes.go`, `backend/main.go`
- 文档：`docs/API.md`
- 仓库卫生：`backend/`, `frontend/dist/`

---

## 一、总体结论

本版本可以运行核心链路（登录、节点管理、规则、DNS、系统更新、远程部署），但存在以下结构性问题：

1. **前后端接口契约存在缺口**：前端保留了 DNS 日志 WebSocket 连接逻辑，但后端无对应路由实现。  
2. **文档与实现不一致**：`/mode` 行为、DNS 日志接口说明与后端现状不一致，容易导致运维/二次开发误判。  
3. **存在可明确清理的遗留代码与工件**：调试函数、补丁脚本、备份文件、未被业务使用的数据表字段/设计。  
4. **部分实现“可用但无效/低价值”**：例如 WireGuard ping 分支中的空正则，导致延迟测量无实际意义。

---

## 二、前后端接口核对（重点）

## 2.1 前端调用到后端可达的主接口（当前有效）

以下主路径在前端被调用，后端也有路由承接（或由参数路由承接）：

- 认证：`/api/login`, `/api/logout`, `/api/password`
- 系统：`/api/status`, `/api/mode`, `/api/cron`
- 节点：`/api/nodes`, `/api/nodes/import`, `/api/nodes/ping`, `/api/nodes/:id`, `/api/nodes/:id/toggle`
- 规则：`/api/rules`, `/api/rules/categories`, `/api/rules/:id`
- DNS：`/api/dns`
- 更新：`/api/xray/versions`, `/api/update/:component`（前端实际调用 `/api/update/xray|geodata|rollback_xray`）
- 远程节点：`/api/remote_nodes` 与 `/:id/check|history|regenerate|rollback|DELETE`
- 配置查看与应用：`/api/config/xray`, `/api/config/mosdns`, `/api/apply`

结论：**核心控制面接口基本连通**。

## 2.2 明确存在的接口缺口 / 冗余

### A. 前端在用、后端未实现：DNS 日志 WS

- 前端在 DNS 页签切换时会主动连接：`ws://<host>/api/dns/logs/ws`。  
- `docs/API.md` 也宣称存在：`GET /dns/logs`、`GET /dns/logs/ws`。  
- 但后端注册路由中无 `/dns/logs` 与 `/dns/logs/ws`。

影响：
- 前端会持续重连一个不存在的 WS 端点，造成无效网络请求与误导性“日志空白”。
- 文档对外承诺了不存在能力。

建议（2 选 1）：
1. **补实现**（推荐）：后端增加日志 ring buffer + WS 推送路由；
2. **删前端/删文档**：彻底下线 DNS 实时日志入口与说明，避免“假功能”。

### B. 文档与实现冲突：`/mode` 的 nftables 行为

- 文档描述：Mode B 应“stop nftables”。
- 实现实际：Mode A / Mode B 都执行 `systemctl start nftables`；Mode B 额外 `start frr`。

影响：
- 运行手册与真实行为不一致，可能导致生产排障时误操作。

建议：
- 若当前实现是正确设计：更新文档；
- 若文档是目标设计：修正 `POST /mode` 实现。

---

## 三、可清理代码与接口（按优先级）

## 3.1 高优先级（建议本周处理）

1. **无效 DNS 日志链路（接口级）**  
   - 前端 WS 连接逻辑存在但无后端支撑。
2. **`/mode` 文档偏差（契约级）**  
   - API 文档与运行逻辑冲突。
3. **WireGuard ping 的空正则（代码级）**  
   - `regexp.MustCompile("")` 无法提取 RTT；当前分支通常返回 1ms 或 -1，测量价值低。  
   - 建议改成明确 regex（如 `time=([0-9.]+)`）或改为统一 TCP/ICMP 测试策略。

## 3.2 中优先级（建议 1~2 个迭代内处理）

1. **后端未被前端消费的管理接口**
   - `POST /api/logout_all` 目前 UI 未使用。
   - 若无运维脚本依赖，可考虑隐藏或下线；若保留，应在 UI 或文档说明用途（如“全端登出”）。

2. **潜在“占位但未落地”的数据模型**
   - `remote_node_templates` 表已创建，但代码中未见读写/管理逻辑。
   - 建议：
     - 要么补齐模板管理功能；
     - 要么迁移时移除该表定义，避免 schema 膨胀。

3. **未使用的全局变量**
   - `dnsLogWSConnections`、`upgrader` 目前未形成完整业务链路（疑似 DNS WS 遗留）。
   - 建议结合 DNS 日志功能去留一并处理。

## 3.3 低优先级（仓库卫生，可批量处理）

1. **遗留补丁/备份工件**
   - `backend/patch_db.py`, `backend/patch_main.txt`, `frontend/dist/index.html.bak`, `frontend/dist/libs/patch_docs.sh`。
2. **调试残留文件**
   - `backend/test_query.go` 仅含 `mainTest()`，不参与主流程。
3. **构建产物入库风险**
   - `backend/proxygw`, `backend/proxygw-backend` 为二进制产物，不建议长期纳入版本库。

建议：
- 建立 `make clean` / release artifact 目录；
- 配合 `.gitignore` 与发布流程统一管理产物，降低仓库噪音。

---

## 四、建议的清理执行顺序（可直接建任务）

### Phase 1（接口契约收敛）
1. 决策 DNS 日志功能“做 or 删”；
2. 同步修正文档与前后端；
3. 回归：DNS 页签、WebSocket 重连行为、API 文档可用性。

### Phase 2（低价值代码清除）
1. 修正 WireGuard ping 解析逻辑；
2. 清理 `logout_all` / `remote_node_templates`（或补 UI/功能）；
3. 删除未使用全局变量。

### Phase 3（仓库卫生）
1. 清理 patch/bak/debug 文件；
2. 移除仓库中二进制，改为 CI 构建产物；
3. 增加一次“路由清单 vs 前端调用清单”的 CI 检查脚本，防止再次漂移。

---

## 五、审计方法（本次）

- 静态路由抽取：后端 `*.routes.go` 与 `api_routes.go`。
- 前端调用抽取：`frontend/dist/index.html` 中 `fetch/apiFetch/WebSocket`。
- 文档一致性校验：`docs/API.md` 与上述两侧交叉比对。
- 仓库卫生检查：定位 `.bak`、patch、调试文件、二进制产物。

