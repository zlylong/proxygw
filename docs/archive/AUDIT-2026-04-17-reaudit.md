# ProxyGW 全面复审审计（2026-04-17）

审计目标：在“前序问题已修复”的基础上，进行一次新的全量静态复审，重点关注**安全性、配置正确性、稳定性与可维护性**。

## 结论摘要

本轮复审确认了前一轮大量问题已被清理，但仍发现 **1 个高优先级功能回归问题** 与 **2 个中低优先级改进项**。

- **高优先级**：`applyMosdnsConfig()` 未读取数据库中的 DNS 设置，导致用户在 `/api/dns` 保存的上游配置不会真正写入 `mosdns` 配置（实际运行回退为内置默认值）。
- **中优先级**：远程部署 SSH 使用 `InsecureIgnoreHostKey()`，存在中间人攻击窗口。
- **低优先级**：`initDB()` 中 `log.Fatal` 后仍有不可达代码，影响可读性与维护质量。

---

## 1) 高优先级：DNS 配置保存后未生效（功能回归）

### 现象

`POST /api/dns` 会把 `dns_local/dns_remote/dns_lazy` 写入 `settings` 表，然后调用 `applyMosdnsConfig()`，理论上应根据用户输入重渲染 `core/mosdns/config.yaml`。

### 证据

`applyMosdnsConfig()` 里声明了 `local, remote, lazyStr`，但**没有从数据库读取这 3 个值**，直接拿空字符串/默认布尔值去渲染模板：

- 变量声明：`var local, remote, lazyStr string`
- 直接渲染：`renderMosdnsConfig(local, remote, lazyStr == "true")`

这会让 `formatUpstreams()` 走默认分支，最终上游退回内置值，而不是用户保存的配置。

### 影响

- 控制台显示“保存成功”，但运行配置与用户预期不一致。
- 在需要特定上游（企业 DNS、境内/境外定制、隐私 DNS）场景中，会造成策略失效。

### 修复建议

在 `applyMosdnsConfig()` 开头显式读取：

- `dns_local`
- `dns_remote`
- `dns_lazy`

读取失败时再做合理 fallback，并记录 warning；然后再调用 `renderMosdnsConfig()`。

---

## 2) 中优先级：远程 SSH 未校验主机指纹

### 现象

远程部署连接中使用：`ssh.InsecureIgnoreHostKey()`。

### 风险

- 无法验证目标主机身份。
- 在不可信网络中可能被中间人劫持，导致下发部署脚本到错误主机，或泄露凭证/部署内容。

### 建议

- 引入 `known_hosts` 校验（`knownhosts.New(...)`）。
- 至少提供“首次信任+指纹持久化”的 TOFU 模式，并在 UI/API 暴露指纹确认机制。

---

## 3) 低优先级：不可达代码

`initDB()` 中 `if err != nil { log.Fatal(err) ... }` 的花括号内，`log.Fatal` 后紧跟 `db.Exec(...)` 不会执行。建议清理为单一错误路径，减少维护歧义。

---

## 建议执行顺序

1. 先修复 `applyMosdnsConfig()` 的 DB 读取回归（高优先级，立即）。
2. 再补 SSH host key 验证（中优先级，安全增强）。
3. 最后做不可达代码与重复错误检查的清理（低优先级，代码质量）。

