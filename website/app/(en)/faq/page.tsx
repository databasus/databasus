import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "FAQ - Frequently Asked Questions | Databasus",
  description:
    "Frequently asked questions about Databasus PostgreSQL backup tool with MySQL, MariaDB and MongoDB support. Learn how to backup localhost databases, understand backup formats, compression methods and more.",
  keywords: [
    "Databasus FAQ",
    "PostgreSQL backup questions",
    "localhost database backup",
    "backup formats",
    "pg_dump compression",
    "zstd compression",
    "PostgreSQL backup help",
    "database backup guide",
  ],
  openGraph: {
    title: "FAQ - Frequently Asked Questions | Databasus",
    description:
      "Frequently asked questions about Databasus PostgreSQL backup tool with MySQL, MariaDB and MongoDB support. Learn how to backup localhost databases, understand backup formats, compression methods and more.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/faq",
  },
  twitter: {
    card: "summary",
    title: "FAQ - Frequently Asked Questions | Databasus",
    description:
      "Frequently asked questions about Databasus PostgreSQL backup tool with MySQL, MariaDB and MongoDB support. Learn how to backup localhost databases, understand backup formats, compression methods and more.",
  },
  alternates: {
    canonical: "https://databasus.com/faq",
    languages: getLanguageAlternates("faq"),
  },
  robots: "index, follow",
};

export default function FAQPage() {
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
                name: "Why does Databasus not use raw SQL dump format for logical PostgreSQL backups?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "For logical backups, Databasus uses pg_dump's custom format with zstd compression because it provides the most efficient backup and restore speed after extensive testing. The custom format with zstd compression level 5 offers the optimal balance between backup creation speed, restore speed and file size.",
                },
              },
              {
                "@type": "Question",
                name: "Where is Databasus installed?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus is installed in /opt/databasus/",
                },
              },
              {
                "@type": "Question",
                name: "How do physical and PITR (Point-in-Time Recovery) backups work?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus runs physical backups remotely from its own host, connecting to your PostgreSQL over the standard replication protocol, so nothing needs to be installed on the database server. Databases in closed networks can be reached through an SSH tunnel. Physical backups use PostgreSQL 17's native stack: full backups via pg_basebackup, block-level incrementals via pg_basebackup --incremental driven by server-side WAL summaries (summarize_wal = on), and continuous WAL streaming via pg_receivewal. Physical backups require PostgreSQL 17 or newer; on older versions you use logical pg_dump backups instead. To restore to a point in time, pg_combinebackup reconstructs a runnable data directory from the full backup and its incremental chain, and PostgreSQL then replays WAL up to the target time you choose, recovering to any second between backups. The Databasus UI gives step-by-step instructions for restoring to a host or a Docker database, either through a ready-made script that makes the restore a single command or by downloading the backups and rebuilding the chain of full, incremental and WAL parts yourself. Incremental and WAL are optional: you can take only a full backup, and WAL is not mandatory. We use PostgreSQL 17 native backups because they reuse PostgreSQL's own battle-tested backup machinery instead of reinventing it, work with remote databases including managed services like RDS and Cloud SQL, and give near-zero data loss.",
                },
              },
              {
                "@type": "Question",
                name: "Why did Databasus move away from agent-based backups?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "An earlier version of Databasus shipped a backup agent: a binary that ran on the database host to stream WAL and create physical backups locally. That first implementation turned out to be a mistake and has been removed. It was a naive implementation that only copied WAL on top of full backups, which led to a long RTO. Users had to configure both Databasus and a separate agent, when doing everything remotely from one place is far simpler. Because the agent lived outside the main system, it was hard to cover every test case. There is really only one problem an agent solves: reaching a database that is not accessible from outside, and for 99% of users that is already handled by running Databasus inside the private network or connecting over SSH, so the agent was reinventing the wheel and making a simple problem far more complicated than needed. It also could not run on managed databases like RDS and Cloud SQL, which forbid host-level installs but already expose the replication protocol, so a remote path was needed anyway. On top of that it came with a lot of edge cases around broken connections, managing agent updates and gathering logs from a separate process, and the fewer moving parts a system has, the more reliable it is in everyday use. Physical backups now run remotely from the Databasus host. Existing backups stay safe: if you upgrade from a version that still has agent backups, Databasus won't do it silently but warns you about the change and lets you either stay on the supported version 3.42.0 or remove the old agent backups yourself before upgrading. The agent-based implementation remains available up to version 3.42.0 and will keep working for a long time.",
                },
              },
              {
                "@type": "Question",
                name: "How to backup Databasus itself?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "To backup Databasus, go to /opt/databasus (or the folder where you installed it), then navigate to the databasus-data directory. You need to backup the secret.key file (encryption key for credentials) and the /pgdata folder (internal database containing configurations and backup metadata). There are two recovery scenarios: 1) You can recover database backups using only secret.key without Databasus UI (see manual recovery guide), 2) To restore Databasus UI with all configurations and history, you need both secret.key and /pgdata folder. To restore, recreate this folder structure on another server.",
                },
              },
              {
                "@type": "Question",
                name: "How is Databasus supported by Anthropic and OpenAI open-source programs?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "In March 2026, Databasus was accepted into both Claude for Open Source by Anthropic and Codex for Open Source by OpenAI. Being backed by these programs is a reliability signal — the project has been independently evaluated and recognized by industry leaders as critical open-source infrastructure worth supporting.",
                },
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="faq">Frequently Asked Questions</h1>

              <p className="text-lg text-gray-400">
                Find answers to the most common questions about Databasus,
                including installation, configuration and backup strategies.
              </p>

              <h2 id="why-no-raw-sql-dump">
                Why does Databasus not use raw SQL dump format for logical
                PostgreSQL backups?
              </h2>

              <p>
                For logical backups, Databasus uses <code>pg_dump</code>&apos;s{" "}
                <strong>custom format</strong> with{" "}
                <strong>zstd compression at level 5</strong> instead of the
                plain SQL format because it provides the most efficient balance
                between:
              </p>

              <ul>
                <li>Backup creation speed</li>
                <li>Restore speed</li>
                <li>
                  File size compression (up to 20x times smaller than plain SQL
                  format)
                </li>
              </ul>

              <p>
                This decision was made after extensive testing and benchmarking
                of different PostgreSQL backup formats and compression methods.
                You can read more about testing here{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  PostgreSQL backups: comparing pg_dump speed in different
                  formats and with different compression
                </a>
                .
              </p>

              <p>Databasus will not include raw SQL dump format, because:</p>

              <ul>
                <li>extra variety is bad for UX;</li>
                <li>makes it harder to support the code;</li>
                <li>current dump format is suitable for 99% of the cases</li>
              </ul>

              <h2 id="installation-directory">
                Where is Databasus installed if installed via .sh script?
              </h2>

              <p>
                Databasus is installed in <code>/opt/databasus/</code>{" "}
                directory.
              </p>

              <h2 id="pitr">
                How do physical and PITR (Point-in-Time Recovery) backups work?
              </h2>

              <p>
                Databasus runs physical backups{" "}
                <strong>remotely from its own host</strong>, connecting to your
                PostgreSQL over the standard{" "}
                <strong>replication protocol</strong>, so nothing needs to be
                installed on the database server. If the database lives in a
                closed network, Databasus can reach it
                through an SSH tunnel to an internal host or a bastion, so the
                database never has to be exposed publicly.
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
                    <strong>Why this is possible now:</strong> for years tools
                    like pgBackRest and WAL-G had to build their own engines for
                    incremental, block-level backups, because PostgreSQL had no
                    native one. That changed with PostgreSQL 17, where the
                    feature was developed by <strong>Robert Haas</strong> with
                    help from <strong>David Steele</strong>, the author of
                    pgBackRest. PostgreSQL now ships native server-side
                    block-level incremental backups under the hood (
                    <code>pg_basebackup --incremental</code> and{" "}
                    <code>summarize_wal</code>), so Databasus builds on that
                    instead of reinventing one.
                  </p>
                </div>
              </div>

              <p>
                <strong>How backups work:</strong>
              </p>

              <ul>
                <li>
                  Full backups are created with <code>pg_basebackup</code>,
                  streamed directly to Databasus
                </li>
                <li>
                  Block-level incrementals use{" "}
                  <code>pg_basebackup --incremental</code>, where PostgreSQL
                  17&apos;s server-side WAL summaries (
                  <code>summarize_wal = on</code>) track changes so only the
                  changed blocks are transferred
                </li>
                <li>
                  WAL is streamed continuously via <code>pg_receivewal</code> to
                  keep the recovery chain complete between backups
                </li>
                <li>
                  Physical backups require{" "}
                  <strong>PostgreSQL 17 or newer</strong>; on older versions you
                  use logical <code>pg_dump</code> backups instead
                </li>
              </ul>

              <p>
                <strong>How restoration works:</strong>
              </p>

              <ul>
                <li>
                  <code>pg_combinebackup</code> reconstructs a runnable data
                  directory from the full backup and its incremental chain
                </li>
                <li>
                  PostgreSQL then replays WAL up to the target time you choose,
                  recovering to any second between backups
                </li>
                <li>
                  Once you start PostgreSQL, it finishes recovery, promotes to
                  primary and resumes normal operations
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
                    <strong>You don&apos;t have to do this by hand.</strong> The
                    Databasus UI gives you step-by-step instructions for
                    restoring to a host or a Docker database, either through a
                    ready-made script or by downloading the backups manually. We
                    prepared the script so a restore is just one command, but you
                    can also rebuild the chain of full, incremental and WAL parts
                    yourself if you prefer. Incremental and WAL are optional too:
                    you can take only a full backup, without incrementals, and
                    WAL is not mandatory.
                  </p>
                </div>
              </div>

              <p>
                <strong>Why we use PG 17 native backups:</strong>
              </p>

              <ul>
                <li>
                  They reuse PostgreSQL&apos;s own backup machinery instead of
                  reinventing it, so you get battle-tested internals with
                  thousands of tests and edge-cases behind them
                </li>
                <li>
                  They work with remote databases, including managed services
                  like Amazon RDS and Google Cloud SQL that expose the
                  replication protocol but forbid installing software on the
                  host
                </li>
                <li>
                  They give near-zero data loss, letting you restore to any
                  second between backups
                </li>
              </ul>

              <h2 id="why-no-agent">
                Why did Databasus move away from agent-based backups?
              </h2>

              <p>
                An earlier version of Databasus shipped a backup{" "}
                <strong>agent</strong>: a binary that ran on the database host to
                stream WAL and create physical backups locally. That first
                implementation turned out to be a mistake, and we removed it.
                Physical backups now run remotely from the Databasus host, as
                described above.
              </p>

              <p>
                <strong>Why the agent was the wrong approach:</strong>
              </p>

              <ul>
                <li>
                  It was a naive implementation that only copied WAL on top of
                  full backups, which led to a long RTO
                </li>
                <li>
                  Users had to configure both Databasus and a separate agent,
                  when doing everything remotely from one place is far simpler
                </li>
                <li>
                  Because the agent lived outside the main system, it was hard to
                  cover every test case
                </li>
                <li>
                  There is really only one problem an agent solves: reaching a
                  database that is not accessible from outside. For 99% of users
                  that is already handled by running Databasus inside the private
                  network or connecting over SSH, so the agent was reinventing
                  the wheel and making a simple problem far more complicated than
                  it needed to be
                </li>
                <li>
                  It could not run on managed databases like RDS and Cloud SQL,
                  which forbid host-level installs but already expose the
                  replication protocol, so a remote path was needed anyway
                </li>
                <li>
                  It also came with a lot of edge cases. Broken connections,
                  managing agent updates and gathering logs from a separate
                  process were all painful, and the fewer moving parts a system
                  has, the more reliable it is in everyday use
                </li>
              </ul>

              <p>
                <strong>We made sure existing backups stay safe.</strong> If you
                upgrade from a version that still has agent backups, Databasus
                won&apos;t do it silently: it warns you about the change and lets
                you either stay on the supported{" "}
                <strong>version 3.42.0</strong> or remove the old agent backups
                yourself before upgrading. The agent-based implementation remains
                available up to version 3.42.0 and will keep working for a long
                time, so nothing breaks.
              </p>

              <p>
                You can read the full reasoning in the architecture decision
                records:{" "}
                <a
                  href="https://github.com/databasus/databasus/blob/main/adr/0008-why-pg17-native-backups-with-mandatory-wal-summary.md"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  ADR-0008: PG17-native backups with mandatory WAL summary
                </a>{" "}
                and{" "}
                <a
                  href="https://github.com/databasus/databasus/blob/main/adr/0009-why-remote-physical-backups-instead-of-agents.md"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  ADR-0009: remote physical backups instead of agents
                </a>
                .
              </p>

              <h2 id="backup-databasus">How to backup Databasus itself?</h2>

              <p>
                If you want to backup your Databasus instance (including all
                configurations, databases and credentials), follow these steps:
              </p>

              <ol>
                <li>
                  Go to <code>/opt/databasus</code> (or the folder where you
                  installed Databasus)
                </li>
                <li>
                  Navigate to the <code>databasus-data</code> directory
                </li>
              </ol>

              <p>
                <strong>You need to backup:</strong>
              </p>

              <ul>
                <li>
                  <code>secret.key</code> — encryption key for your credentials
                </li>
                <li>
                  <code>/pgdata</code> — internal PostgreSQL database of
                  Databasus that contains all your configurations and backup
                  metadata
                </li>
              </ul>

              <p>
                If you use local storage for backups, you can also backup the{" "}
                <code>backups</code> folder.
              </p>

              <p>
                <strong>Important:</strong> There are two different scenarios
                for recovery:
              </p>

              <ul>
                <li>
                  <strong>Recover backups without Databasus UI:</strong> You can
                  recover your database backups using only the{" "}
                  <code>secret.key</code> file, without needing Databasus or its
                  internal data. See the{" "}
                  <a href="/how-to-recover-without-databasus">
                    manual recovery guide
                  </a>{" "}
                  for detailed instructions.
                </li>
                <li>
                  <strong>Restore Databasus UI and all configurations:</strong>{" "}
                  If you want to restore the Databasus interface with all your
                  configurations, scheduled backups and backup history, you need
                  to backup both <code>secret.key</code> and the{" "}
                  <code>/pgdata</code> folder (which contains the encryption
                  metadata and all Databasus configurations).
                </li>
              </ul>

              <p>
                <strong>To restore Databasus on another server:</strong> simply
                recreate the <code>databasus-data</code> folder structure with
                the backed up files and start Databasus.
              </p>

              <h2 id="oss-programs">
                How is Databasus supported by Anthropic and OpenAI open-source
                programs?
              </h2>

              <p>
                In March 2026, Databasus was accepted into both{" "}
                <strong>
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Claude for Open Source
                  </a>
                </strong>{" "}
                by Anthropic and{" "}
                <strong>
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Codex for Open Source
                  </a>
                </strong>{" "}
                by OpenAI. It is really valuable for us that the project has
                been recognized as important open-source software for the
                industry by two of the world&apos;s leading AI companies —
                especially given the high eligibility requirements of both
                programs.
              </p>

              <p>
                What does it mean for users? It just one more reliability
                confirmation that the project has been independently evaluated
                and recognized by industry leaders as critical infrastructure
                worth supporting. So we have even higher code quality, faster
                security reviews and continued active development due to access
                to the latest unlimited AIs.
              </p>

              <img
                src="/images/faq/anthropic-email.png"
                alt="Databasus accepted into Claude for Open Source program by Anthropic"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <img
                src="/images/faq/openai-email.png"
                alt="Databasus accepted into Codex for Open Source program by OpenAI"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
