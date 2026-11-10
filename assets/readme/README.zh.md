<div align="center">
  <img src="../logo.svg" alt="Databasus Logo" width="250"/>

  <h3>PostgreSQL 备份工具</h3>
  <p>Databasus 是一款免费、开源、可自托管的 PostgreSQL 备份工具。备份可以存放到各种存储（S3、Azure Blob Storage、FTP 等），进度通知可以发到 Slack、Discord、Telegram 等渠道。专注于低 RPO/RTO 的时间点恢复</p>
  
  <!-- Badges -->
   [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
  [![MySQL](https://img.shields.io/badge/MySQL-4479A1?logo=mysql&logoColor=white)](https://www.mysql.com/)
  [![MariaDB](https://img.shields.io/badge/MariaDB-003545?logo=mariadb&logoColor=white)](https://mariadb.org/)
  [![MongoDB](https://img.shields.io/badge/MongoDB-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
  <br />
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
  [![Docker Pulls](https://img.shields.io/docker/pulls/databasus/databasus?color=brightgreen)](https://hub.docker.com/r/databasus/databasus)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/databasus/databasus)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/databasus/databasus)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/databasus/databasus)

  <p>
    <a href="../../README.md">English</a> •
    <a href="README.ru.md">Русский</a> •
    <a href="README.es.md">Español</a> •
    <a href="README.pt.md">Português</a> •
    <b>中文</b> •
    <a href="README.fr.md">Français</a> •
    <a href="README.id.md">Bahasa Indonesia</a>
  </p>

  <p>
    <a href="#-功能特性">功能特性</a> •
    <a href="#-安装">安装</a> •
    <a href="#-使用">使用</a> •
    <a href="#-许可证">许可证</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://databasus.com/zh/" target="_blank"><strong>🌐 Databasus 官网</strong></a>
  </p>
  
  <img src="../dashboard-dark.svg" alt="Databasus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="../dashboard.svg" alt="Databasus Dashboard" width="800"/>
</div>

---

## ✨ 功能特性

### 📦 **备份类型**

- **物理备份**：对整个数据库集群做文件级复制，基于 PostgreSQL 原生的增量备份机制（阅读更多）
  - **全量**：集群的完整、自包含副本
  - **增量**：只存储自上次全量备份以来的变化，备份因此又小又快
  - **WAL 流式复制**：持续捕获数据库的写入流，支持时间点恢复（PITR）。为灾难恢复而设计，数据丢失接近于零
- **逻辑备份**：数据库引擎专用二进制格式的原生转储（已压缩，可用于并行恢复）

### 🔄 **定时备份**

- **灵活的计划**：每小时、每天、每周、每月或 cron
- **精确到时刻**：可以指定具体时间执行备份（比如流量低谷的凌晨 4 点）
- **智能压缩**：均衡压缩让备份体积缩小 4-8 倍，运行开销约 20%

### 🧪 **恢复验证** <a href="https://databasus.com/zh/restore-verification/">(文档)</a>

Databasus 会真正执行一次恢复，确认备份可用，而不只是检查文件还在磁盘上或者校验和对得上。

- **触发方式**：每次备份之后，或者按灵活的计划（每小时、每天、每周、每月或 cron）
- **真实恢复**：启动一个数据库容器，执行恢复，并把恢复后的体积与备份做核对
- **报告**：列出每张表及其行数
- **可选通知**：通过任意已配置的通知渠道发送报告，或只发送失败告警

### 🗑️ **保留策略**

- **按时间段**：备份保留固定时长（比如 7 天、3 个月、1 年）
- **按数量**：只保留固定数量的最新备份（比如最近 30 个）
- **GFS（祖父-父亲-儿子）**：分层保留，每小时、每天、每周、每月和每年的备份各自独立保留，用于精细的长期历史（企业常见需求）
- **大小限制**：为单个备份和总存储设置容量上限，控制存储占用

### 🗄️ **多种存储目标** <a href="https://databasus.com/zh/storages/">(查看支持列表)</a>

- **本地存储**：备份留在你的 VPS 或服务器上
- **云存储**：S3、Cloudflare R2、Azure Blob Storage、NAS、FTP、SFTP 等
- **安全**：所有数据都在你的掌控之中

### 📱 **通知** <a href="https://databasus.com/zh/notifiers/">(查看支持列表)</a>

- **多种渠道**：邮件、Telegram、Slack、Discord、Teams、Mattermost、webhook
- **实时更新**：成功和失败都会通知
- **团队协作**：非常适合 DevOps 的工作流

### 🔒 **企业级安全** <a href="https://databasus.com/zh/security/">(文档)</a>

- **AES-256-GCM 加密**：为备份文件提供企业级保护
- **零信任存储**：备份是加密的，落到攻击者手里也毫无用处，因此可以放心存在 S3 之类的共享存储中
- **密钥与凭据加密**：任何敏感数据都会加密，绝不外泄，日志和错误信息里也看不到
- **只读用户**：Databasus 默认使用只读用户来备份，不会保存任何能修改你数据的东西
- **双因素验证**：密码登录可以额外要求发送到账号邮箱的六位验证码。通过 Google 或 GitHub 登录仍然依赖这些服务商自己的验证

### 👥 **适合团队** <a href="https://databasus.com/zh/access-management/">(文档)</a>

- **工作空间**：按项目或团队把数据库、通知渠道和存储分组管理
- **访问管理**：用基于角色的权限控制谁能查看或管理哪些数据库
- **审计日志**：记录系统里的所有活动和用户所做的变更
- **用户角色**：在工作空间内分配 viewer、member、admin 或 owner 角色
- **OpenTelemetry 日志**：把应用日志和审计日志导出到外部系统（默认同时写入本地文件）

### 🎨 **好用的界面**

- **设计师打磨的 UI**：干净直观，细节到位
- **深色与浅色主题**：挑一个顺手的
- **移动端自适应**：随时随地在任何设备上查看备份

### 💾 **支持的数据库**

- **PostgreSQL**：14、15、16、17 和 18（物理和逻辑）
- **MySQL**：5.7、8.0、8.4、9 和 26（仅逻辑）
- **MariaDB**：5.5、10、11、12 和 13（仅逻辑）
- **MongoDB**：4.2+、5、6、7、8 和 9（仅逻辑）

### 🐳 **自托管且安全**

- **基于 Docker**：部署和管理都很简单
- **隐私优先**：所有数据都留在你自己的基础设施上
- **开源**：Apache 2.0 许可证，每一行代码都可以查
- **内置 SSH**：可以通过 SSH 隧道连接到你的 Databasus

### 📦 安装 <a href="https://databasus.com/zh/installation/">(文档)</a>

安装 Databasus 有四种方式：

- 自动安装脚本（推荐）
- 简单的 Docker run
- Docker Compose 部署
- Kubernetes 配合 Helm

<img src="../healthchecks.svg" alt="Databasus Dashboard" width="800"/>

---

## 📦 安装

安装 Databasus 有四种方式：自动安装脚本（推荐）、简单的 Docker run，或者 Docker Compose 部署。

### 方式一：自动安装脚本（推荐，仅限 Linux）

安装脚本会：

- ✅ 安装 Docker 和 Docker Compose（如果还没装）
- ✅ 部署 Databasus
- ✅ 配置系统重启后自动启动

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh \
| sudo bash
```

### 方式二：简单的 Docker run

运行 Databasus 最简单的办法：

```bash
docker run -d \
  --name databasus \
  -p 4005:4005 \
  -v ./databasus-data:/databasus-data \
  --restart unless-stopped \
  databasus/databasus:latest
```

_同一个镜像也发布在 GitHub 的镜像仓库上，如果 Docker Hub 对你的拉取做了限流，可以改用 `ghcr.io/databasus/databasus:latest`。_

这一条命令会：

- ✅ 启动 Databasus
- ✅ 把所有数据存到 `./databasus-data` 目录
- ✅ 系统重启后自动拉起

### 方式三：Docker Compose 部署

创建一个 `docker-compose.yml` 文件，内容如下：

```yaml
services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "databasus", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 60s
```

然后运行：

```bash
docker compose up -d
```

### 方式四：Kubernetes 配合 Helm

在 Kubernetes 上部署时，直接从 OCI 仓库安装。

_在下面任意一条命令后加上 `--set image.repository=ghcr.io/databasus/databasus`，就会从 GHCR 而不是 Docker Hub 拉取镜像。_

**使用 ClusterIP + port-forward（开发/测试）：**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005
```

**使用 LoadBalancer（云环境）：**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set service.type=LoadBalancer
```

```bash
kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005
```

**使用 Ingress（按域名访问）：**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=backup.example.com
```

更多选项（NodePort、TLS、Gateway API 的 HTTPRoute）见 [Helm chart README](../../deploy/helm/README.md)。

---

## 🚀 使用

1. **创建第一个账户**：访问 `http://localhost:4005` 并注册。实例上创建的第一个账户就是它的管理员
2. **添加第一个要备份的数据库**：点击 "New Database"，跟着向导走完
3. **配置计划**：可选每小时、每天、每周、每月或 cron 周期
4. **填写数据库连接**：输入数据库的凭据和连接信息
5. **选择存储**：决定备份存到哪里（本地、S3、Azure Blob Storage 等）
6. **配置保留策略**：按时间段、数量或 GFS 控制备份保留多久
7. **添加通知**（可选）：配置邮件、Telegram、Slack、Mattermost 或 webhook 通知
8. **保存并启动**：Databasus 会校验配置，然后按计划开始备份

### 🔑 重置密码 <a href="https://databasus.com/zh/password/">(文档)</a>

如果需要重置密码，可以用内置的密码重置命令：

```bash
docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"
```

把 `owner@example.com` 换成要重置密码的那个账户的邮箱地址。

如果你不记得哪个地址是该实例的管理员，可以列出管理员账户：

```bash
docker exec -it databasus ./main --list-admins
```

输出会列出每个管理员账户的邮箱、显示名称、创建日期和活动状态，并标记实例所认可的管理员。不会输出任何密码或密码哈希。

如果不想在浏览器里注册，比如要用部署脚本开通账户，可以在控制台直接创建：

```bash
docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin
```

`--role` 取 `admin` 或 `member`，密码至少 8 个字符，`--name` 可选，用来设置显示名称。如果实例上还没有任何账户，第一个账户必须使用 `--role=admin`，它就是该实例的管理员。对同一账户再次运行命令不会做任何改动，所以部署流程可以在每次上线时都运行它；要更换密码，请使用 `--new-password`。

### 💾 给 Databasus 自身做备份

装好之后，建议再<a href="https://databasus.com/zh/faq/#backup-databasus">给 Databasus 自身做一次备份</a>，至少也要把用于加密的密钥复制一份（只要 30 秒）。这样即使装有 Databasus 的服务器丢失访问权限或者损坏，你依然能从加密备份中恢复。

---

## 🛡️ 安全与可靠性工程

详情请参阅[完整的安全与可靠性流程](https://databasus.com/zh/security/#security-and-reliability-engineering)。

每次发布前，Databasus 都会运行自动检查，以降低漏洞、数据泄露以及备份无法恢复的风险：

- **安全扫描**检查代码、依赖和容器镜像中的已知漏洞，并查找泄露的密钥等敏感信息。
- **恢复测试**针对每个受支持的数据库引擎和主要版本，在真实数据库容器中运行完整的备份与恢复流程。
- **发布条件**要求测试、构建和安全检查全部通过。发布分析中尚未处理的 CodeQL 严重漏洞会阻止发布。

OpenAI 的 Codex Security 还会定期开展审计，补充这些发布检查。如需报告漏洞，请按照 [SECURITY.md](https://github.com/databasus/databasus?tab=security-ov-file#readme) 私下联系；安全报告会优先处理。

---

## 📝 许可证

本项目采用 Apache 2.0 许可证，详见 [LICENSE](../../LICENSE) 文件

## 💬 社区

欢迎加入我们在 Telegram 上的开发者、DBA 和 DevOps 工程师社群 [@databasus_community](https://t.me/databasus_community)。
