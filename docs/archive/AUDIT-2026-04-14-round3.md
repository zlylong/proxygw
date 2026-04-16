# ProxyGW 深度治理审计报告（Round 3）

日期：2026-04-14

## 目标
1) 将 Mosdns/Xray 配置生成逻辑从 `main.go` 继续下沉到 service 层
2) 增加 API 级集成测试（`httptest + 临时 sqlite`）
3) 维持线上行为兼容与可回归验证

## 本轮代码变更

### 1. Service 层抽离
新增：
- `backend/mosdns_service.go`
  - `renderMosdnsConfig(local, remote string, lazy bool) string`
- `backend/xray_service.go`
  - `buildBaseXrayConfig() map[string]interface{}`

调整：
- `applyMosdnsConfig()` 改为调用 `renderMosdnsConfig`
- `applyXrayConfig()` 改为调用 `buildBaseXrayConfig`

说明：业务行为保持不变，主要提升可维护性与可测试性。

### 2. API 集成测试
新增：
- `backend/api_integration_test.go`

覆盖：
- `POST /api/login` 成功/失败
- `GET /api/dns` 未授权/授权
- `GET /api/rules` 授权读取

测试环境：
- Gin `TestMode`
- `httptest` 路由调用
- 每个 case 使用临时 sqlite 文件并初始化最小 schema

### 3. 配置更新链路一致性
- 再次确认不使用 ghproxy，GeoData 走 GitHub 官方直连。

## 验证结果

- `go test ./... -v`：通过
- `go vet ./...`：通过
- `go build -o proxygw-backend .`：通过
- `systemctl is-active proxygw`：active
- `systemctl is-active mosdns`：active
- `systemctl is-active xray`：active
- API 冒烟：`/api/login` `/api/status` `/api/dns` `/api/rules` 正常

## 风险与后续建议（Round 4）
1) 将 `applyXrayConfig` 节点拼装与规则拼装再拆成独立函数（纯函数 + DB 访问分离）
2) 为 `/api/update/*` 增加可注入 runner，降低测试中对系统命令依赖
3) 引入 CI：`go test + go vet + golangci-lint + docs link check`
