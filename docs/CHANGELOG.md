# ProxyGW Changelog

## 2026-04-15 (Security Hardening & Sandbox)

### Added
- **Supply Chain Validation**: Added rigorous SHA256/SHA512 hash verification for Xray-core downloads via `.dgst` files.
- **GeoData Validation**: Added `rules.zip.sha256sum` signature verification for geosite and geoip downloads, preventing malicious file replacement.
- **Systemd Sandboxing**: Introduced `ProtectSystem=strict`, `NoNewPrivileges=yes`, `PrivateTmp=yes`, and `ReadWritePaths` whitelisting to the `proxygw.service` to confine the backend root process.
- **Initial Password Generation**: Removed hardcoded `admin` password. First-time installs now generate a secure random password saved to `config/bootstrap_password.txt`.

### Changed
- `scripts/install.sh` has been upgraded to auto-deploy the hardened Systemd service.
- `README.md` updated to reflect the new sandboxing model and security posture.

# ProxyGW Changelog

## 2026-04-14 (Round 3 Deep Governance)

### Added
- Service 层文件：`mosdns_service.go`、`xray_service.go`
- API 集成测试：`api_integration_test.go`（httptest + 临时 sqlite）
- 审计报告：`AUDIT-2026-04-14-round3.md`

### Changed
- `applyMosdnsConfig` 使用 `renderMosdnsConfig()` 生成配置
- `applyXrayConfig` 使用 `buildBaseXrayConfig()` 初始化基础结构
- 继续保持 geodata 官方 GitHub 更新链路

### Verification
- `go test ./... -v` 通过
- `go vet ./...` 通过
- `go build -o proxygw-backend .` 通过
- `systemctl is-active proxygw/mosdns/xray` 均为 active

## 2026-04-14 (Round 2 Deep Governance)

### Added
- 后端模块化路由文件：auth/dns/xray(update)/nodes/rules/system/config/api
- `helpers.go` 可测试纯函数集合
- `helpers_test.go` 回归测试（13 项）
- 文档：`CHANGELOG.md`、`RELEASE_TEMPLATE.md`、Round2 审计报告

### Changed
- `main.go` 从“路由+业务混合”改为“核心逻辑 + 路由装配”
- `/api/update/xray` 下载 URL 构建改为 helper 函数统一校验
- geodata 更新链路统一使用 GitHub 官方直连

### Fixed
- 修复文档中旧接口样例与请求头格式错误
- 修复运维文档中 build 命令与实际代码结构不一致问题

### Verification
- `go test ./... -v` 通过
- `go build -o proxygw-backend .` 通过
- `systemctl is-active proxygw/mosdns/xray` 均为 active
