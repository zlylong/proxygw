# ProxyGW 工程级代码审计报告（Round 4）

审计日期：2026-04-15  
审计范围：`backend/*.go`、`scripts/install.sh`、`frontend/dist/index.html`、`backend/go.mod`

---

## 阶段1：项目结构分析

### 技术栈
- 后端：Go 1.25、Gin、SQLite、Gorilla WebSocket。  
- 前端：单文件 Vue（dist 直出）、Fetch API。  
- 系统集成：systemd、FRR/OSPF、nftables、xray、mosdns。  
- 部署：Shell 安装脚本（root 级系统改写）。

### 目录与核心模块
- `backend/main.go`：启动入口、DB 初始化、OSPF 控制循环、配置下发。  
- `backend/*_routes.go`：API 路由（认证、节点、规则、DNS、系统状态、更新）。  
- `backend/helpers.go`：下载、哈希验证、输入校验辅助函数。  
- `scripts/install.sh`：系统安装与 service 注册。  
- `frontend/dist/index.html`：管理界面与 API 调用逻辑。

### 启动流程
1. 启动时先执行 `syncFRRConfig()` 自动重写 FRR 配置。  
2. 初始化 SQLite、建表、初始化默认配置及密码。  
3. 启动两个后台 goroutine：`ospfController()` 与 `cronUpdater()`。  
4. 启动 Gin HTTP 服务，监听 `:80`。

### 配置加载方式
- 持久化配置主存于 SQLite `settings/nodes/rules/routes_table`。  
- `applyMosdnsConfig`/`applyXrayConfig` 将 DB 内容渲染为配置文件并重启服务。

### 网络入口（攻击面）
- HTTP API：`/api/*`（Bearer Token 认证）。  
- WebSocket：`/api/dns/logs/ws`。  
- 外部网络请求：GitHub Release API 与文件下载。  
- 系统命令调用：`systemctl`、`vtysh`、`tail`、`unzip`、`install` 等。

---

## 阶段2：模块级审计

### 1) 认证与会话模块（`auth_routes.go`, `api_routes.go`）
- 功能：登录、改密、登出、Bearer 校验。  
- 风险等级：**高**。  
- 关键问题：函数内重复定义 `loginAttempts`，导致全局限流状态被遮蔽；会话仅内存存储，重启全失效（可接受但需明确）。

### 2) 路由与网关编排模块（`main.go`, `system_routes.go`）
- 功能：OSPF 发布/撤销、模式切换、FRR 自动配置。  
- 风险等级：**严重~高**。  
- 关键问题：大量 `exec.Command(...).Run()` 忽略错误，可能产生“控制面显示成功、数据面未生效”的失配；FRR 配置写入无原子回滚。

### 3) 节点与规则模块（`nodes_routes.go`, `rules_routes.go`）
- 功能：节点 CRUD、URL 导入、规则配置。  
- 风险等级：**高**。  
- 关键问题：导入与写库链路中错误被忽略；`/nodes/ping` 无并发上限，可被触发为 goroutine 洪泛。

### 4) DNS 与日志模块（`dns_routes.go`）
- 功能：DNS 上游配置、日志读取、WebSocket tail -f。  
- 风险等级：**中~高**。  
- 关键问题：日志 WS 每连接起一个 `tail -f` 子进程，缺少连接与进程总量限制。

### 5) 更新模块（`update_routes.go`, `helpers.go`）
- 功能：拉取 geodata/xray、校验 hash、安装回滚。  
- 风险等级：**高**。  
- 关键问题：临时文件路径固定在 `/tmp`，存在符号链接/覆盖风险；部分写文件和命令错误未处理。

### 6) 前端模块（`frontend/dist/index.html`）
- 功能：登录、管理、调用后端 API、DNS 日志 WS。  
- 风险等级：**中**。  
- 关键问题：token 存 localStorage；WS 强制 `ws://`，在 HTTPS 场景失效且泄露风险增大。

### 7) 部署脚本（`scripts/install.sh`）
- 功能：安装依赖、改系统配置、部署服务。  
- 风险等级：**高**。  
- 关键问题：未启用 `-u/-o pipefail`，重复逻辑，root 下直接覆盖系统文件且缺少备份/回滚。

---

## 阶段3：结构化问题清单

> 说明：
> - **已确认问题**：从代码可直接证明。  
> - **高概率问题**：代码迹象明显，需结合运行环境确认影响面。  
> - **需人工确认**：与部署策略/网络边界强相关。

### P1. 登录限流状态被局部变量遮蔽（已确认）
- 等级：**高**
- 类型：Bug / 安全
- 文件：`backend/auth_routes.go`
- 位置：`registerAuthRoutes`（约 L114-L120）
- 描述：函数内重新声明 `var loginAttempts sync.Map`，遮蔽包级变量，导致限流状态生命周期与初始化语义混乱，后续维护极易误判。  
- 触发条件：任意登录请求。
- 影响：爆破防护不稳定，审计与运维难以验证真实限流状态。
- 修复建议：删除局部声明，统一使用包级变量；增加 TTL 清理与指标。
- 可自动修复：是。

### P2. 大量数据库与系统调用错误被忽略（已确认）
- 等级：**严重**
- 类型：Bug / 稳定性 / 安全
- 文件：`backend/main.go`, `backend/nodes_routes.go`, `backend/rules_routes.go`, `backend/system_routes.go`
- 位置：多处 `db.Query(...); rows.Scan(...); exec.Command(...).Run()` 未判错。
- 描述：关键路径（规则下发、OSPF 发布、模式切换）多处忽略错误。  
- 触发条件：数据库锁冲突、命令失败、权限不足、FRR/xray 异常时。
- 影响：API 返回成功但系统未生效；形成“伪成功”与配置漂移。
- 修复建议：统一错误处理策略（fail-fast + structured log + 指标 + 前端错误透传）。
- 可自动修复：部分可自动。

### P3. OSPF/FRR 配置更新无原子与回滚（已确认）
- 等级：**严重**
- 类型：安全 / 稳定性
- 文件：`backend/main.go`
- 位置：`syncFRRConfig`、`ospfController`。
- 描述：直接覆盖 `/etc/frr/frr.conf` 并重启；失败无事务与回滚，且路由增删命令失败未处理。  
- 触发条件：配置格式问题、磁盘错误、systemctl 失败。
- 影响：路由发布异常、邻居关系抖动，极端可导致路由劫持窗口。
- 修复建议：采用“写临时文件→语法校验→原子替换→失败回滚”流程，并验证 `vtysh` 返回码。
- 可自动修复：部分可自动。

### P4. `/nodes/ping` 可触发 goroutine 洪泛（已确认）
- 等级：**高**
- 类型：性能 / 稳定性
- 文件：`backend/nodes_routes.go`
- 位置：`/nodes/ping`（L104+）
- 描述：每个节点直接 `go`，无并发限制、无 context、无汇聚等待。  
- 触发条件：节点量大或反复调用该接口。
- 影响：goroutine 暴涨、DB 写压力突增、管理面卡顿。
- 修复建议：worker pool + context timeout + 批量写入 + 限流。
- 可自动修复：是。

### P5. WebSocket 日志流每连接启动 `tail -f`（已确认）
- 等级：**高**
- 类型：性能 / 资源管理
- 文件：`backend/dns_routes.go`
- 位置：`/dns/logs/ws`
- 描述：每个 WS 连接启动一个长期子进程，未做连接上限与鉴权细粒度控制。  
- 触发条件：多客户端同时打开 DNS 页面。
- 影响：进程数增长、fd 压力、潜在拒绝服务。
- 修复建议：单后台 tail 广播模型；连接数上限；空闲超时。
- 可自动修复：是。

### P6. shell 命令通过 `sh -c` 执行（已确认）
- 等级：**中**
- 类型：安全
- 文件：`backend/system_routes.go`
- 位置：CPU/RAM 获取命令。
- 描述：虽当前字符串常量无直接用户输入，但 `sh -c` 引入额外攻击面与维护风险。  
- 触发条件：未来改造拼接入参时。
- 影响：命令注入风险放大（演进性风险）。
- 修复建议：改为直接读取 `/proc` 或使用 `exec.Command` 固定参数链。
- 可自动修复：是。

### P7. 更新流程临时文件固定路径（已确认）
- 等级：**高**
- 类型：安全
- 文件：`backend/update_routes.go`
- 位置：`/tmp/rules.zip`, `/tmp/xray.zip`, `/tmp/xray`。
- 描述：固定 `/tmp` 路径，root 进程下可能遭符号链接/竞争攻击。  
- 触发条件：本地低权限用户可写 `/tmp`。
- 影响：覆盖任意文件或植入恶意二进制（结合其他条件）。
- 修复建议：`os.CreateTemp` + `MkdirTemp` + `O_EXCL` + 权限收敛 + 清理。
- 可自动修复：是。

### P8. 部分网络请求未复用统一超时客户端（已确认）
- 等级：**中**
- 类型：性能 / 稳定性
- 文件：`backend/helpers.go`
- 位置：`getGeoDataVersionAndHash` 使用 `http.Get`。
- 描述：项目已有 `httpClient`，但此处仍用默认 client。  
- 触发条件：上游网络慢/阻塞。
- 影响：阻塞链路与 goroutine 堆积风险。
- 修复建议：统一注入带 timeout 的 `http.Client`。
- 可自动修复：是。

### P9. 前端 token 使用 localStorage（已确认）
- 等级：**中**
- 类型：安全（Web）
- 文件：`frontend/dist/index.html`
- 位置：`localStorage.getItem/setItem('token')`。
- 描述：若发生 XSS，token 易被直接窃取。  
- 触发条件：任意前端 XSS 成功。
- 影响：会话接管。
- 修复建议：迁移 HttpOnly/SameSite Cookie + CSRF token；最少缩短 token TTL。
- 可自动修复：部分可自动。

### P10. DNS 日志 WS 强制 `ws://`（已确认）
- 等级：**中**
- 类型：安全 / 兼容性
- 文件：`frontend/dist/index.html`
- 位置：`new WebSocket(\`ws://${window.location.host}...\`)`
- 描述：HTTPS 场景下应使用 `wss://`；当前实现会 mixed-content 失败或明文传输。  
- 触发条件：反代启用 TLS。
- 影响：日志流不可用或被窃听。
- 修复建议：根据 `location.protocol` 动态选择 `ws/wss`。
- 可自动修复：是。

### P11. 安装脚本健壮性不足（已确认）
- 等级：**中**
- 类型：安全 / 运维稳定性
- 文件：`scripts/install.sh`
- 位置：脚本头部与系统文件改写段。
- 描述：仅 `set -e`，缺 `-u -o pipefail`；存在重复配置片段；root 下直接写系统关键文件。  
- 触发条件：未定义变量、管道前段失败、重复运行。
- 影响：部署行为不确定、系统配置污染。
- 修复建议：`set -euo pipefail`，去重逻辑，关键文件改写前备份。
- 可自动修复：是。

### P12. 依赖清理：`gin-contrib/gzip` 未使用（已确认）
- 等级：**低**
- 类型：无效代码 / 供应链面
- 文件：`backend/go.mod`
- 位置：直接依赖列表。
- 描述：未在代码中引用，增加不必要依赖面。  
- 触发条件：常态。
- 影响：构建时间与供应链攻击面微增。
- 修复建议：移除未使用依赖并执行 `go mod tidy`。
- 可自动修复：是。

### P13. 会话状态仅内存存储（高概率问题）
- 等级：中
- 类型：稳定性 / 安全运营
- 文件：`backend/auth_routes.go`
- 描述：重启后所有会话失效，且未限制 session 总数。
- 影响：大规模运维时可能触发异常登出或内存增长。
- 修复建议：会话持久化（可选 Redis/DB）+ 总量/TTL 回收。
- 可自动修复：否（需架构决策）。

### P14. API 访问缺少网络边界约束（需人工确认）
- 等级：高（条件成立时）
- 类型：安全
- 文件：`backend/main.go`, `backend/api_routes.go`
- 描述：服务直接监听 `:80`，若暴露公网且无额外 ACL/WAF，将管理 API 暴露在互联网。  
- 影响：暴力破解、接口滥用、服务编排被远程触发。
- 修复建议：仅监听内网/环回、反代鉴权、IP 白名单、Fail2ban。
- 可自动修复：否（依赖部署拓扑）。

---

## 阶段4：无效代码/可清理项

### 可删除文件（高置信）
- `backend/patch_auth2.js`：一次性补丁脚本，不应进入运行仓库。  
- `backend/patch_auth_fix.js`：同上。  
- `backend/patch_tests.sh`：若仅临时修复用途，建议迁移到 `tools/` 并标注。  

### 可删除/重构函数或片段
- `registerAuthRoutes` 内部局部 `loginAttempts` 声明（重复/有害）。  
- `install.sh` 中重复的 systemd-resolved 处理块。

### 未使用依赖（go.mod）
- `github.com/gin-contrib/gzip`（直接依赖未引用）。

### 重复模块/逻辑
- `geolocation/private` 与 `geoip/private` 处理逻辑重复，可抽函数。

---

## 阶段5：最终结论

## 1) 审计总结
- 项目具备基础输入校验与下载哈希校验，但控制面关键路径存在“错误忽略 + 非原子系统变更”问题，这是当前最大工程风险。  
- 在网关/路由场景中，此类风险会被放大为“控制平面成功、转发平面失败”的隐性故障。  
- 高风险集中在：**OSPF/FRR 编排、批量并发任务、子进程模型、更新临时文件安全**。

## 2) 高风险 Top 10（按优先级）
1. P2 关键错误被广泛忽略（严重）  
2. P3 FRR/OSPF 非原子写入与无回滚（严重）  
3. P4 `/nodes/ping` goroutine 洪泛（高）  
4. P5 DNS WS 每连接起子进程（高）  
5. P7 更新临时文件固定 `/tmp`（高）  
6. P1 登录限流变量遮蔽（高）  
7. P14 管理面暴露边界不明（高，需人工确认）  
8. P6 `sh -c` 习惯性命令执行（中，演进风险）  
9. P9 localStorage 持久 token（中）  
10. P10 WS 未适配 TLS（中）

## 3) 可删除代码清单
- `backend/patch_auth2.js`  
- `backend/patch_auth_fix.js`  
- （条件）`backend/patch_tests.sh`  
- `go.mod` 未使用依赖 `gin-contrib/gzip`

## 4) 修复优先级
- **P0（24h内）**：P2、P3、P7。  
- **P1（1周内）**：P1、P4、P5、P14。  
- **P2（2周内）**：P6、P9、P10、P11、P12。

## 5) 最小整改方案（先修必修）
1. 建立统一 `exec/db` 错误处理框架（返回码、日志、指标、失败回传）。  
2. 路由/配置更新引入原子化与回滚（临时文件+校验+切换）。  
3. `/nodes/ping` 与 DNS WS 引入并发/连接上限。  
4. 更新流程改为安全临时目录与权限控制。  
5. 修复登录限流变量遮蔽并增加防爆破可观测性。

