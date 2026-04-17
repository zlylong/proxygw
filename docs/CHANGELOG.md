# ProxyGW Changelog

## 2026-04-17 (v1.4.0-rc.1: Pre-release)
- 预发布 1.4.0 版本
- 同步最新的 GeoData 和 Core 二进制文件

## 2026-04-16 (v1.3.2: Bugfix - Link Parsing & Architecture Docs)
- **Bug Fix**: 修复前后端 `vless://` 链接解析逻辑，防止在导入节点时漏掉 `flow` (如 `xtls-rprx-vision`) 和 `encryption` 流控参数，导致 Reality 握手失败。
- **Documentation**: 更新 UI 面板中的 "Xray 透明代理架构分析"，精准描述当前采用的最佳实践架构（Nftables 无脑劫持 + Xray Sniffing 与 Routing 分流）。

## 2026-04-16 (v1.3.1: Offline GeoIP Parsing & DNS Optimization)

### Added
- **Dynamic GeoIP Extraction**: 后端基于纯 Go 标准库实现了完全本地、脱机的 Protobuf 解析器。现在添加任何 geoip 代理规则（如 telegram, netflix 等），系统将直接在本地毫秒级从 geoip.dat 二进制文件中剥离对应的 IPv4 CIDR 网段并注入 OSPF。无需再依赖外部 raw.githubusercontent.com 接口。

### Changed
- 移除了前端与后端关于 DNS 实时解析流 (WebSocket) 的逻辑。因 Mosdns v5 默认日志为性能优化已精简，不再全量输出解析流。移除后进一步降低了后台的资源开销与 CPU 占用。
- 前端 DNS 面板 UI 重新排版布局。


## 2026-04-15 (Deep System & UI Optimization, Security, and Fake-IP)

### Added
- **Deep System Optimization**: 后端 SQLite 强制开启 PRAGMA journal_mode=WAL，读写全并发。
- **Deep System Optimization**: API 响应引入 Gzip 压缩。
- **Deep System Optimization**: /api/login 新增动态指数延迟与最大 10 次错误熔断机制。
- **Deep System Optimization**: 前端资源 (Vue3, Tailwind, FontAwesome) 完成全量本地化下载，支持 100% 离线脱机运行。
- **Fake-IP Architecture**: 完整引入了 Fake-IP 零延迟直通架构。Xray 开启内置 fakedns 引擎 (198.18.0.0/16)，OSPF (FRR) 在 Mode B 下静态宣告 198.18.0.0/16 路由。
- **Security**: Supply Chain Validation - Added rigorous SHA256/SHA512 hash verification for Xray-core downloads via .dgst files.
- **Security**: GeoData Validation - Added rules.zip.sha256sum signature verification for geosite and geoip downloads.
- **Security**: Systemd Sandboxing - Introduced ProtectSystem=strict, NoNewPrivileges=yes, PrivateTmp=yes to proxygw.service.
- **Security**: Initial Password Generation - Removed hardcoded admin password. First-time installs now generate a secure random password saved to config/bootstrap_password.txt.

### Changed
- 彻底移除了 watchDnsLogs 协程与相关陈旧的 OSPF 动态路由下发逻辑（由于 Fake-IP 架构已上线，此逻辑为无效高负载开销）。
- 修改 xray_service.go，TProxy 入站开启对 fakedns、http 和 tls 的 Sniffing。
- 修改 mosdns_service.go，命中 proxy_domains 时瞬间向 Xray 的 FakeDNS 请求假 IP 并阻塞式返回。
- scripts/install.sh upgraded to auto-deploy the hardened Systemd service and disable systemd-resolved DNS stub.
- README.md updated to reflect Fake-IP architecture and security posture.

### Fixed
- 彻底解决由于 DNS 解析时间与 OSPF 路由下发时间差导致的首包（TCP SYN）漏网与高延迟断流问题。

## 2026-04-14 (Round 3 Deep Governance)

### Added
- Service 层文件：mosdns_service.go、xray_service.go
- API 集成测试：api_integration_test.go（httptest + 临时 sqlite）
- 审计报告：AUDIT-2026-04-14-round3.md

### Changed
- applyMosdnsConfig 使用 renderMosdnsConfig() 生成配置
- applyXrayConfig 使用 buildBaseXrayConfig() 初始化基础结构
- 继续保持 geodata 官方 GitHub 更新链路

### Verification
- go test ./... -v 通过
- go vet ./... 通过
- go build -o proxygw-backend . 通过
- systemctl is-active proxygw/mosdns/xray 均为 active

## 2026-04-14 (Round 2 Deep Governance)

### Added
- 后端模块化路由文件：auth/dns/xray(update)/nodes/rules/system/config/api
- helpers.go 可测试纯函数集合
- helpers_test.go 回归测试（13 项）
- 文档：CHANGELOG.md、RELEASE_TEMPLATE.md、Round2 审计报告

### Changed
- main.go 从“路由+业务混合”改为“核心逻辑 + 路由装配”
- /api/update/xray 下载 URL 构建改为 helper 函数统一校验
- geodata 更新链路统一使用 GitHub 官方直连

### Fixed
- 修复文档中旧接口样例与请求头格式错误
- 修复运维文档中 build 命令与实际代码结构不一致问题

### Verification
- go test ./... -v 通过
- go build -o proxygw-backend . 通过
- systemctl is-active proxygw/mosdns/xray 均为 active
