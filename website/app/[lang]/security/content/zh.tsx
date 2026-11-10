import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "安全性 - Databasus 如何保护你的数据 | Databasus",
  description:
    "了解 Databasus 如何保障企业级安全：对敏感数据和备份使用 AES-256-GCM 加密、以只读方式访问数据库，并提供完整的审计日志。",
  keywords: [
    "Databasus 安全性",
    "PostgreSQL 备份安全",
    "AES-256-GCM 加密",
    "数据库加密",
    "备份加密",
    "数据库只读访问",
    "企业级安全",
    "数据保护",
    "安全备份",
  ],
  openGraph: {
    title: "安全性 - Databasus 如何保护你的数据 | Databasus",
    description:
      "了解 Databasus 如何保障企业级安全：对敏感数据和备份使用 AES-256-GCM 加密、以只读方式访问数据库，并提供完整的审计日志。",
    type: "article",
    url: getLocalizedUrl("zh", "security"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "安全性 - Databasus 如何保护你的数据 | Databasus",
    description:
      "了解 Databasus 如何保障企业级安全：对敏感数据和备份使用 AES-256-GCM 加密、以只读方式访问数据库，并提供完整的审计日志。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "security"),
    languages: getLanguageAlternates("security"),
  },
  robots: "index, follow",
};

export default function SecurityPage() {
  const encryptionPipeline = `PostgreSQL pg_dump → Compression → Encryption → Cloud Storage`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "安全性 - Databasus 如何保护你的数据",
            description:
              "了解 Databasus 如何保障企业级安全：对敏感数据和备份使用 AES-256-GCM 加密、以只读方式访问数据库，并提供完整的审计日志。",
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
              <h1 id="security">Databasus 如何保障安全？</h1>

              <p className="text-lg text-gray-400">
                Databasus 打交道的都是敏感数据：
              </p>

              <ul>
                <li>它访问你的数据库；</li>
                <li>它执行备份（也就是复制一份数据）；</li>
                <li>它保存凭据，以便定期访问你的数据库；</li>
                <li>它把备份存入你的 S3 或其他云存储（如果你启用了）；</li>
              </ul>

              <p>
                因此，
                <strong>
                  达到企业级的安全性和可靠性是 Databasus 的头号任务
                </strong>
                。
              </p>

              <p>为了确保：</p>

              <ul>
                <li>敏感数据绝不外泄，且始终加密；</li>
                <li>备份是加密的，即使有人在云存储里看到它们也毫无用处；</li>
                <li>Databasus 根本不会获得数据库的写入或修改权限；</li>
                <li>所有操作都被记录，可供审计；</li>
              </ul>

              <p>
                这些措施共同保护你的数据。众所周知，不存在 100%
                安全的系统，但我们尽最大努力让它足够安全。即使遭到入侵，
                也没有人能破坏你的数据。
              </p>

              <p>Databasus 在三个层面保障安全：</p>

              <ol>
                <li>敏感数据加密；</li>
                <li>备份加密；</li>
                <li>数据库只读访问。</li>
              </ol>

              <h2 id="level-1-sensitive-data-encryption">
                第 1 层：敏感数据加密
              </h2>

              <p>
                Databasus 内部使用 PostgreSQL 数据库保存连接信息、配置，
                以及通知渠道和存储（S3、Azure Blob Storage、FTP 等）的设置。
              </p>

              <p>所有敏感数据都会加密。例如：</p>

              <ul>
                <li>密码</li>
                <li>令牌</li>
                <li>带密钥的 webhook</li>
              </ul>

              <p>
                因此，Databasus 的数据库里只存哈希或加密后的值。加密使用{" "}
                <strong>AES-256-GCM</strong> 算法。此外，即便已经加密，
                这些值也从不通过 API 或界面暴露。
              </p>

              <p>
                用于加密的密钥保存在本地存储（默认是{" "}
                <code>./databasus-data/secret.key</code>），不会出现在
                数据库中。所以就算数据库被攻破，也拿不到敏感数据。
              </p>

              <h2 id="level-2-backups-encryption">第 2 层：备份加密</h2>

              <p>
                每个备份文件在创建过程中即时加密。Databasus 使用{" "}
                <strong>AES-256-GCM</strong>{" "}
                加密算法，没有加密密钥就无法读取备份数据，任何篡改都会在
                解密时被发现。
              </p>

              <p>备份经过这条流水线：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{encryptionPipeline}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={encryptionPipeline} />
                </div>
              </div>

              <p>每个备份都有独立的加密密钥，由以下部分派生：</p>

              <ul>
                <li>
                  主密钥（保存在 <code>./databasus-data/secret.key</code>）
                </li>
                <li>备份 ID</li>
                <li>随机盐（每个备份唯一）</li>
              </ul>

              <p>
                <strong>结果</strong>：即使有人拿到了你的云存储（S3、Azure Blob Storage 等）的访问权限，没有你的主密钥也读不了备份。
              </p>

              <h2 id="level-3-read-only-access">第 3 层：数据库只读访问</h2>

              <p>
                Databasus 贯彻最小权限原则：创建备份只需要读权限，
                永远不需要写权限。这样即使经由备份工具，你的数据库也不会
                被意外或恶意地破坏。
              </p>

              <p>接受数据库凭据之前，Databasus 会做三个层面的检查：</p>

              <ol>
                <li>
                  <strong>角色层面</strong>：确认该用户不是
                  superuser，也不能创建角色或数据库
                </li>
                <li>
                  <strong>数据库层面</strong>：确保没有 CREATE 或 TEMP 权限
                </li>
                <li>
                  <strong>表层面</strong>：确认没有任何写权限（INSERT、
                  UPDATE、DELETE、TRUNCATE 等）
                </li>
              </ol>

              <p>
                三项检查全部通过，数据库用户才被认定为只读。只要发现任何
                写权限，Databasus 都会警告你。
              </p>

              <p>Databasus 还会帮你生成权限正确的只读用户：</p>

              <ul>
                <li>授予对当前及未来所有表的 SELECT 权限</li>
                <li>授予模式的 USAGE 权限（但不授予 CREATE）</li>
                <li>显式收回所有写权限</li>
              </ul>

              <p>
                <strong>结果</strong>：即便 Databasus 被攻破、服务器被入侵、
                密钥被盗、凭据被解密，攻击者也破坏不了你的数据库。
              </p>

              <h2 id="two-factor-authentication">登录的双因素验证</h2>

              <p>
                仅凭密码就能拿到实例保管的所有数据库凭据、存储密钥和备份，因此管理员可以为密码登录要求第二重验证。启用后，密码正确会把六位验证码发到账号邮箱，只有输入该验证码才算完成登录。验证码在发送十分钟后或五次错误尝试后失效，重新索取会让上一个验证码作废。启用该设置只保护下一次登录，已经打开的会话仍可继续使用。
              </p>

              <p>
                该设置只覆盖密码登录。通过 Google 或 GitHub
                登录仍然直接签发令牌，因为这些服务商有自己的多因素验证；想让第二重验证覆盖所有人的实例，干脆不要配置它们。
              </p>

              <p>
                只有实例配置了邮件服务器、并且每个在用的管理员都填了真实邮箱时才能启用，免得有人打开开关把自己关在门外。如果邮件之后无法送达，谁都无法用密码登录，这是故意的：坏掉的邮件服务器不该悄悄把登录退回到单因素。有服务器访问权限的运维可以再把它关掉：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{disableTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={disableTwoFactorCommand} />
                </div>
              </div>

              <p>
                命令会说明它改了什么，并把这次改动写入审计日志，于是不再要求验证码的实例能说清原因。它和密码找回一起记录在
                <a
                  href="/zh/password#disable-two-factor"
                  className="text-blue-400 hover:text-blue-600"
                >
                  密码页面
                </a>
                上。
              </p>

              <h2 id="security-and-reliability-engineering">
                🛡️ 安全与可靠性工程
              </h2>

              <p>
                Databasus 处理敏感数据。每次发布前，自动检查都会查找安全问题，并验证备份能否恢复。这些检查可以降低风险，但不能保证软件完全没有漏洞。
              </p>

              <h3 id="static-analysis">静态分析</h3>

              <p>
                CodeQL 检查 Go、JavaScript/TypeScript 和 GitHub Actions 工作流。gitleaks 在本次发布的 Git
                历史中查找泄露的密钥等敏感信息。semgrep 根据安全规则检查源代码、Dockerfile 和 CI
                工作流。这些扫描在每次发布前运行，也会每周定时运行。
              </p>

              <p>
                发布分析中尚未处理的 <strong>CRITICAL</strong> 级 CodeQL 问题会阻止发布。<strong>HIGH</strong>{" "}
                级问题会生成警告。
              </p>

              <p>
                OpenAI 的 Codex Security 还会定期开展更深入的审计。这些审计有助于发现专项扫描可能遗漏的架构问题，与自动发布检查分开进行。
              </p>

              <h3 id="dependency-management">依赖管理</h3>

              <p>
                常规更新需要等待发布满七天，适用于软件包、开发工具和 GitHub
                Actions。镜像、数据库客户端和独立验证扩展需要等待满十四天。维护者会检查发布日期和兼容性，包括可能绕过版本年龄过滤的精确版本选择和安全更新。例外情况会说明原因。
              </p>

              <p>
                Dependency Review Action 将候选版本与上一次发布比较，阻止引入新的 <strong>HIGH</strong> 或{" "}
                <strong>CRITICAL</strong> 级漏洞。当我们的 Go 代码可能调用存在已知漏洞的函数时，govulncheck 会使 Go
                检查失败。<code>go mod verify</code> 命令将下载的 Go 模块与记录的校验和进行比对。
              </p>

              <p>
                发布正式版本或开发镜像前，<code>mise run verify-deps</code> 会检查工具、镜像和 GitHub Actions
                的固定版本。它还会检查相关运行环境的版本是否一致，以及仓库中数据库客户端的哈希值和可执行权限。这些检查只覆盖指定的本地版本声明和文件。包管理器会另行检查锁定的版本及下载软件包的校验和。
              </p>

              <p>
                本地依赖检查不能证明第三方代码安全，也不会为每个依赖强制执行最短发布等待期。版本选择与审查仍由维护者负责。
              </p>

              <h3 id="container-and-ci-hardening">容器与 CI 安全</h3>

              <ul>
                <li>
                  Trivy 在 amd64 和 arm64 上扫描应用镜像。有可用修复的 <strong>CRITICAL</strong>{" "}
                  级漏洞会使检查失败。有修复的 <strong>HIGH</strong> 级漏洞会列入报告，但不会阻止检查通过。尚无修复的漏洞不在此项检查范围内。
                </li>
                <li>
                  独立的 Trivy 配置扫描会阻止应用、验证代理和指定测试镜像的 Dockerfile 中出现 <strong>HIGH</strong> 或{" "}
                  <strong>CRITICAL</strong> 级问题。
                </li>
                <li>
                  第三方 GitHub Actions 固定到完整的提交 SHA。这样，即使标签指向发生变化，也不会悄悄替换实际运行的 action 代码。
                </li>
                <li>
                  工作流默认只有仓库读取权限。各项任务仅在需要时获得额外权限。
                </li>
              </ul>

              <h3 id="testing-and-verification">测试与验证</h3>

              <p>
                单元测试和集成测试覆盖关键行为。每次发布前，都会针对每个受支持的数据库引擎和主要版本，在真实数据库容器中运行完整的备份与恢复流程。测试验证的是数据能否恢复，而不只是是否生成了备份文件。
              </p>

              <p>
                发布检查还包括 lint、类型检查、浏览器测试、镜像冒烟测试，以及在 amd64 和 arm64
                上进行的原生构建与测试。只有所有必需任务通过后才会发布，其中也包括上述安全检查。
              </p>

              <h3 id="reporting-a-vulnerability">报告漏洞</h3>

              <p>
                请通过{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>{" "}
                 中列出的联系方式私下报告漏洞，不要创建公开 issue。安全报告会优先处理。
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
