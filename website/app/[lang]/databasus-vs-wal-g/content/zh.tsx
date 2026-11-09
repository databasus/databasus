import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs WAL-G：PostgreSQL 备份工具对比",
  description:
    "对比 Databasus 和 WAL-G 两款 PostgreSQL 备份工具：备份方式、多数据库支持、易用性、团队功能上的差异，以及各自适用的场景。",
  keywords: [
    "Databasus vs WAL-G",
    "PostgreSQL 备份对比",
    "WAL-G 替代方案",
    "PostgreSQL 备份工具",
    "数据库备份对比",
    "pg_dump 与 WAL 归档",
    "自托管备份",
    "PostgreSQL PITR",
    "WAL 归档",
    "多数据库备份",
  ],
  openGraph: {
    title: "Databasus vs WAL-G：PostgreSQL 备份工具对比",
    description:
      "对比 Databasus 和 WAL-G 两款 PostgreSQL 备份工具：备份方式、多数据库支持、易用性、团队功能上的差异，以及各自适用的场景。",
    type: "article",
    url: getLocalizedUrl("zh", "databasus-vs-wal-g"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs WAL-G：PostgreSQL 备份工具对比",
    description:
      "对比 Databasus 和 WAL-G 两款 PostgreSQL 备份工具：备份方式、多数据库支持、易用性、团队功能上的差异，以及各自适用的场景。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "databasus-vs-wal-g"),
    languages: getLanguageAlternates("databasus-vs-wal-g"),
  },
  robots: "index, follow",
};

export default function DatabasusVsWalGPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Databasus vs WAL-G：PostgreSQL 备份工具对比",
            description:
              "全面对比 Databasus 和 WAL-G 两款 PostgreSQL 备份工具，涵盖备份方式、多数据库支持、易用性、团队功能，以及各自适用的场景。",
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
              <h1 id="databasus-vs-wal-g">Databasus vs WAL-G</h1>

              <p className="text-lg text-gray-400">
                Databasus 和 WAL-G 都以最小化 RTO 和 RPO
                的灾难恢复为目标，都支持 PostgreSQL 物理备份、WAL
                归档和按时间点恢复（PITR）。Databasus 基于 PostgreSQL 17
                的原生工具栈远程执行这些备份，复用 PostgreSQL
                自带的、久经考验的工具而不是重新造轮子，这一切都在直观的 Web
                界面里管理。它适用于任何规模和复杂度的数据库。物理备份要求
                PostgreSQL 17 及以上版本；在更早的版本上只能使用逻辑{" "}
                <code>pg_dump</code> 备份。WAL-G
                是一个命令行工具，自带备份引擎，因此能在旧得多的 PostgreSQL
                版本上做物理备份，使用自定义流式协议获得略优的性能，支持增量
                delta 备份（只备份变化的页），并覆盖更多数据库引擎，包括 MS
                SQL、FoundationDB 和 Greenplum。
              </p>

              <h2 id="quick-comparison">快速对比</h2>

              <p>下面是 Databasus 与 WAL-G 主要差异的速览：</p>

              <table>
                <thead>
                  <tr>
                    <th>功能</th>
                    <th>Databasus</th>
                    <th>WAL-G</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>备份管理</td>
                    <td data-label="Databasus">✅ 支持（多个数据库）</td>
                    <td data-label="WAL-G">❌ 不支持（仅单个数据库）</td>
                  </tr>
                  <tr>
                    <td>其他数据库支持</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL、MySQL、MariaDB、MongoDB
                    </td>
                    <td data-label="WAL-G">✅ PostgreSQL、MySQL、MS SQL</td>
                  </tr>
                  <tr>
                    <td>界面</td>
                    <td data-label="Databasus">Web 界面</td>
                    <td data-label="WAL-G">仅命令行</td>
                  </tr>
                  <tr>
                    <td>备份类型</td>
                    <td data-label="Databasus">逻辑 + 物理</td>
                    <td data-label="WAL-G">物理（WAL 归档）</td>
                  </tr>
                  <tr>
                    <td>物理备份所需 PostgreSQL 版本</td>
                    <td data-label="Databasus">17+（原生）</td>
                    <td data-label="WAL-G">9.x+（自带引擎）</td>
                  </tr>
                  <tr>
                    <td>备份调度</td>
                    <td data-label="Databasus">✅ 内置调度器</td>
                    <td data-label="WAL-G">需外部工具（cron）</td>
                  </tr>
                  <tr>
                    <td>恢复选项</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="WAL-G">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>增量备份</td>
                    <td data-label="Databasus">✅ 块级（PG 17+）</td>
                    <td data-label="WAL-G">Delta 备份（只备份变化的页）</td>
                  </tr>
                  <tr>
                    <td>远程备份</td>
                    <td data-label="Databasus">✅ 支持</td>
                    <td data-label="WAL-G">❌ 不支持（本地运行）</td>
                  </tr>
                  <tr>
                    <td>团队功能</td>
                    <td data-label="Databasus">✅ 工作区、RBAC、审计日志</td>
                    <td data-label="WAL-G">❌ 仅操作系统级权限</td>
                  </tr>
                  <tr>
                    <td>通知</td>
                    <td data-label="Databasus">
                      ✅ Slack、Teams、Telegram、邮件
                    </td>
                    <td data-label="WAL-G">❌ 需要自写脚本</td>
                  </tr>
                  <tr>
                    <td>加密</td>
                    <td data-label="Databasus">内置 AES-256-GCM</td>
                    <td data-label="WAL-G">GPG 或 libsodium</td>
                  </tr>
                  <tr>
                    <td>学习曲线</td>
                    <td data-label="Databasus">很低</td>
                    <td data-label="WAL-G">要求熟练使用命令行</td>
                  </tr>
                  <tr>
                    <td>安装</td>
                    <td data-label="Databasus">一行脚本或 Docker</td>
                    <td data-label="WAL-G">下载二进制文件 + 配置</td>
                  </tr>
                  <tr>
                    <td>适合自托管数据库</td>
                    <td data-label="Databasus">✅ 适合</td>
                    <td data-label="WAL-G">✅ 适合</td>
                  </tr>
                  <tr>
                    <td>适合云数据库</td>
                    <td data-label="Databasus">
                      ✅ 适合（RDS、Cloud SQL、Azure）
                    </td>
                    <td data-label="WAL-G">❌ 只能备份（无法恢复到云端）</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="database-focus">数据库定位</h2>

              <p>两款工具最显著的差异之一是覆盖的数据库范围：</p>

              <h3 id="focus-databasus">Databasus：全面的备份管理</h3>

              <p>
                Databasus 面向多数据库系统的全面备份管理，并以易用性为核心：
              </p>

              <ul>
                <li>
                  <strong>多数据库支持</strong>：在同一个界面里管理
                  PostgreSQL、MySQL、MariaDB 和 MongoDB 的数据库备份。
                </li>
                <li>
                  <strong>统一体验</strong>
                  ：界面、工作流和功能在所有受支持的数据库上保持一致。
                </li>
                <li>
                  <strong>版本支持</strong>：支持 PostgreSQL 12 到 18
                  的各个版本，并针对不同版本做了优化。
                </li>
                <li>
                  <strong>专注管理体验</strong>
                  ：全部开发精力都投入到改进备份管理体验上。
                </li>
              </ul>

              <h3 id="focus-wal-g">WAL-G：多数据库支持</h3>

              <p>
                WAL-G 最初是 PostgreSQL 备份工具，后来扩展到支持多种数据库系统：
              </p>

              <ul>
                <li>
                  <strong>PostgreSQL</strong>：最早、最成熟的实现。
                </li>
                <li>
                  <strong>MySQL/MariaDB</strong>：支持基于 binlog 的备份。
                </li>
                <li>
                  <strong>MS SQL Server</strong>：Windows 上的 SQL Server 备份。
                </li>
                <li>
                  <strong>MongoDB</strong>：文档数据库备份支持。
                </li>
                <li>
                  <strong>FoundationDB</strong>：分布式数据库支持。
                </li>
                <li>
                  <strong>Greenplum</strong>：数据仓库备份支持。
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">
                    当全面管理很重要时：
                  </strong>{" "}
                  如果你需要通过统一界面管理多个数据库的备份，Databasus
                  提供了更顺畅的体验。你可以集中管理所有备份，还能使用团队功能，不必为不同数据库切换不同工具。
                </p>
              </div>

              <h2 id="target-audience">目标用户</h2>

              <p>基于各自的设计理念，两款工具服务于不同的用户群体：</p>

              <h3 id="audience-databasus">Databasus 的用户</h3>

              <p>Databasus 面向广泛的用户群体，从个人开发者到大型企业：</p>

              <ul>
                <li>
                  <strong>个人开发者</strong>
                  ：安装简单、界面直观，无需深入了解 PostgreSQL
                  也能保护个人项目。
                </li>
                <li>
                  <strong>开发团队</strong>
                  ：工作区、基于角色的访问控制和审计日志让团队成员安全协作。
                </li>
                <li>
                  <strong>企业</strong>
                  ：完善的安全机制、多种存储目标和通知渠道，可满足企业级需求。
                </li>
                <li>
                  <strong>多数据库环境</strong>：同时运行
                  PostgreSQL、MySQL、MariaDB 或 MongoDB
                  的组织能从集中式备份管理中受益。
                </li>
                <li>
                  <strong>DBA 和灾难恢复</strong>：物理备份、WAL 归档和
                  PITR，为要求数据近零丢失的关键系统服务。
                </li>
                <li>
                  <strong>DevOps 工程师</strong>：Agent
                  模式可融入现有基础设施，Web 界面和 API
                  提供可见性和控制力，无需自写脚本。
                </li>
              </ul>

              <h3 id="audience-wal-g">WAL-G 的用户</h3>

              <p>WAL-G 面向习惯命令行工具的用户：</p>

              <ul>
                <li>
                  <strong>DevOps 工程师</strong>：偏好基础设施即代码和基于 CLI
                  的工作流的人。
                </li>
                <li>
                  <strong>多数据库环境</strong>：同时运行 PostgreSQL 和
                  MySQL、MongoDB 或其他受支持数据库的组织。
                </li>
                <li>
                  <strong>云原生部署</strong>：使用 Kubernetes 或容器化环境、CLI
                  工具容易集成的团队。
                </li>
                <li>
                  <strong>扩展数据库支持</strong>：除了 PostgreSQL 还需要备份 MS
                  SQL、FoundationDB 或 Greenplum 的团队。
                </li>
              </ul>

              <h2 id="backup-approach">备份方式</h2>

              <p>两款工具采用截然不同的备份策略，各有优势：</p>

              <h3 id="backup-databasus">Databasus：逻辑 + 物理备份</h3>

              <p>Databasus 同时支持逻辑和物理两种备份策略：</p>

              <ul>
                <li>
                  <strong>物理备份、增量备份和 WAL 备份</strong>：通过
                  PostgreSQL 复制协议远程执行，基于 PostgreSQL 17 的原生工具栈：
                  <code>pg_basebackup</code>、由服务端 WAL 摘要驱动的块级{" "}
                  <code>pg_basebackup --incremental</code>、
                  <code>pg_receivewal</code> 和 <code>pg_combinebackup</code>
                  。Databasus 复用 PostgreSQL
                  自带的、久经考验的工具，而不是重新造轮子。要求 PostgreSQL 17
                  及以上版本。
                </li>
                <li>
                  <strong>逻辑备份</strong>：使用 <code>pg_dump</code>{" "}
                  生成可移植的备份，能恢复到不同的 PostgreSQL 版本。这也是
                  PostgreSQL 17 以下版本唯一可用的备份类型，以及 MySQL、MariaDB
                  和 MongoDB 的备份途径。
                </li>
                <li>
                  <strong>数据库上无需安装任何东西</strong>
                  ：备份通过远程连接完成；封闭网络可以通过 SSH
                  隧道连接到内部主机或跳板机，数据库无需暴露到公网。
                </li>
                <li>
                  <strong>高效压缩</strong>：两种备份类型都使用 zstd（级别
                  5），可将体积缩小 4-8 倍。
                </li>
                <li>
                  <strong>只读访问</strong>：逻辑备份只需要 SELECT
                  权限，最大限度降低安全风险。
                </li>
              </ul>

              <h3 id="backup-wal-g">WAL-G：物理备份加 WAL 归档</h3>

              <p>WAL-G 执行文件级（物理）备份并持续归档 WAL：</p>

              <ul>
                <li>
                  <strong>基础备份</strong>：对 PostgreSQL
                  数据目录做完整的文件级复制。
                </li>
                <li>
                  <strong>Delta 备份</strong>
                  ：只备份发生变化的页，减少存储占用和传输时间。
                </li>
                <li>
                  <strong>WAL 归档</strong>
                  ：持续归档预写日志（WAL），实现按时间点恢复。
                </li>
                <li>
                  <strong>写时复制优化</strong>：高效处理未变化的数据块。
                </li>
              </ul>

              <h2 id="recovery-options">恢复选项</h2>

              <p>两款工具都提供恢复能力，但粒度不同：</p>

              <h3 id="recovery-databasus">Databasus 的恢复</h3>

              <ul>
                <li>
                  <strong>按时间点恢复</strong>：通过 WAL
                  回放恢复到任意指定的一秒。
                </li>
                <li>
                  <strong>整集群恢复</strong>
                  ：从物理备份将整个数据库集群恢复到指定时间点。
                </li>
                <li>
                  <strong>逻辑恢复</strong>
                  ：从定时逻辑备份恢复到任意一个备份点。
                </li>
                <li>
                  <strong>一键恢复</strong>：直接在 Web 界面下载并恢复逻辑备份。
                </li>
                <li>
                  <strong>跨版本兼容</strong>：逻辑备份可以恢复到不同的
                  PostgreSQL 版本。
                </li>
              </ul>

              <h3 id="recovery-wal-g">WAL-G 的恢复</h3>

              <ul>
                <li>
                  <strong>按时间点恢复（PITR）</strong>：通过 WAL
                  回放恢复到任意指定的一秒，最大限度减少数据丢失。
                </li>
                <li>
                  <strong>整集群恢复</strong>
                  ：将整个数据库集群恢复到指定时间点。
                </li>
                <li>
                  <strong>Delta 恢复</strong>
                  ：只获取变化的页，恢复速度更快。
                </li>
                <li>
                  <strong>创建备用节点</strong>：从备份创建 PostgreSQL
                  副本，用于高可用架构。
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">注意：</strong>{" "}
                  两款工具都支持 PITR。WAL-G 额外提供 delta
                  恢复（只获取变化的页），并使用自定义流式协议，在大规模场景下性能略优。{" "}
                  <a
                    href="/zh/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    了解 Databasus 如何支持 PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">易用性</h2>

              <p>两款工具在用户体验上的取向差别很大：</p>

              <h3 id="ease-databasus">Databasus 的使用体验</h3>

              <ul>
                <li>
                  <strong>Web 界面</strong>
                  ：所有备份设置都可以点选配置，不需要命令行。
                </li>
                <li>
                  <strong>2 分钟安装</strong>：一行 cURL 脚本或一条简单的 Docker
                  命令，立即上手。
                </li>
                <li>
                  <strong>可视化监控</strong>
                  ：仪表盘一目了然地展示备份状态、健康检查和历史记录。
                </li>
                <li>
                  <strong>内置通知</strong>：直接在界面里配置
                  Slack、Teams、Telegram、邮件或 webhook 告警。
                </li>
                <li>
                  <strong>无需 PostgreSQL 专业知识</strong>
                  ：为想获得可靠备份、又不想成为数据库专家的开发者而设计。
                </li>
              </ul>

              <h3 id="ease-wal-g">WAL-G 的使用体验</h3>

              <ul>
                <li>
                  <strong>命令行界面</strong>：所有操作都通过终端命令完成，如{" "}
                  <code>wal-g backup-push</code>、
                  <code>wal-g backup-fetch</code>。
                </li>
                <li>
                  <strong>环境变量</strong>
                  ：配置主要通过环境变量而不是配置文件。
                </li>
                <li>
                  <strong>外部调度</strong>：自动备份需要 cron 任务或外部编排。
                </li>
                <li>
                  <strong>WAL 归档配置</strong>：必须配置 PostgreSQL 的{" "}
                  <code>archive_command</code> 才能与 WAL-G 集成。
                </li>
                <li>
                  <strong>要求熟练使用 CLI</strong>
                  ：文档默认读者熟悉命令行工具和 shell 脚本。
                </li>
              </ul>

              <p>
                <a
                  href="/zh/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看 Databasus 安装指南 →
                </a>
              </p>

              <h2 id="team-features">团队功能</h2>

              <p>对于由多名成员共同管理备份的组织：</p>

              <h3 id="team-databasus">Databasus 的团队能力</h3>

              <ul>
                <li>
                  <strong>工作区</strong>
                  ：按项目或团队组织数据库、通知渠道和存储。用户只能看到被邀请加入的工作区。
                </li>
                <li>
                  <strong>基于角色的访问控制</strong>
                  ：分配查看者、编辑者或管理员权限，控制每位成员能做什么。
                </li>
                <li>
                  <strong>审计日志</strong>
                  ：记录系统中的所有操作和变更，是安全合规与责任追溯的基础。
                </li>
                <li>
                  <strong>共享通知</strong>
                  ：团队频道自动收到备份状态更新。
                </li>
              </ul>

              <h3 id="team-wal-g">WAL-G 的团队能力</h3>

              <p>WAL-G 是命令行工具，没有内置团队功能：</p>

              <ul>
                <li>没有用户管理或访问控制</li>
                <li>没有操作审计日志</li>
                <li>团队协作需要借助外部工具和流程</li>
                <li>访问通过操作系统级权限和云 IAM 策略控制</li>
              </ul>

              <p>
                <a
                  href="/zh/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  进一步了解 Databasus 的访问管理 →
                </a>
              </p>

              <h2 id="security">安全性</h2>

              <p>两款工具都提供安全功能，但思路不同：</p>

              <h3 id="security-databasus">Databasus 的安全性</h3>

              <ul>
                <li>
                  <strong>AES-256-GCM 加密</strong>
                  ：所有密码、令牌和凭据都加密存储，加密密钥与数据库分开保存。
                </li>
                <li>
                  <strong>每个备份独立加密</strong>
                  ：每个备份文件用主密钥、备份 ID 和随机盐派生的唯一密钥加密。
                </li>
                <li>
                  <strong>只读数据库访问</strong>：只需 SELECT
                  权限，即使被攻破也不会破坏数据。
                </li>
              </ul>

              <h3 id="security-wal-g">WAL-G 的安全性</h3>

              <ul>
                <li>
                  <strong>GPG 加密</strong>：支持对备份文件做基于 GPG 的加密。
                </li>
                <li>
                  <strong>libsodium 加密</strong>：另一种基于 libsodium
                  库的加密方式。
                </li>
                <li>
                  <strong>云 IAM 集成</strong>：利用云厂商的 IAM
                  控制对存储的访问。
                </li>
                <li>
                  <strong>没有内置凭据管理</strong>
                  ：依赖环境变量或外部密钥管理。
                </li>
              </ul>

              <p>
                <a
                  href="/zh/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  进一步了解 Databasus 的安全性 →
                </a>
              </p>

              <h2 id="storage-options">存储选项</h2>

              <p>两款工具都支持云存储，但侧重点不同：</p>

              <h3 id="storage-databasus">Databasus 的存储</h3>

              <p>面向各种使用场景的友好选项：</p>

              <ul>
                <li>本地存储</li>
                <li>Amazon S3 及 S3 兼容服务</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS（网络附加存储）</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-wal-g">WAL-G 的存储</h3>

              <p>云原生存储选项：</p>

              <ul>
                <li>Amazon S3</li>
                <li>Google Cloud Storage（GCS）</li>
                <li>Azure Blob Storage</li>
                <li>Swift（OpenStack）</li>
                <li>本地文件系统</li>
                <li>SSH/SFTP</li>
              </ul>

              <p>
                <a
                  href="/zh/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看 Databasus 的所有存储选项 →
                </a>
              </p>

              <h2 id="notifications">通知</h2>

              <p>随时掌握备份状态：</p>

              <h3 id="notifications-databasus">Databasus 的通知</h3>

              <p>内置多种通知渠道：</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>邮件</li>
                <li>Webhook</li>
              </ul>

              <h3 id="notifications-wal-g">WAL-G 的通知</h3>

              <p>WAL-G 没有内置通知支持，实现通知需要：</p>

              <ul>
                <li>围绕备份命令自写脚本</li>
                <li>集成外部监控工具</li>
                <li>手动解析日志并配置告警</li>
                <li>接入 Prometheus、Grafana 等工具或自研方案</li>
              </ul>

              <p>
                <a
                  href="/zh/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看 Databasus 的所有通知渠道 →
                </a>
              </p>

              <h2 id="compression">压缩</h2>

              <p>两款工具都提供压缩来缩小备份体积：</p>

              <h3 id="compression-databasus">Databasus 的压缩</h3>

              <ul>
                <li>
                  <strong>zstd 压缩</strong>：使用 zstd 级别
                  5，在速度和压缩率之间取得平衡。
                </li>
                <li>
                  <strong>体积缩小 4-8 倍</strong>
                  ：典型压缩率下运行时开销只增加约 20%。
                </li>
                <li>
                  <strong>自动启用</strong>：压缩默认开启，无需任何配置。
                </li>
              </ul>

              <h3 id="compression-wal-g">WAL-G 的压缩</h3>

              <ul>
                <li>
                  <strong>多种算法</strong>：支持 LZ4、LZMA、Brotli 和 zstd。
                </li>
                <li>
                  <strong>级别可调</strong>
                  ：可精细权衡压缩率与速度。
                </li>
                <li>
                  <strong>按文件类型压缩</strong>：WAL
                  文件和基础备份可以使用不同的压缩设置。
                </li>
              </ul>

              <h2 id="conclusion">结论</h2>

              <p>
                在 PostgreSQL 备份生态中，Databasus 和 WAL-G
                满足的是不同的需求。如何选择取决于你的数据库环境、团队结构和运维偏好。
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    选择 Databasus，如果：
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>你需要在同一个界面里全面管理 PostgreSQL 备份</li>
                  <li>相比命令行工具，你更喜欢 Web 界面</li>
                  <li>你需要团队协作功能（工作区、RBAC、审计日志）</li>
                  <li>你想要内置的 Slack、Teams、Telegram 等通知</li>
                  <li>你想要内置调度，不必额外配置 cron</li>
                  <li>
                    你想在一个带调度、通知和团队功能的仪表盘里管理多个数据库的备份
                  </li>
                  <li>你想快速上手，不需要太多数据库知识</li>
                  <li>内置备份加密对你很重要</li>
                  <li>
                    你使用云托管数据库（AWS RDS、Google Cloud
                    SQL、Azure）或自托管数据库
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>选择 WAL-G，如果：</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    你需要在 PostgreSQL 17 以下版本上做物理或增量备份（WAL-G
                    自带备份引擎）
                  </li>
                  <li>
                    你需要 delta 备份（只备份变化的页）来减少存储占用和传输时间
                  </li>
                  <li>你需要支持 MS SQL、FoundationDB 或 Greenplum</li>
                  <li>你偏好命令行工具和基础设施即代码的工作流</li>
                  <li>
                    你想要多种压缩算法（LZ4、LZMA、Brotli、zstd）并做精细调优
                  </li>
                  <li>你的团队有管理 CLI 工具的 DevOps 经验</li>
                </ul>
              </div>

              <p>
                两款工具都支持物理备份、WAL 归档和 PITR，都以最小化 RTO 和 RPO
                的灾难恢复为目标。Databasus
                适用于任何规模和复杂度的数据库，并提供 Web
                界面、团队功能，以及覆盖自托管和云托管数据库的逻辑与物理备份。
                <br />
                <br />
                对于偏好 CLI 工作流的团队，WAL-G
                仍然是出色的选择，它有独特的优势：delta
                备份（只备份变化的页）、性能略优的自定义流式协议，以及对
                PostgreSQL 之外更多数据库引擎的支持。
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
