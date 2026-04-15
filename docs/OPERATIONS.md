# ProxyGW 部署与运维手册

## 一、系统要求
- 仅支持原生 Debian 12 / 13 或 Ubuntu 22+ 环境。
- **严禁使用 Docker 运行**，本系统依赖内核路由（nftables / IP rule）及本地 FRR 进程。
- 需要 root 权限。

## 二、部署指南 (Git 纳管模式)

可以直接克隆仓库并执行安装脚本，脚本将自动处理依赖安装、内核路由和编译部署：

```bash
git clone https://github.com/zlylong/proxygw.git /root/proxygw
cd /root/proxygw/scripts
chmod +x install.sh
./install.sh
```

**关于初始密码：**
为了安全，ProxyGW **没有默认固定密码**。首次启动时，如果你没有配置 `PROXYGW_BOOTSTRAP_PASSWORD` 环境变量，后端将随机生成一个 12 位的初始密码，并保存在 `/root/proxygw/config/bootstrap_password.txt` 中。
你可以通过以下命令查看初始密码：
```bash
cat /root/proxygw/config/bootstrap_password.txt
```
登录 Web 界面后，请**立即修改密码**，修改后该 txt 文件中的随机密码将作废，后续将使用数据库中的 bcrypt 哈希密码。

## 三、版本更新 (Pull Deployment)

更新部署非常简单，由于整个环境现在被 Git 纳管，只需要拉取并重启即可。

### 标准更新流程

1. 进入项目目录：
   ```bash
   cd /root/proxygw
   ```
2. 拉取远端最新代码：
   ```bash
   git pull origin main
   ```
3. 编译后端（如果后端代码有更新）：
   ```bash
   cd backend
   go build -o proxygw-backend .
   ```
4. 重启服务以应用更改：
   ```bash
   systemctl restart proxygw
   ```

### Git 更新异常处理
如果之前有手动修改过代码或执行过强制回退，导致 Git Pull 遇到冲突，可以使用强行同步远端的策略（**注意：这会丢失未提交的本地代码修改，但不会影响运行期数据库和配置，因为数据库已经被 Git Ignore**）：

```bash
cd /root/proxygw
git fetch origin
git reset --hard origin/main
```

## 四、服务管理

```bash
systemctl status proxygw --no-pager
systemctl status xray --no-pager
systemctl status mosdns --no-pager
systemctl status frr --no-pager
systemctl status nftables --no-pager
```

```bash
systemctl restart proxygw
systemctl restart xray
systemctl restart mosdns
```

## 五、常见故障排查

### 5.1 Mosdns 启动失败

```bash
journalctl -u mosdns -n 200 --no-pager
/usr/local/bin/mosdns start -c /root/proxygw/core/mosdns/config.yaml
```

重点检查：
- 插件引用顺序（被 exec 的插件必须先定义）
- YAML 行拼接污染
- `ip_set` 必须使用 CIDR 文本，不可直接喂 `geoip.dat`

### 5.2 Xray 更新失败

```bash
journalctl -u proxygw -n 100 --no-pager
ls -l /tmp/xray.zip /tmp/xray/xray
xray version
```

### 5.3 密码遗忘处理

由于数据库存储的是 Bcrypt 哈希，无法逆向。在紧急情况下，可以通过 SQLite 命令修改数据库，将密码强制重置。

进入包含 `proxygw.db` 的目录并执行 SQL 替换。

## 六、备份与回滚

数据库：
```bash
cp /root/proxygw/config/proxygw.db /root/proxygw/config/proxygw.db.bak.$(date +%F-%H%M%S)
```

后端二进制：
```bash
cp /root/proxygw/backend/proxygw-backend /root/proxygw/backend/proxygw-backend.bak.$(date +%F-%H%M%S)
```

Xray：
```bash
# 接口级触发
curl -s -X POST http://127.0.0.1/api/update/rollback_xray -H "Authorization: Bearer <TOKEN>"
```
