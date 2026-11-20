import type { Metadata } from "next";
import { getLanguageAlternates, getLocalizedHref } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

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
  const startupRecoveryCommand = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  -e RECOVERY_DUMP_PATH=/databasus-data/recovery/backup.dbbackup \\
  -e RECOVERY_MASTER_KEY_PATH=/databasus-data/recovery/master.key \\
  --restart unless-stopped \\
  databasus/databasus:<version>`;

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
                  text: "Databasus runs physical backups remotely from its own host, connecting to your PostgreSQL over the standard replication protocol, so nothing needs to be installed on the database server. Databases in closed networks can be reached through an SSH tunnel. Physical backups use PostgreSQL 17's native stack: full backups via pg_basebackup, block-level incrementals via pg_basebackup --incremental driven by server-side WAL summaries (summarize_wal = on), and continuous WAL streaming via pg_receivewal. Physical backups require PostgreSQL 17 or newer; on older versions you use logical pg_dump backups instead. To restore to a point in time, pg_combinebackup reconstructs a runnable data directory from the full backup and its incremental chain, and PostgreSQL then replays WAL up to the target time you choose, recovering to any second between backups. The Databasus UI gives step-by-step instructions for restoring to a host or a Docker database, either through a ready-made script that makes the restore a single command or by downloading the backups and rebuilding the chain of full, incremental and WAL parts yourself. Incremental and WAL are optional: you can take only a full backup, and WAL is not mandatory. We use PostgreSQL 17 native backups because they reuse PostgreSQL's own battle-tested backup machinery instead of reinventing it and give near-zero data loss.",
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
                  text: "Databasus is built to stay recoverable even if its own server is lost or corrupted. It can back up its own internal database: accounts, settings, the encrypted credentials of databases, storages and notifiers, schedules and the backup catalog. Turn it on in Databasus settings → Databasus backups by selecting one or more existing storages; by default a Databasus backup runs every hour and is kept for 30 days. The file is encrypted with the master key and contains neither the key nor the backup files of your databases, so save the master key separately: Databasus asks you to download it, and it is also . /databasus-data/secret. key on the host. Recovery needs the backup file, its master key and exactly the same Databasus version, patch included. Either put both files into the mounted data directory and start Databasus with RECOVERY_DUMP_PATH and RECOVERY_MASTER_KEY_PATH on a server without accounts or changed settings, or upload the file and paste the key in Databasus settings → Databasus backups, which replaces the current installation. Stop the old installation before you recover. A recovered installation starts its backups, notifications and storage writes at once, so two running installations would write to the same storages and databases.",
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
              {
                "@type": "Question",
                name: "How is AI used in Databasus development?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "AI helps us develop Databasus, from planning and implementation to tests and documentation. People set the direction and make the final decision. A human reads every line before a change is merged. We use spec-driven development (SDD) with OpenSpec to agree on behavior before implementation. Independent review at each stage Automatic hooks start separate reviewer sessions with read-only access to the repository. Each agent inspects the relevant code and returns its own findings. The implementing agent cannot approve its own work. Review starts before implementation. Separate planning agents look for simpler solutions, check proposed names and assess the design. They also check that requirements and verification tasks agree. Code reviewer: Looks for bugs, missed edge cases, broken error handling, races, resource leaks and security problems. Requirements reviewer: Traces each requirement to its implementation and tests. Checks that the tests actually prove the promised behavior. Architecture reviewer: Checks module boundaries, dependency direction, data ownership, migrations and API rules. Code structure reviewer: Checks names, file and function responsibilities, comments and readability. Test reviewer: Checks whether tests would catch missing behavior or a regression. Also checks condition waits, cleanup and verification evidence. Documentation and task reviewer: Checks that documentation matches the code, tasks are complete and verification claims have evidence. Full implementation reviews run in parallel. Each changed specification gets its own requirements reviewer, alongside separate architecture, test and documentation reviewers. A change covering two specifications therefore gets five full-review agents. Blocking findings stop acceptance. The implementing agent fixes them or explains why a finding is incorrect. A follow-up reviewer checks that resolution and the subsequent edits. A review that could not run does not count as passed. Automated checks and real restores Independent review works alongside strict automated checks. golangci-lint and our architecture linter check backend dependencies, module boundaries and migration rules. API checks validate OpenAPI contracts and generated code. Frontend checks cover TypeScript, React, layer imports, styles and localization. Ruff and ty check Python formatting, lint rules and types. ShellCheck checks shell scripts. actionlint checks GitHub Actions workflows. Pre-commit hooks check affected backend, frontend and Python files without rewriting them. CI also runs linting. Tests exercise real behavior. Backend tests use the application API and check engines against real database containers. End-to-end scenarios create backups, restore them and inspect the restored data. Other tests cover upgrades, frontend screens and browser workflows with visual comparisons. Checks are chosen for each change. Required release suites run on amd64 and arm64. The implementing agent runs the relevant linters and tests before review; review agents inspect the evidence without rerunning those commands. Before publication, CodeQL and Semgrep analyze code, Gitleaks scans for secrets and dependency checks look for vulnerabilities. Trivy scans container images and Dockerfiles. Required CI checks must pass before release.",
                },
              },
              {
                "@type": "Question",
                name: "Is there a Databasus community?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes. The Databasus community talks in chats on Telegram (https://t.me/databasus_community) and Discord (https://databasus.com/discord). There you can ask questions, share how you run Databasus, discuss features and hear about new releases first. Pick the app you already use.",
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

              <h2 id="ai-usage">How is AI used in Databasus development?</h2>

              <p>
                AI helps us develop Databasus, from planning and implementation to tests and documentation.
                People set the direction and make the final decision.{" "}
                <strong>A human reads every line before a change is merged.</strong>
              </p>

              <p>
                We use spec-driven development (SDD) with{" "}
                <a href="https://github.com/databasus/databasus/tree/main/openspec">OpenSpec</a> to agree on
                behavior before implementation.
              </p>

              <h3 id="ai-review-hooks">Independent review at each stage</h3>

              <p>
                Automatic hooks start separate reviewer sessions with read-only access to the repository. Each
                agent inspects the relevant code and returns its own findings. The implementing agent cannot
                approve its own work.
              </p>

              <p>
                Review starts before implementation. Separate planning agents look for simpler solutions, check
                proposed names and assess the design. They also check that requirements and verification tasks
                agree.
              </p>

              <ul>
                <li>
                  <strong>Code reviewer:</strong> Looks for bugs, missed edge cases, broken error handling,
                  races, resource leaks and security problems.
                </li>
                <li>
                  <strong>Requirements reviewer:</strong> Traces each requirement to its implementation and
                  tests. Checks that the tests actually prove the promised behavior.
                </li>
                <li>
                  <strong>Architecture reviewer:</strong> Checks module boundaries, dependency direction, data
                  ownership, migrations and API rules.
                </li>
                <li>
                  <strong>Code structure reviewer:</strong> Checks names, file and function responsibilities,
                  comments and readability.
                </li>
                <li>
                  <strong>Test reviewer:</strong> Checks whether tests would catch missing behavior or a
                  regression. Also checks condition waits, cleanup and verification evidence.
                </li>
                <li>
                  <strong>Documentation and task reviewer:</strong> Checks that documentation matches the code,
                  tasks are complete and verification claims have evidence.
                </li>
              </ul>

              <p>
                Full implementation reviews run in parallel. Each changed specification gets its own
                requirements reviewer, alongside separate architecture, test and documentation reviewers. A
                change covering two specifications therefore gets five full-review agents.
              </p>

              <p>
                <strong>Blocking findings stop acceptance.</strong> The implementing agent fixes them or
                explains why a finding is incorrect. A follow-up reviewer checks that resolution and the
                subsequent edits. A review that could not run does not count as passed.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/.agent-hooks/review_gate.py">
                  Review hook implementation
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/tree/main/.agent-hooks">
                  Reviewer instructions
                </a>
              </p>

              <h3 id="ai-linting">Automated checks and real restores</h3>

              <p>
                Independent review works alongside strict automated checks. golangci-lint and our architecture
                linter check backend dependencies, module boundaries and migration rules. API checks validate
                OpenAPI contracts and generated code. Frontend checks cover TypeScript, React, layer imports,
                styles and localization.
              </p>

              <p>
                Ruff and ty check Python formatting, lint rules and types. ShellCheck checks shell scripts.
                actionlint checks GitHub Actions workflows. Pre-commit hooks check affected backend, frontend
                and Python files without rewriting them. CI also runs linting.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/mise-tasks/backend/lint">
                  Backend lint checks
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/frontend/lint/check.sh">
                  Frontend lint checks
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/scripts/python_quality.py">
                  Python checks
                </a>
              </p>

              <p>
                Tests exercise real behavior. Backend tests use the application API and check engines against
                real database containers. End-to-end scenarios create backups, restore them and inspect the
                restored data. Other tests cover upgrades, frontend screens and browser workflows with visual
                comparisons.
              </p>

              <p>
                Checks are chosen for each change. Required release suites run on amd64 and arm64. The
                implementing agent runs the relevant linters and tests before review; review agents inspect the
                evidence without rerunning those commands.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/docs/testing.md">
                  Test strategy and commands
                </a>
              </p>

              <p>
                Before publication, CodeQL and Semgrep analyze code, Gitleaks scans for secrets and dependency
                checks look for vulnerabilities. Trivy scans container images and Dockerfiles.
              </p>

              <p>
                <strong>Required CI checks must pass before release.</strong>{" "}
                <a href={`${getLocalizedHref("en", "security")}#security-and-reliability-engineering`}>
                  Security and reliability engineering
                </a>
                .
              </p>

              <h2 id="backup-databasus">How to backup Databasus itself?</h2>

              <p>
                Databasus is built to stay recoverable even if its own server is lost or corrupted.
                It can back up its own internal database: accounts,
                settings, the encrypted credentials of your databases, storages
                and notifiers, schedules and the backup catalog. Turn it on in{" "}
                <strong>Databasus settings → Databasus backups</strong> by
                selecting one or more existing storages. By default a Databasus
                backup runs every hour and is kept for 30 days. Each file is
                encrypted with the master key.
              </p>

              <p>
                A Databasus backup does not contain the master key or the
                backup files of your databases, which stay in their own
                storages. <strong>Save the master key separately.</strong>{" "}
                Databasus asks you to download it on the first visit,
                and the key is also
                the <code>./databasus-data/secret.key</code>{" "}
                file on the host.
              </p>

              <ul>
                <li>
                  <strong>Choose a remote storage.</strong> A local storage on
                  the same server is lost together with that server.
                </li>
                <li>
                  <strong>Alerts.</strong> Databasus shows a notice when a
                  Databasus backup fails, saves to only some storages, stops or
                  is overdue, meaning no copy succeeded 15 minutes after a
                  scheduled run. The same alerts are emailed through the{" "}
                  <a href={`${getLocalizedHref("en", "advanced-config")}#email-smtp`}>
                  instance mail server
                  </a>{" "}
                  when it is configured.
                </li>
              </ul>

              <h3 id="recover-databasus">
                How to recover Databasus from a Databasus backup?
              </h3>

              <p>
                You need two things: a Databasus backup file with the{" "}
                <code>.dbbackup</code> extension, downloaded from the history or
                taken directly from the storage, and the master key of the
                installation that made it.{" "}
                <strong>
                Recovery works only on exactly the same Databasus version, patch
                included.
                </strong>{" "}
                For any other version Databasus refuses and names the version to
                start: recover on that version, then upgrade as usual.
              </p>

              <p>
                <strong>Stop the old installation before you recover.</strong> A
                recovered installation starts its backups, notifications and
                storage writes at once, so two running installations would write
                to the same storages and databases.
              </p>

              <p>
                <strong>Option 1: files at startup.</strong> Use it for a new
                server. Put both files into the mounted data directory, for
                example <code>./databasus-data/recovery/</code>, and start the
                image of the version that made the backup with two extra
                settings. Replace <code>&lt;version&gt;</code> with that
                version:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startupRecoveryCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startupRecoveryCommand} />
                </div>
              </div>

              <ul>
                <li>
                  If you already started Databasus normally on this server but
                  created no account and changed no settings, you do not have to
                  delete its data. Stop it, add the two files and the two
                  settings, then start it again.
                </li>
                <li>
                  Startup recovery never replaces an installation with an
                  account, even a single one, with other user data or with
                  changed settings. In that case Databasus does not start. Its
                  log says that recovery did not run, why, that the current
                  installation is unchanged and what to do next. To start the
                  current installation, remove the two settings, or use the
                  upload in the interface instead.
                </li>
                <li>
                  After a successful recovery, the files and settings may stay.
                  Databasus remembers the SHA-256 of the applied file, so
                  restarts and later upgrades skip it and say so in the log,
                  even if the file was renamed. A different file at the same
                  path is not skipped.
                </li>
              </ul>

              <p>
                <strong>Option 2: upload in the interface.</strong> Use it on an
                installation that is already running. On a new installation,
                first sign up. Open <strong>Databasus settings → Databasus
                backups</strong>, choose the backup file and paste or type the
                master key of the installation that made it.
                Everything in the current installation is replaced: accounts,
                settings and the backup catalog. Afterwards, sign in with an
                account from the backup.
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

              <h2 id="community">Is there a Databasus community?</h2>

              <p>
                Yes. The Databasus community talks in chats on{" "}
                <a
                  href="https://t.me/databasus_community"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Telegram
                </a>{" "}
                and{" "}
                <a
                  href="/discord"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Discord
                </a>
                . There you can ask questions, share how you run Databasus, discuss features and hear about new releases first.
              </p>

              <p>Pick the app you already use.</p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
