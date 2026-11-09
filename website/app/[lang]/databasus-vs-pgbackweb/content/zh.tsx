import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs PgBackWeb：PostgreSQL 备份工具对比",
  description:
    "对比 Databasus 和 PgBackWeb 两款 PostgreSQL 备份工具：功能、安全性、团队支持、存储选项、通知和易用性上的差异。",
  keywords: [
    "Databasus vs PgBackWeb",
    "PostgreSQL 备份对比",
    "PgBackWeb 替代方案",
    "PostgreSQL 备份工具",
    "数据库备份对比",
    "pg_dump 图形界面",
    "自托管备份",
    "PostgreSQL 备份安全",
  ],
  openGraph: {
    title: "Databasus vs PgBackWeb：PostgreSQL 备份工具对比",
    description:
      "对比 Databasus 和 PgBackWeb 两款 PostgreSQL 备份工具：功能、安全性、团队支持、存储选项、通知和易用性上的差异。",
    type: "article",
    url: getLocalizedUrl("zh", "databasus-vs-pgbackweb"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs PgBackWeb：PostgreSQL 备份工具对比",
    description:
      "对比 Databasus 和 PgBackWeb 两款 PostgreSQL 备份工具：功能、安全性、团队支持、存储选项、通知和易用性上的差异。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "databasus-vs-pgbackweb"),
    languages: getLanguageAlternates("databasus-vs-pgbackweb"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackWebPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Databasus vs PgBackWeb：PostgreSQL 备份工具对比",
            description:
              "全面对比 Databasus 和 PgBackWeb 两款 PostgreSQL 备份工具，涵盖功能、安全性、团队支持、存储选项和易用性。",
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
              <h1 id="databasus-vs-pgbackweb">Databasus vs PgBackWeb</h1>

              <p className="text-lg text-gray-400">
                Databasus 和 PgBackWeb 都是通过 Web 界面简化 PostgreSQL
                备份管理的开源工具。虽然目标相同，都是让数据库备份更容易上手，但两者在功能、安全性、团队支持和易用性上差别很大。
              </p>

              <h2 id="quick-comparison">快速对比</h2>

              <p>下面是 Databasus 与 PgBackWeb 主要差异的速览：</p>

              <table>
                <thead>
                  <tr>
                    <th>功能</th>
                    <th>Databasus</th>
                    <th>PgBackWeb</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>许可证</td>
                    <td data-label="Databasus">Apache 2.0</td>
                    <td data-label="PgBackWeb">AGPL-3.0</td>
                  </tr>
                  <tr>
                    <td>备份管理</td>
                    <td data-label="Databasus">✅ 多个数据库</td>
                    <td data-label="PgBackWeb">✅ 多个数据库</td>
                  </tr>
                  <tr>
                    <td>其他数据库支持</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL、MySQL、MariaDB、MongoDB
                    </td>
                    <td data-label="PgBackWeb">❌ 仅 PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>存储选项</td>
                    <td data-label="Databasus">
                      本地、S3、Cloudflare
                      R2、Azure、NAS、FTP
                    </td>
                    <td data-label="PgBackWeb">仅本地和 S3 兼容存储</td>
                  </tr>
                  <tr>
                    <td>通知</td>
                    <td data-label="Databasus">
                      Slack、Discord、Telegram、Teams、邮件、Webhook
                    </td>
                    <td data-label="PgBackWeb">仅 Webhook</td>
                  </tr>
                  <tr>
                    <td>安全性</td>
                    <td data-label="Databasus">
                      ✅ AES-256-GCM、每个备份独立密钥、强制只读
                    </td>
                    <td data-label="PgBackWeb">✅ PGP 加密</td>
                  </tr>
                  <tr>
                    <td>团队功能</td>
                    <td data-label="Databasus">
                      ✅ 工作区、基于角色的访问控制、审计日志
                    </td>
                    <td data-label="PgBackWeb">❌ 不提供</td>
                  </tr>
                  <tr>
                    <td>健康监控</td>
                    <td data-label="Databasus">✅ 内置</td>
                    <td data-label="PgBackWeb">❌ 不提供</td>
                  </tr>
                  <tr>
                    <td>安装</td>
                    <td data-label="Databasus">
                      一行脚本、Docker 或 Helm
                    </td>
                    <td data-label="PgBackWeb">手动 Docker 配置</td>
                  </tr>
                  <tr>
                    <td>物理备份</td>
                    <td data-label="Databasus">✅ 支持</td>
                    <td data-label="PgBackWeb">❌ 不提供</td>
                  </tr>
                  <tr>
                    <td>增量备份</td>
                    <td data-label="Databasus">✅ 块级（PG 17+）</td>
                    <td data-label="PgBackWeb">❌ 不提供</td>
                  </tr>
                  <tr>
                    <td>WAL 归档</td>
                    <td data-label="Databasus">✅ 持续流式归档</td>
                    <td data-label="PgBackWeb">❌ 不提供</td>
                  </tr>
                  <tr>
                    <td>按时间点恢复</td>
                    <td data-label="Databasus">✅ 支持</td>
                    <td data-label="PgBackWeb">❌ 不提供</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="backup-features">备份功能</h2>

              <p>两款工具都支持灵活的定时备份：</p>

              <ul>
                <li>
                  <strong>Databasus</strong>：支持每小时、每天、每周、每月或
                  cron 计划，并可精确到具体时间（例如凌晨 4 点）。采用
                  <strong>均衡的 zstd（级别 5）压缩</strong>，可将备份体积缩小
                  4-8 倍，而运行时开销只增加约 20%，效率明显高于 gzip。
                </li>
                <li>
                  <strong>PgBackWeb</strong>：支持基于 cron
                  的备份计划。备份使用 gzip 压缩，比 zstd 更慢、效率更低。
                </li>
              </ul>

              <p>
                除逻辑备份外，Databasus 还支持物理备份、增量备份和 WAL
                备份。这些能力基于 PostgreSQL 17
                的原生备份工具栈并远程运行，数据库服务器上无需安装任何东西，封闭网络可以通过
                SSH 隧道访问。你因此获得块级增量备份、持续的 WAL
                流式归档和按时间点恢复，可恢复到两次备份之间的任意一秒，实现数据近零丢失的灾难恢复。PgBackWeb
                不提供其中任何一项。
              </p>

              <h2 id="storage-options">存储选项</h2>

              <p>
                存储的灵活性对备份策略至关重要。两款工具的对比如下：
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>：支持广泛的存储目标：
                  <ul>
                    <li>本地存储</li>
                    <li>Amazon S3 及 S3 兼容服务</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS（网络附加存储）</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>：只支持本地存储和 S3
                  兼容存储。
                </li>
              </ul>

              <p>
                <a
                  href="/zh/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看 Databasus 的所有存储选项 →
                </a>
              </p>

              <h2 id="security">安全性</h2>

              <p>
                安全是备份管理的核心环节。Databasus
                在三个层面实现企业级安全：
              </p>

              <h3 id="security-databasus">Databasus 的安全模型</h3>

              <ol>
                <li>
                  <strong>敏感数据加密</strong>：所有密码、令牌和凭据都用
                  AES-256-GCM
                  加密。加密密钥与数据库分开保存，即使数据库被攻破，敏感数据依然安全。
                </li>
                <li>
                  <strong>备份加密</strong>：每个备份文件用主密钥、备份 ID
                  和随机盐派生的唯一密钥加密。即使有人拿到了你的云存储访问权限，没有加密密钥也读不了备份。
                </li>
                <li>
                  <strong>只读数据库访问</strong>：Databasus
                  会检查角色级、数据库级和表级权限来强制只读访问。它只需要
                  SELECT 权限，并会在检测到写权限时发出警告。即使 Databasus
                  被攻破，也不会破坏数据。
                </li>
              </ol>

              <h3 id="security-pgbackweb">PgBackWeb 的安全模型</h3>

              <ul>
                <li>
                  <strong>PGP 加密</strong>：PgBackWeb 提供对备份文件的 PGP
                  加密。
                </li>
                <li>
                  <strong>不强制只读</strong>：PgBackWeb
                  不强制、也不验证只读数据库访问，意味着备份可能由拥有写权限的用户创建。
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

              <h2 id="notifications">通知</h2>

              <p>随时掌握备份状态是可靠运维的基础：</p>

              <ul>
                <li>
                  <strong>Databasus</strong>：通过多种渠道实时通知：
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>邮件</li>
                    <li>Webhook</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>：通知只支持 Webhook。想通过
                  Slack、Telegram
                  或其他平台接收告警，需要额外搭建中间层或第三方服务。
                </li>
              </ul>

              <p>
                <a
                  href="/zh/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  查看 Databasus 的所有通知渠道 →
                </a>
              </p>

              <h2 id="team-features">团队功能</h2>

              <p>
                对于组织和 DevOps 团队来说，协作功能必不可少。这正是
                Databasus 明显领先 PgBackWeb 的地方：
              </p>

              <h3 id="team-databasus">Databasus 的团队能力</h3>

              <ul>
                <li>
                  <strong>工作区</strong>
                  ：按项目或团队把数据库、通知渠道和存储分组管理。用户只能看到被邀请加入的工作区。
                </li>
                <li>
                  <strong>基于角色的访问控制</strong>
                  ：通过权限级别控制每位成员在工作区内能做什么。
                </li>
                <li>
                  <strong>审计日志</strong>
                  ：记录用户在系统中的所有操作和变更，是安全合规与团队责任追溯的基础。
                </li>
              </ul>

              <h3 id="team-pgbackweb">PgBackWeb 的团队能力</h3>

              <p>
                PgBackWeb
                没有内置用户管理、工作区或审计日志，主要面向单用户场景。
              </p>

              <p>
                <a
                  href="/zh/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  进一步了解 Databasus 的访问管理 →
                </a>
              </p>

              <h2 id="ease-of-use">易用性</h2>

              <p>
                <strong>Databasus 在设计上比 PgBackWeb 好用得多</strong>
                ，重点放在直观的交互和最短的上手时间上：
              </p>

              <h3 id="ease-databasus">Databasus 的使用体验</h3>

              <ul>
                <li>
                  <strong>安装简单</strong>：直接用 Docker，或运行一行脚本自动安装
                  Docker（如有需要）、部署 Databasus
                  并配置自启动。总耗时约 2 分钟。
                </li>
                <li>
                  <strong>直观的 Web 界面</strong>
                  ：经设计师打磨的界面，一步步引导你完成备份配置，无需
                  PostgreSQL 专业知识。
                </li>
                <li>
                  <strong>深色和浅色主题</strong>
                  ：按自己的习惯选择外观。
                </li>
                <li>
                  <strong>移动端适配</strong>
                  ：随时随地在任何设备上查看备份。
                </li>
                <li>
                  <strong>内置健康监控</strong>
                  ：可配置的健康检查，配以可视化的可用性图表。
                </li>
                <li>
                  <strong>一键恢复</strong>
                  ：任意备份都能一键下载和恢复。
                </li>
              </ul>

              <h3 id="ease-pgbackweb">PgBackWeb 的使用体验</h3>

              <ul>
                <li>
                  <strong>手动 Docker 配置</strong>
                  ：需要配置环境变量，并额外准备一个外部 PostgreSQL
                  数据库来存放配置。
                </li>
                <li>
                  <strong>基础的 Web 界面</strong>：能用，但打磨程度不及
                  Databasus。提供深色主题。
                </li>
                <li>
                  <strong>没有健康监控</strong>
                  ：数据库可用性监控需要单独搭建。
                </li>
              </ul>

              <h2 id="installation">安装与部署</h2>

              <h3 id="install-databasus">安装 Databasus</h3>

              <p>
                Databasus 提供三种安装方式，其中自动脚本最快：
              </p>

              <ul>
                <li>
                  <strong>自动脚本（推荐）</strong>：一行 cURL 命令安装
                  Docker、部署 Databasus 并配置自启动。
                </li>
                <li>
                  <strong>Docker run</strong>：一条命令启动内置 PostgreSQL 的
                  Databasus。
                </li>
                <li>
                  <strong>Docker Compose</strong>
                  ：需要更精细控制部署时使用。
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

              <h3 id="install-pgbackweb">安装 PgBackWeb</h3>

              <p>
                PgBackWeb 需要 Docker
                和手动配置环境变量，还要额外搭建一个外部 PostgreSQL
                数据库来存放 PgBackWeb 的配置。
              </p>

              <h2 id="licensing">许可证</h2>

              <p>
                许可模式会直接影响你使用和修改软件的方式：
              </p>

              <ul>
                <li>
                  <strong>Databasus（Apache 2.0）</strong>
                  ：宽松许可证，允许不受限制的商业使用、修改和分发。你可以在专有项目中使用
                  Databasus，没有任何许可顾虑。
                </li>
                <li>
                  <strong>PgBackWeb（AGPL-3.0）</strong>
                  ：Copyleft 许可证，任何衍生作品或修改也必须以 AGPL-3.0
                  开源。如果你修改了 PgBackWeb
                  并以服务形式提供，就必须公开你的修改。
                </li>
              </ul>

              <h2 id="conclusion">结论</h2>

              <p>
                Databasus 和 PgBackWeb 都是称职的 PostgreSQL
                备份工具，但它们满足的需求不同：
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    选择 Databasus，如果你需要：
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>三层防护的企业级安全</li>
                  <li>带工作区和审计日志的团队协作</li>
                  <li>
                    多种存储目标（FTP、SFTP、Azure 等）
                  </li>
                  <li>内置的 Slack、Teams、Telegram 等通知</li>
                  <li>一行脚本或 Docker 的快速安装</li>
                  <li>直观现代、几乎无学习成本的界面</li>
                  <li>可商用的宽松 Apache 2.0 许可证</li>
                  <li>
                    面向灾难恢复的物理备份、增量备份、WAL 归档和 PITR
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>选择 PgBackWeb，如果你需要：</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>面向单用户场景的简单备份方案</li>
                  <li>只用本地或 S3 存储</li>
                  <li>只有 Webhook 通知也够用</li>
                  <li>AGPL-3.0 许可证对你的场景可以接受</li>
                </ul>
              </div>

              <p>
                对大多数用户，尤其是需要可靠安全、多种存储选项和完善通知渠道的团队和组织来说，
                <strong>Databasus 是推荐的选择</strong>。
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
