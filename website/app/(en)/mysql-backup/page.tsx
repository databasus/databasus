import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { getLanguageAlternates } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "MySQL backup with MariaDB support",
  description:
    "Free and open source tool for MySQL and MariaDB scheduled backups. Alternative to mysqldump with web UI, cloud storage (S3, Google Drive), notifications (Slack, Discord, Telegram) and AES-256 encryption.",
  keywords:
    "MySQL backup, MariaDB backup, mysqldump alternative, mysqldump GUI, MySQL backup automation, MySQL backup tool, MariaDB backup tool, MySQL scheduled backup, MySQL cloud backup, MySQL S3 backup, MySQL Docker backup, MySQL backup encryption",
  robots: "index, follow",
  alternates: {
    canonical: "https://databasus.com/mysql-backup",
    languages: getLanguageAlternates("mysql-backup"),
  },
  openGraph: {
    type: "website",
    url: "https://databasus.com/mysql-backup",
    title: "MySQL backup with MariaDB support",
    description:
      "Free and open source tool for MySQL and MariaDB scheduled backups. Alternative to mysqldump with web UI, cloud storage, notifications and encryption.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Databasus dashboard interface showing MySQL backup management",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "MySQL backup with MariaDB support",
    description:
      "Free and open source tool for MySQL and MariaDB scheduled backups. Alternative to mysqldump with web UI, cloud storage, notifications and encryption.",
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
              "Free and open source tool for MySQL and MariaDB scheduled backups. Alternative to mysqldump with web UI, cloud storage, notifications and encryption.",
            url: "https://databasus.com/mysql-backup",
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
              "Scheduled MySQL and MariaDB backups",
              "Multiple storage destinations (S3, Google Drive, Dropbox, SFTP, rclone, etc.)",
              "Real-time notifications (Slack, Telegram, Discord, Webhook, email, etc.)",
              "MySQL database health monitoring",
              "Self-hosted via Docker",
              "Open source and free",
              "Support for MySQL 5.7, 8.0, 8.4, 9, 26",
              "Support for MariaDB 5.5, 10, 11, 12, 13",
              "Backup compression and AES-256-GCM encryption",
              "mysqldump alternative with web UI",
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
                name: "What is Databasus and how does it backup MySQL databases?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus is an Apache 2.0 licensed, self-hosted backup tool that uses mysqldump under the hood to create consistent MySQL backups. It wraps mysqldump with a modern web interface, automated scheduling, cloud storage integration, real-time notifications and AES-256-GCM encryption — eliminating the need for custom shell scripts.",
                },
              },
              {
                "@type": "Question",
                name: "Is Databasus a mysqldump alternative or replacement?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus extends mysqldump rather than replacing it. Under the hood, it executes mysqldump with optimized parameters (--single-transaction, --routines, --triggers, --events) then handles compression, encryption and upload to your configured storage. You get all mysqldump reliability plus scheduling, cloud storage, notifications and team features.",
                },
              },
              {
                "@type": "Question",
                name: "Which MySQL versions does Databasus support?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus supports MySQL 5.7, 8.0, 8.4, 9 and 26. For MySQL 8.0 and newer, Databasus uses zstd network compression for faster transfers. For MySQL 5.7, legacy compression is used automatically. MySQL 5.7 is supported on x86-64 only.",
                },
              },
              {
                "@type": "Question",
                name: "Which MariaDB versions does Databasus support?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus supports MariaDB 5.5, 10, 11, 12 and 13. For MariaDB backups, Databasus uses the native mariadb-dump tool instead of mysqldump, ensuring full compatibility with MariaDB-specific features and optimizations.",
                },
              },
              {
                "@type": "Question",
                name: "Can I backup MySQL databases hosted on AWS RDS, Google Cloud SQL or Azure?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes, Databasus works seamlessly with cloud-hosted MySQL databases including AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL and PlanetScale. Since Databasus uses logical backups via mysqldump, it only requires standard MySQL connection credentials — no special cloud permissions needed.",
                },
              },
              {
                "@type": "Question",
                name: "How does Databasus secure MySQL database credentials and backups?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus implements multi-layer security: (1) All MySQL passwords and credentials are encrypted with AES-256-GCM before storage; (2) Each backup file is encrypted with a unique key derived from master key, backup ID and random salt; (3) Credentials are passed via temporary .my.cnf files with 0600 permissions, never exposed in command line or logs.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        sectionLinks={[
          { href: "#features", label: "Features" },
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
              MySQL backup tool with MariaDB support
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus is a free, open source and self-hosted tool to backup
              MySQL and MariaDB databases. Schedule automated dumps, store them
              in S3, Google Drive or locally. Receive notifications via Slack,
              Discord or Telegram when backups complete
            </p>

            <div className="flex flex-col sm:flex-row sm:flex-wrap items-center justify-center gap-2 sm:gap-2 max-w-[400px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
              <a
                href="#installation"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-3"
              >
                Self-host via Docker
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
                alt="Databasus dashboard interface for MySQL backup management"
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
              <span className="text-sm font-medium">Overview</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Features for MySQL backup
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus wraps mysqldump with enterprise features: automated
              scheduling, cloud storage integration, real-time notifications and
              AES-256-GCM encryption. Perfect for developers, DevOps teams and
              organizations managing MySQL databases
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
                Scheduled MySQL dumps
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="MySQL scheduled backups configuration"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Schedule mysqldump at optimal times when database load is low.
                Choose hourly, daily, weekly, monthly intervals or use cron
                expressions for precise control
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                MySQL health monitoring
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="MySQL health checks"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Monitor MySQL connection availability with configurable health
                checks. Get notified when your database becomes unreachable or
                recovers
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Set check intervals (every minute, 5 minutes, etc.) and failure
                thresholds before marking the database as down
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Store MySQL dumps anywhere
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Keep MySQL backup files locally, in S3-compatible storage,
                Google Drive, Dropbox, NAS or other destinations. Your data
                stays under your control.{" "}
                <a
                  href="/storages"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  View all →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="MySQL backup storage destinations"
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
                Backup notifications
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Get alerts when MySQL backups complete or fail. Send
                notifications to your DevOps team chat or personal channels.{" "}
                <a
                  href="/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  View all →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="MySQL backup notifications"
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
                Self hosted via Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Run Databasus on your own infrastructure. All MySQL credentials
                and backup data stay on servers you control. Deploy in about 2
                minutes via script, Docker or Kubernetes
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Docker deployment"
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
                Open source and free
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus is fully open source with Apache 2.0 license. Inspect
                every line of code and fork it. Free for personal
                and enterprise use
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="GitHub open source"
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
                  MySQL and MariaDB versions
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  MySQL 5.7, 8.0, 8.4, 9 and 26 are supported. MariaDB 5.5, 10, 11, 12
                  and 13 are supported with native mariadb-dump tool for full
                  compatibility
                </p>

                <div>
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="MySQL versions"
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
                  Security
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  MySQL credentials are encrypted with AES-256-GCM before
                  storage. Each backup file is encrypted with a unique key.
                  Passwords are passed via temporary config files, never exposed
                  in command line.{" "}
                  <a
                    href="/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Read more →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="MySQL backup security"
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
                  Access management
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  for teams
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="MySQL backup access management"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Control who can view or manage MySQL databases. Create
                workspaces for different projects. Assign viewer, editor or
                admin roles.{" "}
                <a
                  href="/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Read more →
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
                  Audit logs
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  for teams
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="MySQL backup audit logs"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Track all activities: backup downloads, schedule changes,
                configuration updates. See who did what and when for compliance
                and accountability.{" "}
                <a
                  href="/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Read more →
                </a>
              </p>
            </div>

            {/* Card 10: Security - Mobile/Tablet only */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Security
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                MySQL credentials are encrypted with AES-256-GCM before storage.
                Each backup file is encrypted with a unique key. Passwords are
                passed via temporary config files, never exposed in command
                line.{" "}
                <a
                  href="/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Read more →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="MySQL backup security"
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
                  Works with cloud MySQL databases
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus connects to cloud-hosted MySQL databases including
                  AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL
                  and PlanetScale. Since it uses logical backups via mysqldump,
                  you only need standard connection credentials — no special
                  cloud permissions or filesystem access required
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
                <span className="text-sm font-medium">Built on mysqldump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                MySQL backup
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus uses <strong>mysqldump</strong> under the hood — the
                  official MySQL backup utility. When you trigger a backup,
                  Databasus executes mysqldump with optimized parameters:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --single-transaction
                    </code>{" "}
                    for consistent snapshots without locking tables
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --routines
                    </code>{" "}
                    to include stored procedures and functions
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --triggers
                    </code>{" "}
                    to include database triggers
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --events
                    </code>{" "}
                    to include scheduled events
                  </li>
                </ul>

                <p className="text-gray-400">
                  For MySQL 8.0 and newer, Databasus enables{" "}
                  <strong>zstd network compression</strong> for faster transfers
                  over slow connections. MySQL 5.7 uses legacy compression
                  automatically.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Supported MySQL versions:
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
                  alt="MySQL database"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Official MySQL backup via mysqldump with compression,
                  encryption and cloud storage
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
                  Native mariadb-dump tool
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                MariaDB backup
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  For MariaDB databases, Databasus uses the native{" "}
                  <strong>mariadb-dump</strong> tool instead of mysqldump. This
                  ensures full compatibility with MariaDB-specific features and
                  optimizations.
                </p>

                <p className="text-gray-400">
                  mariadb-dump is executed with the same optimized parameters:
                  single-transaction mode for consistent backups, routines,
                  triggers and events included by default.
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>Full MariaDB feature compatibility</li>
                  <li>SSL/TLS connection support</li>
                  <li>Network compression for faster transfers</li>
                  <li>Same encryption and storage options as MySQL</li>
                </ul>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Supported MariaDB versions:
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
                  alt="MariaDB database"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Native MariaDB backup via mariadb-dump with full feature
                  compatibility
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
                <span className="text-sm font-medium">Get started</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                How to install?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus supports multiple installation methods. Deploy on your
                VPS, local machine or Kubernetes cluster in about 2 minutes.
                Same installation works for MySQL, MariaDB, PostgreSQL and
                MongoDB backups
              </p>
            </div>

            <InstallationComponent />
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
              MySQL backup questions
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Common questions about backing up MySQL and MariaDB databases with
              Databasus. If you have other questions, join our community on
              Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="What is Databasus and how does it backup MySQL databases?"
              answer="Databasus is an Apache 2.0 licensed, self-hosted backup tool that uses mysqldump under the hood to create consistent MySQL backups. It wraps mysqldump with a modern web interface, automated scheduling, cloud storage integration (S3, Google Drive, Dropbox), real-time notifications (Slack, Discord, Telegram) and AES-256-GCM encryption — eliminating the need for custom shell scripts and cron jobs."
            />
            <FaqItem
              number="2"
              question="Is Databasus a mysqldump alternative or replacement?"
              answer="Databasus extends mysqldump rather than replacing it. Under the hood, it executes mysqldump with optimized parameters (--single-transaction, --routines, --triggers, --events) then handles compression, encryption and upload to your configured storage. You get all mysqldump reliability plus scheduling, cloud storage, notifications and team collaboration features through a web UI."
            />
            <FaqItem
              number="3"
              question="Which MySQL versions does Databasus support?"
              answer="Databasus supports MySQL 5.7, 8.0, 8.4, 9 and 26. For MySQL 8.0 and newer, Databasus uses zstd network compression for faster transfers over slow connections. For MySQL 5.7, legacy compression is used automatically. MySQL 5.7 is supported on x86-64 only. All versions get the same features: scheduling, encryption, cloud storage and notifications."
            />
            <FaqItem
              number="4"
              question="Which MariaDB versions does Databasus support?"
              answer="Databasus supports MariaDB 5.5, 10, 11, 12 and 13. For MariaDB backups, Databasus uses the native mariadb-dump tool instead of mysqldump, ensuring full compatibility with MariaDB-specific features and optimizations. The same web UI, scheduling, storage and notification features work for both MySQL and MariaDB."
            />
            <FaqItem
              number="5"
              question="Can I backup MySQL databases hosted on AWS RDS, Google Cloud SQL or Azure?"
              answer="Yes, Databasus works seamlessly with cloud-hosted MySQL databases including AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL and PlanetScale. Since Databasus uses logical backups via mysqldump, it only requires standard MySQL connection credentials (host, port, username, password) — no special cloud permissions, IAM roles or filesystem access needed."
            />
            <FaqItem
              number="6"
              question="How does MySQL backup scheduling work in Databasus?"
              answer="You can schedule MySQL backups at hourly, daily, weekly or monthly intervals, or use cron expressions for precise control. Choose specific times (like 3 AM when traffic is low) and days. Databasus handles the rest: connecting to MySQL, running mysqldump, compressing the output, encrypting it and uploading to your storage destination."
            />
            <FaqItem
              number="7"
              question="Where are my MySQL backups stored?"
              answer="You choose where to store MySQL backup files: locally on your server, in S3-compatible storage (AWS S3, MinIO, Backblaze B2), Google Drive, Cloudflare R2, Dropbox, NAS via SFTP, or any destination supported by rclone. Each database can have its own storage destination. Your backup files stay under your control."
            />
            <FaqItem
              number="8"
              question="How does Databasus secure MySQL database credentials and backups?"
              answer={
                <>
                  Databasus implements multi-layer security:
                  <br />
                  <br />
                  <strong>1. Credential encryption:</strong> All MySQL
                  passwords, tokens and connection strings are encrypted with
                  AES-256-GCM before storage. The encryption key is stored
                  separately from the database.
                  <br />
                  <br />
                  <strong>2. Backup encryption:</strong> Each backup file is
                  encrypted with a unique key derived from master key, backup ID
                  and random salt.
                  <br />
                  <br />
                  <strong>3. Secure password handling:</strong> MySQL passwords
                  are passed via temporary .my.cnf files with 0600 permissions,
                  never exposed in command line arguments or logs.
                </>
              }
            />
            <FaqItem
              number="9"
              question="Can I restore MySQL backups to a different server or cloud provider?"
              answer="Yes, since Databasus creates standard mysqldump logical backups, you can restore them to any MySQL server — different version, different cloud provider or local development machine. Download the backup from Databasus (automatically decrypted), then use mysql command to restore. Databasus shows the exact restore command for each backup."
            />
            <FaqItem
              number="10"
              question="Does Databasus support backing up MySQL replication slaves?"
              answer="Yes, you can backup MySQL replica (slave) servers. Databasus uses --single-transaction which creates a consistent snapshot without stopping replication. This is useful for offloading backup load from your primary server. Just point Databasus to your replica's connection details."
            />
            <FaqItem
              number="11"
              question="How is Databasus different from MySQL Enterprise Backup or Percona XtraBackup?"
              answer={
                <>
                  MySQL Enterprise Backup and Percona XtraBackup create physical
                  (binary) backups that require direct filesystem access to the
                  MySQL data directory. They&apos;re faster for very large
                  databases but:
                  <br />
                  <br />
                  • Cannot backup cloud-hosted MySQL (RDS, Cloud SQL, Azure)
                  <br />
                  • Require installation on the database server itself
                  <br />
                  • More complex setup and restore process
                  <br />
                  <br />
                  Databasus uses logical backups (mysqldump) which work with any
                  MySQL server you can connect to, including all cloud
                  providers. For most databases under 100GB, logical backups are
                  practical and much simpler to manage.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Can I backup both MySQL and PostgreSQL databases with the same Databasus instance?"
              answer="Yes, Databasus supports MySQL, MariaDB, PostgreSQL and MongoDB in a single installation. You can manage backups for all your databases from one web interface, with different schedules, storage destinations and notification channels for each. This is useful for teams managing diverse database infrastructure."
            />
          </div>
        </div>
      </section>

      <LandingFooterComponent />
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
