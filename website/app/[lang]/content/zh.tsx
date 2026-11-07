import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import HeroStatsComponent from "@/app/components/HeroStatsComponent";
import InstallationComponent from "@/app/components/InstallationComponent";
import LiteYouTubeEmbed from "@/app/components/LiteYouTubeEmbed";

export const metadata: Metadata = {
  title: "PostgreSQL 备份 | Databasus",
  description:
    "免费开源的 PostgreSQL 定时备份工具（同时支持 MySQL 和 MongoDB）。备份可保存到本地和云端。支持 Slack、Discord、Telegram、邮件、webhook 等多种通知方式。",
  keywords:
    "PostgreSQL、备份、监控、数据库、定时备份、Docker、自托管、开源、S3、Google Drive、Slack 通知、Discord、DevOps、数据库监控、pg_dump、数据库恢复、加密、AES-256、备份加密",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("zh", ""),
    languages: getLanguageAlternates(""),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("zh", ""),
    title: "PostgreSQL 备份 | Databasus",
    description:
      "免费开源的 PostgreSQL 定时备份工具（同时支持 MySQL 和 MongoDB）。备份可保存到本地和云端。支持 Slack、Discord、Telegram、邮件、webhook 等多种通知方式。",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Databasus 备份管理控制台界面",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary_large_image",
    title: "PostgreSQL 备份 | Databasus",
    description:
      "免费开源的 PostgreSQL 定时备份工具（同时支持 MySQL 和 MongoDB）。备份可保存到本地和云端。支持 Slack、Discord、Telegram、邮件、webhook 等多种通知方式。",
    images: ["https://databasus.com/images/index/dashboard.png"],
  },
  applicationName: "Databasus",
  appleWebApp: {
    title: "Databasus",
    capable: true,
  },
  icons: {
    icon: [
      { url: "/favicon.ico", type: "image/x-icon" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: "/favicon.svg",
    shortcut: "/favicon.ico",
  },
};

export default function Index() {
  return (
    <div className="overflow-x-hidden">
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            name: "Databasus",
            description:
              "免费开源的 PostgreSQL 定时备份工具（同时支持 MySQL 和 MongoDB）。备份可保存到本地和云端。支持 Slack、Discord、Telegram、邮件、webhook 等多种通知方式。",
            url: "https://databasus.com",
            image: "https://databasus.com/images/index/dashboard.png",
            logo: "https://databasus.com/logo.svg",
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
            featureList: [
              "PostgreSQL 定时备份",
              "多种存储目标（S3、Google Drive、Dropbox、SFTP、rclone 等）",
              "实时通知（Slack、Telegram、Discord、Webhook、邮件等）",
              "数据库健康监控",
              "通过 Docker 自托管部署",
              "开源且免费",
              "支持 PostgreSQL 12-18",
              "备份压缩与 AES-256-GCM 加密",
              "支持 PostgreSQL、MySQL、MariaDB 和 MongoDB",
              "保留策略：按时间段、数量、GFS 和大小限制",
              "基于 WAL 归档的时间点恢复（PITR）",
              "恢复验证：在真实数据库 Docker 容器中自动测试恢复",
            ],
            screenshot: "https://databasus.com/images/index/dashboard.png",
            softwareVersion: "latest",
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "Organization",
            name: "Databasus",
            url: "https://databasus.com/",
            alternateName: ["databasus", "Databasus"],
            logo: "https://databasus.com/logo.svg",
            sameAs: ["https://github.com/databasus/databasus"],
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: "Databasus",
            alternateName: ["databasus", "Databasus"],
            url: "https://databasus.com/",
            description: "PostgreSQL 备份工具",
            publisher: { "@type": "Organization", name: "Databasus" },
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Databasus 是什么？为什么应该用它而不是自己写脚本？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 是一个采用 Apache 2.0 许可证的自托管服务，用于备份 PostgreSQL v13 到 v18。它与 shell 脚本的区别在于：它提供了前端界面，可以调度任务、压缩归档并存储到多个目标（本地磁盘、S3、Google Drive、Dropbox、SFTP、rclone 等）、配置保留策略来自动清理旧备份，并在任务完成或失败时通知你的团队，这一切都无需手写代码",
                },
              },
              {
                "@type": "Question",
                name: "怎样以最快的方式安装 Databasus？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "最直接的方式是运行单行 cURL 安装脚本。它会拉取最新的 Docker 镜像，启动一个 PostgreSQL 容器，然后创建 docker-compose.yml 并启动服务，服务器重启后也会自动重新启动。在一台普通的 VPS 上，整个过程通常不到两分钟。",
                },
              },
              {
                "@type": "Question",
                name: "恢复验证是如何工作的？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 会在你控制的主机上运行一个小型验证代理。每次按计划运行时，代理会下载最新的备份，把它恢复到一个一次性的数据库容器中，然后将恢复后的数据库与源库做完整性核对。结果会被回传，包括恢复的退出码和每张表的行数。计划支持备份完成后、每小时、每天、每周、每月或 UTC cron 表达式。失败可以通过连接到该数据库的任意通知渠道发送，如 Slack、Teams、Discord、邮件等。",
                },
              },
              {
                "@type": "Question",
                name: "我的备份存放在哪里？会占用多少空间？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "归档可以保存到本地卷、S3 兼容存储桶、Google Drive、Dropbox 和其他云端目标。Databasus 采用均衡压缩，通常能把转储文件缩小 4-8 倍，而运行时开销只增加约 20%，从而节省存储和带宽。",
                },
              },
              {
                "@type": "Question",
                name: "我如何知道备份成功了，或者更糟的是失败了？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 可以通过邮件、Slack、Telegram、webhook、Mattermost、Discord 等方式实时通知。你可以自由选择通知渠道，让你的 DevOps 团队实时了解成功与失败，使恢复流程和合规审计更轻松。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 如何保证安全性？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 从三个层面保障安全：（1）敏感数据加密——所有密码、令牌和凭据均使用 AES-256-GCM 加密，并与数据库分开存储；（2）备份加密——每个备份文件都使用由主密钥、备份 ID 和随机盐派生出的唯一密钥加密，即使有人获得了存储的访问权限，没有你的加密密钥备份也毫无用处；（3）只读数据库访问——Databasus 只需要 SELECT 权限，并会做全面检查确保不存在任何写权限，即使工具本身被攻破也不会破坏数据。登录本身也可以要求第二重验证：启用后，密码正确之后账号邮箱会收到六位验证码，而通过 Google 或 GitHub 登录仍然依赖这些服务商自己的验证。在运行时之外，安全性和可靠性也被落实到每一次发布：CodeQL 静态分析、gitleaks 密钥扫描、semgrep 安全规则检查、Dependabot CVE 监控、Trivy 镜像与 Dockerfile 扫描，以及 OpenAI 定期进行的 Codex Security 审计。集成测试针对真实的 PostgreSQL、MySQL、MariaDB 和 MongoDB 容器运行，并在每次发布前验证完整的备份加恢复流程。GitHub Actions 固定到提交 SHA，工作流遵循最小权限原则。所有操作都在你控制的容器和你拥有的服务器上运行，而且因为它是开源的，你的安全团队可以在部署前审计每一行代码。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 得到了 Anthropic 和 OpenAI 开源计划的支持吗？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "是的，2026 年 3 月，Databasus 同时入选了 Anthropic 的 Claude for Open Source 和 OpenAI 的 Codex for Open Source。该项目经过独立评估，被行业领军企业认可为值得支持的关键开源基础设施。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 与 PgBackRest、Barman 或 pg_dump 有什么不同？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 追求简单：它提供现代化的 Web 界面来同时管理多个数据库的备份，内置调度、压缩、多存储目标、健康监控和实时通知。同时，与 pgBackRest 和 WAL-G 不同，Databasus 基于 PostgreSQL 17 的原生机制来做物理备份、增量备份和 WAL 备份，不去重新发明自己的备份引擎。它以远程方式连接你的数据库，可通过到服务器或堡垒机的 SSH 隧道进入封闭网络，因此未公开暴露的数据库同样可以在一个控制台上完成备份和管理。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 支持哪些数据库？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 支持 PostgreSQL、MySQL、MariaDB 和 MongoDB。不过 Databasus 最初就是专为 PostgreSQL 打造的，并始终以它为核心，为 PostgreSQL 备份提供 100% 完善的支持和最高的效率。虽然也支持 MySQL、MariaDB 和 MongoDB，但 PostgreSQL 仍是首要重点，功能优化最充分，开发也在持续推进。例如，Databasus 为 PostgreSQL 灾难恢复原生支持物理备份和 WAL 备份。所以 Databasus 本质上是 PostgreSQL 备份工具，其他数据库只是扩展。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 的采用程度如何？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 是当今使用最广泛的开源 PostgreSQL 备份工具。截至 2026 年 6 月 17 日，它在 Docker 上已被全球的 DBA、DevOps 工程师、开发者和团队拉取超过 180 万次。它拥有 8,500+ GitHub star，超过了 pgBackRest（约 4,200 star，2014 年发布）和 WAL-G（约 4,100 star，2017 年发布）。Databasus 于 2025 年发布，在第一年内就超越了两者。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 支持哪些备份类型？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 支持物理备份、全量备份、增量备份、WAL 备份和逻辑备份。物理备份是对整个数据库集群的文件级复制，对大数据集来说备份和恢复都比逻辑转储更快，并且构建在 PostgreSQL 17 的原生备份机制之上，我们依赖 PostgreSQL 自身久经考验的工具，而不是重新造轮子。全量备份是集群的完整、自包含副本，是每条备份链的起点。增量备份只存储自上次备份以来的变化，让备份保持小而快。WAL 流式复制持续捕获数据库的写入流，支持时间点恢复（PITR），用于灾难恢复并将数据丢失降到接近于零。逻辑备份是数据库引擎专用二进制格式的原生转储，压缩后直接流式写入存储，没有中间文件。如果你要求非公开连接，所有这些备份都可以通过 SSH 隧道运行，数据库永远无需公开暴露。SSH 隧道是内置功能。",
                },
              },
              {
                "@type": "Question",
                name: "我忘记了管理员邮箱或密码",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: '实例上创建的第一个账户就是它的管理员。如果你不记得那个地址，可以在运行 Databasus 的服务器上执行 docker exec -it databasus ./main --list-admins：它会列出每个管理员账户的邮箱、显示名称、创建日期和活动状态，并且不会输出任何密码。要为该账户设置新密码，请执行 docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"。在第一个账户成为管理员之前创建的实例，在其所有者替换之前仍使用占位地址 admin。如果实例还要求登录验证码，而邮件服务器已经无法送达，请执行 docker exec -it databasus ./main --disable-2fa，让实例不再要求验证码。这三条命令都记录在密码页面上。',
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="zh"
        sectionLinks={[
          { href: "#how-to-use", label: "使用方法" },
        ]}
      />

      {/* MAIN SECTION */}
      <main className="relative overflow-hidden pt-[60px] md:pt-[68px]">
        <div className="relative mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px] px-4 md:px-6 lg:px-0 pt-12 md:pt-[100px] pb-12 md:pb-[100px]">
          {/* Background ellipse */}
          <div className="relative">
            <div className="absolute left-1/2 -translate-x-1/2 -translate-y-1/4 w-[400px] h-[400px] md:w-[900px] md:h-[900px] bg-[#155dfc]/4 top-0 rounded-full blur-3xl -z-10" />
          </div>

          {/* Content */}
          <div className="text-center mb-8 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Databasus</span>
            </div>

            <h1 className="text-2xl sm:text-4xl sm:max-w-[300px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[650px]">
              PostgreSQL 备份工具，支持时间点恢复与恢复验证
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus 是一款免费、开源、可自托管的 PostgreSQL
              备份工具。支持多种存储（S3、Google Drive、FTP
              等）并可将进度通知发送到 Slack、Discord、Telegram 等渠道。专注于{" "}
              <span className="underline decoration-2 underline-offset-2 decoration-blue-600">
                低 RPO/RTO
              </span>{" "}
              的时间点恢复
            </p>

            <div>
              <div className="flex flex-col items-center justify-center gap-2 max-w-[370px] sm:max-w-[340px] mx-auto">
                <a
                  href="#installation"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-1"
                >
                  通过 Docker 自托管部署
                </a>

                <a
                  href="https://github.com/databasus/databasus"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-2 sm:order-2"
                >
                  <svg
                    aria-hidden={true}
                    width="24"
                    height="24"
                    viewBox="0 0 20 20"
                    fill="none"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <g clipPath="url(#clip0_1_2459)">
                      <path
                        fillRule="evenodd"
                        clipRule="evenodd"
                        d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                        fill="white"
                      />
                    </g>
                    <defs>
                      <clipPath id="clip0_1_2459">
                        <rect width="20" height="20" fill="white" />
                      </clipPath>
                    </defs>
                  </svg>

                  <span>GitHub</span>
                </a>
              </div>

              <div className="mt-2 max-w-[370px] sm:max-w-[340px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
                <HeroStatsComponent />
              </div>
            </div>
          </div>

          {/* Dashboard Screenshot */}
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Databasus 控制台界面"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
            </div>
          </div>

          <div className="mt-10 md:mt-15 mb-12 md:mb-20 flex justify-center px-4 md:px-0">
            <div className="flex flex-col md:flex-row items-center">
              <img
                className="h-[45px] md:h-[55px]"
                src="/images/index/ais.svg"
                alt="获得 Anthropic 和 OpenAI 开源计划支持"
              />

              <div className="flex justify-center text-base md:text-xl mt-4 md:mt-0 md:ml-10">
                <div className="max-w-[370px] text-gray-400 text-center md:text-left">
                  同时获得 Anthropic 与 OpenAI 开源计划的支持。{" "}
                  <a
                    href="/zh/faq#oss-programs"
                    target="_blank"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    了解更多&nbsp;→
                  </a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* FEATURES OVERVIEW SECTION */}
      <section id="features" className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">概览</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              功能特性
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus
              提供可靠的生产环境备份管理所需的一切，从自动调度到备份加密。既适合有个人项目的独立开发者，也适合
              DevOps 团队和企业
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          {/* Feature Cards Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 border border-[#ffffff20] rounded-xl">
            {/* Card 1: Scheduled backups */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                1
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                定时备份
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="定时备份"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                备份应当在指定时间定期执行。因此我们提供多种选项：每小时、每天、每周、每月、cron
                等
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                可配置的健康检查
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="健康检查"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                系统每分钟（或你设定的任意间隔）ping
                一次你的数据库，并展示检测历史
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                比如连续 3
                次失败后即可判定数据库宕机。一旦数据库恢复正常，你同样会收到通知
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                多种存储目标
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                文件可存放在 VPS、云存储等多种位置。你可以任选存储方式，
                文件始终归你所有。{" "}
                <a
                  href="/zh/storages"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  查看全部 →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="存储"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 4: Notifications */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                4
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                通知
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                你可以接收备份成功或失败的通知，对开发者和 DevOps 团队都很实用。{" "}
                <a
                  href="/zh/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  查看全部 →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="通知"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 5: Self hosted via Docker */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                5
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                通过 Docker 自托管
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus 运行在你自己的 PC 或 VPS
                上，因此所有数据由你掌控并受到保护。通过脚本、Docker 或 k8s
                部署大约只需 2 分钟
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Docker"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 6: Open source and free */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                6
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                开源且免费
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                项目完全开源、免费，采用 Apache 2.0 许可证。你可以自由复制和
                fork 代码，同样对企业开放
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="GitHub"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 7: Restore verification - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Restore verification */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  恢复验证
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  备份顺利完成并不等于备份能够恢复。Databasus
                  会定期拉取最新备份，将其恢复到一次性数据库容器中并报告结果。{" "}
                  <a
                    href="/zh/restore-verification"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    阅读更多 →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-postgresql.svg"
                    alt="PostgreSQL"
                    loading="lazy"
                  />
                </div>
              </div>

              {/* Card 10: Security - Only visible on desktop, merged with card 7 */}
              <div className="hidden lg:block p-5 md:p-6">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  10
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  安全性
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  企业级加密保护敏感数据和备份，只读数据库访问防止数据损坏。一切开箱即用，无需额外配置。{" "}
                  <a
                    href="/zh/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    阅读更多 →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="安全性"
                    loading="lazy"
                  />
                </div>
              </div>
            </div>

            {/* Card 8: Access management */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  8
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  访问管理
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  面向团队
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="访问管理"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                为用户分配查看或管理数据库的权限，将团队和项目分开管理，适合
                DevOps 团队和开发者。{" "}
                <a
                  href="/zh/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  阅读更多 →
                </a>
              </p>
            </div>

            {/* Card 9: Audit logs */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  9
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  审计日志
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  面向团队
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="审计日志"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                通过完整的审计日志追踪系统的所有活动。你可以查看每个用户的访问和变更历史（备份下载、计划变更、配置更新等）。{" "}
                <a
                  href="/zh/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  阅读更多 →
                </a>
              </p>
            </div>

            {/* Card 10: Security - Mobile/Tablet only */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                安全性
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                企业级加密保护敏感数据和备份，只读数据库访问防止数据损坏。一切开箱即用，无需额外配置。{" "}
                <a
                  href="/zh/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  阅读更多 →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="安全性"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 11: Backup types and modes */}
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>

              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  逻辑、物理、增量与 WAL 备份
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus 支持逻辑备份、物理备份（全量和增量），并通过 WAL
                  流式复制实现时间点恢复。这使 Databasus
                  适用于灾难恢复，对自托管数据库和云数据库同样适用；对云托管或可公开访问的数据库可使用远程模式。物理备份基于
                  PG 17 的原生备份机制实现。{" "}
                  <a
                    href="/zh/faq/#pitr"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    阅读更多 →
                  </a>
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* VIDEO SECTION */}
      <section className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0" id="how-to-use">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[450px] lg:shrink-0">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">4 分钟概览</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                如何使用 Databasus？
              </h2>

              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                在这段视频中你将看到：如何连接数据库、如何配置备份计划、如何下载和恢复备份、如何添加团队成员，以及什么是用户审计日志
              </p>

              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                className="flex items-center gap-3 md:gap-4 hover:opacity-70 transition-colors"
              >
                <img
                  src="/images/index/rostislav.png"
                  alt="Rostislav Dugin"
                  className="w-10 h-10 md:w-12 md:h-12 rounded-full object-cover"
                  loading="lazy"
                />

                <div>
                  <p className="font-medium text-base md:text-lg">
                    Rostislav Dugin
                  </p>
                  <p className="text-sm text-gray-400">Databasus 开发者</p>
                </div>
              </a>
            </div>

            {/* Right side: Video */}
            <div className="flex-1 relative">
              <div className="rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
                <LiteYouTubeEmbed
                  videoId="1qsAnijJfJE"
                  title="How to use Databasus (overview)?"
                  thumbnailSrc="/images/index/how-to-use-preview.svg"
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* Databases section */}
      <section className="pt-12 md:pt-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">数据库</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              支持的数据库
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Databasus 支持 PostgreSQL、MySQL、MariaDB 和
              MongoDB，你可以用同一个工具完成它们的备份与恢复。核心重点是
              PostgreSQL，同时也支持 MySQL、MariaDB 和 MongoDB
            </p>
          </div>

          {/* Databases list */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 border border-[#ffffff20] rounded-xl">
            {/* PostgreSQL */}
            <div className="border-b md:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-postgresql.svg"
                    alt="PostgreSQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                PostgreSQL
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                PostgreSQL 是 Databasus 的核心支持数据库，支持从 12 到 18
                的所有版本
              </p>
            </div>

            {/* MySQL */}
            <div className="border-b lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="MySQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MySQL
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MySQL 是全球第二流行的数据库。你可以同样轻松地备份和恢复 MySQL
                数据库
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/zh/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  阅读更多 →
                </a>
              </div>
            </div>

            {/* MariaDB */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mariadb.svg"
                    alt="MariaDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MariaDB
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MariaDB 拥有与 MySQL 相同的功能支持，你可以顺畅地备份和恢复
                MariaDB 数据库
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/zh/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  阅读更多 →
                </a>
              </div>
            </div>

            {/* MongoDB */}
            <div className="p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="MongoDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MongoDB
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MongoDB 是最流行的 NoSQL
                数据库。你可以用同样易用的界面备份和恢复 MongoDB 数据库
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/zh/mongodb-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  阅读更多 →
                </a>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* PROCESS SECTION */}
      <section className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">流程</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              如何创建数据库备份？
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Databasus 的首要目标是简单。目前它是全世界最容易上手的 PostgreSQL
              备份工具。创建备份只需 4 步，之后即可一键恢复
            </p>
          </div>

          {/* Steps */}
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            {/* Step 1 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                第 1 步
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  选择所需的备份计划
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    你可以任选时间：每天、每周、每月、指定时刻（比如凌晨 4
                    点）或 cron 周期
                  </p>
                  <p>
                    按周执行需要指定具体的星期几，按月执行需要指定具体的日期
                  </p>
                  <p>如果数据库较大，我们建议选择流量低谷时段执行备份</p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Step 1"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 2 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                第 2 步
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  填写数据库信息
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    输入 PostgreSQL 数据库的连接凭据，选择版本和目标
                    DB，并选择是否需要 SSL
                  </p>
                  <p>
                    Databasus
                    默认以均衡级别压缩备份，既不会明显拖慢备份过程（约慢
                    20%），又能节省 4-8 倍的空间（同时减少网络流量）
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-2.svg"
                  alt="Step 2"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 3 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                第 3 步
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  选择备份的存储位置
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    备份文件可以保存在本地、S3、Google Drive、NAS、Dropbox
                    等服务中
                  </p>
                  <p>请注意，存储位置需要有足够的空间</p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-3.svg"
                  alt="Step 3"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 4 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                第 4 步
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  选择接收通知的渠道（可选）
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    当备份成功或失败时，Databasus 可以向你发送通知，可以是
                    DevOps 群聊、你的邮箱，甚至是团队的 webhook
                  </p>
                  <p>我们计划支持绝大多数主流即时通讯工具和平台</p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-4.svg"
                  alt="Step 4"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
          </div>

          {/* CTA Button */}
          <div className="text-center mt-8 md:mt-12">
            <a
              href="#installation"
              className="inline-flex items-center gap-2 px-6 py-3 bg-white text-black rounded-lg text-[15px] font-medium hover:opacity-70 transition-colors"
            >
              开始使用
              <svg
                aria-hidden={true}
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
          </div>
        </div>
      </section>

      {/* INSTALLATION SECTION */}
      <section id="installation" className="px-4 md:px-6 lg:px-0">
        <div className="max-w-[1000px] 2xl:max-w-[1200px] mx-auto border border-[#ffffff20] rounded-xl py-10 md:py-20 px-4 md:px-6">
          <div className="max-w-[1100px] mx-auto">
            <div className="text-center mb-8 md:mb-10">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">开始使用</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                如何安装？
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus
                支持多种安装方式，本地和云端均可。两种方式都极其简单，即使没有任何运维或
                DevOps 经验也能轻松上手
              </p>
            </div>

            <InstallationComponent lang="zh" />
          </div>
        </div>
      </section>

      {/* FAQ SECTION */}
      <section id="faq" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-8 md:mb-12">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">FAQ</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              常见问题
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Databasus 的目标是让备份对独立开发者（以及
              DevOps）和团队都尽可能简单。界面让创建备份变得轻松，直观展示进度，几次点击即可恢复任何内容
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Databasus 是什么？为什么应该用它而不是自己写脚本？"
              answer="Databasus 是一个采用 Apache 2.0 许可证的自托管数据库备份服务。它与 shell 脚本的区别在于：它提供了前端界面，可以调度任务、压缩归档并存储到多个目标（本地磁盘、S3、Google Drive、NAS、Dropbox、SFTP、rclone 等）、配置保留策略来自动清理旧备份，并在任务完成或失败时通知你的团队，这一切都无需手写代码"
            />
            <FaqItem
              number="2"
              question="怎样以最快的方式安装 Databasus？"
              answer="Databasus 支持多种安装方式：自动脚本、Docker、Docker Compose 以及使用 Helm 的 Kubernetes。最快的方式是运行单行 cURL 安装脚本，它会拉取最新的 Docker 镜像，创建 docker-compose.yml 并启动服务，服务器重启后也会自动重启。在 Kubernetes 环境中，可使用官方 Helm chart 进行生产级部署。在一台普通的 VPS 上，整个过程通常不到两分钟。"
            />
            <FaqItem
              number="3"
              question="恢复验证是如何工作的？"
              answer="Databasus 会在你控制的主机上运行一个小型验证代理。每次按计划运行时，代理会下载最新的备份，把它恢复到一个一次性的数据库容器中，然后将恢复后的数据库与源库做完整性核对。结果会被回传，包括恢复的退出码和每张表的行数。计划支持备份完成后、每小时、每天、每周、每月或 UTC cron 表达式。失败可以通过连接到该数据库的任意通知渠道发送，如 Slack、Teams、Discord、邮件等。"
            />
            <FaqItem
              number="4"
              question="Databasus 如何保证安全性？"
              answer={
                <>
                  Databasus
                  从三个层面保障安全：（1）敏感数据加密：所有密码、令牌和凭据均使用
                  AES-256-GCM
                  加密，并与数据库分开存储；（2）备份加密：每个备份文件都使用由主密钥、备份
                  ID
                  和随机盐派生出的唯一密钥加密，即使有人获得了存储的访问权限，没有你的加密密钥备份也毫无用处；（3）只读数据库访问：Databasus
                  只需要 SELECT
                  权限，并会做全面检查确保不存在任何写权限，即使工具本身被攻破也不会破坏数据。
                  <br />
                  <br />
                  登录本身也可以要求第二重验证：启用后，密码正确之后账号邮箱会收到六位验证码，而通过
                  Google 或 GitHub 登录仍然依赖这些服务商自己的验证。
                  <br />
                  <br />
                  在运行时之外，安全性和可靠性也被落实到每一次发布：CodeQL 静态分析、gitleaks 密钥扫描、semgrep
                  安全规则检查、Dependabot CVE 监控、Trivy 镜像与 Dockerfile 扫描，以及 OpenAI
                  定期进行的 Codex Security 审计。集成测试针对真实的 PostgreSQL、MySQL、MariaDB 和
                  MongoDB 容器运行，并在每次发布前验证完整的备份加恢复流程。GitHub Actions 固定到提交
                  SHA，工作流遵循最小权限原则。
                  <br />
                  <br />
                  完整流程请参阅{" "}
                  <a
                    href="/zh/security#security-and-reliability-engineering"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    安全与可靠性工程
                  </a>
                  。
                  <br />
                  <br />
                  此外，所有日志都可以通过 OpenTelemetry
                  标准导出到任意外部系统（VictoriaLogs、SigNoz、Graylog
                  等）。默认情况下，日志（包括审计日志）也会写入本地文件，因此审计日志不会丢失。详见
                  <a
                    href="/zh/advanced-config#logging"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    高级配置
                  </a>
                  。
                </>
              }
            />
            <FaqItem
              number="5"
              question="如何在 Databasus 中设置并运行我的第一个备份任务？"
              answer={
                <>
                  要开始你的第一次 Databasus 备份，只需登录控制台，点击 New
                  Backup，选择一个间隔：每小时、每天、每周、每月或
                  cron，然后指定确切的运行时间（例如 02:30 这样的低峰时段）。
                  <br />
                  <br />
                  接着填写 PostgreSQL 的主机、端口号、数据库名称、凭据和 SSL
                  选项，并选择归档的存放位置（本地路径、S3 存储桶、Google Drive
                  文件夹、Dropbox 等）。 <br />
                  <br />
                  如有需要，可添加邮件、Slack、Telegram 或 webhook
                  等通知渠道，然后点击 Save。Databasus
                  会立即校验信息、启动计划、执行首次任务并实时反馈状态。备份完成后，你即可一键恢复。
                </>
              }
            />
            <FaqItem
              number="6"
              question="Databasus 的采用程度如何？"
              answer="Databasus 是当今使用最广泛的开源 PostgreSQL 备份工具。截至 2026 年 6 月 17 日，它在 Docker 上已被全球的 DBA、DevOps 工程师、开发者和团队拉取超过 180 万次。它拥有 8,500+ GitHub star，超过了 pgBackRest（约 4,200 star，2014 年发布）和 WAL-G（约 4,100 star，2017 年发布）。Databasus 于 2025 年发布，在第一年内就超越了两者。"
            />
            <FaqItem
              number="7"
              question="Databasus 与 PgBackRest、Barman 或 pg_dump 有什么不同？在哪里可以看到对比？"
              answer={
                <>
                  Databasus 追求简单：它提供现代化的 Web
                  界面来同时管理多个数据库的备份，而不是复杂的配置文件和命令行工具。与原始的
                  pg_dump
                  脚本不同，它内置调度、压缩、多存储目标、健康监控和实时通知，一切都通过简洁的
                  Web 界面管理。
                  <br />
                  <br />
                  同时，与 pgBackRest 和 WAL-G 不同，Databasus 基于 PostgreSQL
                  17 的原生机制来做物理备份、增量备份和 WAL
                  备份，不去重新发明自己的备份引擎。它以远程方式连接你的数据库，可通过到服务器或堡垒机的
                  SSH
                  隧道进入封闭网络，因此未公开暴露的数据库同样可以在一个控制台上完成备份和管理。{" "}
                  <a
                    href="/zh/faq/#pitr"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    阅读物理备份与 PITR 备份的实现方式
                  </a>
                  。
                  <br />
                  <br />
                  我们为主流备份工具准备了详细的对比页面：{" "}
                  <a
                    href="/zh/pgdump-alternative"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pg_dump
                  </a>
                  、{" "}
                  <a
                    href="/zh/databasus-vs-pgbackrest"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackRest
                  </a>
                  、{" "}
                  <a
                    href="/zh/databasus-vs-barman"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs Barman
                  </a>
                  、{" "}
                  <a
                    href="/zh/databasus-vs-wal-g"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs WAL-G
                  </a>{" "}
                  和{" "}
                  <a
                    href="/zh/databasus-vs-pgbackweb"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackWeb
                  </a>
                  。每篇对比都解释了关键差异和优缺点，帮助你选择适合自己的工具。
                </>
              }
            />
            <FaqItem
              number="8"
              question="Databasus 得到了 Anthropic 和 OpenAI 开源计划的支持吗？"
              answer={
                <>
                  是的，我们很自豪 Databasus 被全球两家领先的 AI
                  公司认可为有价值的开源项目。2026 年 3 月，Databasus 同时入选了
                  Anthropic 的{" "}
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Claude for Open Source
                  </a>{" "}
                  和 OpenAI 的{" "}
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Codex for Open Source
                  </a>
                  。这对我们来说是一份独立的可靠性认证：项目经过评估，被认可为值得支持的关键基础设施。{" "}
                  <a
                    href="/zh/faq#oss-programs"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    阅读更多 →
                  </a>
                </>
              }
            />
            <FaqItem
              number="9"
              question="Databasus 是 pg_dump 的替代品吗？"
              answer="不完全是。Databasus 专注于低 RTO 和 RPO 的灾难恢复，因此更接近 pgBackRest 或 WAL-G 的替代品——它努力让灾难恢复像 pg_dump 一样简单。不过在逻辑备份方面，它确实可以作为 pg_dump 的替代品，并且底层就使用 pg_dump，在此之上增加了友好的 Web 界面、自动调度、多存储目标、实时通知、健康监控和备份加密。逻辑备份同样适用于 MySQL、MariaDB 和 MongoDB。"
            />
            <FaqItem
              number="10"
              question="Databasus 支持哪些数据库？"
              answer={
                <>
                  Databasus 支持 PostgreSQL、MySQL、MariaDB 和 MongoDB。不过
                  Databasus 最初就是专为 PostgreSQL 打造的，并始终以它为核心，为
                  PostgreSQL 备份提供 100% 完善的支持和最高的效率。
                  <br />
                  <br />
                  虽然也支持 MySQL、MariaDB 和 MongoDB，但 PostgreSQL
                  仍是首要重点，功能优化最充分，开发也在持续推进。
                  <br />
                  <br />
                  例如，Databasus 为 PostgreSQL 灾难恢复原生支持物理备份和 WAL
                  备份。所以 Databasus 本质上是 PostgreSQL
                  备份工具，其他数据库只是扩展。
                </>
              }
            />
            <FaqItem
              number="11"
              question="Databasus 支持哪些备份类型？"
              answer={
                <>
                  Databasus 支持物理备份、全量备份、增量备份、WAL
                  备份和逻辑备份，既适合只想要简单逻辑转储的用户，也适合需要可靠灾难恢复工具的用户。
                  <ul className="list-disc list-inside mt-3 space-y-2">
                    <li>
                      <strong>物理备份</strong>
                      ：对整个数据库集群的文件级复制。对大数据集来说，备份和恢复都比逻辑转储更快。构建在
                      PostgreSQL 17 的原生备份机制之上，我们依赖 PostgreSQL
                      自身久经考验的工具，而不是重新造轮子
                    </li>
                    <li>
                      <strong>全量备份</strong>
                      ：集群的完整、自包含副本，是每条备份链的起点
                    </li>
                    <li>
                      <strong>增量备份</strong>
                      ：只存储自上次备份以来的变化，让备份保持小而快
                    </li>
                    <li>
                      <strong>WAL 流式复制</strong>
                      ：持续捕获数据库的写入流，支持时间点恢复（PITR）。专为灾难恢复设计，将数据丢失降到接近于零
                    </li>
                    <li>
                      <strong>逻辑备份</strong>
                      ：数据库引擎专用二进制格式的原生转储。压缩后直接流式写入存储，没有中间文件
                    </li>
                  </ul>
                  <br />
                  物理备份、增量备份和 WAL 备份构建在 PostgreSQL 17
                  的原生机制之上，因此需要 PostgreSQL 17
                  或更高版本；更旧的版本只能使用逻辑备份。这是有意为之：大多数生产数据库已经运行在
                  PostgreSQL 17
                  及以上版本，而更旧的版本大约两年内就会到达生命周期终点。Databasus
                  的目标是成为 PostgreSQL 17 起的标准备份工具。
                  <br />
                  <br />
                  如果你要求非公开连接，所有这些备份都可以通过 SSH
                  隧道运行，数据库永远无需公开暴露。SSH 隧道是内置功能。
                </>
              }
            />
            <FaqItem
              number="12"
              question="如何加入 Databasus 社区？"
              answer={
                <>
                  你可以在{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    t.me/databasus_community
                  </a>{" "}
                  加入我们由开发者、DBA 和 DevOps
                  工程师组成的大型社区。在社区里可以提问、交流经验、获得配置帮助，并及时了解最新功能和版本发布。
                </>
              }
            />
            <FaqItem
              number="13"
              question="Databasus 的普及程度如何？"
              answer={
                <>
                  Databasus 拥有超过 180 万次 Docker 拉取和 8.5k GitHub
                  star。作为对比，pgBackRest 和 WAL-G 各约 4.2k star，Barman 约
                  3.1k star，这使 Databasus 成为 GitHub
                  上最受欢迎的数据库备份工具。
                  <br />
                  <br />
                  它作为重要的关键项目入选了 Anthropic 和 OpenAI
                  的开源计划。如今 Databasus 已被企业、团队和 DevOps
                  工程师使用，并有庞大而活跃的社区支持。
                  <br />
                  <br />
                  Databasus 自 2023 年起开发并投入使用，自 2025
                  年初起开源并被广泛使用。它已经在真实生产环境中运行了相当长的时间，经受了大量边界情况的考验。关键在于，Databasus
                  不会发明自定义的备份方式——它依赖 PostgreSQL
                  原生的、经过验证的实现，而不是为边界情况构建自己的变通方案。
                  <br />
                  <br />
                  我们的目标是成为 PostgreSQL 17
                  及以上版本的标准备份工具。Databasus 是第一个构建在 PostgreSQL
                  原生、高效且如今已成为标准的备份协议之上的备份工具，而不是编写自己的实现。
                </>
              }
            />
            <FaqItem
              number="14"
              question="我忘记了管理员邮箱或密码"
              answer={
                <>
                  实例上创建的第一个账户就是它的管理员。如果你不记得那个地址，可以在运行
                  Databasus 的服务器上执行 docker exec -it databasus ./main
                  --list-admins：它会列出每个管理员账户的邮箱、显示名称、创建日期和活动状态，并且不会输出任何密码。要为该账户设置新密码，请执行
                  docker exec -it databasus ./main
                  --new-password=&quot;YourNewSecurePassword123&quot;
                  --email=&quot;owner@example.com&quot;。在第一个账户成为管理员之前创建的实例，在其所有者替换之前仍使用占位地址
                  admin。如果实例还要求登录验证码，而邮件服务器已经无法送达，请执行
                  docker exec -it databasus ./main
                  --disable-2fa，让实例不再要求验证码。这三条命令都记录在
                  <a
                    href="/zh/password"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    密码页面
                  </a>
                  上。
                </>
              }
            />
          </div>
        </div>
      </section>

      <LandingFooterComponent lang="zh" />
    </div>
  );
}

function FaqItem({
  number,
  question,
  answer,
}: {
  number: string;
  question: string;
  answer: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-[#ffffff20] p-4 md:p-6">
      <div className="flex items-center justify-center w-6 h-6 rounded border border-[#ffffff20] text-sm font-semibold mb-3 md:mb-4">
        {number}
      </div>

      <h3 className="text-base md:text-lg font-bold mb-2 md:mb-3">
        {question}
      </h3>

      <div className="text-gray-400 text-sm md:text-base">{answer}</div>
    </div>
  );
}
