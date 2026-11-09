import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "pg_dump 替代方案 - Databasus PostgreSQL 备份工具",
  description:
    "Databasus 基于 pg_dump 构建，并在其之上扩展了备份管理、Web 界面、自动调度、云存储、通知、团队协作和加密功能。",
  keywords: [
    "pg_dump 替代方案",
    "pg_dump 图形界面",
    "pg_dump 自动化",
    "pg_dump Web 界面",
    "PostgreSQL 备份工具",
    "pg_dump 调度器",
    "pg_dump 云存储",
    "pg_dump 加密",
    "PostgreSQL 备份自动化",
    "pg_dump 封装工具",
  ],
  openGraph: {
    title: "pg_dump 替代方案 - Databasus PostgreSQL 备份工具",
    description:
      "Databasus 基于 pg_dump 构建，并在其之上扩展了备份管理、Web 界面、自动调度、云存储、通知、团队协作和加密功能。",
    type: "article",
    url: getLocalizedUrl("zh", "pgdump-alternative"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "pg_dump 替代方案 - Databasus PostgreSQL 备份工具",
    description:
      "Databasus 基于 pg_dump 构建，并在其之上扩展了备份管理、Web 界面、自动调度、云存储、通知、团队协作和加密功能。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "pgdump-alternative"),
    languages: getLanguageAlternates("pgdump-alternative"),
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "pg_dump 替代方案 - Databasus PostgreSQL 备份工具",
            description:
              "全面介绍作为 pg_dump 替代方案的 Databasus：它如何基于 pg_dump 构建，并通过自动化、云存储、通知和团队功能扩展其能力。",
            author: {
              "@type": "Organization",
              name: "Databasus",
            },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
          }),
        }}
      />

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="pgdump-alternative">pg_dump 替代方案</h1>

              <p className="text-lg text-gray-400">
                在逻辑备份方面，Databasus 构建在 <code>pg_dump</code>{" "}
                之上。Databasus 并不是要取代 <code>pg_dump</code>
                ，而是在其基础上扩展了备份管理、Web
                界面、自动调度、云存储集成、通知、团队协作功能和内置加密。除了逻辑备份，Databasus
                还支持物理备份、基于 WAL
                归档的增量备份和时间点恢复（Point-in-Time Recovery）。
              </p>

              <h2 id="quick-comparison">快速对比</h2>

              <p>
                下表概览 Databasus 如何扩展 <code>pg_dump</code> 的核心功能：
              </p>

              <table>
                <thead>
                  <tr>
                    <th>功能</th>
                    <th>pg_dump</th>
                    <th>Databasus</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>备份引擎</td>
                    <td data-label="pg_dump">pg_dump</td>
                    <td data-label="Databasus">基于 pg_dump 构建</td>
                  </tr>
                  <tr>
                    <td>备份管理</td>
                    <td data-label="pg_dump">❌ 无</td>
                    <td data-label="Databasus">✅ 有</td>
                  </tr>
                  <tr>
                    <td>其他数据库支持</td>
                    <td data-label="pg_dump">仅 PostgreSQL</td>
                    <td data-label="Databasus">
                      PostgreSQL、MySQL、MariaDB、MongoDB
                    </td>
                  </tr>
                  <tr>
                    <td>界面</td>
                    <td data-label="pg_dump">命令行</td>
                    <td data-label="Databasus">Web 界面 + API</td>
                  </tr>
                  <tr>
                    <td>调度</td>
                    <td data-label="pg_dump">手动或 cron 脚本</td>
                    <td data-label="Databasus">✅ 内置调度器</td>
                  </tr>
                  <tr>
                    <td>存储目标</td>
                    <td data-label="pg_dump">仅本地文件系统</td>
                    <td data-label="Databasus">
                      本地、S3、R2、Azure、NAS、FTP
                    </td>
                  </tr>
                  <tr>
                    <td>压缩</td>
                    <td data-label="pg_dump">gzip、LZ4、zstd（手动）</td>
                    <td data-label="Databasus">zstd（自动、已优化）</td>
                  </tr>
                  <tr>
                    <td>加密</td>
                    <td data-label="pg_dump">需要外部工具</td>
                    <td data-label="Databasus">✅ 内置 AES-256-GCM</td>
                  </tr>
                  <tr>
                    <td>通知</td>
                    <td data-label="pg_dump">❌ 无</td>
                    <td data-label="Databasus">
                      ✅ Slack、Teams、Telegram、邮件、Webhook
                    </td>
                  </tr>
                  <tr>
                    <td>团队功能</td>
                    <td data-label="pg_dump">❌ 无</td>
                    <td data-label="Databasus">✅ 工作区、RBAC、审计日志</td>
                  </tr>
                  <tr>
                    <td>保留策略</td>
                    <td data-label="pg_dump">手动清理脚本</td>
                    <td data-label="Databasus">✅ 自动保留</td>
                  </tr>
                  <tr>
                    <td>健康监控</td>
                    <td data-label="pg_dump">❌ 无</td>
                    <td data-label="Databasus">✅ 内置健康检查</td>
                  </tr>
                  <tr>
                    <td>物理备份</td>
                    <td data-label="pg_dump">❌ 无</td>
                    <td data-label="Databasus">✅ 有</td>
                  </tr>
                  <tr>
                    <td>增量备份</td>
                    <td data-label="pg_dump">❌ 无</td>
                    <td data-label="Databasus">✅ 块级（PG 17+）</td>
                  </tr>
                  <tr>
                    <td>时间点恢复（PITR）</td>
                    <td data-label="pg_dump">❌ 无</td>
                    <td data-label="Databasus">✅ 有</td>
                  </tr>
                  <tr>
                    <td>远程备份</td>
                    <td data-label="pg_dump">✅ 有（命令行）</td>
                    <td data-label="Databasus">✅ 有</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="what-is-pgdump">pg_dump 是什么？</h2>

              <p>
                <code>pg_dump</code> 是 PostgreSQL
                自带的逻辑备份工具。它从一开始就是 PostgreSQL
                的一部分，是数据库导出的标准工具。
              </p>

              <h3 id="pgdump-strengths">pg_dump 的优势</h3>

              <ul>
                <li>
                  <strong>可移植的备份</strong>：生成 SQL 或自定义格式的
                  dump，可以恢复到不同版本的 PostgreSQL。
                </li>
                <li>
                  <strong>选择性备份</strong>：可以导出指定的表、schema
                  或整个数据库。
                </li>
                <li>
                  <strong>一致性快照</strong>：利用 PostgreSQL 的 MVCC
                  机制生成一致的备份，不会阻塞写入。
                </li>
                <li>
                  <strong>广泛支持</strong>：每个 PostgreSQL
                  安装都自带，文档完善，久经考验。
                </li>
                <li>
                  <strong>灵活的输出格式</strong>：纯 SQL、自定义、目录或 tar
                  格式。
                </li>
              </ul>

              <h3 id="pgdump-limitations">pg_dump 的局限</h3>

              <p>
                <code>pg_dump</code>{" "}
                固然强大，但在生产环境中使用通常还需要额外写脚本：
              </p>

              <ul>
                <li>
                  <strong>没有内置调度</strong>：需要 cron 任务或外部调度器。
                </li>
                <li>
                  <strong>只写本地存储</strong>
                  ：输出到本地文件系统，上传到云端需要额外脚本。
                </li>
                <li>
                  <strong>没有加密</strong>：备份文件默认不加密，需要通过 gpg
                  等工具管道处理。
                </li>
                <li>
                  <strong>没有通知</strong>
                  ：备份成功或失败时无法告警，除非自己写脚本。
                </li>
                <li>
                  <strong>没有保留管理</strong>
                  ：旧备份必须手动或用脚本清理。
                </li>
                <li>
                  <strong>只有命令行</strong>：没有可视化界面用于监控和管理。
                </li>
              </ul>

              <h2 id="how-databasus-extends">Databasus 如何扩展 pg_dump</h2>

              <p>
                Databasus 使用 <code>pg_dump</code>{" "}
                作为备份引擎，保留了逻辑备份的所有优点，并在此之上加入了企业级功能。
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">底层原理：</strong> 当你在
                  Databasus 中触发备份时，它会用优化过的参数执行{" "}
                  <code className="bg-[#374151] text-gray-200">pg_dump</code>
                  ，然后自动完成压缩、加密并上传到你配置的存储目标。
                </p>
              </div>

              <h3 id="web-interface">Web 界面</h3>

              <p>
                不用再记 <code>pg_dump</code> 的命令行参数，Databasus 提供 Web
                界面，你可以在其中：
              </p>

              <ul>
                <li>通过引导式连接向导添加数据库</li>
                <li>用可视化控件配置备份计划</li>
                <li>一眼看清备份历史和状态</li>
                <li>一键下载或恢复备份</li>
                <li>查看数据库健康与可用性图表</li>
              </ul>

              <h3 id="optimized-compression">优化的压缩</h3>

              <p>Databasus 默认使用 zstd 压缩（级别 5），它带来：</p>

              <ul>
                <li>
                  <strong>体积缩小 4-8 倍</strong>（相对未压缩的 dump）
                </li>
                <li>
                  <strong>约 20% 的运行时开销</strong>，远快于 gzip
                </li>
                <li>
                  <strong>自动处理</strong>，无需再通过压缩工具管道处理
                </li>
              </ul>

              <h2 id="beyond-pgdump">超越 pg_dump：物理备份与 PITR</h2>

              <p>
                Databasus 的逻辑备份基于 <code>pg_dump</code>
                ，但它还能做到 <code>pg_dump</code> 做不到的事：
              </p>

              <ul>
                <li>
                  <strong>物理备份</strong>：通过 <code>pg_basebackup</code>{" "}
                  对整个数据库集群做文件级复制。对大型数据库，备份和恢复都更快。
                </li>
                <li>
                  <strong>增量与 WAL 备份</strong>：通过{" "}
                  <code>pg_basebackup --incremental</code>{" "}
                  实现块级增量备份（由服务端 WAL 摘要驱动），再加上通过{" "}
                  <code>pg_receivewal</code> 的持续 WAL
                  流式传输，实现时间点恢复：可以恢复到两次备份之间的任意一秒。
                </li>
                <li>
                  <strong>灾难恢复</strong>：物理基础备份加持续 WAL
                  流式传输，为接近零数据丢失的需求而设计。
                </li>
              </ul>

              <p>
                这些备份构建在 PostgreSQL 17 的原生备份机制之上，Databasus
                复用的是 PostgreSQL
                自身久经考验的工具链，而不是重新造轮子。它们要求 PostgreSQL 17
                或更新的版本；在更早的版本上只能使用逻辑 <code>pg_dump</code>{" "}
                备份。所有操作都由 Databasus
                主机通过复制协议远程执行，数据库服务器上不需要安装任何软件。封闭网络可以通过
                SSH 隧道连接到内部主机或跳板机，数据库无需暴露到公网。{" "}
                <a
                  href="/zh/faq#pitr"
                  className="text-blue-400 hover:text-blue-600"
                >
                  了解物理备份和 PITR 备份的工作原理
                </a>
                。
              </p>

              <h2 id="backup-automation">备份自动化</h2>

              <p>
                使用 <code>pg_dump</code>{" "}
                时最常见的难题之一，就是搭建可靠的自动化备份。
              </p>

              <h3 id="automation-pgdump">传统的 pg_dump 自动化</h3>

              <p>
                一个典型的 <code>pg_dump</code> 自动化脚本大概是这样：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
                </pre>
              </div>

              <p>
                这个脚本需要维护、测试和监控。每个数据库还需要单独的 cron 条目。
              </p>

              <h3 id="automation-databasus">Databasus 的自动化</h3>

              <p>同样的功能在 Databasus 中是内置的：</p>

              <ul>
                <li>
                  <strong>可视化调度器</strong>
                  ：按小时、天、周、月或 cron 设置备份，并指定具体时间。
                </li>
                <li>
                  <strong>自动压缩</strong>：自动应用 zstd 压缩。
                </li>
                <li>
                  <strong>内置加密</strong>：AES-256-GCM
                  加密，每个备份使用唯一密钥。
                </li>
                <li>
                  <strong>云端上传</strong>：直接上传到 S3、Cloudflare R2、Azure 等目标。
                </li>
                <li>
                  <strong>保留策略</strong>
                  ：根据你的保留设置自动清理旧备份。
                </li>
                <li>
                  <strong>通知</strong>：成功或失败时向
                  Slack、Teams、Telegram、邮件发送告警。
                </li>
              </ul>

              <h2 id="storage-options">存储选项</h2>

              <p>
                <code>pg_dump</code>{" "}
                只写本地文件系统。要把备份放到云存储，需要额外的工具和脚本。
              </p>

              <h3 id="storage-databasus">Databasus 的存储目标</h3>

              <p>Databasus 开箱即用地支持多种存储目标：</p>

              <ul>
                <li>本地存储</li>
                <li>Amazon S3 及 S3 兼容服务</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS（网络附加存储）</li>
                <li>FTP</li>
              </ul>

              <p>每个数据库可以有自己的存储目标，也可以配置多个目标做冗余。</p>

              <p>
                <a
                  href="/zh/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看所有存储选项 →
                </a>
              </p>

              <h2 id="notifications">通知</h2>

              <p>及时知道备份成功还是失败，对数据保护至关重要。</p>

              <h3 id="notifications-pgdump">pg_dump 的通知</h3>

              <p>
                <code>pg_dump</code> 没有通知系统。你需要：
              </p>

              <ul>
                <li>写检查退出码的封装脚本</li>
                <li>接入外部监控工具</li>
                <li>搭建自定义告警流水线</li>
              </ul>

              <h3 id="notifications-databasus">Databasus 的通知</h3>

              <p>Databasus 内置了对以下渠道的通知：</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>邮件</li>
                <li>Webhook（用于自定义集成）</li>
              </ul>

              <p>
                你可以配置哪些事件触发通知：备份成功、备份失败，或两者都通知。
              </p>

              <p>
                <a
                  href="/zh/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看所有通知渠道 →
                </a>
              </p>

              <h2 id="team-features">团队功能</h2>

              <p>
                <code>pg_dump</code> 是单用户命令行工具。Databasus
                为团队加入了协作功能：
              </p>

              <h3 id="team-databasus">Databasus 的团队能力</h3>

              <ul>
                <li>
                  <strong>工作区</strong>
                  ：按项目或团队组织数据库、通知渠道和存储。用户只能看到自己被邀请加入的工作区。
                </li>
                <li>
                  <strong>基于角色的访问控制</strong>
                  ：为团队成员分配查看者、编辑者或管理员权限，控制每个人能做什么。
                </li>
                <li>
                  <strong>审计日志</strong>
                  ：跟踪系统里的所有操作和变更，是安全合规与责任追溯的基础。
                </li>
                <li>
                  <strong>共享通知</strong>
                  ：团队频道自动收到备份状态更新。
                </li>
              </ul>

              <p>
                <a
                  href="/zh/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  进一步了解访问管理 →
                </a>
              </p>

              <h2 id="security">安全</h2>

              <p>
                与直接使用 <code>pg_dump</code> 相比，安全是 Databasus
                提供显著附加价值的领域。
              </p>

              <h3 id="security-pgdump">pg_dump 的安全</h3>

              <p>
                <code>pg_dump</code> 生成的是未加密的备份文件。要保护它们需要：
              </p>

              <ul>
                <li>把输出通过加密工具（gpg、openssl）管道处理</li>
                <li>单独管理加密密钥</li>
                <li>确保密钥安全存放并定期轮换</li>
                <li>设置正确的文件权限</li>
              </ul>

              <h3 id="security-databasus">Databasus 的安全</h3>

              <p>Databasus 在多个层面实现安全：</p>

              <ul>
                <li>
                  <strong>AES-256-GCM 加密</strong>
                  ：所有密码、令牌和凭据都被加密，加密密钥与数据库分开存放。
                </li>
                <li>
                  <strong>每个备份独立加密</strong>
                  ：每个备份文件都用由主密钥、备份 ID
                  和随机盐派生出的唯一密钥加密。
                </li>
                <li>
                  <strong>只读数据库访问</strong>：强制只使用 SELECT
                  权限，即使被攻破也不会破坏数据。
                </li>
              </ul>

              <p>
                <a
                  href="/zh/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  进一步了解 Databasus 的安全机制 →
                </a>
              </p>

              <h2 id="restore-process">恢复流程</h2>

              <p>两个工具都支持恢复备份，但流程不同。</p>

              <h3 id="restore-pgdump">恢复 pg_dump 备份</h3>

              <p>
                恢复一个 <code>pg_dump</code> 备份需要：
              </p>

              <ol>
                <li>找到备份文件</li>
                <li>如果加密了，先解密</li>
                <li>如果压缩了，先解压</li>
                <li>
                  用正确的参数运行 <code>pg_restore</code> 或 <code>psql</code>
                </li>
              </ol>

              <h3 id="restore-databasus">恢复 Databasus 备份</h3>

              <p>Databasus 简化了恢复：</p>

              <ul>
                <li>
                  <strong>一键下载</strong>：直接在 Web 界面下载任意备份。
                </li>
                <li>
                  <strong>自动解密</strong>：下载时备份会自动解密。
                </li>
                <li>
                  <strong>提供恢复命令</strong>：Databasus
                  会为每个备份给出准确的 <code>pg_restore</code> 命令。
                </li>
                <li>
                  <strong>支持并行恢复</strong>：利用多个 CPU
                  核心，加快大型数据库的恢复。
                </li>
              </ul>

              <h2 id="installation">安装</h2>

              <h3 id="install-pgdump">安装 pg_dump</h3>

              <p>
                <code>pg_dump</code> 随 PostgreSQL 一起提供。只要安装了
                PostgreSQL，就有 <code>pg_dump</code>。
              </p>

              <h3 id="install-databasus">安装 Databasus</h3>

              <p>Databasus 提供多种安装方式：</p>

              <ul>
                <li>
                  <strong>一行脚本</strong>：安装 Docker（如果需要）、部署
                  Databasus 并配置自动启动。
                </li>
                <li>
                  <strong>Docker run</strong>：一条命令启动，附带内嵌
                  PostgreSQL。
                </li>
                <li>
                  <strong>Docker Compose</strong>：对部署有更多控制。
                </li>
              </ul>

              <p>
                <a
                  href="/zh/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看安装指南 →
                </a>
              </p>

              <h2 id="conclusion">结论</h2>

              <p>
                <code>pg_dump</code> 是 PostgreSQL 久经验证的备份工具，而
                Databasus 正是直接构建在它之上。直接使用 <code>pg_dump</code>{" "}
                还是通过 Databasus 使用，取决于你的需求。
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>直接使用 pg_dump，如果：</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>你只需要一次性或临时的数据库导出</li>
                  <li>你习惯编写并维护 shell 脚本</li>
                  <li>你已经有自动化基础设施（Ansible、Terraform 等）</li>
                  <li>你只需要本地备份，不需要云存储</li>
                  <li>你是单人开发者，需求简单</li>
                </ul>
              </div>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    使用 Databasus，如果：
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>你想要按计划自动备份，而不用写脚本</li>
                  <li>你需要把备份存到云端（S3、Azure Blob Storage 等）</li>
                  <li>你想要内置加密，而不用手动管理密钥</li>
                  <li>你需要在备份成功或失败时收到通知</li>
                  <li>你在团队中工作，需要协作功能</li>
                  <li>你更喜欢可视化界面而不是命令行工具</li>
                  <li>你想要自动的保留策略和清理</li>
                  <li>你需要物理备份、增量备份或时间点恢复来做灾难恢复</li>
                </ul>
              </div>

              <p>
                Databasus 的逻辑备份基于 <code>pg_dump</code>
                ，并在其之上扩展了自动化、安全和团队功能。除此之外，Databasus
                还支持物理备份、基于 WAL 归档的增量备份和时间点恢复，而这些是{" "}
                <code>pg_dump</code> 根本无法提供的能力。
              </p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
