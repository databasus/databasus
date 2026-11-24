import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "备份恢复验证 - Databasus 文档",
  description:
    "证明你的数据库备份真的可以恢复。Databasus 拉取最新备份，把它恢复到一次性数据库容器中，对照源库检查恢复出的数据库，并在每次运行时报告各表的行数。",
  keywords: [
    "备份恢复验证",
    "数据库恢复",
    "备份验证",
    "灾难恢复",
    "数据库备份测试",
    "Databasus 验证代理",
    "备份完整性",
    "自动恢复测试",
  ],
  openGraph: {
    title: "备份恢复验证 - Databasus 文档",
    description:
      "证明你的数据库备份真的可以恢复。Databasus 拉取最新备份，把它恢复到一次性数据库容器中，对照源库检查恢复出的数据库，并在每次运行时报告各表的行数。",
    type: "article",
    url: getLocalizedUrl("zh", "restore-verification"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "备份恢复验证 - Databasus 文档",
    description:
      "证明你的数据库备份真的可以恢复。Databasus 拉取最新备份，把它恢复到一次性数据库容器中，对照源库检查恢复出的数据库，并在每次运行时报告各表的行数。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "restore-verification"),
    languages: getLanguageAlternates("restore-verification"),
  },
  robots: "index, follow",
};

export default function RestoreVerificationPage() {
  const downloadAgent = `curl -L -o verification-agent "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" \\
  && chmod +x verification-agent`;

  const startAgent = `./verification-agent start \\
  --databasus-host=https://your-databasus-host \\
  --agent-id=<AGENT_ID> \\
  --token=<TOKEN> \\
  --max-cpu=2 \\
  --max-ram-mb=2048 \\
  --max-disk-gb=20 \\
  --max-concurrent-jobs=1`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "备份恢复验证 - Databasus 文档",
            description:
              "证明你的数据库备份真的可以恢复。Databasus 拉取最新备份，把它恢复到一次性数据库容器中，对照源库检查恢复出的数据库，并在每次运行时报告各表的行数。",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "如何在 Databasus 中配置备份恢复验证",
            description:
              "注册验证代理、在服务器上启动它并配置定时恢复验证的分步指南。",
            step: [
              {
                "@type": "HowToStep",
                name: "在界面中创建验证代理",
                text: "打开 Settings → Verification agents，点击 Create verification agent，起个名字，然后从对话框中复制令牌和代理 ID。",
              },
              {
                "@type": "HowToStep",
                name: "下载代理二进制文件",
                text: "在要运行验证的主机上执行 curl 命令，按架构选择 amd64 或 arm64。",
              },
              {
                "@type": "HowToStep",
                name: "启动代理",
                text: "带上 --agent-id、--token 和资源预算参数（--max-cpu、--max-ram-mb、--max-disk-gb、--max-concurrent-jobs）启动代理。",
              },
              {
                "@type": "HowToStep",
                name: "设置验证计划",
                text: "打开数据库的验证设置，启用 Scheduled verification，并选择周期（After backup、Hourly、Daily、Weekly、Monthly 或 Cron）。",
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
              <h1 id="restore-verification">备份恢复验证</h1>

              <p className="text-lg text-gray-400">
                备份顺利完成，不等于备份真的能恢复。唯一的实证就是把它恢复
                一遍。Databasus 会按计划替你完成这件事：
              </p>

              <ul>
                <li>取最新的备份</li>
                <li>把它恢复到一次性数据库容器中</li>
                <li>对照源库检查恢复出的数据库</li>
                <li>销毁容器</li>
                <li>报告结果</li>
              </ul>

              <img
                src="/images/restore-verification/verified-backups.png"
                alt="Verified backups tab"
                loading="lazy"
              />

              <img
                className="mt-5"
                src="/images/restore-verification/verifications.png"
                alt="Verifications tab"
                loading="lazy"
              />

              <h2 id="what-is-verification-agent">什么是验证代理？</h2>

              <p>
                验证代理是一个小巧的 Go 二进制程序，运行在你自己掌控的机器
                上，只要有空闲的 CPU、内存和磁盘就行。代理向 Databasus
                注册后，从队列领取验证任务，在本地执行并把结果回报。
              </p>

              <h3 id="what-you-need">你需要什么</h3>

              <ul>
                <li>一台能通过 HTTPS 出站访问你 Databasus 地址的主机。</li>
                <li>
                  主机上装有 Docker：代理会为每个任务启动与源库主版本一致的
                  临时数据库容器。
                </li>
                <li>
                  每个验证任务所需的磁盘容量，包括<strong>备份文件大小</strong>
                  、<strong>数据库原始大小</strong>、预留空间，以及同时构建的
                  <strong>最大的几个索引</strong>。
                </li>
                <li>每个并发任务至少 1 个 CPU 核心和 512 MB 可用内存。</li>
              </ul>

              <h3 id="why-not-just-checksums">为什么校验和不够？</h3>

              <p>校验和与退出码能发现一部分故障，但对另一些完全无能为力：</p>

              <ul>
                <li>
                  <strong>校验和</strong>能发现归档文件的位衰减，但完全说明
                  不了转储本身是否完整、语义是否有效。
                </li>
                <li>
                  <strong>转储退出码</strong>只说明转储命令跑完了。它发现不了
                  角色对某些对象缺少读权限、源库缺扩展或表空间不匹配这类
                  问题，而这些都会导致对象被悄悄跳过或丢失。
                </li>
                <li>
                  <strong>恢复验证</strong>则真的把归档交给数据库自带的恢复
                  工具执行，并逐表统计行数。它是唯一能抓住上述全部问题的
                  检查：如果一个备份恢复不了，你会在需要它之前发现，而不是
                  在灾难发生时。
                </li>
              </ul>

              <h2 id="configuration">配置</h2>

              <h3 id="create-on-ui">在界面中创建代理</h3>

              <p>
                打开 <strong>Settings → Verification agents</strong>，点击{" "}
                <strong>Create verification agent</strong>。取一个能说明用途
                的名字，比如 <code>staging-verifier</code> 或{" "}
                <code>eu-west-host-1</code>。接下来的对话框会显示代理的
                <strong>令牌</strong>和 <strong>ID</strong>。
              </p>

              <p>
                令牌<strong>只显示一次</strong>，关闭对话框前务必复制。之后
                如果丢失，在代理所在行使用 <strong>Rotate token</strong>{" "}
                操作签发新令牌；旧令牌会在代理下一次心跳时失效。随后的对话框
                会按你服务器的架构显示安装命令，与下文描述的命令相同。
              </p>

              <h3 id="launch">在服务器上启动代理</h3>

              <p>
                SSH 登录将要运行验证的机器。先下载代理二进制文件。把{" "}
                <code>https://your-databasus-host</code> 换成你自己的 Databasus
                地址；如果服务器是 ARM 架构，把 <code>amd64</code> 换成{" "}
                <code>arm64</code>：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={downloadAgent} />
                </div>
              </div>

              <p>然后启动代理。代理 ID 和令牌来自上一步的对话框：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={startAgent} />
                </div>
              </div>

              <p>
                <code>start</code> 会让代理以守护进程运行，并把启动参数写入
                工作目录下的 <code>databasus-verification.json</code>，之后
                重启只需执行 <code>./verification-agent start</code>，无需
                任何参数。日志写入二进制文件旁边的{" "}
                <code>databasus-verification.log</code>。
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] mb-3 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    Databasus 地址必须是 <code>https://</code>。只有加上{" "}
                    <code>--allow-insecure-http</code> 才允许纯 HTTP，而它
                    仅用于本地测试：绝不要让生产环境的代理走未加密的 HTTP。
                  </p>
                </div>
              </div>

              <p>
                四个 <code>--max-*</code> 参数是<strong>总预算</strong>，
                不是单个任务的配额。代理在每次心跳时把它们上报给
                Databasus，Databasus 再把预算分给你允许的并发任务。使用{" "}
                <code>
                  --max-cpu=2 --max-ram-mb=2048 --max-concurrent-jobs=1
                </code>{" "}
                时，唯一的任务独占 2 个 CPU 和 2 GB 内存；改为{" "}
                <code>--max-concurrent-jobs=2</code> 后，每个任务分到 1 个 CPU
                和 1 GB。每个任务的下限是 1 个 CPU 和 512
                MB：如果预算满足不了下限，代理会上报更低的并发数。磁盘预算
                最容易设错：每个任务需要容纳<strong>备份文件大小</strong>、
                <strong>数据库原始大小</strong>、约{" "}
                <strong>1.5 GB 的预留空间</strong>，以及
                <strong>最大的几个索引</strong>，每个并行恢复任务对应一个，
                因为每个正在构建的索引都会在磁盘上占用与自身大小相当的排序空间。
                Databasus 会减少并行恢复任务数，直到这些都能放进代理的空闲磁盘。
                所以 <code>--max-cpu</code> 越大，需要的磁盘越多；
                <code>--max-disk-gb</code> 越大，大数据库恢复得越快。
              </p>

              <h3 id="manage">管理代理</h3>

              <p>同一个二进制文件提供四个子命令：</p>

              <ul>
                <li>
                  <code>./verification-agent status</code> —
                  显示守护进程是否在运行以及当前正在处理哪些任务。
                </li>
                <li>
                  <code>./verification-agent stop</code> —
                  停止守护进程。执行中的验证会以失败状态回报给
                  Databasus，并重新排队。
                </li>
                <li>
                  <code>./verification-agent start</code> —
                  重新启动守护进程。参数沿用首次启动时的记录；令牌轮换后 传入{" "}
                  <code>--token=&lt;NEW&gt;</code> 更新保存的令牌。
                </li>
                <li>
                  <code>./verification-agent run</code> —
                  在前台运行而不是守护进程。把代理放进 systemd 单元或 Docker
                  容器时用这个：这些管理器要求进程不要自行 fork。
                </li>
              </ul>

              <p>
                Settings 页面在每个代理所在行提供三个图标操作：再次查看安装
                命令（不会显示令牌）、轮换令牌、删除代理。删除是安全的：
                当前分配给该代理的验证会退回队列，如果有其他可用代理，会被
                它们接手。
              </p>

              <h2 id="schedules-and-notifications">计划与通知</h2>

              <p>
                恢复验证按数据库单独配置。打开数据库的验证设置，启用{" "}
                <strong>Scheduled verification</strong>，然后选择周期。
              </p>

              <h3 id="interval-options">周期选项</h3>

              <ul>
                <li>
                  <strong>After backup</strong> —
                  最强的保障：每个成功的备份一完成就立即验证。
                </li>
                <li>
                  <strong>Hourly、daily、weekly、monthly</strong> —
                  选择频率和一天中的时间。
                </li>
                <li>
                  <strong>Cron</strong> — 用 UTC cron
                  表达式覆盖预设之外的任何需求。例如：<code>0 4 * * 0</code>{" "}
                  （每周日 4:00 AM UTC）和 <code>0 */6 * * *</code> （每 6
                  小时一次）。
                </li>
              </ul>

              <h3 id="how-the-queue-works">
                队列如何处理 &quot;After backup&quot;
              </h3>

              <p>
                验证通常比产生它的备份更慢，如果备份到来的速度超过验证完成的
                速度，队列会无限增长。Databasus 的解决办法是：
                <strong>
                  每当同一数据库有新备份到达，就取消它所有待处理的验证
                </strong>
                ，队列里只留最新的备份。这个取舍是有意为之：与其花几个小时
                验证一个反正不会用来恢复的过时备份，不如跳过它。
              </p>

              <h3 id="manual-runs">手动运行</h3>

              <p>
                你也可以在数据库的 <strong>Restore verifications</strong>{" "}
                标签页发起一次性验证，无需改动计划。适合抽查某个特定备份，
                或者在把定时任务交给新代理之前先端到端地试运行一次。
              </p>

              <h3 id="notifications">通知</h3>

              <p>
                成功和失败都可以通过数据库已接入的任意通知渠道发送。两个 复选框
                — <strong>Verification success</strong> 和{" "}
                <strong>Verification failed</strong> —
                彼此独立。多数团队只开失败通知，避免通知疲劳。接入
                Slack、Microsoft Teams、Discord、邮件等渠道请参阅{" "}
                <a
                  href="/zh/notifiers"
                  className="text-blue-400 hover:text-blue-300"
                >
                  通知渠道文档
                </a>
                。
              </p>

              <h3 id="results">解读结果</h3>

              <p>
                每次验证尝试都会在数据库的{" "}
                <strong>Restore verifications</strong> 标签页显示为一行。状态是{" "}
                <strong>Pending</strong>、<strong>Running</strong>、
                <strong>Successful</strong>、<strong>Failed</strong> 或{" "}
                <strong>Canceled</strong>{" "}
                之一。点击某一行会打开抽屉面板，显示完整时间线、恢复退出码、
                恢复出的数据库大小、模式与表的数量，以及逐表的行数明细。
                失败的运行会在抽屉顶部显示失败信息。
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
