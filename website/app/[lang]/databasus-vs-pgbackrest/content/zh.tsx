import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs pgBackRest：PostgreSQL 备份工具对比",
  description:
    "对比 Databasus 和 pgBackRest 两款 PostgreSQL 备份工具：备份方式、目标用户、易用性、恢复选项上的差异，以及各自适用的场景。",
  keywords: [
    "Databasus vs pgBackRest",
    "PostgreSQL 备份对比",
    "pgBackRest 替代方案",
    "PostgreSQL 备份工具",
    "数据库备份对比",
    "pg_dump 与物理备份",
    "自托管备份",
    "PostgreSQL PITR",
    "大型数据库备份",
    "DBA 备份工具",
  ],
  openGraph: {
    title: "Databasus vs pgBackRest：PostgreSQL 备份工具对比",
    description:
      "对比 Databasus 和 pgBackRest 两款 PostgreSQL 备份工具：备份方式、目标用户、易用性、恢复选项上的差异，以及各自适用的场景。",
    type: "article",
    url: getLocalizedUrl("zh", "databasus-vs-pgbackrest"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs pgBackRest：PostgreSQL 备份工具对比",
    description:
      "对比 Databasus 和 pgBackRest 两款 PostgreSQL 备份工具：备份方式、目标用户、易用性、恢复选项上的差异，以及各自适用的场景。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "databasus-vs-pgbackrest"),
    languages: getLanguageAlternates("databasus-vs-pgbackrest"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackRestPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Databasus vs pgBackRest：PostgreSQL 备份工具对比",
            description:
              "全面对比 Databasus 和 pgBackRest 两款 PostgreSQL 备份工具，涵盖备份方式、目标用户、易用性、恢复选项，以及各自适用的场景。",
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
              <h1 id="databasus-vs-pgbackrest">Databasus vs pgBackRest</h1>

              <p className="text-lg text-gray-400">
                Databasus 和 pgBackRest 都以最小化 RTO 和 RPO
                的灾难恢复为目标，都支持物理备份、WAL
                归档和按时间点恢复（PITR）。Databasus 基于 PostgreSQL 17
                的原生工具栈远程执行这些备份，复用 PostgreSQL
                自带的、久经考验的工具而不是重新造轮子，这一切都在直观的 Web
                界面里管理。它适用于任何规模和复杂度的数据库。物理备份要求
                PostgreSQL 17 及以上版本；在更早的版本上只能使用逻辑{" "}
                <code>pg_dump</code> 备份。pgBackRest
                自带备份引擎，因此也能在旧得多的 PostgreSQL
                版本上做物理和增量备份，还提供差异备份、delta 恢复等高级功能。
              </p>

              <h2 id="quick-comparison">快速对比</h2>

              <p>下面是 Databasus 与 pgBackRest 主要差异的速览：</p>

              <table>
                <thead>
                  <tr>
                    <th>功能</th>
                    <th>Databasus</th>
                    <th>pgBackRest</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>目标用户</td>
                    <td data-label="Databasus">个人、团队、DBA、企业</td>
                    <td data-label="pgBackRest">DBA、命令行和 IaC 工作流</td>
                  </tr>
                  <tr>
                    <td>备份管理</td>
                    <td data-label="Databasus">✅ 多个数据库</td>
                    <td data-label="pgBackRest">❌ 仅单个数据库</td>
                  </tr>
                  <tr>
                    <td>其他数据库支持</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL、MySQL、MariaDB、MongoDB
                    </td>
                    <td data-label="pgBackRest">❌ 仅 PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>界面</td>
                    <td data-label="Databasus">Web 界面</td>
                    <td data-label="pgBackRest">命令行、配置文件</td>
                  </tr>
                  <tr>
                    <td>备份类型</td>
                    <td data-label="Databasus">逻辑 + 物理</td>
                    <td data-label="pgBackRest">物理（文件级）</td>
                  </tr>
                  <tr>
                    <td>物理备份所需 PostgreSQL 版本</td>
                    <td data-label="Databasus">17+（原生）</td>
                    <td data-label="pgBackRest">9.4+（自带引擎）</td>
                  </tr>
                  <tr>
                    <td>恢复选项</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="pgBackRest">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>并行操作</td>
                    <td data-label="Databasus">✅ 支持</td>
                    <td data-label="pgBackRest">✅ 支持</td>
                  </tr>
                  <tr>
                    <td>增量备份</td>
                    <td data-label="Databasus">✅ 块级（PG 17+）</td>
                    <td data-label="pgBackRest">块级增量</td>
                  </tr>
                  <tr>
                    <td>差异备份</td>
                    <td data-label="Databasus">❌ 不支持</td>
                    <td data-label="pgBackRest">✅ 支持</td>
                  </tr>
                  <tr>
                    <td>Delta 恢复</td>
                    <td data-label="Databasus">❌ 不支持</td>
                    <td data-label="pgBackRest">✅ 支持（只恢复变化的文件）</td>
                  </tr>
                  <tr>
                    <td>远程备份</td>
                    <td data-label="Databasus">✅ 支持</td>
                    <td data-label="pgBackRest">
                      ❌ 不支持（需要文件系统访问）
                    </td>
                  </tr>
                  <tr>
                    <td>团队功能</td>
                    <td data-label="Databasus">✅ 工作区、RBAC、审计日志</td>
                    <td data-label="pgBackRest">❌ 单用户</td>
                  </tr>
                  <tr>
                    <td>学习曲线</td>
                    <td data-label="Databasus">很低</td>
                    <td data-label="pgBackRest">要求 DBA 专业知识</td>
                  </tr>
                  <tr>
                    <td>安装</td>
                    <td data-label="Databasus">一行脚本或 Docker</td>
                    <td data-label="pgBackRest">需要手动配置</td>
                  </tr>
                  <tr>
                    <td>适合自托管数据库</td>
                    <td data-label="Databasus">✅ 适合</td>
                    <td data-label="pgBackRest">✅ 适合</td>
                  </tr>
                  <tr>
                    <td>适合云数据库</td>
                    <td data-label="Databasus">
                      ✅ 适合（RDS、Cloud SQL、Azure）
                    </td>
                    <td data-label="pgBackRest">
                      ❌ 不适合（需要文件系统访问）
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="target-audience">目标用户</h2>

              <p>两款工具最重要的差异在于它们为谁而设计：</p>

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
                  <strong>DBA 和灾难恢复</strong>：物理备份、WAL 归档和
                  PITR，为要求数据近零丢失的关键系统服务。
                </li>
              </ul>

              <h3 id="audience-pgbackrest">pgBackRest 的用户</h3>

              <p>pgBackRest 是命令行工具，面向想自己运维备份引擎的团队：</p>

              <ul>
                <li>
                  <strong>命令行和 IaC 工作流</strong>：偏好通过文件和脚本而不是
                  Web 界面配置备份的团队。
                </li>
                <li>
                  <strong>较老的 PostgreSQL 版本</strong>：pgBackRest
                  自带引擎，能在 PostgreSQL 17 以下的版本上做物理和增量备份。
                </li>
                <li>
                  <strong>大规模场景的高级功能</strong>：当差异备份、delta
                  恢复和创建备用节点值得付出额外配置成本时。
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
                  生成可移植的备份，能恢复到不同的 PostgreSQL
                  版本，还可以选择性恢复指定的表或 schema。这也是 PostgreSQL 17
                  以下版本唯一可用的备份类型，以及 MySQL、MariaDB 和 MongoDB
                  的备份途径。
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

              <h3 id="backup-pgbackrest">pgBackRest：物理备份</h3>

              <p>pgBackRest 对 PostgreSQL 数据目录执行文件级（物理）备份：</p>

              <ul>
                <li>
                  <strong>块级增量</strong>
                  ：只备份发生变化的数据块，缩短超大数据库的备份时间、减少存储占用。
                </li>
                <li>
                  <strong>WAL 归档</strong>
                  ：持续归档预写日志（WAL），实现精确的按时间点恢复。
                </li>
                <li>
                  <strong>全量、差异、增量</strong>
                  ：多种备份策略，应对不同的恢复场景。
                </li>
                <li>
                  <strong>为大规模而优化</strong>
                  ：专为逻辑备份耗时过长的数据库设计。
                </li>
              </ul>

              <h2 id="recovery-options">恢复选项</h2>

              <p>两款工具都提供灵活的恢复选项，但粒度不同：</p>

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

              <h3 id="recovery-pgbackrest">pgBackRest 的恢复</h3>

              <ul>
                <li>
                  <strong>按时间点恢复（PITR）</strong>：通过 WAL
                  回放恢复到任意指定的一秒。
                </li>
                <li>
                  <strong>整集群恢复</strong>
                  ：从物理备份将整个数据库集群恢复到指定时间点。
                </li>
                <li>
                  <strong>并行恢复</strong>
                  ：多线程恢复，加快大型数据库的恢复速度。
                </li>
                <li>
                  <strong>Delta 恢复</strong>
                  ：只恢复变化的文件，缩短恢复时间。
                </li>
                <li>
                  <strong>创建备用节点</strong>：从备份创建 PostgreSQL 副本。
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">注意：</strong>{" "}
                  两款工具都支持 PITR。pgBackRest 额外提供 delta
                  恢复（只获取变化的文件）、差异备份，以及从备份创建备用节点。{" "}
                  <a
                    href="/zh/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    了解 Databasus 如何支持 PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">易用性</h2>

              <p>两款工具在用户体验上的取向差别极大：</p>

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

              <h3 id="ease-pgbackrest">pgBackRest 的使用体验</h3>

              <ul>
                <li>
                  <strong>命令行界面</strong>
                  ：所有操作都通过终端命令完成。
                </li>
                <li>
                  <strong>配置文件</strong>：需要手动编辑 INI 风格的配置文件。
                </li>
                <li>
                  <strong>WAL 归档配置</strong>：必须配置 PostgreSQL 的{" "}
                  <code>archive_command</code> 及相关设置。
                </li>
                <li>
                  <strong>学习曲线陡峭</strong>：需要理解 PostgreSQL
                  内部机制、WAL 原理和各种备份策略。
                </li>
                <li>
                  <strong>要求 DBA 专业知识</strong>
                  ：文档默认读者熟悉数据库管理概念。
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

              <h3 id="team-pgbackrest">pgBackRest 的团队能力</h3>

              <p>pgBackRest 是命令行工具，没有内置团队功能：</p>

              <ul>
                <li>没有用户管理或访问控制</li>
                <li>没有操作审计日志</li>
                <li>团队协作需要借助外部工具和流程</li>
                <li>访问通过配置文件的操作系统级权限控制</li>
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

              <p>两款工具都提供扎实的安全功能：</p>

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

              <h3 id="security-pgbackrest">pgBackRest 的安全性</h3>

              <ul>
                <li>
                  <strong>仓库加密</strong>：备份仓库可以使用 AES-256 加密。
                </li>
                <li>
                  <strong>TLS/SSH 传输</strong>
                  ：远程操作通过安全通道通信。
                </li>
                <li>
                  <strong>校验和验证</strong>
                  ：在创建和恢复时验证备份完整性。
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

              <p>
                两款工具都支持多种存储目标，Databasus
                提供了更多面向普通用户的选项：
              </p>

              <h3 id="storage-databasus">Databasus 的存储</h3>

              <ul>
                <li>本地存储</li>
                <li>Amazon S3 及 S3 兼容服务</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS（网络附加存储）</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-pgbackrest">pgBackRest 的存储</h3>

              <ul>
                <li>本地存储（POSIX、CIFS）</li>
                <li>Amazon S3 及 S3 兼容服务</li>
                <li>Cloudflare R2（S3 兼容）</li>
                <li>Azure Blob Storage</li>
                <li>NAS（网络附加存储）</li>
                <li>Google Cloud Storage</li>
                <li>SFTP</li>
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

              <h3 id="notifications-pgbackrest">pgBackRest 的通知</h3>

              <p>pgBackRest 没有内置通知支持，实现通知需要：</p>

              <ul>
                <li>围绕备份命令自写脚本</li>
                <li>集成外部监控工具</li>
                <li>手动解析日志并配置告警</li>
              </ul>

              <p>
                <a
                  href="/zh/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看 Databasus 的所有通知渠道 →
                </a>
              </p>

              <h2 id="conclusion">结论</h2>

              <p>
                在 PostgreSQL 备份生态中，Databasus 和 pgBackRest
                满足的是不同的需求。如何选择取决于你的数据库规模、团队结构和技术要求。
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    选择 Databasus，如果：
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>你是个人开发者、团队或企业，想要一个直观的备份方案</li>
                  <li>相比命令行工具，你更喜欢 Web 界面</li>
                  <li>你需要团队协作功能（工作区、RBAC、审计日志）</li>
                  <li>你想要内置的 Slack、Teams、Telegram 等通知</li>
                  <li>
                    你想在一个带调度、通知和团队功能的仪表盘里管理多个数据库的备份
                  </li>
                  <li>你想快速上手，不需要太多 PostgreSQL 知识</li>
                  <li>
                    你使用云托管数据库（AWS RDS、Google Cloud
                    SQL、Azure）或自托管 PostgreSQL
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>选择 pgBackRest，如果：</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    你需要在 PostgreSQL 17
                    以下版本上做物理或增量备份（pgBackRest 自带备份引擎）
                  </li>
                  <li>你需要差异备份或 delta 恢复（只处理变化的文件）</li>
                  <li>你需要从备份创建备用节点来实现高可用</li>
                  <li>你偏好命令行工具和基础设施即代码的工作流</li>
                  <li>你的团队有运维和调优它所需的 PostgreSQL 专业知识</li>
                </ul>
              </div>

              <p>
                两款工具都支持物理备份、WAL 归档和 PITR，都以最小化 RTO 和 RPO
                的灾难恢复为目标。Databasus
                适用于任何规模和复杂度的数据库，并提供 Web
                界面、团队功能，以及覆盖自托管和云托管数据库的逻辑与物理备份。
                <br />
                <br />
                当你想自己运维备份引擎、需要在 PostgreSQL 17
                以下版本上做物理备份，或者依赖它的差异备份和 delta
                恢复时，pgBackRest 更合适。
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
