import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

export const metadata: Metadata = {
  title: "Neon PostgreSQL 备份：图文分步指南 | Databasus",
  description:
    "带截图的分步指南：用 Databasus 定时进行 Neon PostgreSQL 备份，把备份保存在你自己的存储中，需要时再恢复到 Neon。",
  keywords: [
    "Databasus",
    "Neon 备份",
    "Neon PostgreSQL 备份",
    "备份 Neon 数据库",
    "Neon 恢复",
    "Neon 只读用户",
    "PostgreSQL 备份",
    "数据库备份",
  ],
  openGraph: {
    title: "Neon PostgreSQL 备份：图文分步指南 | Databasus",
    description:
      "带截图的分步指南：用 Databasus 定时进行 Neon PostgreSQL 备份，把备份保存在你自己的存储中，需要时再恢复到 Neon。",
    type: "article",
    url: getLocalizedUrl("zh", "faq/neon"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "Neon PostgreSQL 备份：图文分步指南 | Databasus",
    description:
      "带截图的分步指南：用 Databasus 定时进行 Neon PostgreSQL 备份，把备份保存在你自己的存储中，需要时再恢复到 Neon。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "faq/neon"),
    languages: getLanguageAlternates("faq/neon"),
  },
  robots: "index, follow",
};

export default function NeonPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "如何用 Databasus 进行 Neon PostgreSQL 备份",
            description:
              "用 Databasus 备份 Neon PostgreSQL 数据库并恢复到 Neon 的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "从 Neon 复制连接字符串",
                text: "在 Neon 控制台中打开项目，点击 Connect，再点击 Copy snippet。",
                url: `${getLocalizedUrl("zh", "faq/neon")}#copy-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "把 Neon 数据库添加到 Databasus",
                text: "点击 Add database，选择 Neon，用 Paste from clipboard 粘贴连接字符串，然后点击 Test connection。",
                url: `${getLocalizedUrl("zh", "faq/neon")}#paste-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "创建只读用户",
                text: "点击 Yes, create read-only user，这样 Databasus 保存的凭据无法修改你的数据。",
                url: `${getLocalizedUrl("zh", "faq/neon")}#read-only-user`,
              },
              {
                "@type": "HowToStep",
                name: "设置备份计划、存储和保留期限",
                text: "选择多久备份一次、备份存到哪里以及保留多久。",
                url: `${getLocalizedUrl("zh", "faq/neon")}#schedule-storage-retention`,
              },
              {
                "@type": "HowToStep",
                name: "选择通知渠道",
                text: "选择接收备份通知的渠道，然后点击 Complete。Databasus 会开始第一次备份。",
                url: `${getLocalizedUrl("zh", "faq/neon")}#notifiers`,
              },
              {
                "@type": "HowToStep",
                name: "把备份恢复到 Neon",
                text: "点击备份旁边的恢复图标，粘贴目标 Neon 数据库的连接字符串，然后点击 Restore to this DB。",
                url: `${getLocalizedUrl("zh", "faq/neon")}#restore`,
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
              <h1 id="neon-backup">如何进行 Neon PostgreSQL 备份</h1>

              <p className="text-lg text-gray-400">
                Neon 是一种 serverless 托管 PostgreSQL 服务。要用 Databasus 备份它，只需在 Neon 控制台复制连接字符串，在 Databasus 中以 Neon 类型添加数据库，再选好备份的存放位置。之后 Databasus 会按你设定的计划自动备份，并且可以把备份恢复到任意 Neon 数据库。
              </p>

              <p>
                你只需要一个正在运行的 Databasus。如果还没有，请按照
                <a href={getLocalizedHref("zh", "installation")}>安装指南</a>
                安装。存储和通知渠道可以在配置过程中直接创建。
              </p>

              <h2 id="copy-connection-string">
                第 1 步：从 Neon 复制连接字符串
              </h2>

              <p>
                在 Neon 控制台中打开你的项目，点击 <strong>Connect</strong>。
              </p>

              <img
                src="/images/faq/neon/neon-connect-button.png"
                alt="Neon 项目 Computes 页面上的 Connect 按钮"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
              />

              <p>
                在 <strong>Connection details</strong> 窗口中，选择要备份的分支、数据库和角色，然后点击 <strong>Copy snippet</strong>。
              </p>

              <img
                src="/images/faq/neon/neon-copy-connection-string.png"
                alt="Neon 的 Connection details 窗口和 Copy snippet 按钮"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="add-database">第 2 步：把 Neon 数据库添加到 Databasus</h2>

              <h3 id="paste-connection-string">
                选择 Neon 并粘贴连接字符串
              </h3>

              <p>
                在 Databasus 中点击 <strong>Add database</strong>，给数据库起个名字，选择 <strong>Neon</strong>，然后点击 <strong>Continue</strong>。
              </p>

              <img
                src="/images/faq/neon/databasus-choose-neon.png"
                alt="Databasus 的 Add database 窗口，已选择 Neon 类型"
                width={1440}
                height={898}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                点击 <strong>Paste from clipboard</strong>。Databasus 会读取你在 Neon 中复制的连接字符串，自动填好主机、端口、用户名、密码、数据库名和 SSL 模式。
              </p>

              <img
                src="/images/faq/neon/databasus-paste-connection-string.png"
                alt="Databasus 中的 Neon 连接表单和 Paste from clipboard 按钮"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                点击 <strong>Test connection</strong>。测试通过后，点击 <strong>Continue</strong>。
              </p>

              <img
                src="/images/faq/neon/databasus-test-connection.png"
                alt="连接测试通过后，Databasus 中已填好的 Neon 连接表单"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="read-only-user">创建只读用户</h3>

              <p>
                Databasus 会建议为备份创建一个<a href={`${getLocalizedHref("zh", "security")}#level-3-read-only-access`}>只读用户</a>。点击 <strong>Yes, create read-only user</strong>。这样 Databasus 保存的凭据只能读取你的数据，无法修改。
              </p>

              <img
                src="/images/faq/neon/databasus-create-read-only-user.png"
                alt="Databasus 建议为 Neon 备份创建只读用户"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="schedule-storage-retention">
                设置备份计划、存储和保留期限
              </h3>

              <p>在这一步选择：</p>

              <ul>
                <li>
                  <strong>Backup interval</strong> 和时间：Databasus 多久做一次备份，例如每天 03:00。
                </li>
                <li>
                  <strong>Storage</strong>：备份存放的位置。你可以直接在这个列表中<a href={getLocalizedHref("zh", "storages")}>添加存储</a>。想把备份也放在 Neon？请看<a href={getLocalizedHref("zh", "storages/neon-object-storage")}>如何备份到 Neon Object Storage</a>。
                </li>
                <li>
                  <strong>Encryption</strong>：上传前是否<a href={`${getLocalizedHref("zh", "security")}#level-2-backups-encryption`}>加密备份文件</a>。
                </li>
                <li>
                  <strong>Retention policy</strong>：旧备份保留多久，例如 3 个月。
                </li>
                <li>
                  <strong>Notifications</strong>：通知哪些事件，是只通知备份失败，还是成功也通知。
                </li>
              </ul>

              <img
                src="/images/faq/neon/databasus-schedule-storage-retention.png"
                alt="Databasus 中的备份计划、存储、加密和保留期限设置"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="notifiers">选择通知渠道</h3>

              <p>
                选择接收这些通知的<a href={getLocalizedHref("zh", "notifiers")}>通知渠道</a>，例如 <a href={getLocalizedHref("zh", "notifiers/slack")}>Slack</a> 或 <a href={getLocalizedHref("zh", "notifiers/teams")}>Microsoft Teams</a>。可以选多个，也可以先不选，以后再添加。点击 <strong>Complete</strong>。
              </p>

              <img
                src="/images/faq/neon/databasus-choose-notifiers.png"
                alt="完成数据库设置前，在 Databasus 中选择通知渠道"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="first-backup">第 3 步：检查第一次备份</h2>

              <p>
                点击 <strong>Complete</strong> 后，Databasus 会立即开始第一次备份。它会出现在 <strong>Backups</strong> 标签页上，并显示大小和耗时。之后的备份按计划进行，点击 <strong>Make backup right now</strong> 可以随时手动备份一次。
              </p>

              <img
                src="/images/faq/neon/databasus-first-neon-backup.png"
                alt="Databasus 的 Backups 标签页上第一次成功的 Neon 备份"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                备份成功并不代表一定能恢复。要确认每个备份都能真正恢复，请设置<a href={getLocalizedHref("zh", "restore-verification")}>恢复验证</a>。
              </p>

              <h2 id="restore">如何恢复 Neon PostgreSQL 备份</h2>

              <p>
                点击备份旁边的恢复图标（带箭头的云朵）。按照<a href="#copy-connection-string">第 1 步</a>的方法，在 Neon 控制台中复制目标数据库的连接字符串，点击 <strong>Paste from clipboard</strong>，再点击 <strong>Restore to this DB</strong>。
              </p>

              <img
                src="/images/faq/neon/databasus-restore-neon-backup.png"
                alt="Databasus 中的备份恢复窗口，目标是 Neon 数据库"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                恢复前，Databasus 会清空目标数据库。目标可以是：
              </p>

              <ul>
                <li>同一 Neon 项目中的新数据库</li>
                <li>Neon 分支上的数据库</li>
                <li>已有的数据库，包括源数据库本身</li>
              </ul>

              <p>
                最稳妥的做法是恢复到一个新的 Neon 分支。你可以先在那里检查恢复出的数据，在你决定切换之前，生产数据库不会受到任何影响。
              </p>

              <p>恢复完成后，状态会变为 Successful。</p>

              <img
                src="/images/faq/neon/databasus-restore-done.png"
                alt="Databasus 中已完成的 Neon 备份恢复"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="health-check">
                可用性检查会让 Neon 计算节点无法休眠吗？
              </h2>

              <p>
                默认不会。没有任何连接时，Neon 会让计算节点（compute）休眠，休眠期间不计费。可用性检查每分钟连接一次，会让计算节点一直运行，全天候计费。因此 Databasus 默认为 Neon 数据库关闭可用性检查。如果你要开启它，Databasus 会先请你确认。定时备份只在运行期间唤醒计算节点，结束后 Neon 会让它重新休眠。
              </p>

              <p>
                还要备份 Supabase？请看<a href={getLocalizedHref("zh", "faq/supabase")}>如何备份 Supabase</a>。
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("zh", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← 返回常见问题
                </a>
              </div>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
