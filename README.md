# ProxyGW

ProxyGW 是一个 Debian 原生部署的透明代理网关，整合 Xray、Mosdns、nftables(TProxy)、FRR(OSPF)，并提供 Web 管理界面。

## 当前架构（第二轮治理后）

- 后端：Go + Gin（模块化路由）
- 前端：Vue 构建产物（`frontend/dist`）
- 数据库：`/root/proxygw/config/proxygw.db`
- 运行配置输出：
  - Xray: `core/xray/config.json`
  - Mosdns: `core/mosdns/config.yaml`

后端目录（关键）：

- `backend/main.go`：核心运行逻辑、配置生成、守护任务
- `backend/api_routes.go`：API 总装配与鉴权
- `backend/auth_routes.go`
- `backend/system_routes.go`
- `backend/config_routes.go`
- `backend/nodes_routes.go`
- `backend/rules_routes.go`
- `backend/dns_routes.go`
- `backend/update_routes.go`
- `backend/helpers.go`：可测试的纯函数
- `backend/mosdns_service.go`：Mosdns 配置渲染 service
- `backend/xray_service.go`：Xray 基础配置构建 service
- `backend/helpers_test.go`：最小回归测试（13 项）
- `backend/api_integration_test.go`：API 集成测试（httptest + 临时sqlite）

## 快速启动

```bash
cd /root/proxygw/backend
go test ./... -v
go build -o proxygw-backend .
systemctl restart proxygw
systemctl status proxygw --no-pager
```

## 访问

- Web: `http://<server-ip>/`
- 默认密码：`admin`

## 关键特性

1) 运行模式切换
- Mode A: start nftables + stop frr
- Mode B: stop nftables + start frr

2) DNS 与分流
- 路由分流中 `policy=proxy` 的 domain 规则会同步到 `core/mosdns/proxy_domains.txt`
- 远端 DNS 上游支持 socks5 出站（127.0.0.1:10808）

3) 组件更新
- Xray 版本列表：`GET /api/xray/versions`
- Xray 指定版本升级：`POST /api/update/xray` + `{ "version": "v26.3.27" }`
- Xray 回滚：`POST /api/update/rollback_xray`
- GeoData 更新链路使用 GitHub 官方直连（不使用 ghproxy）

## 文档索引

- `docs/README.md`
- `docs/API.md`
- `docs/OPERATIONS.md`
- `docs/CHANGELOG.md`
- `docs/RELEASE_TEMPLATE.md`
- `docs/AUDIT-2026-04-14.md`
- `docs/AUDIT-2026-04-14-round2.md`
