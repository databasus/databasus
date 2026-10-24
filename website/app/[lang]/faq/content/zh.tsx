import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "常见问题 FAQ | Databasus",
  description:
    "关于 Databasus PostgreSQL 备份工具（支持 MySQL、MariaDB 和 MongoDB）的常见问题。了解如何备份 localhost 数据库、备份格式、压缩方式等。",
  keywords: [
    "Databasus 常见问题",
    "PostgreSQL 备份问题",
    "localhost 数据库备份",
    "备份格式",
    "pg_dump 压缩",
    "zstd 压缩",
    "PostgreSQL 备份帮助",
    "数据库备份指南",
  ],
  openGraph: {
    title: "常见问题 FAQ | Databasus",
    description:
      "关于 Databasus PostgreSQL 备份工具（支持 MySQL、MariaDB 和 MongoDB）的常见问题。了解如何备份 localhost 数据库、备份格式、压缩方式等。",
    type: "article",
    url: getLocalizedUrl("zh", "faq"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "常见问题 FAQ | Databasus",
    description:
      "关于 Databasus PostgreSQL 备份工具（支持 MySQL、MariaDB 和 MongoDB）的常见问题。了解如何备份 localhost 数据库、备份格式、压缩方式等。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "faq"),
    languages: getLanguageAlternates("faq"),
  },
  robots: "index, follow",
};

export default function FAQPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "为什么 Databasus 的逻辑 PostgreSQL 备份不使用原始 SQL dump 格式？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "在逻辑备份方面，Databasus 使用 pg_dump 的自定义格式加 zstd 压缩，因为经过大量测试，它提供了最高效的备份和恢复速度。自定义格式配合 zstd 压缩级别 5，在备份创建速度、恢复速度和文件大小之间达到最佳平衡。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 安装在哪里？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 安装在 /opt/databasus/",
                },
              },
              {
                "@type": "Question",
                name: "物理备份和 PITR（时间点恢复）备份是如何工作的？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 从自己的主机远程执行物理备份，通过标准复制协议连接你的 PostgreSQL，因此数据库服务器上无需安装任何软件。封闭网络中的数据库可以通过 SSH 隧道访问。物理备份使用 PostgreSQL 17 的原生工具链：通过 pg_basebackup 做全量备份，通过由服务端 WAL 摘要 (summarize_wal = on) 驱动的 pg_basebackup --incremental 做块级增量备份，并通过 pg_receivewal 做持续 WAL 流式传输。物理备份要求 PostgreSQL 17 或更新版本；更早的版本使用逻辑 pg_dump 备份。要恢复到某个时间点，pg_combinebackup 会从全量备份及其增量链重建可运行的数据目录，然后 PostgreSQL 重放 WAL 到你选择的目标时间，可以恢复到两次备份之间的任意一秒。Databasus 界面提供恢复到主机或 Docker 数据库的分步指引：既可以用现成脚本把恢复变成一条命令，也可以自己下载备份并重建全量、增量和 WAL 的链条。增量和 WAL 是可选的：可以只做全量备份，WAL 不是必需的。我们使用 PostgreSQL 17 原生备份，因为它复用 PostgreSQL 自身久经考验的备份机制而不是重新造轮子，支持包括 RDS 和 Cloud SQL 在内的托管服务等远程数据库，并提供接近零的数据丢失。",
                },
              },
              {
                "@type": "Question",
                name: "为什么 Databasus 放弃了基于 agent 的备份？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "早期版本的 Databasus 附带一个备份 agent：一个运行在数据库主机上的二进制程序，用来传输 WAL 并在本地创建物理备份。这个最初的实现被证明是个错误，已经移除。它是一个简陋的实现，只是在全量备份之上复制 WAL，导致 RTO 很长。用户必须同时配置 Databasus 和一个独立的 agent，而在一个地方远程完成所有事情要简单得多。因为 agent 位于主系统之外，很难覆盖所有测试用例。agent 真正解决的问题只有一个：访问从外部无法触达的数据库，而对 99% 的用户来说，把 Databasus 部署在私有网络里或通过 SSH 连接已经解决了这个问题，所以 agent 是在重新造轮子，把一个简单问题弄得复杂得多。它也无法在 RDS 和 Cloud SQL 这类托管数据库上运行，这些服务禁止在主机上安装软件但已经暴露复制协议，所以无论如何都需要一条远程路径。此外它还带来大量边界情况：连接中断、agent 更新管理、从独立进程收集日志，而系统的活动部件越少，日常使用就越可靠。物理备份现在由 Databasus 主机远程执行。现有备份是安全的：如果你从仍有 agent 备份的版本升级，Databasus 不会悄悄升级，而是提示这一变化，让你选择留在受支持的 3.42.0 版本，或在升级前自行删除旧的 agent 备份。基于 agent 的实现在 3.42.0 及之前的版本中仍然可用，并会长期继续工作。",
                },
              },
              {
                "@type": "Question",
                name: "如何备份 Databasus 自身？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "要备份 Databasus，进入 /opt/databasus（或你安装它的目录），然后进入 databasus-data 目录。需要备份 secret.key 文件（凭据的加密密钥）和 /pgdata 文件夹（包含配置和备份元数据的内部数据库）。有两种恢复场景：1) 只用 secret.key 就能恢复数据库备份，不需要 Databasus 界面（见手动恢复指南）；2) 要恢复带有全部配置和历史记录的 Databasus 界面，需要 secret.key 和 /pgdata 文件夹两者。恢复时，在另一台服务器上重建这个文件夹结构即可。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 如何获得 Anthropic 和 OpenAI 开源项目计划的支持？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "2026 年 3 月，Databasus 同时被 Anthropic 的 Claude for Open Source 和 OpenAI 的 Codex for Open Source 计划接纳。获得这些计划的支持是一个可靠性信号：项目经过独立评估，被行业领导者认可为值得支持的关键开源基础设施。",
                },
              },
            ],
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
              <h1 id="faq">常见问题</h1>

              <p className="text-lg text-gray-400">
                这里汇总了关于 Databasus
                最常见的问题的答案，包括安装、配置和备份策略。
              </p>

              <h2 id="why-no-raw-sql-dump">
                为什么 Databasus 的逻辑 PostgreSQL 备份不使用原始 SQL dump
                格式？
              </h2>

              <p>
                在逻辑备份方面，Databasus 使用 <code>pg_dump</code> 的
                <strong>自定义格式</strong>加{" "}
                <strong>zstd 压缩（级别 5）</strong>，而不是纯 SQL
                格式，因为它在以下几方面之间达到了最高效的平衡：
              </p>

              <ul>
                <li>备份创建速度</li>
                <li>恢复速度</li>
                <li>文件压缩比（最多比纯 SQL 格式小 20 倍）</li>
              </ul>

              <p>
                这个决定是在对不同的 PostgreSQL
                备份格式和压缩方式做了大量测试和基准评测之后做出的。测试详情见这篇文章：{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  PostgreSQL backups: comparing pg_dump speed in different
                  formats and with different compression
                </a>
                。
              </p>

              <p>Databasus 不会加入原始 SQL dump 格式，因为：</p>

              <ul>
                <li>额外的多样性对用户体验没有好处；</li>
                <li>会增加代码维护成本；</li>
                <li>当前的 dump 格式适用于 99% 的场景</li>
              </ul>

              <h2 id="installation-directory">
                通过 .sh 脚本安装时，Databasus 安装在哪里？
              </h2>

              <p>
                Databasus 安装在 <code>/opt/databasus/</code> 目录。
              </p>

              <h2 id="pitr">物理备份和 PITR（时间点恢复）备份是如何工作的？</h2>

              <p>
                Databasus <strong>从自己的主机远程</strong>
                执行物理备份，通过标准的
                <strong>复制协议</strong>连接你的
                PostgreSQL，因此数据库服务器上无需安装任何软件。如果数据库位于封闭网络中，Databasus
                可以通过连到内部主机或跳板机的 SSH
                隧道访问它，数据库永远不必暴露到公网。
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>为什么这在现在成为可能：</strong>多年来，pgBackRest
                    和 WAL-G 这类工具不得不自己实现块级增量备份引擎，因为
                    PostgreSQL 没有原生的。PostgreSQL 17 改变了这一点：该功能由{" "}
                    <strong>Robert Haas</strong> 开发，pgBackRest 的作者{" "}
                    <strong>David Steele</strong> 参与协助。PostgreSQL
                    现在原生提供服务端块级增量备份（
                    <code>pg_basebackup --incremental</code> 和{" "}
                    <code>summarize_wal</code>），所以 Databasus
                    直接构建在它之上，而不是自己再造一个。
                  </p>
                </div>
              </div>

              <p>
                <strong>备份的工作方式：</strong>
              </p>

              <ul>
                <li>
                  全量备份用 <code>pg_basebackup</code> 创建，直接流式传输到
                  Databasus
                </li>
                <li>
                  块级增量备份使用 <code>pg_basebackup --incremental</code>，由
                  PostgreSQL 17 的服务端 WAL 摘要（
                  <code>summarize_wal = on</code>
                  ）跟踪变更，因此只传输发生变化的块
                </li>
                <li>
                  WAL 通过 <code>pg_receivewal</code>{" "}
                  持续流式传输，保证两次备份之间的恢复链完整
                </li>
                <li>
                  物理备份要求 <strong>PostgreSQL 17 或更新版本</strong>
                  ；在更早的版本上使用逻辑 <code>pg_dump</code> 备份
                </li>
              </ul>

              <p>
                <strong>恢复的工作方式：</strong>
              </p>

              <ul>
                <li>
                  <code>pg_combinebackup</code>{" "}
                  从全量备份及其增量链重建出可运行的数据目录
                </li>
                <li>
                  然后 PostgreSQL 重放 WAL
                  到你选择的目标时间，可以恢复到两次备份之间的任意一秒
                </li>
                <li>
                  启动 PostgreSQL 后，它会完成恢复，提升为主库并恢复正常运行
                </li>
              </ul>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>这些不需要你手动完成。</strong> Databasus
                    界面提供恢复到主机或 Docker
                    数据库的分步指引：既可以用现成脚本，也可以手动下载备份。我们准备了脚本，让恢复只需一条命令；如果你愿意，也可以自己重建全量、增量和
                    WAL 部分组成的链条。增量和 WAL
                    同样是可选的：可以只做全量备份而不做增量，WAL 也不是必需的。
                  </p>
                </div>
              </div>

              <p>
                <strong>为什么我们使用 PG 17 原生备份：</strong>
              </p>

              <ul>
                <li>
                  它复用 PostgreSQL
                  自身的备份机制而不是重新发明，你得到的是背后有数千个测试和边界用例保障、久经考验的内部实现
                </li>
                <li>
                  它适用于远程数据库，包括 Amazon RDS 和 Google Cloud SQL
                  这类暴露复制协议但禁止在主机上安装软件的托管服务
                </li>
                <li>
                  它带来接近零的数据丢失，可以恢复到两次备份之间的任意一秒
                </li>
              </ul>

              <h2 id="why-no-agent">
                为什么 Databasus 放弃了基于 agent 的备份？
              </h2>

              <p>
                早期版本的 Databasus 附带一个备份 <strong>agent</strong>
                ：一个运行在数据库主机上的二进制程序，用来传输 WAL
                并在本地创建物理备份。这个最初的实现被证明是个错误，我们把它移除了。物理备份现在由
                Databasus 主机远程执行，如上文所述。
              </p>

              <p>
                <strong>为什么 agent 是错误的方向：</strong>
              </p>

              <ul>
                <li>
                  它是一个简陋的实现，只是在全量备份之上复制 WAL，导致 RTO 很长
                </li>
                <li>
                  用户必须同时配置 Databasus 和一个独立的
                  agent，而在一个地方远程完成所有事情要简单得多
                </li>
                <li>因为 agent 位于主系统之外，很难覆盖所有测试用例</li>
                <li>
                  agent 真正解决的问题只有一个：访问从外部无法触达的数据库。对
                  99% 的用户来说，把 Databasus 部署在私有网络里或通过 SSH
                  连接就已经解决了，所以 agent
                  是在重新造轮子，把一个简单问题弄得比实际需要复杂得多
                </li>
                <li>
                  它无法在 RDS 和 Cloud SQL
                  这类托管数据库上运行：这些服务禁止在主机上安装软件，却已经暴露了复制协议，所以无论如何都需要一条远程路径
                </li>
                <li>
                  它还带来大量边界情况。连接中断、agent
                  更新管理、从独立进程收集日志都很痛苦，而系统的活动部件越少，日常使用就越可靠
                </li>
              </ul>

              <p>
                <strong>我们确保现有备份安全无虞。</strong>如果你从仍有 agent
                备份的版本升级，Databasus
                不会悄悄进行：它会提示你这一变化，让你选择留在受支持的{" "}
                <strong>3.42.0 版本</strong>，或在升级前自行删除旧的 agent
                备份。基于 agent 的实现在 3.42.0
                及之前的版本中仍然可用，并会长期继续工作，所以不会有任何东西坏掉。
              </p>

              <h2 id="backup-databasus">如何备份 Databasus 自身？</h2>

              <p>
                如果你想备份自己的 Databasus
                实例（包括所有配置、数据库和凭据），按以下步骤操作：
              </p>

              <ol>
                <li>
                  进入 <code>/opt/databasus</code>（或你安装 Databasus 的目录）
                </li>
                <li>
                  进入 <code>databasus-data</code> 目录
                </li>
              </ol>

              <p>
                <strong>需要备份的内容：</strong>
              </p>

              <ul>
                <li>
                  <code>secret.key</code>：你的凭据的加密密钥
                </li>
                <li>
                  <code>/pgdata</code>：Databasus 的内部 PostgreSQL
                  数据库，包含你的全部配置和备份元数据
                </li>
              </ul>

              <p>
                如果你使用本地存储保存备份，也可以一并备份 <code>backups</code>{" "}
                文件夹。
              </p>

              <p>
                <strong>重要：</strong>恢复有两种不同的场景：
              </p>

              <ul>
                <li>
                  <strong>不通过 Databasus 界面恢复备份：</strong>只需要{" "}
                  <code>secret.key</code> 文件就能恢复数据库备份，不需要
                  Databasus 或它的内部数据。详细步骤见
                  <a href="/zh/how-to-recover-without-databasus">
                    手动恢复指南
                  </a>
                  。
                </li>
                <li>
                  <strong>恢复 Databasus 界面及全部配置：</strong>
                  如果你想恢复带有全部配置、备份计划和备份历史的 Databasus
                  界面，需要同时备份 <code>secret.key</code> 和{" "}
                  <code>/pgdata</code> 文件夹（后者包含加密元数据和 Databasus
                  的全部配置）。
                </li>
              </ul>

              <p>
                <strong>在另一台服务器上恢复 Databasus：</strong>
                用备份好的文件重建 <code>databasus-data</code>{" "}
                目录结构，然后启动 Databasus 即可。
              </p>

              <h2 id="oss-programs">
                Databasus 如何获得 Anthropic 和 OpenAI 开源项目计划的支持？
              </h2>

              <p>
                2026 年 3 月，Databasus 同时被 Anthropic 的{" "}
                <strong>
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Claude for Open Source
                  </a>
                </strong>{" "}
                和 OpenAI 的{" "}
                <strong>
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Codex for Open Source
                  </a>
                </strong>{" "}
                计划接纳。项目被两家全球领先的 AI
                公司认可为对行业重要的开源软件，这对我们来说非常宝贵，尤其考虑到两个计划的准入门槛都很高。
              </p>

              <p>
                这对用户意味着什么？这是又一个可靠性的佐证：项目经过独立评估，被行业领导者认可为值得支持的关键基础设施。因此，得益于对最新且不限量的
                AI
                的使用权限，我们有了更高的代码质量、更快的安全审查和持续活跃的开发。
              </p>

              <img
                src="/images/faq/anthropic-email.png"
                alt="Databasus 被 Anthropic 的 Claude for Open Source 计划接纳"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <img
                src="/images/faq/openai-email.png"
                alt="Databasus 被 OpenAI 的 Codex for Open Source 计划接纳"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
