import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "MongoDB 备份",
  description:
    "免费开源的 MongoDB 定时备份工具。通过 Web 界面自动化 mongodump，将归档存放到 S3、Azure Blob Storage 或本地。通过 Slack、Discord、Telegram 接收通知。BSON 数据 AES-256 加密。",
  keywords:
    "MongoDB 备份、mongodump 替代方案、MongoDB 备份自动化、MongoDB 备份工具、MongoDB 定时备份、MongoDB 云备份、MongoDB S3 备份、MongoDB Docker 备份、MongoDB 备份加密、MongoDB Atlas 备份、副本集备份、文档数据库备份、BSON 备份、NoSQL 备份",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("zh", "mongodb-backup"),
    languages: getLanguageAlternates("mongodb-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("zh", "mongodb-backup"),
    title: "MongoDB 备份",
    description:
      "免费开源的 MongoDB 定时备份工具。通过 Web 界面自动化 mongodump，提供云存储、通知和加密。",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Databasus 控制台界面，展示 MongoDB 备份管理",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary_large_image",
    title: "MongoDB 备份",
    description:
      "免费开源的 MongoDB 定时备份工具。通过 Web 界面自动化 mongodump，提供云存储、通知和加密。",
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

export default function MongodbBackupPage() {
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
              "免费开源的 MongoDB 定时备份工具。通过 Web 界面自动化 mongodump，提供云存储、通知和加密。",
            url: "https://databasus.com/zh/mongodb-backup/",
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
              "通过 mongodump 定时备份 MongoDB",
              "多种存储目标（S3、Azure Blob Storage、FTP、SFTP、NAS 等）",
              "实时通知（Slack、Telegram、Discord、Webhook、邮件等）",
              "MongoDB 连接健康监控",
              "通过 Docker 自托管",
              "开源且免费",
              "支持 MongoDB 4、5、6、7、8 和 9",
              "使用 gzip 压缩 BSON 归档",
              "备份文件 AES-256-GCM 加密",
              "支持 MongoDB Atlas 和副本集",
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
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Databasus 是什么？它如何备份 MongoDB 数据库？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 是一款采用 Apache 2.0 许可证的自托管备份工具，底层使用 mongodump 创建一致的 MongoDB 备份。它在 mongodump 之上提供现代化 Web 界面、自动定时调度、云存储集成、实时通知和 AES-256-GCM 加密，让你无需再编写自定义 Shell 脚本。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 支持 MongoDB 副本集吗？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "支持。Databasus 完整支持 MongoDB 副本集。你可以使用标准的 MongoDB 连接 URI 格式连接副本集的任意成员。Databasus 会从指定节点读取数据，因此可以从从节点备份，减轻主节点的负载。",
                },
              },
              {
                "@type": "Question",
                name: "可以用 Databasus 备份 MongoDB Atlas 数据库吗？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "可以。Databasus 可以无缝备份 MongoDB Atlas。由于 Databasus 通过 mongodump 做逻辑备份，只需要标准的 MongoDB 连接凭据，不需要特殊的 Atlas 权限或角色。只需提供你的 Atlas 连接字符串，其余的交给 Databasus。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 支持哪些 MongoDB 版本？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 支持 MongoDB 4、5、6、7、8 和 9。所有备份都使用原生 mongodump 工具并带 --archive 和 --gzip 参数，生成高效压缩的 BSON 归档，可用 mongorestore 恢复。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 如何保护 MongoDB 凭据和备份的安全？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 采用多层安全机制：（1）所有 MongoDB 密码和连接字符串在存储前使用 AES-256-GCM 加密；（2）每个备份文件使用由主密钥、备份 ID 和随机盐派生的唯一密钥加密；（3）连接 URI 以安全方式传递给 mongodump，绝不出现在日志或命令行输出中。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 支持 MongoDB 增量备份吗？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 专注于使用 mongodump 的完整逻辑备份，而非增量备份。对大多数场景来说，定时完整备份（每小时、每天、每周）已能提供足够的恢复点。MongoDB Atlas 本身已提供原生的时间点恢复，而外部增量备份难以恢复到 Atlas 集群。",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="zh"
        sectionLinks={[
          { href: "#features", label: "功能" },
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

            <h1 className="text-2xl sm:text-4xl sm:max-w-[400px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[500px]">
              MongoDB 备份工具
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus 是一款免费、开源、可自托管的 MongoDB
              文档数据库备份工具。按计划自动执行 mongodump，将 BSON 归档存放到
              S3、Azure Blob Storage 或本地。备份完成后通过 Slack、Discord 或 Telegram
              接收通知
            </p>

            <div className="flex flex-col sm:flex-row sm:flex-wrap items-center justify-center gap-2 sm:gap-2 max-w-[400px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
              <a
                href="#installation"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-3"
              >
                通过 Docker 自托管
              </a>

              <a
                href="https://github.com/databasus/databasus"
                target="_blank"
                rel="noopener noreferrer"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-4 sm:order-4"
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
          </div>

          {/* Dashboard Screenshot */}
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="用于管理 MongoDB 备份的 Databasus 控制台界面"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
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
              MongoDB 备份功能
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus 在 mongodump 之上提供企业级能力：自动定时调度、
              云存储集成、实时通知和 AES-256-GCM 加密。适合管理 MongoDB
              文档数据库和集合的开发者与 DevOps 团队
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
                MongoDB 定时转储
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="MongoDB 定时备份配置"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                在应用负载较低的时段定时执行
                mongodump。可选择每小时、每天、每周、每月的间隔，也可以用 cron
                表达式做精确控制
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                MongoDB 健康监控
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="MongoDB 健康检查"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                通过可配置的健康检查监控 MongoDB
                连接可用性。数据库或副本集不可达时收到通知
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                可设置检查间隔（每分钟、每 5
                分钟等）以及标记数据库不可用前的失败次数阈值
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                将 BSON 归档存放到任何地方
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                将 MongoDB 备份归档存放在本地、S3 兼容存储、Azure Blob Storage、FTP、NAS 或其他目标。文档数据始终在你的掌控之中。{" "}
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
                  alt="MongoDB 备份存储目标"
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
                备份通知
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                MongoDB 备份完成或失败时收到提醒。将通知发送到 DevOps
                团队群聊或个人频道。{" "}
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
                  alt="MongoDB 备份通知"
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
                在自己的基础设施上运行 Databasus。所有 MongoDB
                连接字符串和备份数据都保存在你掌控的服务器上。通过脚本、Docker
                或 Kubernetes 约 2 分钟即可部署完成
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Docker 部署"
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
                Databasus 完全开源，采用 Apache 2.0
                许可证。你可以审阅每一行代码并 Fork
                它。个人和企业均可免费使用
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="GitHub 开源"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 7: Many MongoDB versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MongoDB versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  支持的 MongoDB 版本
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  支持 MongoDB 4、5、6、7、8 和 9。Databasus 为每个版本使用原生
                  mongodump 工具，确保与你的文档数据库完全兼容
                </p>

                <div>
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="MongoDB 版本"
                    className="w-[75px] h-[75px]"
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
                  MongoDB 连接字符串在存储前使用 AES-256-GCM 加密。每个 BSON
                  归档都使用唯一密钥加密。凭据以安全方式传递给
                  mongodump，绝不出现在日志中。{" "}
                  <a
                    href="/zh/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    了解更多 →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="MongoDB 备份安全"
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
                  alt="MongoDB 备份访问管理"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                控制谁可以查看或管理 MongoDB
                数据库。为不同项目创建工作区。分配查看者、编辑者或管理员角色。{" "}
                <a
                  href="/zh/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  了解更多 →
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
                  alt="MongoDB 备份审计日志"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                跟踪所有操作：备份下载、计划变更、配置更新。清楚记录谁在何时做了什么，
                便于合规审查和责任追溯。{" "}
                <a
                  href="/zh/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  了解更多 →
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
                MongoDB 连接字符串在存储前使用 AES-256-GCM 加密。每个 BSON
                归档都使用唯一密钥加密。凭据以安全方式传递给
                mongodump，绝不出现在日志中。{" "}
                <a
                  href="/zh/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  了解更多 →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="MongoDB 备份安全"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 11: Suitable for clouds */}
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>

              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  支持 MongoDB Atlas 和自托管部署
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus 可以连接云端托管的 MongoDB 数据库，包括 MongoDB
                  Atlas、AWS DocumentDB 以及自托管部署。由于它通过 mongodump
                  做逻辑备份，只需要标准的连接凭据，不需要特殊的云权限或文件系统访问
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MONGODUMP SECTION */}
      <section id="mongodump" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">基于 mongodump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                MongoDB 备份是如何工作的
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus 底层使用 <strong>mongodump</strong>，即 MongoDB
                  官方的备份工具。当你触发一次备份时，Databasus 会以优化参数执行
                  mongodump：
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --archive
                    </code>{" "}
                    输出单个 BSON 文件而非目录结构
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --gzip
                    </code>{" "}
                    压缩归档，减少存储和传输体积
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --db
                    </code>{" "}
                    备份 MongoDB 实例中的指定数据库
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --uri
                    </code>{" "}
                    安全地处理带认证的连接字符串
                  </li>
                </ul>

                <p className="text-gray-400">
                  备份流直接以管道传输到你配置的存储，并可在写入前使用
                  AES-256-GCM 加密。这种方式将磁盘 I/O
                  降到最低，对大集合也能高效工作。
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    支持的 MongoDB 版本：
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 6
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 8
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 9
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Right side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mongodb.svg"
                  alt="MongoDB 数据库"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  通过 mongodump 完成官方 MongoDB 数据库备份，支持 gzip
                  压缩、加密和云存储
                </p>
              </div>
            </div>
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
                Databasus 支持多种安装方式。可在 VPS、本地机器或 Kubernetes
                集群上约 2 分钟完成部署。同一套安装即可用于
                MongoDB、PostgreSQL、MySQL 和 MariaDB 的数据库备份
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
              <span className="text-sm font-medium">常见问题</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              MongoDB 备份常见问题
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              关于使用 Databasus 备份 MongoDB
              文档数据库的常见问题。如有其他问题，欢迎加入我们的 Telegram 社区
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Databasus 是什么？它如何备份 MongoDB 数据库？"
              answer="Databasus 是一款采用 Apache 2.0 许可证的自托管备份工具，底层使用 mongodump 创建一致的 MongoDB 备份。它在 mongodump 之上提供现代化 Web 界面、自动定时调度、云存储集成（S3、Azure Blob Storage、FTP）、实时通知（Slack、Discord、Telegram）和 AES-256-GCM 加密，让你无需再编写自定义 Shell 脚本和 cron 任务。"
            />
            <FaqItem
              number="2"
              question="Databasus 支持 MongoDB 副本集吗？"
              answer="支持。Databasus 完整支持 MongoDB 副本集。你可以使用带副本集选项的标准 MongoDB 连接 URI 格式连接副本集的任意成员。Databasus 会从指定节点读取数据，因此可以从从节点备份，减轻主节点的负载。这在生产环境中特别有用，可以避免影响主节点的性能。"
            />
            <FaqItem
              number="3"
              question="可以用 Databasus 备份 MongoDB Atlas 数据库吗？"
              answer="可以。Databasus 可以无缝备份 MongoDB Atlas。由于 Databasus 通过 mongodump 做逻辑备份，只需要标准的 MongoDB 连接凭据，除了将 Databasus 服务器加入 IP 白名单外，不需要特殊的 Atlas 权限或管理员角色。只需提供你的 Atlas 连接字符串（可在 Atlas 控制台获取），其余的交给 Databasus。"
            />
            <FaqItem
              number="4"
              question="Databasus 支持哪些 MongoDB 版本？"
              answer="Databasus 支持 MongoDB 4、5、6、7、8 和 9。所有备份都使用原生 mongodump 工具并带 --archive 和 --gzip 参数，生成高效压缩的 BSON 归档。归档可以用 mongorestore 恢复到任何兼容的 MongoDB 版本，版本间迁移也因此变得简单。"
            />
            <FaqItem
              number="5"
              question="Databasus 如何处理大型 MongoDB 集合？"
              answer="Databasus 将 mongodump 的输出直接以流的方式传输到你的存储目标，并可在传输过程中加密。这种方式无需在磁盘上写入临时文件，对包含大集合的数据库非常高效。--archive 参数生成单个压缩文件而非目录结构，减少了 I/O 开销，也让存储管理更简单。"
            />
            <FaqItem
              number="6"
              question="可以用 Databasus 备份 MongoDB 分片集群吗？"
              answer={
                <>
                  Databasus 目前专注于备份单个 MongoDB
                  数据库，而非协调式的分片集群备份。
                  <br />
                  <br />
                  对于分片集群，你可以：
                  <br />
                  <br />
                  • 连接各分片的副本集，逐个备份每个分片
                  <br />
                  • 通过 mongos 路由备份（不过这可能影响性能）
                  <br />
                  <br />
                  对于生产环境的分片集群，建议考虑 MongoDB Atlas
                  原生备份，或使用带 --oplog 的
                  mongodump，以保证跨分片的时间点一致性。
                </>
              }
            />
            <FaqItem
              number="7"
              question="Databasus 如何保护 MongoDB 凭据和备份的安全？"
              answer={
                <>
                  Databasus 采用多层安全机制：
                  <br />
                  <br />
                  <strong>1. 凭据加密：</strong>所有 MongoDB 连接
                  URI、密码和认证信息在存储前均使用 AES-256-GCM 加密。
                  <br />
                  <br />
                  <strong>2. 备份加密：</strong>每个 BSON 归档使用由主密钥、备份
                  ID 和随机盐派生的唯一密钥加密。
                  <br />
                  <br />
                  <strong>3. 安全的凭据处理：</strong>连接 URI
                  通过安全参数直接传递给 mongodump，绝不出现在日志或进程列表中。
                </>
              }
            />
            <FaqItem
              number="8"
              question="Databasus 支持 MongoDB 增量备份或 oplog 跟踪吗？"
              answer="Databasus 专注于使用 mongodump 的完整逻辑备份，而非增量备份或基于 oplog 的时间点恢复。对大多数场景来说，定时完整备份（每小时、每天、每周）已能提供足够的恢复点，也免去了管理 oplog 的复杂性。MongoDB Atlas 本身已提供带时间点恢复的原生持续备份，而外部增量备份难以恢复到 Atlas 集群。"
            />
            <FaqItem
              number="9"
              question="可以把 MongoDB 备份恢复到其他版本或集群吗？"
              answer="可以。Databasus 创建的是 BSON 格式的标准 mongodump 归档，可以恢复到任何兼容的 MongoDB 服务器：不同版本、不同云服务商或本地开发机。从 Databasus 下载备份（自动解密），然后使用带 --archive 和 --gzip 参数的 mongorestore 即可。Databasus 会为每个备份展示确切的恢复命令。"
            />
            <FaqItem
              number="10"
              question="Databasus 中的 mongodump 压缩是如何工作的？"
              answer="Databasus 使用 mongodump 内置的 --gzip 参数，在转储过程中压缩 BSON 数据。与未压缩的 BSON 相比，归档体积通常可减少 60-80%。压缩发生在 mongodump 数据流内、可选加密之前，因此压缩和加密后的归档都保持高效。使用带 --gzip 参数的 mongorestore 时会自动解压。"
            />
            <FaqItem
              number="11"
              question="可以只备份特定的 MongoDB 集合而不是整个数据库吗？"
              answer="目前 Databasus 备份的是整个 MongoDB 数据库，而非单个集合。这能保证备份完整且一致，包含所有集合、索引和元数据。如果需要集合级别的备份，可以按数据领域拆分成不同的数据库，并在 Databasus 中为每个数据库设置各自的备份计划。"
            />
            <FaqItem
              number="12"
              question="Databasus 支持运行在 Docker 或 Kubernetes 中的 MongoDB 吗？"
              answer="支持。Databasus 通过标准连接 URI 经网络连接 MongoDB，因此无论 MongoDB 部署在哪里都能使用：Docker 容器、Kubernetes Pod、虚拟机或裸机。只需保证 Databasus 与 MongoDB 实例之间的网络连通。对于 Kubernetes 部署，可以使用内部服务 DNS 名称或外部负载均衡器地址。"
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
