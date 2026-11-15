import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

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
  const startupRecoveryCommand = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  -e RECOVERY_DUMP_PATH=/databasus-data/recovery/backup.dbbackup \\
  -e RECOVERY_MASTER_KEY_PATH=/databasus-data/recovery/master.key \\
  --restart unless-stopped \\
  databasus/databasus:<version>`;

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
                  text: "Databasus 从自己的主机远程执行物理备份，通过标准复制协议连接你的 PostgreSQL，因此数据库服务器上无需安装任何软件。封闭网络中的数据库可以通过 SSH 隧道访问。物理备份使用 PostgreSQL 17 的原生工具链：通过 pg_basebackup 做全量备份，通过由服务端 WAL 摘要 (summarize_wal = on) 驱动的 pg_basebackup --incremental 做块级增量备份，并通过 pg_receivewal 做持续 WAL 流式传输。物理备份要求 PostgreSQL 17 或更新版本；更早的版本使用逻辑 pg_dump 备份。要恢复到某个时间点，pg_combinebackup 会从全量备份及其增量链重建可运行的数据目录，然后 PostgreSQL 重放 WAL 到你选择的目标时间，可以恢复到两次备份之间的任意一秒。Databasus 界面提供恢复到主机或 Docker 数据库的分步指引：既可以用现成脚本把恢复变成一条命令，也可以自己下载备份并重建全量、增量和 WAL 的链条。增量和 WAL 是可选的：可以只做全量备份，WAL 不是必需的。我们使用 PostgreSQL 17 原生备份，因为它复用 PostgreSQL 自身久经考验的备份机制而不是重新造轮子，并提供接近零的数据丢失。",
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
                  text: "Databasus 的设计目标是：即使自身所在的服务器丢失或损坏，也能恢复。它可以备份自己的内部数据库：账号、设置、数据库、存储和通知渠道的加密凭据、备份计划以及备份清单。在 Databasus settings → Databasus backups 中选择一个或多个已有存储即可开启；默认每小时执行一次 Databasus 备份，保留 30 天。备份文件用主密钥加密，既不包含主密钥，也不包含你的数据库的备份文件，所以请单独保存主密钥：Databasus 会请你下载它，它也是主机上的 ./databasus-data/secret.key。恢复需要备份文件、对应的主密钥以及完全相同的 Databasus 版本，补丁版本也要一致。一种方式是把两个文件放进挂载的数据目录，在一台没有账号、也没改过设置的服务器上用 RECOVERY_DUMP_PATH 和 RECOVERY_MASTER_KEY_PATH 启动 Databasus；另一种方式是在 Databasus settings → Databasus backups 中上传文件并粘贴密钥，这会替换当前安装。",
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
              {
                "@type": "Question",
                name: "Databasus 开发中如何使用 AI？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "AI 帮助我们开发 Databasus，参与规划、实现、测试和文档工作。开发者确定方向并做出最终决定。 合并变更前，开发者会逐行阅读代码。 我们使用 OpenSpec 进行规格驱动开发（SDD），在实现前约定行为。 每个阶段都有独立审查 自动钩子启动独立的审查会话，对仓库的访问权限为只读。每个代理自行检查相关代码，给出自己的审查结果。执行实现的代理不能批准自己的工作。 审查在实现前就开始了。独立的规划审查代理寻找更简单的方案，检查拟定的名称并评估设计。它们还检查需求与验证任务是否一致。 代码审查代理：查找缺陷、遗漏的边界情况、错误处理问题、竞态条件、资源泄漏和安全问题。 需求审查代理：找到每项需求对应的实现和测试，检查测试是否真正证明了承诺的行为。 架构审查代理：检查模块边界、依赖方向、数据归属、迁移和 API 规则。 代码结构审查代理：检查命名、文件和函数的职责、注释以及可读性。 测试审查代理：评估测试能否发现行为缺失或回归问题，还检查条件等待、清理和验证证据。 文档和任务审查代理：检查文档是否与代码一致、任务是否完成，以及所声称的检查是否有证据支持。 完整的实现审查并行运行。每项修改后的规格说明都有自己的需求审查代理。架构、测试和文档也各有独立代理。因此，涉及两项规格说明的变更会有五个完整审查代理。 阻塞性问题会阻止工作被接受。 执行实现的代理修复问题，或解释某项审查意见为何有误。后续审查代理检查处理结果和之后的修改。未能运行的审查不算通过。 自动检查与真实恢复 独立审查还配有严格的自动检查。golangci-lint 和项目自有的架构检查工具检查后端依赖、模块边界和迁移规则。API 检查验证 OpenAPI 契约和生成代码。前端检查 TypeScript、React、分层导入、样式和本地化。 Ruff 和 ty 检查 Python 的格式、代码规则和类型。ShellCheck 检查 shell 脚本，actionlint 检查 GitHub Actions 工作流。提交前的钩子检查涉及的后端、前端和 Python 文件，不改写文件。CI 也会运行静态检查。 测试检查真实行为。后端测试通过应用的 API 操作，数据库引擎使用真实的数据库容器进行测试。端到端场景创建备份、执行恢复并检查恢复后的数据。其他测试覆盖升级、前端页面和浏览器操作流程，包括视觉对比。 检查范围取决于变更内容。发布前，在 amd64 和 arm64 上运行必需的测试套件。执行实现的代理在审查前运行所需的静态检查和测试。审查代理检查证据，不重复运行这些命令。 发布前，CodeQL 和 Semgrep 分析代码，Gitleaks 扫描密钥等敏感信息，依赖检查查找漏洞。Trivy 扫描容器镜像和 Dockerfile。 发布前必须通过所有必需的 CI 检查。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 有社区吗？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "有。Databasus 社区在 Telegram（https://t.me/databasus_community）和 Discord（https://databasus.com/discord）群里交流。你可以在那里提问、分享你使用 Databasus 的方式、讨论功能，并第一时间了解新版本。选择你已经在用的应用即可。",
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

              <h2 id="ai-usage">Databasus 开发中如何使用 AI？</h2>

              <p>
                AI 帮助我们开发 Databasus，参与规划、实现、测试和文档工作。开发者确定方向并做出最终决定。{" "}
                <strong>合并变更前，开发者会逐行阅读代码。</strong>
              </p>

              <p>
                我们使用 <a href="https://github.com/databasus/databasus/tree/main/openspec">OpenSpec</a>{" "}
                进行规格驱动开发（SDD），在实现前约定行为。
              </p>

              <h3 id="ai-review-hooks">每个阶段都有独立审查</h3>

              <p>
                自动钩子启动独立的审查会话，对仓库的访问权限为只读。每个代理自行检查相关代码，给出自己的审查结果。执行实现的代理不能批准自己的工作。
              </p>

              <p>
                审查在实现前就开始了。独立的规划审查代理寻找更简单的方案，检查拟定的名称并评估设计。它们还检查需求与验证任务是否一致。
              </p>

              <ul>
                <li>
                  <strong>代码审查代理：</strong>{" "}
                  查找缺陷、遗漏的边界情况、错误处理问题、竞态条件、资源泄漏和安全问题。
                </li>
                <li>
                  <strong>需求审查代理：</strong>{" "}
                  找到每项需求对应的实现和测试，检查测试是否真正证明了承诺的行为。
                </li>
                <li>
                  <strong>架构审查代理：</strong> 检查模块边界、依赖方向、数据归属、迁移和 API 规则。
                </li>
                <li>
                  <strong>代码结构审查代理：</strong> 检查命名、文件和函数的职责、注释以及可读性。
                </li>
                <li>
                  <strong>测试审查代理：</strong>{" "}
                  评估测试能否发现行为缺失或回归问题，还检查条件等待、清理和验证证据。
                </li>
                <li>
                  <strong>文档和任务审查代理：</strong>{" "}
                  检查文档是否与代码一致、任务是否完成，以及所声称的检查是否有证据支持。
                </li>
              </ul>

              <p>
                完整的实现审查并行运行。每项修改后的规格说明都有自己的需求审查代理。架构、测试和文档也各有独立代理。因此，涉及两项规格说明的变更会有五个完整审查代理。
              </p>

              <p>
                <strong>阻塞性问题会阻止工作被接受。</strong>{" "}
                执行实现的代理修复问题，或解释某项审查意见为何有误。后续审查代理检查处理结果和之后的修改。未能运行的审查不算通过。
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/.agent-hooks/review_gate.py">
                  审查钩子的实现
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/tree/main/.agent-hooks">审查代理的指令</a>
              </p>

              <h3 id="ai-linting">自动检查与真实恢复</h3>

              <p>
                独立审查还配有严格的自动检查。golangci-lint
                和项目自有的架构检查工具检查后端依赖、模块边界和迁移规则。API 检查验证 OpenAPI
                契约和生成代码。前端检查 TypeScript、React、分层导入、样式和本地化。
              </p>

              <p>
                Ruff 和 ty 检查 Python 的格式、代码规则和类型。ShellCheck 检查 shell 脚本，actionlint 检查
                GitHub Actions 工作流。提交前的钩子检查涉及的后端、前端和 Python 文件，不改写文件。CI
                也会运行静态检查。
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/mise-tasks/backend/lint">
                  后端静态检查
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/frontend/lint/check.sh">
                  前端静态检查
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/scripts/python_quality.py">
                  Python 检查
                </a>
              </p>

              <p>
                测试检查真实行为。后端测试通过应用的 API
                操作，数据库引擎使用真实的数据库容器进行测试。端到端场景创建备份、执行恢复并检查恢复后的数据。其他测试覆盖升级、前端页面和浏览器操作流程，包括视觉对比。
              </p>

              <p>
                检查范围取决于变更内容。发布前，在 amd64 和 arm64
                上运行必需的测试套件。执行实现的代理在审查前运行所需的静态检查和测试。审查代理检查证据，不重复运行这些命令。
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/docs/testing.md">测试策略与命令</a>
              </p>

              <p>
                发布前，CodeQL 和 Semgrep 分析代码，Gitleaks 扫描密钥等敏感信息，依赖检查查找漏洞。Trivy
                扫描容器镜像和 Dockerfile。
              </p>

              <p>
                <strong>发布前必须通过所有必需的 CI 检查。</strong>{" "}
                <a href={`${getLocalizedHref("zh", "security")}#security-and-reliability-engineering`}>
                  安全与可靠性工程
                </a>
                .
              </p>

              <h2 id="backup-databasus">如何备份 Databasus 自身？</h2>

              <p>
                Databasus 的设计目标是：即使自身所在的服务器丢失或损坏，也能恢复。它可以备份自己的内部数据库：账号、设置、你的数据库、存储和通知渠道的加密凭据、备份计划以及备份清单。在{" "}
                <strong>Databasus settings → Databasus backups</strong>{" "}
                中选择一个或多个已有存储即可开启这项功能。默认每小时执行一次 Databasus
                备份，保留 30 天。每个文件都用主密钥加密。
              </p>

              <p>
                Databasus 备份不包含主密钥，也不包含你的数据库的备份文件，后者留在各自的存储中。
                <strong>请单独保存主密钥。</strong>Databasus 会在你第一次访问时请你下载主密钥，它也就是主机上的{" "}
                <code>./databasus-data/secret.key</code>{" "}
                文件。
              </p>

              <ul>
                <li>
                  <strong>选择远程存储。</strong>同一台服务器上的本地存储会随这台服务器一起丢失。
                </li>
                <li>
                  <strong>警报。</strong>当 Databasus 备份失败、只保存到部分存储、停止或逾期时，Databasus
                  会显示提示。逾期是指计划运行后 15 分钟内没有任何副本保存成功。配置了
                  <a href={`${getLocalizedHref("zh", "advanced-config")}#email-smtp`}>
                  实例邮件服务器
                  </a>
                  时，同样的警报也会通过邮件发送。
                </li>
              </ul>

              <h3 id="recover-databasus">
                如何从 Databasus 备份恢复 Databasus？
              </h3>

              <p>
                你需要两样东西：一个 Databasus 备份文件（<code>.dbbackup</code>
                ），可以从历史记录中下载，也可以直接从存储中取；以及生成它的那个安装的主密钥。
                <strong>
                恢复只能在完全相同的 Databasus 版本上进行，补丁版本也要一致。
                </strong>
                版本不同时，Databasus 会拒绝恢复，并告诉你该启动哪个版本：先在那个版本上恢复，再照常升级。
              </p>

              <p>
                <strong>方式 1：启动时提供文件。</strong>
                适用于新服务器。把两个文件放进挂载的数据目录，例如{" "}
                <code>./databasus-data/recovery/</code>
                ，然后用生成该备份的那个版本的镜像启动，并加上两项额外设置。把{" "}
                <code>&lt;version&gt;</code> 替换成那个版本：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startupRecoveryCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={startupRecoveryCommand} />
                </div>
              </div>

              <ul>
                <li>
                  如果你已经在这台服务器上正常启动过
                  Databasus，但没有创建账号，也没有改过设置，就不必删除它的数据。停止它，加上这两个文件和两项设置，再重新启动。
                </li>
                <li>
                  只要安装中有账号（哪怕只有一个）、有其他用户数据或改过设置，启动时恢复就绝不会替换它。这种情况下 Databasus
                  不会启动。日志会说明恢复没有执行、原因、当前安装保持不变，以及下一步该怎么做。要启动当前安装，请去掉这两项设置，或者改用界面中的上传功能。
                </li>
                <li>
                  恢复成功后，文件和设置可以留着。Databasus 会记住已应用文件的
                  SHA-256，所以重启和之后的升级都会跳过它，并在日志中说明，即使文件改了名也一样。同一路径下换成另一个文件则不会被跳过。
                </li>
              </ul>

              <p>
                <strong>方式 2：在界面中上传。</strong>
                适用于已经在运行的安装。如果是新安装，先注册。打开{" "}
                <strong>Databasus settings → Databasus backups</strong>
                ，选择备份文件，然后粘贴或输入生成它的那个安装的主密钥。当前安装中的一切都会被替换：账号、设置和备份清单。之后请用备份中的账号登录。
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

              <h2 id="community">Databasus 有社区吗？</h2>

              <p>
                有。Databasus 社区在{" "}
                <a
                  href="https://t.me/databasus_community"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Telegram
                </a>{" "}
                和{" "}
                <a
                  href="/discord"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Discord
                </a>{" "}
                群里交流。你可以在那里提问、分享你使用 Databasus 的方式、讨论功能，并第一时间了解新版本。
              </p>

              <p>选择你已经在用的应用即可。</p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
