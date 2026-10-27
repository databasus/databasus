import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Agent Installation - Databasus Documentation",
  description:
    "Install the Databasus agent for physical backups, incremental backups, WAL archiving and Point-in-Time Recovery (PITR) of PostgreSQL databases.",
  keywords: [
    "Databasus agent",
    "PostgreSQL physical backup",
    "WAL archiving",
    "PITR",
    "Point-in-Time Recovery",
    "pg_basebackup",
    "incremental backup",
    "disaster recovery",
    "PostgreSQL agent",
    "database backup agent",
  ],
  openGraph: {
    title: "Agent Installation - Databasus Documentation",
    description:
      "Install the Databasus agent for physical backups, incremental backups, WAL archiving and Point-in-Time Recovery (PITR) of PostgreSQL databases.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/installation/agent",
  },
  twitter: {
    card: "summary",
    title: "Agent Installation - Databasus Documentation",
    description:
      "Install the Databasus agent for physical backups, incremental backups, WAL archiving and Point-in-Time Recovery (PITR) of PostgreSQL databases.",
  },
  alternates: {
    canonical: "https://databasus.com/installation/agent",
    languages: getLanguageAlternates("installation/agent"),
  },
  robots: "index, follow",
};

export default function AgentInstallationPage() {
  const downloadCommand = `curl -L -o databasus-agent "<DATABASUS_HOST>/api/v1/system/agent?arch=<ARCH>" && chmod +x databasus-agent`;

  const postgresqlConf = `wal_level = replica
archive_mode = on
archive_command = 'cp %p <WAL_QUEUE_DIR>/%f.tmp && mv <WAL_QUEUE_DIR>/%f.tmp <WAL_QUEUE_DIR>/%f'`;

  const postgresqlConfDocker = `wal_level = replica
archive_mode = on
archive_command = 'cp %p /wal-queue/%f.tmp && mv /wal-queue/%f.tmp /wal-queue/%f'`;

  const pgHbaEntry = `host    replication   all   127.0.0.1/32   md5`;

  const grantReplication = `ALTER ROLE <YOUR_PG_USER> WITH REPLICATION;`;

  const createWalDir = `mkdir -p /opt/databasus/wal-queue`;

  const walDirPermissions = `chown postgres:postgres /opt/databasus/wal-queue
chmod 755 /opt/databasus/wal-queue`;

  const dockerVolumeExample = `# In your docker run command:
docker run ... -v /opt/databasus/wal-queue:/wal-queue ...

# Or in docker-compose.yml:
volumes:
  - /opt/databasus/wal-queue:/wal-queue`;

  const dockerWalDirPermissions = `# Inside the container (or via docker exec):
chown postgres:postgres /wal-queue`;

  const startCommandHost = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const startCommandFolder = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-host-bin-dir=<PATH_TO_PG_BIN_DIR> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const startCommandDocker = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=docker \\
  --pg-docker-container-name=<CONTAINER_NAME> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const restoreCommand = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR>`;

  const restoreCommandDocker = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --pg-type=docker \\
  --target-dir=<HOST_PGDATA_PATH>`;

  const restoreCommandPitr = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR> \\
  --target-time=<RFC3339_TIMESTAMP>`;

  const archiveCommandCleanup = `# In <PGDATA_DIR>/postgresql.auto.conf, remove or comment out:
# archive_mode = on
# archive_command = '...'`;

  const dockerVolumeMountExample = `# PostgreSQL 17 and earlier
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/data postgres:17

# PostgreSQL 18+
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/18/docker postgres:18`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Agent Installation - Databasus Documentation",
            description:
              "Install the Databasus agent for physical backups, incremental backups, WAL archiving and Point-in-Time Recovery (PITR) of PostgreSQL databases.",
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

      <DocsNavbarComponent />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="agent-installation">Agent mode</h1>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Agent backups are deprecated.</strong> Databasus now
                    runs physical and PITR backups remotely using PostgreSQL 17
                    native backups, with no agent installed on the database
                    server.{" "}
                    <a
                      href="/faq/#why-no-agent"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      Read why and how PITR backups work now
                    </a>
                    .
                  </p>
                </div>
              </div>

              <p className="text-lg text-gray-400">
                The Databasus agent enables physical backups, incremental
                backups, WAL archiving and Point-in-Time Recovery (PITR) for
                PostgreSQL databases.
              </p>

              {/* When to use */}
              <h2 id="when-to-use">When to use the agent</h2>

              <p>
                For most databases,{" "}
                <strong>remote backups are the simplest option</strong>.
                Databasus connects directly to the database over the network,
                performs logical backups using pg_dump, and requires no
                additional software on the database server. Remote backups work
                with cloud-managed databases (RDS, Cloud SQL, Supabase) and
                self-hosted instances alike.
              </p>

              <p>
                The agent is designed for scenarios where remote backups are
                not sufficient:
              </p>

              <ul>
                <li>
                  <strong>Disaster recovery with PITR</strong> — restore to any
                  second between backups with near-zero data loss
                </li>
                <li>
                  <strong>Physical backups</strong> — file-level copy of the
                  entire database cluster, faster backup and restore for large
                  datasets
                </li>
                <li>
                  <strong>Databases not exposed publicly</strong> — the agent
                  connects outbound to Databasus, so the database never needs a
                  public endpoint
                </li>
                <li>
                  <strong>Incremental backups</strong> — continuous WAL segment
                  archiving combined with periodic base backups
                </li>
              </ul>

              {/* In-app guided setup */}
              <h2 id="in-app-setup">In-app guided setup</h2>

              <p>
                Databasus provides interactive installation and restore
                instructions directly in the UI. When you open the agent
                settings for a database, all commands are pre-filled with your
                specific values: architecture, database ID, agent token,
                Databasus host, and PostgreSQL deployment type. You can copy
                each command and run it on your server.
              </p>

              <p>
                The documentation below covers the same steps for reference and
                for users who prefer to follow a guide outside the UI.
              </p>

              {/* Requirements */}
              <h2 id="requirements">Requirements</h2>

              <ul>
                <li>
                  <strong>PostgreSQL 15 or newer</strong>
                </li>
                <li>
                  <strong>Linux</strong> (amd64 or arm64)
                </li>
                <li>
                  <strong>Network access</strong> from the agent to your
                  Databasus instance (outbound only — the database does not need
                  to be reachable from Databasus)
                </li>
              </ul>

              {/* Installation */}
              <h2 id="installation">Installation</h2>

              <h3 id="step-1-download">Step 1 — Download the agent</h3>

              <p>
                Download the agent binary on the server where PostgreSQL runs.
                Replace <code>&lt;DATABASUS_HOST&gt;</code> with your Databasus
                instance URL and <code>&lt;ARCH&gt;</code> with{" "}
                <code>amd64</code> or <code>arm64</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={downloadCommand} />
                </div>
              </div>

              <h3 id="step-2-postgresql-conf">
                Step 2 — Configure postgresql.conf
              </h3>

              <p>
                Add or update these settings in your{" "}
                <code>postgresql.conf</code>, then{" "}
                <strong>restart PostgreSQL</strong>.
              </p>

              <p>
                <strong>For host installations</strong> (replace{" "}
                <code>&lt;WAL_QUEUE_DIR&gt;</code> with the actual path, e.g.{" "}
                <code>/opt/databasus/wal-queue</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConf}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlConf} />
                </div>
              </div>

              <p>
                <strong>For Docker installations</strong>, the{" "}
                <code>archive_command</code> path (<code>/wal-queue</code>) is
                the path <strong>inside the container</strong>. It must match
                the volume mount target — see Step 5.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConfDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlConfDocker} />
                </div>
              </div>

              <h3 id="step-3-pg-hba">Step 3 — Configure pg_hba.conf</h3>

              <p>
                Add this line to <code>pg_hba.conf</code>. This is required for{" "}
                <code>pg_basebackup</code> to take full backups — not for
                streaming replication. Adjust the address and auth method as
                needed, then reload PostgreSQL.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pgHbaEntry}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={pgHbaEntry} />
                </div>
              </div>

              <h3 id="step-4-replication">
                Step 4 — Grant replication privilege
              </h3>

              <p>
                This is a PostgreSQL requirement for running{" "}
                <code>pg_basebackup</code> — it does not set up a replica.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{grantReplication}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={grantReplication} />
                </div>
              </div>

              <h3 id="step-5-wal-queue">
                Step 5 — Create WAL queue directory
              </h3>

              <p>
                PostgreSQL places WAL archive files here for the agent to
                upload.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createWalDir}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={createWalDir} />
                </div>
              </div>

              <p>
                Ensure the directory is writable by PostgreSQL and readable by
                the agent:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{walDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={walDirPermissions} />
                </div>
              </div>

              <p>
                <strong>For Docker installations</strong>, the WAL queue
                directory must be a volume mount shared between the PostgreSQL
                container and the host. The agent reads WAL files from the host
                path, while PostgreSQL writes to the container path via{" "}
                <code>archive_command</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerVolumeExample} />
                </div>
              </div>

              <p>
                Ensure the directory inside the container is owned by the{" "}
                <code>postgres</code> user:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerWalDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerWalDirPermissions} />
                </div>
              </div>

              <h3 id="step-6-start">Step 6 — Start the agent</h3>

              <p>
                Replace placeholders in <code>&lt;ANGLE_BRACKETS&gt;</code> with
                your actual values.
              </p>

              <p>
                <strong>System-wide PostgreSQL</strong> (pg_basebackup available
                in PATH):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startCommandHost} />
                </div>
              </div>

              <p>
                <strong>PostgreSQL in a specific folder</strong> (e.g.{" "}
                <code>/usr/lib/postgresql/17/bin</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandFolder}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startCommandFolder} />
                </div>
              </div>

              <p>
                <strong>Docker</strong> (use the PostgreSQL port{" "}
                <strong>inside the container</strong>, usually 5432, not the
                host-mapped port):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startCommandDocker} />
                </div>
              </div>

              <h3 id="after-installation">After installation</h3>

              <ul>
                <li>
                  The agent runs in the background after{" "}
                  <code>start</code>
                </li>
                <li>
                  Check status: <code>./databasus-agent status</code>
                </li>
                <li>
                  View logs: <code>databasus.log</code> in the working directory
                </li>
                <li>
                  Stop the agent: <code>./databasus-agent stop</code>
                </li>
              </ul>

              {/* Restore */}
              <h2 id="restore">Restore from agent backup</h2>

              <p>
                Restore a physical or incremental backup to a target directory.
                For Point-in-Time Recovery, add the{" "}
                <code>--target-time</code> flag to restore to a specific moment.
              </p>

              <h3 id="restore-step-1">Step 1 — Download the agent</h3>

              <p>
                Download the agent binary on the server where you want to
                restore (same command as installation Step 1).
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={downloadCommand} />
                </div>
              </div>

              <h3 id="restore-step-2">Step 2 — Stop PostgreSQL</h3>

              <p>
                PostgreSQL must be stopped before restoring. The target directory
                must be empty.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; stop</code>
                </pre>
              </div>

              <p>For Docker:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker stop &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-3">Step 3 — Run restore</h3>

              <p>
                Replace <code>&lt;YOUR_AGENT_TOKEN&gt;</code> with your agent
                token and <code>&lt;PGDATA_DIR&gt;</code> with the path to an
                empty PostgreSQL data directory.
              </p>

              <p>
                <strong>Host installation:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={restoreCommand} />
                </div>
              </div>

              <p>
                <strong>Docker installation</strong> (
                <code>&lt;HOST_PGDATA_PATH&gt;</code> is the path on the host
                that will be mounted as the container&apos;s pgdata volume):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={restoreCommandDocker} />
                </div>
              </div>

              <p>
                Mount <code>&lt;HOST_PGDATA_PATH&gt;</code> at the
                container&apos;s PGDATA path when (re)creating the postgres
                container. The path depends on the major version: PostgreSQL
                18+ uses <code>/var/lib/postgresql/&lt;major&gt;/docker</code>;
                PostgreSQL 17 and earlier use{" "}
                <code>/var/lib/postgresql/data</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeMountExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerVolumeMountExample} />
                </div>
              </div>

              <p>
                For <strong>Point-in-Time Recovery</strong> (PITR), add{" "}
                <code>--target-time</code> with an RFC 3339 timestamp (e.g.{" "}
                <code>2025-01-15T14:30:00Z</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandPitr}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={restoreCommandPitr} />
                </div>
              </div>

              <h3 id="restore-step-4">
                Step 4 — Handle archive_command
              </h3>

              <p>
                The restored backup includes the original{" "}
                <code>archive_command</code> configuration. PostgreSQL will fail
                to archive WAL files after recovery unless you either:
              </p>

              <ul>
                <li>
                  <strong>Re-attach the agent</strong> — mount the WAL queue
                  directory and start the Databasus agent on the restored
                  instance, same as the original setup.
                </li>
                <li>
                  <strong>Disable archiving</strong> — if you don&apos;t need
                  continuous backups yet, comment out or reset the archive
                  settings in <code>postgresql.auto.conf</code>:
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{archiveCommandCleanup}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={archiveCommandCleanup} />
                </div>
              </div>

              <h3 id="restore-step-5">Step 5 — Start PostgreSQL</h3>

              <p>
                Start PostgreSQL to begin WAL recovery. It will automatically
                replay WAL segments.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; start</code>
                </pre>
              </div>

              <p>For Docker:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker start &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-6">Step 6 — Clean up</h3>

              <p>
                After recovery completes, remove the WAL restore directory:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>rm -rf &lt;PGDATA_DIR&gt;/databasus-wal-restore/</code>
                </pre>
              </div>

              {/* How it works */}
              <h2 id="how-it-works">How it works</h2>

              <p>
                The Databasus agent is a lightweight Go binary that runs two
                concurrent processes:
              </p>

              <ul>
                <li>
                  <strong>WAL streaming</strong> — picks up WAL segment files
                  from the queue directory approximately every 10 seconds and
                  uploads them to Databasus
                </li>
                <li>
                  <strong>Periodic base backups</strong> — runs{" "}
                  <code>pg_basebackup</code> on the configured schedule to
                  create full physical backups of the database cluster
                </li>
              </ul>

              <p>
                During restoration, the agent downloads the base backup and all
                relevant WAL segments, then configures{" "}
                <code>recovery.signal</code> and <code>restore_command</code> in{" "}
                <code>postgresql.auto.conf</code>. When PostgreSQL starts, it
                replays the WAL segments to reach the target recovery point.
              </p>

              <p>
                The agent always initiates the connection to Databasus
                (outbound). The database server does not need to accept incoming
                connections from Databasus, making it suitable for private
                networks and firewalled environments.
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
