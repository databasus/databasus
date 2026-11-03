import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { getLanguageAlternates } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "MongoDB backup",
  description:
    "Free and open source tool for MongoDB scheduled backups. Automate mongodump with web UI, store archives in S3, Google Drive or locally. Get notifications via Slack, Discord, Telegram. AES-256 encryption for BSON data.",
  keywords:
    "MongoDB backup, mongodump alternative, MongoDB backup automation, MongoDB backup tool, MongoDB scheduled backup, MongoDB cloud backup, MongoDB S3 backup, MongoDB Docker backup, MongoDB backup encryption, MongoDB Atlas backup, replica set backup, document database backup, BSON backup, NoSQL backup",
  robots: "index, follow",
  alternates: {
    canonical: "https://databasus.com/mongodb-backup",
    languages: getLanguageAlternates("mongodb-backup"),
  },
  openGraph: {
    type: "website",
    url: "https://databasus.com/mongodb-backup",
    title: "MongoDB backup",
    description:
      "Free and open source tool for MongoDB scheduled backups. Automate mongodump with web UI, cloud storage, notifications and encryption.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Databasus dashboard interface showing MongoDB backup management",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "MongoDB backup",
    description:
      "Free and open source tool for MongoDB scheduled backups. Automate mongodump with web UI, cloud storage, notifications and encryption.",
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
              "Free and open source tool for MongoDB scheduled backups. Automate mongodump with web UI, cloud storage, notifications and encryption.",
            url: "https://databasus.com/mongodb-backup",
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
              "Scheduled MongoDB backups via mongodump",
              "Multiple storage destinations (S3, Google Drive, Dropbox, SFTP, rclone, etc.)",
              "Real-time notifications (Slack, Telegram, Discord, Webhook, email, etc.)",
              "MongoDB connection health monitoring",
              "Self-hosted via Docker",
              "Open source and free",
              "Support for MongoDB 4, 5, 6, 7, 8 and 9",
              "BSON archive compression with gzip",
              "AES-256-GCM encryption for backup files",
              "MongoDB Atlas and replica set support",
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
                name: "What is Databasus and how does it backup MongoDB databases?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus is an Apache 2.0 licensed, self-hosted backup tool that uses mongodump under the hood to create consistent MongoDB backups. It wraps mongodump with a modern web interface, automated scheduling, cloud storage integration, real-time notifications and AES-256-GCM encryption — eliminating the need for custom shell scripts.",
                },
              },
              {
                "@type": "Question",
                name: "Does Databasus support MongoDB replica sets?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes, Databasus fully supports MongoDB replica sets. You can connect to any member of a replica set using the standard MongoDB connection URI format. Databasus will read from the specified node, allowing you to backup from secondary nodes to reduce load on your primary.",
                },
              },
              {
                "@type": "Question",
                name: "Can I backup MongoDB Atlas databases with Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes, Databasus works seamlessly with MongoDB Atlas. Since Databasus uses logical backups via mongodump, it only requires standard MongoDB connection credentials — no special Atlas permissions or roles needed. Just provide your Atlas connection string and Databasus handles the rest.",
                },
              },
              {
                "@type": "Question",
                name: "Which MongoDB versions does Databasus support?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus supports MongoDB versions 4, 5, 6, 7, 8 and 9. All backups use the native mongodump tool with --archive and --gzip flags for efficient, compressed BSON archives that can be restored with mongorestore.",
                },
              },
              {
                "@type": "Question",
                name: "How does Databasus secure MongoDB credentials and backups?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus implements multi-layer security: (1) All MongoDB passwords and connection strings are encrypted with AES-256-GCM before storage; (2) Each backup file is encrypted with a unique key derived from master key, backup ID and random salt; (3) Connection URIs are passed securely to mongodump, never exposed in logs or command line output.",
                },
              },
              {
                "@type": "Question",
                name: "Does Databasus support incremental MongoDB backups?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus focuses on full logical backups using mongodump rather than incremental backups. For most use cases, scheduled full backups (hourly, daily, weekly) provide sufficient recovery points. MongoDB Atlas already offers native point-in-time recovery, and external incremental backups cannot be easily restored to Atlas clusters.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        sectionLinks={[
          { href: "#features", label: "Features" },
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
              MongoDB backup tool
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus is a free, open source and self-hosted tool to backup
              MongoDB document databases. Automate mongodump with scheduling,
              store BSON archives in S3, Google Drive or locally. Get notified
              via Slack, Discord or Telegram when backups complete
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
                alt="Databasus dashboard interface for MongoDB backup management"
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
              Features for MongoDB backup
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus wraps mongodump with enterprise features: automated
              scheduling, cloud storage integration, real-time notifications and
              AES-256-GCM encryption. Ideal for developers and DevOps teams
              managing MongoDB document databases and collections
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
                Scheduled MongoDB dumps
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="MongoDB scheduled backups configuration"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Schedule mongodump at optimal times when your application load
                is low. Choose hourly, daily, weekly, monthly intervals or use
                cron expressions for precise timing control
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                MongoDB health monitoring
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="MongoDB health checks"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Monitor MongoDB connection availability with configurable health
                checks. Get notified when your database or replica set becomes
                unreachable
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Set check intervals (every minute, 5 minutes, etc.) and failure
                thresholds before marking the database as unavailable
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Store BSON archives anywhere
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Keep MongoDB backup archives locally, in S3-compatible storage,
                Google Drive, Dropbox, NAS or other destinations. Your document
                data stays under your control.{" "}
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
                  alt="MongoDB backup storage destinations"
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
                Get alerts when MongoDB backups complete or fail. Send
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
                  alt="MongoDB backup notifications"
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
                Run Databasus on your own infrastructure. All MongoDB connection
                strings and backup data stay on servers you control. Deploy in
                about 2 minutes via script, Docker or Kubernetes
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

            {/* Card 7: Many MongoDB versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MongoDB versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  MongoDB versions supported
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  MongoDB 4, 5, 6, 7, 8 and 9 are supported. Databasus uses the
                  native mongodump tool for each version to ensure full
                  compatibility with your document database
                </p>

                <div>
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="MongoDB versions"
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
                  MongoDB connection strings are encrypted with AES-256-GCM
                  before storage. Each BSON archive is encrypted with a unique
                  key. Credentials are passed securely to mongodump, never
                  exposed in logs.{" "}
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
                    alt="MongoDB backup security"
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
                  alt="MongoDB backup access management"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Control who can view or manage MongoDB databases. Create
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
                  alt="MongoDB backup audit logs"
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
                MongoDB connection strings are encrypted with AES-256-GCM before
                storage. Each BSON archive is encrypted with a unique key.
                Credentials are passed securely to mongodump, never exposed in
                logs.{" "}
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
                  alt="MongoDB backup security"
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
                  Works with MongoDB Atlas and self-hosted
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus connects to cloud-hosted MongoDB databases including
                  MongoDB Atlas, AWS DocumentDB and self-hosted deployments.
                  Since it uses logical backups via mongodump, you only need
                  standard connection credentials — no special cloud permissions
                  or filesystem access required
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
                <span className="text-sm font-medium">Built on mongodump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                How MongoDB backup works
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus uses <strong>mongodump</strong> under the hood — the
                  official MongoDB backup utility. When you trigger a backup,
                  Databasus executes mongodump with optimized parameters:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --archive
                    </code>{" "}
                    for single-file BSON output instead of directory structure
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --gzip
                    </code>{" "}
                    for compressed archives reducing storage and transfer size
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --db
                    </code>{" "}
                    to backup specific databases from your MongoDB instance
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --uri
                    </code>{" "}
                    for secure connection string handling with authentication
                  </li>
                </ul>

                <p className="text-gray-400">
                  The backup stream is piped directly to your configured
                  storage, optionally encrypted with AES-256-GCM before writing.
                  This approach minimizes disk I/O and works efficiently with
                  large collections.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Supported MongoDB versions:
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
                  alt="MongoDB database"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Official MongoDB backup via mongodump with gzip compression,
                  encryption and cloud storage
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
                Same installation works for MongoDB, PostgreSQL, MySQL and
                MariaDB backups
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
              MongoDB backup questions
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Common questions about backing up MongoDB document databases with
              Databasus. If you have other questions, join our community on
              Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="What is Databasus and how does it backup MongoDB databases?"
              answer="Databasus is an Apache 2.0 licensed, self-hosted backup tool that uses mongodump under the hood to create consistent MongoDB backups. It wraps mongodump with a modern web interface, automated scheduling, cloud storage integration (S3, Google Drive, Dropbox), real-time notifications (Slack, Discord, Telegram) and AES-256-GCM encryption — eliminating the need for custom shell scripts and cron jobs."
            />
            <FaqItem
              number="2"
              question="Does Databasus support MongoDB replica sets?"
              answer="Yes, Databasus fully supports MongoDB replica sets. You can connect to any member of a replica set using the standard MongoDB connection URI format with replica set options. Databasus will read from the specified node, allowing you to backup from secondary nodes to reduce load on your primary. This is particularly useful for production environments where you want to avoid impacting primary node performance."
            />
            <FaqItem
              number="3"
              question="Can I backup MongoDB Atlas databases with Databasus?"
              answer="Yes, Databasus works seamlessly with MongoDB Atlas. Since Databasus uses logical backups via mongodump, it only requires standard MongoDB connection credentials — no special Atlas permissions, IP whitelisting beyond your Databasus server, or administrative roles needed. Just provide your Atlas connection string (available in the Atlas dashboard) and Databasus handles the rest."
            />
            <FaqItem
              number="4"
              question="Which MongoDB versions does Databasus support?"
              answer="Databasus supports MongoDB versions 4, 5, 6, 7, 8 and 9. All backups use the native mongodump tool with --archive and --gzip flags for efficient, compressed BSON archives. The archives can be restored using mongorestore to any compatible MongoDB version, making migrations between versions straightforward."
            />
            <FaqItem
              number="5"
              question="How does Databasus handle large MongoDB collections?"
              answer="Databasus streams mongodump output directly to your storage destination, optionally encrypting the stream in transit. This approach avoids writing temporary files to disk, making it efficient for databases with large collections. The --archive flag creates a single compressed file rather than a directory structure, reducing I/O overhead and simplifying storage management."
            />
            <FaqItem
              number="6"
              question="Can I backup sharded MongoDB clusters with Databasus?"
              answer={
                <>
                  Databasus currently focuses on backing up individual MongoDB
                  databases rather than coordinated sharded cluster backups.
                  <br />
                  <br />
                  For sharded clusters, you can:
                  <br />
                  <br />
                  • Backup each shard individually by connecting to shard
                  replica sets
                  <br />
                  • Backup via a mongos router (though this may impact
                  performance)
                  <br />
                  <br />
                  For production sharded clusters, consider MongoDB Atlas native
                  backups or mongodump with --oplog for point-in-time
                  consistency across shards.
                </>
              }
            />
            <FaqItem
              number="7"
              question="How does Databasus secure MongoDB credentials and backups?"
              answer={
                <>
                  Databasus implements multi-layer security:
                  <br />
                  <br />
                  <strong>1. Credential encryption:</strong> All MongoDB
                  connection URIs, passwords and authentication details are
                  encrypted with AES-256-GCM before storage.
                  <br />
                  <br />
                  <strong>2. Backup encryption:</strong> Each BSON archive is
                  encrypted with a unique key derived from master key, backup ID
                  and random salt.
                  <br />
                  <br />
                  <strong>3. Secure credential handling:</strong> Connection
                  URIs are passed directly to mongodump via secure parameters,
                  never exposed in logs or process listings.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Does Databasus support incremental MongoDB backups or oplog tailing?"
              answer="Databasus focuses on full logical backups using mongodump rather than incremental backups or oplog-based point-in-time recovery. For most use cases, scheduled full backups (hourly, daily, weekly) provide sufficient recovery points without the complexity of oplog management. MongoDB Atlas already offers native continuous backups with point-in-time recovery, and external incremental backups cannot be easily restored to Atlas clusters."
            />
            <FaqItem
              number="9"
              question="Can I restore MongoDB backups to a different version or cluster?"
              answer="Yes, since Databasus creates standard mongodump archives in BSON format, you can restore them to any compatible MongoDB server — different version, different cloud provider or local development machine. Download the backup from Databasus (automatically decrypted), then use mongorestore with --archive and --gzip flags. Databasus shows the exact restore command for each backup."
            />
            <FaqItem
              number="10"
              question="How does mongodump compression work in Databasus?"
              answer="Databasus uses mongodump's built-in --gzip flag which compresses BSON data during the dump process. This typically reduces archive size by 60-80% compared to uncompressed BSON. The compression happens in the mongodump stream before optional encryption, so both compressed and encrypted archives remain efficient. Decompression is automatic when using mongorestore with the --gzip flag."
            />
            <FaqItem
              number="11"
              question="Can I backup specific MongoDB collections instead of entire databases?"
              answer="Currently, Databasus backs up entire MongoDB databases rather than individual collections. This ensures you have complete, consistent backups including all collections, indexes and metadata. If you need collection-level backups, you can create separate databases for different data domains, each with its own backup schedule in Databasus."
            />
            <FaqItem
              number="12"
              question="Does Databasus work with MongoDB running in Docker or Kubernetes?"
              answer="Yes, Databasus connects to MongoDB over the network using standard connection URIs, so it works with MongoDB regardless of where it's deployed — Docker containers, Kubernetes pods, VMs or bare metal. Just ensure network connectivity between Databasus and your MongoDB instance. For Kubernetes deployments, you can use internal service DNS names or external load balancer endpoints."
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
