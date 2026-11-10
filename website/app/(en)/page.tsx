import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { getLanguageAlternates, getLocalizedHref } from "@/app/i18n";
import HeroStatsComponent from "@/app/components/HeroStatsComponent";
import InstallationComponent from "@/app/components/InstallationComponent";
import LiteYouTubeEmbed from "@/app/components/LiteYouTubeEmbed";

export const metadata: Metadata = {
  title: "PostgreSQL backup | Databasus",
  description:
    "Free and open source tool for PostgreSQL scheduled backups (with MySQL and MongoDB support). Save them locally and to clouds. Notifications to Slack, Discord, Telegram, email, webhook, etc.",
  keywords:
    "PostgreSQL, backup, monitoring, database, scheduled backups, Docker, self-hosted, open source, S3, Azure Blob Storage, Slack notifications, Discord, DevOps, database monitoring, pg_dump, database restore, encryption, AES-256, backup encryption",
  robots: "index, follow",
  alternates: {
    canonical: "https://databasus.com",
    languages: getLanguageAlternates(""),
  },
  openGraph: {
    type: "website",
    url: "https://databasus.com",
    title: "PostgreSQL backup | Databasus",
    description:
      "Free and open source tool for PostgreSQL scheduled backups (with MySQL and MongoDB support). Save them locally and to clouds. Notifications to Slack, Discord, Telegram, email, webhook, etc.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Databasus dashboard interface showing backup management",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "PostgreSQL backup | Databasus",
    description:
      "Free and open source tool for PostgreSQL scheduled backups (with MySQL and MongoDB support). Save them locally and to clouds. Notifications to Slack, Discord, Telegram, email, webhook, etc.",
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
              "Free and open source tool for PostgreSQL scheduled backups (with MySQL and MongoDB support). Save them locally and to clouds. Notifications to Slack, Discord, Telegram, email, webhook, etc.",
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
              "Scheduled PostgreSQL backups",
              "Multiple storage destinations (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Real-time notifications (Slack, Telegram, Discord, Webhook, email, etc.)",
              "Database health monitoring",
              "Self-hosted via Docker",
              "Open source and free",
              "Support for PostgreSQL 12-18",
              "Backup compression and AES-256-GCM encryption",
              "Support for PostgreSQL, MySQL, MariaDB and MongoDB",
              "Retention policies: time period, count, GFS and size limits",
              "Point-in-Time Recovery (PITR) with WAL archiving",
              "Restore verification: automated restore testing in real database Docker containers",
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
            description: "PostgreSQL backup tool",
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
                name: "What is Databasus and why should I use it instead of hand-rolled scripts?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus is an Apache 2.0 licensed, self-hosted service backing up PostgreSQL, v13 to v18. It differs from shell scripts in that it has a frontend for scheduling tasks, compressing and storing archives on multiple targets (local disk, S3, Azure Blob Storage, FTP, SFTP, NAS, etc.), configuring retention policies to automatically prune old backups and notifying your team when tasks finish or fail — all without hand-rolled code",
                },
              },
              {
                "@type": "Question",
                name: "How do I install Databasus in the quickest manner?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "The most direct route is to run the one-line cURL installer. It fetches the current Docker image, spins up a single PostgreSQL container. Then creates a docker-compose.yml and boots up the service so it will automatically start again when reboots occur. Overall time is usually less than two minutes on a typical VPS.",
                },
              },
              {
                "@type": "Question",
                name: "How restore verification works?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus runs a small verification agent on a host you control. On every scheduled run the agent downloads the latest backup. It restores the backup into a throwaway database container. Then it sanity-checks the restored database against the source. The result is reported back — including the restore exit code and per-table row counts. Schedules support after backup, hourly, daily, weekly, monthly or a UTC cron expression. Failures can be sent through any notifier wired to the database — Slack, Teams, Discord, email and others.",
                },
              },
              {
                "@type": "Question",
                name: "Where do my backups live and how much space will they occupy?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Archives can be saved to local volumes, S3-compatible buckets, Azure Blob Storage, FTP and other cloud targets. Databasus implements balanced compression, which typically shrinks dump size by 4-8x with incremental only about 20% of runtime overhead, so you have storage and bandwidth savings.",
                },
              },
              {
                "@type": "Question",
                name: "How will I know a backup succeeded — or worse, failed?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus can notify with real-time emails, Slack, Telegram, webhooks, Mattermost, Discord and more. You have the choice of what channels to ping so that your DevOps team hears about successes and failures in real time, making recovery routines and compliance audits easier.",
                },
              },
              {
                "@type": "Question",
                name: "How does Databasus ensure security?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus enforces security on three levels: (1) Sensitive data encryption — all passwords, tokens and credentials are encrypted with AES-256-GCM and stored separately from the database; (2) Backup encryption — each backup file is encrypted with a unique key derived from a master key, backup ID and random salt, making backups useless without your encryption key even if someone gains storage access; (3) Read-only database access — Databasus only requires SELECT permissions and performs comprehensive checks to ensure no write privileges exist, preventing data corruption even if the tool is compromised. Sign-in itself can require a second factor: with it on, a correct password is followed by a six-digit code emailed to the account, while sign-in through Google or GitHub keeps relying on the provider's own checks. Before each release, automated checks scan code, dependencies, secrets and container images. Tests verify full backup-and-restore cycles against real PostgreSQL, MySQL, MariaDB and MongoDB containers. Required checks must pass before publication. Regular Codex Security audits from OpenAI complement these checks and are separate from the release pipeline. All operations run in containers you control on servers you own, and because it's open source, your security team can audit every line of code before deployment.",
                },
              },
              {
                "@type": "Question",
                name: "Is Databasus supported by Anthropic and OpenAI OSS programs?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes, in March 2026 Databasus was accepted into both Claude for Open Source by Anthropic and Codex for Open Source by OpenAI. The project has been independently evaluated and recognized by industry leaders as critical open-source infrastructure worth supporting.",
                },
              },
              {
                "@type": "Question",
                name: "How is Databasus different from PgBackRest, Barman or pg_dump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus prefers simplicity — it provides a modern web interface to manage backups for many databases at once, with built-in scheduling, compression, multiple storage destinations, health monitoring and real-time notifications. At the same time, unlike pgBackRest and WAL-G, Databasus makes physical, incremental and WAL backups on top of PostgreSQL 17's native approach, so it does not reinvent its own backup engine. It connects to your databases remotely, reaching closed networks through an SSH tunnel to the server or a bastion, so databases that are not publicly exposed can still be backed up and managed from a single dashboard.",
                },
              },
              {
                "@type": "Question",
                name: "Which databases are supported by Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus supports PostgreSQL, MySQL, MariaDB and MongoDB. However, Databasus was originally created specifically for PostgreSQL and maintains its primary focus on it — providing 100% excellent support and maximum efficiency for PostgreSQL backups. While MySQL, MariaDB and MongoDB are supported, PostgreSQL remains the core priority with the most optimized features and ongoing development. For example, Databasus provide native support of physical and WAL backups for PostgreSQL disaster recovery. So Databasus is actually PostgreSQL backup tools, other DBs are just extensions.",
                },
              },
              {
                "@type": "Question",
                name: "What is Databasus adoption level?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus is the most widely adopted open-source PostgreSQL backup tool today. At the moment of 17 June 2026, it has been pulled over 1,800,000 times on Docker by DBAs, DevOps engineers, developers and teams worldwide. With 8,500+ GitHub stars, it surpasses pgBackRest (~4,200 stars, available since 2014) and WAL-G (~4,100 stars, available since 2017). Databasus launched in 2025 and outpaced both within its first year. This adoption level reflects strong community trust and preference among database professionals.",
                },
              },
              {
                "@type": "Question",
                name: "What backup types does Databasus support?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus supports physical, full, incremental, WAL and logical backups. Physical backups are a file-level copy of the entire database cluster — faster to back up and restore for large datasets than logical dumps, and built on PostgreSQL 17's native backup mechanism, so we rely on PostgreSQL's own battle-tested tooling instead of re-inventing it. Full backups are a complete, self-contained copy of the cluster, the base every backup chain starts from. Incremental backups store only what changed since the previous backup, so backups stay small and fast. WAL streaming continuously captures the database write stream, enabling Point-in-time recovery (PITR) for disaster recovery and near-zero data loss. Logical backups are a native dump of the database in its engine-specific binary format, compressed and streamed directly to storage with no intermediate files. All of these backups can run over an SSH tunnel if you have a requirement for non-public connections, so the database never has to be exposed publicly. SSH tunneling is built in.",
                },
              },
              {
                "@type": "Question",
                name: "I forgot my admin email or password",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: 'The first account created on the instance administers it. If you cannot remember which address that is, run docker exec -it databasus ./main --list-admins on the server where Databasus runs — it names every administrator account with its email address, display name, creation date and active state, and prints no password material. To set a new password for that account, run docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com". An instance created before the first account administered it still carries the placeholder address admin until its owner replaces it. If the instance also requires an emailed code at sign-in and the mail server has stopped delivering it, run docker exec -it databasus ./main --disable-2fa to stop requiring the code. All three commands are documented on the password page.',
                },
              },
              {
                "@type": "Question",
                name: "How is AI used in Databasus development?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "AI helps us develop Databasus within a defined engineering process. People set the direction and make the final decision. A human reads every line before a change is merged. Independent agents review correctness, architecture, tests and documentation in separate, read-only sessions. Strict linting and tests with real databases support their reviews. Required CI checks must pass before release.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        sectionLinks={[
          { href: "#how-to-use", label: "How to use" },
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
              PostgreSQL backup tool
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus is a free, open source and self-hosted tool to backup
              PostgreSQL. Make backups with different storages (S3, Azure Blob Storage, FTP, etc.) and notifications about progress (Slack,
              Discord, Telegram, etc.). With a focus on Point-in-Time Recovery{" "}
              <span className="underline decoration-2 underline-offset-2 decoration-blue-600">
                at low RPO/RTO
              </span>
            </p>

            <div>
              <div className="flex flex-col items-center justify-center gap-2 max-w-[370px] sm:max-w-[340px] mx-auto">
                <a
                  href="#installation"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-1"
                >
                  Self-host via Docker
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
                alt="Databasus dashboard interface"
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
                alt="Support by Anthropic and OpenAI OSS"
              />

              <div className="flex justify-center text-base md:text-xl mt-4 md:mt-0 md:ml-10">
                <div className="max-w-[370px] text-gray-400 text-center md:text-left">
                  Supported by both Anthropic and OpenAI open source programs.{" "}
                  <a
                    href="/faq#oss-programs"
                    target="_blank"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Learn&nbsp;more&nbsp;→
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
              <span className="text-sm font-medium">Overview</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Features
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus provides everything you need for reliable production
              backup management. From automated scheduling to backups
              encryption. Suitable well for both individual developers with
              personal projects, DevOps teams and enterprises
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
                Scheduled backups
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Scheduled backups"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Backup is a thing that should be done in specified time on
                regular basis. So we provide many options: hourly, daily,
                weekly, monthly, cron, etc.
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Configurable health checks
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Health checks"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Each minute (or any another amount of time) the system will ping
                your database and show you the history of attempts
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                The database can be considered as down after 3 failed attempts
                or so. Once DB is healthy again - you receive notification too
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Many destinations to store
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Files are kept on VPS, cloud storages and other places. You can
                choose any storage you. Files are always owned by you.{" "}
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
                  alt="Storage"
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
                Notifications
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                You can receive notifications about success or fail of the
                process. This is useful for developers or DevOps teams.{" "}
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
                  alt="Notifications"
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
                Databasus runs on your PC or VPS. Therefore, all your data is
                owned by you and secured. Deploy takes about 2 minutes via
                script, Docker or k8s
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
                Open source and free
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                The project is fully open source, free and have Apache 2.0
                license. You can copy and fork the code on your own. Open for
                enterprise as well
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
                  Restore verification
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  A backup that finishes without error is not the same as a
                  backup you can restore. Databasus periodically pulls the
                  latest backup, restores it into a throwaway database container
                  and reports the outcome.{" "}
                  <a
                    href="/restore-verification"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Read more →
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
                  Security
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Enterprise-grade encryption protects sensitive data and
                  backups. Read-only database access prevents data corruption.
                  Everything this does not require any knowledge and ready out
                  of the box from the start automatically.{" "}
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
                    alt="Security"
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
                  alt="Access management"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Provide access for users to view or manage DBs. Separate teams
                and projects. Suitable for DevOps teams and developers.{" "}
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
                  alt="Audit logs"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Track all system activities with comprehensive audit logs. You
                can view access and changes history for each user (backup
                downloads, schedule changes, config updates, etc.).{" "}
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
                Enterprise-grade encryption protects sensitive data and backups.
                Read-only database access prevents data corruption. Everything
                this does not require any knowledge and ready out of the box
                from the start automatically.{" "}
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
                  alt="Security"
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
                  Logical, physical, incremental and WAL backups
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus supports logical, physical (full and incremental)
                  backups with WAL-streaming for Point-in-Time Recovery. This
                  makes Databasus suitable for disaster recovery, and works
                  equally well with self-hosted and cloud databases — use remote
                  mode for cloud-managed or publicly accessible DBs. Physical
                  backups are made over PG 17 native backups.{" "}
                  <a
                    href="/faq/#pitr"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Read more →
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
                <span className="text-sm font-medium">4-minutes overview</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                How to use Databasus?
              </h2>

              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                Watch in this video how to connect your database, how to
                configure backups schedule, how to download and restore backups,
                how to add team members and what is users&apos; audit logs
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
                  <p className="text-sm text-gray-400">
                    Developer of Databasus
                  </p>
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
              <span className="text-sm font-medium">Databases</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Supported databases
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Databasus supports PostgreSQL, MySQL, MariaDB and MongoDB. You can
              backup and restore all of them with the same tool. Primary focus
              is on PostgreSQL, but MySQL, MariaDB and MongoDB are supported too
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
                PostgreSQL is the primary database supported by Databasus. All
                versions from 12 to 18 are supported
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
                MySQL is the second most popular database in the world. You can
                backup and restore your MySQL databases with the same simplicity
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Read more →
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
                MariaDB is supported with the same features as MySQL. You can
                backup and restore your MariaDB databases seamlessly
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Read more →
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
                MongoDB is the most popular NoSQL database. You can backup and
                restore your MongoDB databases with the same easy-to-use
                interface
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/mongodb-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Read more →
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
              <span className="text-sm font-medium">Process</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              How to make backups?
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              The main priority for Databasus is simplicity, right now this is
              the easiest tool to backup PostgreSQL in the world. To make
              backups, you need to follow 4 steps. After that, you will be able
              to restore in one click
            </p>
          </div>

          {/* Steps */}
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            {/* Step 1 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Step 1
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Select required schedule
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    You can choose any time you need: daily, weekly, monthly,
                    particular time (like 4 AM) and cron cycles
                  </p>
                  <p>
                    For week interval you need to specify particular day, for
                    month you need to specify particular day
                  </p>
                  <p>
                    If your database is large, we recommend you choosing the
                    time when there are decrease in traffic
                  </p>
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
                Step 2
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Enter your database info
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Enter credentials of your PostgreSQL database, select
                    version and target DB. Also choose whether SSL is required
                  </p>
                  <p>
                    Databasus, by default, compress backups at balance level to
                    not slow down backup process (~20% slower) and save x4-x8 of
                    the space (that decreasing network traffic)
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
                Step 3
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Choose storage for backups
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    You can keep files with backups locally, in S3, in Azure Blob Storage, NAS, FTP and other services
                  </p>
                  <p>
                    Please keep in mind that you need to have enough space on
                    the storage
                  </p>
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
                Step 4
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Choose where you want to receive notifications (optional)
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    When backup succeed or failed, Databasus is able to send you
                    notification. It can be chat with DevOps, your emails or
                    even webhook of your team
                  </p>
                  <p>
                    We are going to support the most of popular messangers and
                    platforms
                  </p>
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
              Get started
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
                <span className="text-sm font-medium">Get started</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                How to install?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus support many ways of installation. All ways are
                extremely simple and easy to use even for those who has no
                experience in administration or DevOps. Both local and cloud DBs
                are supported.
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
              Frequent questions
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              The goal of Databasus — make backing up as simple as possible for
              single developers (as well as DevOps) and teams. UI makes it easy
              to create backups and visualizes the progress and restores
              anything in couple of clicks
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="What is Databasus and why should I use it instead of hand-rolled scripts?"
              answer="Databasus is an Apache 2.0 licensed, self-hosted service backing up databases. It differs from shell scripts in that it has a frontend for scheduling tasks, compressing and storing archives on multiple targets (local disk, S3, Azure Blob Storage, NAS, FTP, SFTP, etc.), configuring retention policies to automatically prune old backups and notifying your team when tasks finish or fail — all without hand-rolled code"
            />
            <FaqItem
              number="2"
              question="How do I install Databasus in the quickest manner?"
              answer="Databasus supports multiple installation methods: automated script, Docker, Docker Compose and Kubernetes with Helm. The quickest route is to run the one-line cURL installer, which fetches the current Docker image, creates a docker-compose.yml and boots up the service so it will automatically restart on reboots. For Kubernetes environments, use the official Helm chart for production-ready deployments. Overall time is usually less than two minutes on a typical VPS."
            />
            <FaqItem
              number="3"
              question="How restore verification works?"
              answer="Databasus runs a small verification agent on a host you control. On every scheduled run the agent downloads the latest backup. It restores the backup into a throwaway database container. Then it sanity-checks the restored database against the source. The result is reported back — including the restore exit code and per-table row counts. Schedules support after backup, hourly, daily, weekly, monthly or a UTC cron expression. Failures can be sent through any notifier wired to the database — Slack, Teams, Discord, email and others."
            />
            <FaqItem
              number="4"
              question="How does Databasus ensure security?"
              answer={
                <>
                  Databasus enforces security on three levels: (1) Sensitive
                  data encryption — all passwords, tokens and credentials are
                  encrypted with AES-256-GCM and stored separately from the
                  database; (2) Backup encryption — each backup file is
                  encrypted with a unique key derived from a master key, backup
                  ID and random salt, making backups useless without your
                  encryption key even if someone gains storage access; (3)
                  Read-only database access — Databasus only requires SELECT
                  permissions and performs comprehensive checks to ensure no
                  write privileges exist, preventing data corruption even if the
                  tool is compromised.
                  <br />
                  <br />
                  Sign-in itself can require a second factor: with it on, a
                  correct password is followed by a six-digit code emailed to
                  the account, while sign-in through Google or GitHub keeps
                  relying on the provider&apos;s own checks.
                  <br />
                  <br />
                  Before each release, automated checks scan code, dependencies, secrets and
                  container images. Tests verify full backup-and-restore cycles against real
                  PostgreSQL, MySQL, MariaDB and MongoDB containers. Required checks must pass
                  before publication. Regular Codex Security audits from OpenAI complement
                  these checks and are separate from the release pipeline.
                  <br />
                  <br />
                  See{" "}
                  <a
                    href={`${getLocalizedHref("en", "security")}#security-and-reliability-engineering`}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Security and reliability engineering
                  </a>{" "}
                  for the full pipeline.
                  <br />
                  <br />
                  Moreover, all the logs can be exported to any external system
                  (VictoriaLogs, SigNoz, Graylog, etc.) via the OpenTelemetry
                  standard. By default, logs (including audit logs) are also
                  written to local files, so audit logs cannot be lost. See{" "}
                  <a
                    href="/advanced-config#logging"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    advanced config
                  </a>{" "}
                  here.
                </>
              }
            />
            <FaqItem
              number="5"
              question="How do I set up and run my first backup job in Databasus?"
              answer={
                <>
                  To start your very first Databasus backup, simply log in to
                  the dashboard, click on New Backup, select an interval —
                  hourly, daily, weekly, monthly or cron. Then specify the exact
                  run time (e.g., 02:30 for off-peak hours).
                  <br />
                  <br />
                  Then input your PostgreSQL host, port number, database name,
                  credentials and SSL preference. Choose where the archive
                  should be sent (local path, S3 bucket, Azure Blob Storage,
                  FTP, etc.). <br />
                  <br />
                  If you need, add notification channels such as email, Slack,
                  Telegram or a webhook and click Save. Databasus instantly
                  validates the info, starts the schedule, runs the initial job
                  and sends live status. So you may restore with one touch when
                  the backup is complete.
                </>
              }
            />
            <FaqItem
              number="6"
              question="What is Databasus adoption level?"
              answer="Databasus is the most widely adopted open-source PostgreSQL backup tool today. At the moment of 17 June 2026, it has been pulled over 1,800,000 times on Docker by DBAs, DevOps engineers, developers and teams worldwide. With 8,500+ GitHub stars, it surpasses pgBackRest (~4,200 stars, available since 2014) and WAL-G (~4,100 stars, available since 2017). Databasus launched in 2025 and outpaced both within its first year. This adoption level reflects strong community trust and preference among database professionals."
            />
            <FaqItem
              number="7"
              question="How is Databasus different from PgBackRest, Barman or pg_dump? Where can I read comparisons?"
              answer={
                <>
                  Databasus prefers simplicity — it provides a modern web
                  interface to manage backups for many databases at once,
                  instead of complex configuration files and command-line tools.
                  Unlike raw pg_dump scripts, it includes built-in scheduling,
                  compression, multiple storage destinations, health monitoring
                  and real-time notifications — all managed through a simple web
                  UI.
                  <br />
                  <br />
                  At the same time, unlike pgBackRest and WAL-G, Databasus makes
                  physical, incremental and WAL backups on top of PostgreSQL
                  17&apos;s native approach, so it does not reinvent its own
                  backup engine. It connects to your databases remotely,
                  reaching closed networks through an SSH tunnel to the server
                  or a bastion, so databases that are not publicly exposed can
                  still be backed up and managed from a single dashboard.{" "}
                  <a
                    href="/faq/#pitr"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Read how physical and PITR backups implemented
                  </a>
                  .
                  <br />
                  <br />
                  We have detailed comparison pages for popular backup tools:{" "}
                  <a
                    href="/pgdump-alternative"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pg_dump
                  </a>
                  ,{" "}
                  <a
                    href="/databasus-vs-pgbackrest"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackRest
                  </a>
                  ,{" "}
                  <a
                    href="/databasus-vs-barman"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs Barman
                  </a>
                  ,{" "}
                  <a
                    href="/databasus-vs-wal-g"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs WAL-G
                  </a>{" "}
                  and{" "}
                  <a
                    href="/databasus-vs-pgbackweb"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackWeb
                  </a>
                  . Each comparison explains the key differences, pros and cons,
                  and helps you choose the right tool for your needs.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Is Databasus supported by Anthropic and OpenAI OSS programs?"
              answer={
                <>
                  Yes, we are proud that Databasus has been recognized as a
                  valuable open-source project by two of the world&apos;s
                  leading AI companies. In March 2026, Databasus was accepted
                  into both{" "}
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Claude for Open Source
                  </a>{" "}
                  by Anthropic and{" "}
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Codex for Open Source
                  </a>{" "}
                  by OpenAI. This is an independent reliability confirmation for
                  us that the project has been evaluated and recognized as
                  critical infrastructure worth supporting.{" "}
                  <a
                    href="/faq#oss-programs"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Read more →
                  </a>
                </>
              }
            />
            <FaqItem
              number="9"
              question="Is Databasus an alternative to pg_dump?"
              answer="Not exactly. Databasus focuses on disaster recovery with low RTO and RPO, so it is closer to an alternative to pgBackRest or WAL-G — it tries to make disaster recovery as simple as pg_dump. That said, for logical backups it does serve as a pg_dump alternative and uses pg_dump under the hood, adding a user-friendly web interface, automated scheduling, multiple storage destinations, real-time notifications, health monitoring and backup encryption. Logical backups are also available for MySQL, MariaDB and MongoDB."
            />
            <FaqItem
              number="10"
              question="Which databases does Databasus support?"
              answer={
                <>
                  Databasus supports PostgreSQL, MySQL, MariaDB and MongoDB.
                  However, Databasus was originally created specifically for
                  PostgreSQL and maintains its primary focus on it — providing
                  100% excellent support and maximum efficiency for PostgreSQL
                  backups.
                  <br />
                  <br />
                  While MySQL, MariaDB and MongoDB are supported, PostgreSQL
                  remains the core priority with the most optimized features and
                  ongoing development.
                  <br />
                  <br />
                  For example, Databasus provide native support of physical and
                  WAL backups for PostgreSQL disaster recovery. So Databasus is
                  actually PostgreSQL backup tools, other DBs are just
                  extensions.
                </>
              }
            />
            <FaqItem
              number="11"
              question="What backup types does Databasus support?"
              answer={
                <>
                  Databasus supports physical, full, incremental, WAL and
                  logical backups — so it suits both those who want simple
                  logical dumps and those who need a solid disaster recovery
                  tool.
                  <ul className="list-disc list-inside mt-3 space-y-2">
                    <li>
                      <strong>Physical</strong> — file-level copy of the entire
                      database cluster. Faster backup and restore for large
                      datasets than logical dumps. Built on PostgreSQL 17&apos;s
                      native backup mechanism, so we rely on PostgreSQL&apos;s
                      own battle-tested tooling instead of re-inventing it
                    </li>
                    <li>
                      <strong>Full</strong> — a complete, self-contained copy of
                      the cluster, the base every backup chain starts from
                    </li>
                    <li>
                      <strong>Incremental</strong> — stores only what changed
                      since the previous backup, so backups stay small and fast
                    </li>
                    <li>
                      <strong>WAL streaming</strong> — continuously captures the
                      database write stream, enabling Point-in-time recovery
                      (PITR). Designed for disaster recovery and near-zero data
                      loss
                    </li>
                    <li>
                      <strong>Logical</strong> — native dump of the database in
                      its engine-specific binary format. Compressed and streamed
                      directly to storage with no intermediate files
                    </li>
                  </ul>
                  <br />
                  Physical, incremental and WAL backups build on PostgreSQL
                  17&apos;s native mechanism, so they require PostgreSQL 17 or
                  newer; on older versions only logical backups are available.
                  This is intentional: most production databases already run on
                  PostgreSQL 17 or above, and within roughly two years the older
                  versions reach end of life. Databasus aims to become the
                  standard backup tool for databases from PostgreSQL 17 onward.
                  <br />
                  <br />
                  All of these backups can run over an SSH tunnel if you have a
                  requirement for non-public connections, so the database never
                  has to be exposed publicly. SSH tunneling is built in.
                </>
              }
            />
            <FaqItem
              number="12"
              question="How is AI used in Databasus development?"
              answer={
                <>
                  <p>
                    AI helps us develop Databasus within a defined engineering process. People set the direction and
                    make the final decision. <strong>A human reads every line before a change is merged.</strong>
                  </p>
                  <p className="mt-3">
                    Independent agents review correctness, architecture, tests and documentation in separate,
                    read-only sessions. Strict linting and tests with real databases support their reviews. Required
                    CI checks must pass before release.{" "}
                    <a
                      href={`${getLocalizedHref("en", "faq")}#ai-usage`}
                      className="text-blue-400 hover:text-blue-600"
                    >
                      Read about the development process
                    </a>
                    .
                  </p>
                </>
              }
            />
            <FaqItem
              number="13"
              question="How can I join the Databasus community?"
              answer={
                <>
                  You can join our large community of developers, DBAs and
                  DevOps engineers at{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    t.me/databasus_community
                  </a>
                  . The community is a great place to ask questions, share
                  experiences, get help with configuration and stay updated with
                  the latest features and releases.
                </>
              }
            />
            <FaqItem
              number="14"
              question="What is the adoption level of Databasus?"
              answer={
                <>
                  Databasus has over 1.8 million Docker pulls and 8.5k GitHub
                  stars. For comparison, pgBackRest and WAL-G sit at around 4.2k
                  stars each and Barman at about 3.1k, which makes Databasus the
                  most popular database backup tool on GitHub.
                  <br />
                  <br />
                  It has been accepted into the open-source programs of
                  Anthropic and OpenAI as an important, critical project. Today
                  Databasus is used by enterprises, teams and DevOps engineers,
                  backed by a large and active community.
                  <br />
                  <br />
                  Databasus has been developed and used since 2023, and open
                  source in widespread use since early 2025. It has been in real
                  production use for a while, so it is battle-tested across many
                  edge cases. Crucially, Databasus does not invent custom ways
                  to back up your data — it relies on PostgreSQL&apos;s native,
                  tested implementation instead of building its own workarounds
                  for edge cases.
                  <br />
                  <br />
                  Our goal is to become the standard backup tool for PostgreSQL
                  from version 17 and above. Databasus is the first backup tool
                  built on PostgreSQL&apos;s native, efficient and now standard
                  backup protocol instead of writing its own implementations.
                </>
              }
            />
            <FaqItem
              number="15"
              question="I forgot my admin email or password"
              answer={
                <>
                  The first account created on the instance administers it. If
                  you cannot remember which address that is, run docker exec -it
                  databasus ./main --list-admins on the server where Databasus
                  runs — it names every administrator account with its email
                  address, display name, creation date and active state, and
                  prints no password material. To set a new password for that
                  account, run docker exec -it databasus ./main
                  --new-password=&quot;YourNewSecurePassword123&quot;
                  --email=&quot;owner@example.com&quot;. An instance created
                  before the first account administered it still carries the
                  placeholder address admin until its owner replaces it. If the
                  instance also requires an emailed code at sign-in and the mail
                  server has stopped delivering it, run docker exec -it
                  databasus ./main --disable-2fa to stop requiring the code. All
                  three commands are documented on the{" "}
                  <a
                    href="/password"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    password page
                  </a>
                  .
                </>
              }
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
