import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "MySQL 备份工具，支持 MariaDB",
  description:
    "免费开源的 MySQL 和 MariaDB 定时备份工具。mysqldump 的替代方案，提供 Web 界面、云存储（S3、Azure Blob Storage）、通知（Slack、Discord、Telegram）和 AES-256 加密。",
  keywords:
    "MySQL 备份、MariaDB 备份、mysqldump 替代方案、mysqldump GUI、MySQL 备份自动化、MySQL 备份工具、MariaDB 备份工具、MySQL 定时备份、MySQL 云备份、MySQL S3 备份、MySQL Docker 备份、MySQL 备份加密",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("zh", "mysql-backup"),
    languages: getLanguageAlternates("mysql-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("zh", "mysql-backup"),
    title: "MySQL 备份工具，支持 MariaDB",
    description:
      "免费开源的 MySQL 和 MariaDB 定时备份工具。mysqldump 的替代方案，提供 Web 界面、云存储、通知和加密。",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Databasus 控制台界面，展示 MySQL 备份管理",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary_large_image",
    title: "MySQL 备份工具，支持 MariaDB",
    description:
      "免费开源的 MySQL 和 MariaDB 定时备份工具。mysqldump 的替代方案，提供 Web 界面、云存储、通知和加密。",
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

export default function MysqlBackupPage() {
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
              "免费开源的 MySQL 和 MariaDB 定时备份工具。mysqldump 的替代方案，提供 Web 界面、云存储、通知和加密。",
            url: "https://databasus.com/zh/mysql-backup/",
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
              "MySQL 和 MariaDB 定时备份",
              "多种存储目标（S3、Azure Blob Storage、FTP、SFTP、NAS 等）",
              "实时通知（Slack、Telegram、Discord、Webhook、邮件等）",
              "MySQL 数据库健康监控",
              "通过 Docker 自托管",
              "开源且免费",
              "支持 MySQL 5.7、8.0、8.4、9、26",
              "支持 MariaDB 5.5、10、11、12、13",
              "备份压缩和 AES-256-GCM 加密",
              "带 Web 界面的 mysqldump 替代方案",
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
                name: "Databasus 是什么？它如何备份 MySQL 数据库？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 是一款采用 Apache 2.0 许可证的自托管备份工具，底层使用 mysqldump 创建一致的 MySQL 备份。它在 mysqldump 之上提供现代化 Web 界面、自动定时调度、云存储集成、实时通知和 AES-256-GCM 加密，让你无需再编写自定义 Shell 脚本。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 是 mysqldump 的替代品还是取代它？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 是对 mysqldump 的扩展而非取代。它在底层以优化参数（--single-transaction、--routines、--triggers、--events）执行 mysqldump，然后处理压缩、加密并上传到你配置的存储。你既保留 mysqldump 的全部可靠性，又获得定时调度、云存储、通知和团队协作功能。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 支持哪些 MySQL 版本？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 支持 MySQL 5.7、8.0、8.4、9 和 26。对于 MySQL 8.0 及更新版本，Databasus 使用 zstd 网络压缩加快传输速度。对于 MySQL 5.7，会自动使用传统压缩方式。MySQL 5.7 仅支持 x86-64。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 支持哪些 MariaDB 版本？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 支持 MariaDB 5.5、10、11、12 和 13。对于 MariaDB 备份，Databasus 使用原生的 mariadb-dump 工具而非 mysqldump，确保与 MariaDB 特有功能和优化完全兼容。",
                },
              },
              {
                "@type": "Question",
                name: "可以备份托管在 AWS RDS、Google Cloud SQL 或 Azure 上的 MySQL 数据库吗？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "可以。Databasus 可以无缝备份云端托管的 MySQL 数据库，包括 AWS RDS for MySQL、Google Cloud SQL、Azure Database for MySQL 和 PlanetScale。由于 Databasus 通过 mysqldump 做逻辑备份，只需要标准的 MySQL 连接凭据，不需要任何特殊的云权限。",
                },
              },
              {
                "@type": "Question",
                name: "Databasus 如何保护 MySQL 数据库凭据和备份的安全？",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus 采用多层安全机制：（1）所有 MySQL 密码和凭据在存储前使用 AES-256-GCM 加密；（2）每个备份文件使用由主密钥、备份 ID 和随机盐派生的唯一密钥加密；（3）凭据通过权限为 0600 的临时 .my.cnf 文件传递，绝不出现在命令行或日志中。",
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
          { href: "#mysql-backup", label: "MySQL" },
          { href: "#mariadb-backup", label: "MariaDB" },
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
              MySQL 备份工具，支持 MariaDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus 是一款免费、开源、可自托管的 MySQL 和 MariaDB
              数据库备份工具。按计划自动执行转储，将备份存放到 S3、Azure Blob Storage
              或本地。备份完成后通过 Slack、Discord 或 Telegram 接收通知
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
                alt="用于管理 MySQL 备份的 Databasus 控制台界面"
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
              MySQL 备份功能
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus 在 mysqldump 之上提供企业级能力：自动定时调度、
              云存储集成、实时通知和 AES-256-GCM 加密。适合管理 MySQL
              数据库的开发者、DevOps 团队和企业
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
                MySQL 定时转储
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="MySQL 定时备份配置"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                在数据库负载较低的时段定时执行
                mysqldump。可选择每小时、每天、每周、每月的间隔，也可以用 cron
                表达式做精确控制
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                MySQL 健康监控
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="MySQL 健康检查"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                通过可配置的健康检查监控 MySQL
                连接可用性。数据库不可达或恢复时收到通知
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                可设置检查间隔（每分钟、每 5
                分钟等）以及标记数据库离线前的失败次数阈值
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                将 MySQL 转储存放到任何地方
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                将 MySQL 备份文件存放在本地、S3 兼容存储、Azure Blob Storage、FTP、NAS 或其他目标。数据始终在你的掌控之中。{" "}
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
                  alt="MySQL 备份存储目标"
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
                MySQL 备份完成或失败时收到提醒。将通知发送到 DevOps
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
                  alt="MySQL 备份通知"
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
                在自己的基础设施上运行 Databasus。所有 MySQL
                凭据和备份数据都保存在你掌控的服务器上。通过脚本、Docker 或
                Kubernetes 约 2 分钟即可部署完成
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

            {/* Card 7: Many MySQL versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MySQL versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  MySQL 和 MariaDB 版本
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  支持 MySQL 5.7、8.0、8.4、9 和 26。支持 MariaDB 5.5、10、11、12
                  和 13，并使用原生 mariadb-dump 工具确保完全兼容
                </p>

                <div>
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="MySQL 版本"
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
                  MySQL 凭据在存储前使用 AES-256-GCM
                  加密。每个备份文件都使用唯一密钥加密。密码通过临时配置文件传递，
                  绝不出现在命令行中。{" "}
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
                    alt="MySQL 备份安全"
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
                  alt="MySQL 备份访问管理"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                控制谁可以查看或管理 MySQL
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
                  alt="MySQL 备份审计日志"
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
                MySQL 凭据在存储前使用 AES-256-GCM
                加密。每个备份文件都使用唯一密钥加密。密码通过临时配置文件传递，
                绝不出现在命令行中。{" "}
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
                  alt="MySQL 备份安全"
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
                  支持云端 MySQL 数据库
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus 可以连接云端托管的 MySQL 数据库，包括 AWS RDS for
                  MySQL、Google Cloud SQL、Azure Database for MySQL 和
                  PlanetScale。由于它通过 mysqldump
                  做逻辑备份，只需要标准的连接凭据，不需要特殊的云权限或文件系统访问
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MYSQL BACKUP SECTION */}
      <section
        id="mysql-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">基于 mysqldump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                MySQL 备份
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus 底层使用 <strong>mysqldump</strong>，即 MySQL
                  官方的备份工具。当你触发一次备份时，Databasus 会以优化参数执行
                  mysqldump：
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --single-transaction
                    </code>{" "}
                    在不锁表的情况下获得一致性快照
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --routines
                    </code>{" "}
                    包含存储过程和函数
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --triggers
                    </code>{" "}
                    包含数据库触发器
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --events
                    </code>{" "}
                    包含计划事件
                  </li>
                </ul>

                <p className="text-gray-400">
                  对于 MySQL 8.0 及更新版本，Databasus 会启用{" "}
                  <strong>zstd 网络压缩</strong>，在低速连接下加快传输。MySQL
                  5.7 会自动使用传统压缩方式。
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    支持的 MySQL 版本：
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 5.7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.0
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 9
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 26
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Right side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mysql.svg"
                  alt="MySQL 数据库"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  通过 mysqldump 完成官方 MySQL
                  数据库备份，支持压缩、加密和云存储
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* MARIADB BACKUP SECTION */}
      <section
        id="mariadb-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row-reverse gap-8 lg:gap-16">
            {/* Right side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">
                  原生 mariadb-dump 工具
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                MariaDB 备份
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  对于 MariaDB 数据库，Databasus 使用原生的{" "}
                  <strong>mariadb-dump</strong> 工具而非 mysqldump，从而与
                  MariaDB 特有功能和优化完全兼容。
                </p>

                <p className="text-gray-400">
                  mariadb-dump 以同样的优化参数执行：single-transaction
                  模式保证备份一致性，存储过程、触发器和计划事件默认包含在内。
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>与 MariaDB 功能完全兼容</li>
                  <li>支持 SSL/TLS 连接</li>
                  <li>网络压缩加快传输</li>
                  <li>与 MySQL 相同的加密和存储选项</li>
                </ul>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    支持的 MariaDB 版本：
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 5.5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 10
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 11
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 12
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 13
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Left side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mariadb.svg"
                  alt="MariaDB 数据库"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  通过 mariadb-dump 完成原生 MariaDB 数据库备份，功能完全兼容
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
                MySQL、MariaDB、PostgreSQL 和 MongoDB 的数据库备份
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
              MySQL 备份常见问题
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              关于使用 Databasus 备份 MySQL 和 MariaDB
              数据库的常见问题。如有其他问题，欢迎加入我们的 Telegram 社区
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Databasus 是什么？它如何备份 MySQL 数据库？"
              answer="Databasus 是一款采用 Apache 2.0 许可证的自托管备份工具，底层使用 mysqldump 创建一致的 MySQL 备份。它在 mysqldump 之上提供现代化 Web 界面、自动定时调度、云存储集成（S3、Azure Blob Storage、FTP）、实时通知（Slack、Discord、Telegram）和 AES-256-GCM 加密，让你无需再编写自定义 Shell 脚本和 cron 任务。"
            />
            <FaqItem
              number="2"
              question="Databasus 是 mysqldump 的替代品还是取代它？"
              answer="Databasus 是对 mysqldump 的扩展而非取代。它在底层以优化参数（--single-transaction、--routines、--triggers、--events）执行 mysqldump，然后处理压缩、加密并上传到你配置的存储。你既保留 mysqldump 的全部可靠性，又通过 Web 界面获得定时调度、云存储、通知和团队协作功能。"
            />
            <FaqItem
              number="3"
              question="Databasus 支持哪些 MySQL 版本？"
              answer="Databasus 支持 MySQL 5.7、8.0、8.4、9 和 26。对于 MySQL 8.0 及更新版本，Databasus 使用 zstd 网络压缩，在低速连接下加快传输。对于 MySQL 5.7，会自动使用传统压缩方式。MySQL 5.7 仅支持 x86-64。所有版本享有相同的功能：定时调度、加密、云存储和通知。"
            />
            <FaqItem
              number="4"
              question="Databasus 支持哪些 MariaDB 版本？"
              answer="Databasus 支持 MariaDB 5.5、10、11、12 和 13。对于 MariaDB 备份，Databasus 使用原生的 mariadb-dump 工具而非 mysqldump，确保与 MariaDB 特有功能和优化完全兼容。MySQL 和 MariaDB 共用同一套 Web 界面、定时调度、存储和通知功能。"
            />
            <FaqItem
              number="5"
              question="可以备份托管在 AWS RDS、Google Cloud SQL 或 Azure 上的 MySQL 数据库吗？"
              answer="可以。Databasus 可以无缝备份云端托管的 MySQL 数据库，包括 AWS RDS for MySQL、Google Cloud SQL、Azure Database for MySQL 和 PlanetScale。由于 Databasus 通过 mysqldump 做逻辑备份，只需要标准的 MySQL 连接凭据（主机、端口、用户名、密码），不需要特殊的云权限、IAM 角色或文件系统访问。"
            />
            <FaqItem
              number="6"
              question="Databasus 中的 MySQL 备份调度是如何工作的？"
              answer="你可以按每小时、每天、每周或每月的间隔安排 MySQL 备份，也可以用 cron 表达式做精确控制。可选择具体的时间（比如流量较低的凌晨 3 点）和日期。其余的交给 Databasus：连接 MySQL、运行 mysqldump、压缩输出、加密并上传到你的存储目标。"
            />
            <FaqItem
              number="7"
              question="我的 MySQL 备份存放在哪里？"
              answer="MySQL 备份文件的存放位置由你决定：服务器本地、S3 兼容存储（AWS S3、MinIO、Backblaze B2）、Azure Blob Storage、Cloudflare R2、FTP、通过 SFTP 访问的 NAS。每个数据库都可以有自己的存储目标。备份文件始终在你的掌控之中。"
            />
            <FaqItem
              number="8"
              question="Databasus 如何保护 MySQL 数据库凭据和备份的安全？"
              answer={
                <>
                  Databasus 采用多层安全机制：
                  <br />
                  <br />
                  <strong>1. 凭据加密：</strong>所有 MySQL
                  密码、令牌和连接字符串在存储前均使用 AES-256-GCM
                  加密。加密密钥与数据库分开存放。
                  <br />
                  <br />
                  <strong>2. 备份加密：</strong>每个备份文件使用由主密钥、备份
                  ID 和随机盐派生的唯一密钥加密。
                  <br />
                  <br />
                  <strong>3. 安全的密码处理：</strong>MySQL 密码通过权限为 0600
                  的临时 .my.cnf 文件传递，绝不出现在命令行参数或日志中。
                </>
              }
            />
            <FaqItem
              number="9"
              question="可以把 MySQL 备份恢复到其他服务器或云服务商吗？"
              answer="可以。Databasus 创建的是标准的 mysqldump 逻辑备份，可以恢复到任何 MySQL 服务器：不同版本、不同云服务商或本地开发机。从 Databasus 下载备份（自动解密），然后用 mysql 命令恢复即可。Databasus 会为每个备份展示确切的恢复命令。"
            />
            <FaqItem
              number="10"
              question="Databasus 支持备份 MySQL 复制从库吗？"
              answer="支持。你可以备份 MySQL 副本（从库）服务器。Databasus 使用 --single-transaction，在不停止复制的情况下创建一致性快照。这有助于把备份负载从主库上转移出去。只需将 Databasus 指向副本的连接信息即可。"
            />
            <FaqItem
              number="11"
              question="Databasus 与 MySQL Enterprise Backup 或 Percona XtraBackup 有什么不同？"
              answer={
                <>
                  MySQL Enterprise Backup 和 Percona XtraBackup
                  创建的是物理（二进制）备份，需要直接访问 MySQL
                  数据目录所在的文件系统。它们对超大数据库更快，但是：
                  <br />
                  <br />
                  • 无法备份云端托管的 MySQL（RDS、Cloud SQL、Azure）
                  <br />
                  • 必须安装在数据库服务器本机上
                  <br />
                  • 部署和恢复流程更复杂
                  <br />
                  <br />
                  Databasus 使用逻辑备份（mysqldump），适用于任何能连接上的
                  MySQL 服务器，包括所有云服务商。对大多数 100 GB
                  以下的数据库来说，逻辑备份足够实用，管理起来也简单得多。
                </>
              }
            />
            <FaqItem
              number="12"
              question="可以用同一个 Databasus 实例同时备份 MySQL 和 PostgreSQL 数据库吗？"
              answer="可以。Databasus 在单个安装中同时支持 MySQL、MariaDB、PostgreSQL 和 MongoDB。你可以在一个 Web 界面中管理所有数据库的备份，并为每个数据库设置不同的计划、存储目标和通知渠道。这对管理多种数据库基础设施的团队很有帮助。"
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
