import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "pg_dump Alternative - Databasus PostgreSQL Backup Tool",
  description:
    "Databasus is built on pg_dump and extends its features with backups management, a web UI, automated scheduling, cloud storage, notifications, team collaboration and encryption.",
  keywords: [
    "pg_dump alternative",
    "pg_dump GUI",
    "pg_dump automation",
    "pg_dump web interface",
    "PostgreSQL backup tool",
    "pg_dump scheduler",
    "pg_dump cloud storage",
    "pg_dump encryption",
    "PostgreSQL backup automation",
    "pg_dump wrapper",
  ],
  openGraph: {
    title: "pg_dump Alternative - Databasus PostgreSQL Backup Tool",
    description:
      "Databasus is built on pg_dump and extends its features with backups management, a web UI, automated scheduling, cloud storage, notifications, team collaboration and encryption.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/pgdump-alternative",
  },
  twitter: {
    card: "summary",
    title: "pg_dump Alternative - Databasus PostgreSQL Backup Tool",
    description:
      "Databasus is built on pg_dump and extends its features with backups management, a web UI, automated scheduling, cloud storage, notifications, team collaboration and encryption.",
  },
  alternates: {
    canonical: "https://databasus.com/pgdump-alternative",
    languages: getLanguageAlternates("pgdump-alternative"),
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "pg_dump Alternative - Databasus PostgreSQL Backup Tool",
            description:
              "A comprehensive guide to Databasus as a pg_dump alternative, explaining how it builds on pg_dump and extends its capabilities with automation, cloud storage, notifications and team features.",
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
              <h1 id="pgdump-alternative">pg_dump Alternative</h1>

              <p className="text-lg text-gray-400">
                For logical backups, Databasus is built on top of{" "}
                <code>pg_dump</code>. Rather than replacing <code>pg_dump</code>
                , Databasus extends its capabilities with backups management, a
                web interface, automated scheduling, cloud storage integration,
                notifications, team collaboration features and built-in
                encryption. Beyond logical backups, Databasus also supports
                physical backups, incremental backups with WAL archiving and
                Point-in-Time Recovery.
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s an overview of how Databasus extends the core{" "}
                <code>pg_dump</code> functionality:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Feature</th>
                    <th>pg_dump</th>
                    <th>Databasus</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Backup engine</td>
                    <td data-label="pg_dump">pg_dump</td>
                    <td data-label="Databasus">Built on pg_dump</td>
                  </tr>
                  <tr>
                    <td>Backups management</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">✅ Yes</td>
                  </tr>
                  <tr>
                    <td>Support of other DBs</td>
                    <td data-label="pg_dump">PostgreSQL only</td>
                    <td data-label="Databasus">
                      PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="pg_dump">Command-line</td>
                    <td data-label="Databasus">Web UI + API</td>
                  </tr>
                  <tr>
                    <td>Scheduling</td>
                    <td data-label="pg_dump">Manual or cron scripts</td>
                    <td data-label="Databasus">✅ Built-in scheduler</td>
                  </tr>
                  <tr>
                    <td>Storage destinations</td>
                    <td data-label="pg_dump">Local filesystem</td>
                    <td data-label="Databasus">
                      Local, S3, Google Drive, R2, Azure, NAS, Dropbox
                    </td>
                  </tr>
                  <tr>
                    <td>Compression</td>
                    <td data-label="pg_dump">gzip, LZ4, zstd (manual)</td>
                    <td data-label="Databasus">zstd (automatic, optimized)</td>
                  </tr>
                  <tr>
                    <td>Encryption</td>
                    <td data-label="pg_dump">External tools required</td>
                    <td data-label="Databasus">✅ AES-256-GCM built-in</td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="pg_dump">❌ None</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, Email, Webhooks
                    </td>
                  </tr>
                  <tr>
                    <td>Team features</td>
                    <td data-label="pg_dump">❌ None</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, RBAC, audit logs
                    </td>
                  </tr>
                  <tr>
                    <td>Retention policies</td>
                    <td data-label="pg_dump">Manual cleanup scripts</td>
                    <td data-label="Databasus">✅ Automatic retention</td>
                  </tr>
                  <tr>
                    <td>Health monitoring</td>
                    <td data-label="pg_dump">❌ None</td>
                    <td data-label="Databasus">✅ Built-in health checks</td>
                  </tr>
                  <tr>
                    <td>Physical backups</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">✅ Yes</td>
                  </tr>
                  <tr>
                    <td>Incremental backups</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">✅ Block-level (PG 17+)</td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">✅ Yes</td>
                  </tr>
                  <tr>
                    <td>Remote backups</td>
                    <td data-label="pg_dump">✅ Yes (CLI)</td>
                    <td data-label="Databasus">✅ Yes</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="what-is-pgdump">What is pg_dump?</h2>

              <p>
                <code>pg_dump</code> is PostgreSQL&apos;s native utility for
                creating logical backups. It&apos;s been part of PostgreSQL
                since the beginning and is the standard tool for database
                exports.
              </p>

              <h3 id="pgdump-strengths">pg_dump strengths</h3>

              <ul>
                <li>
                  <strong>Portable backups</strong>: Creates SQL or custom
                  format dumps that can be restored to different PostgreSQL
                  versions.
                </li>
                <li>
                  <strong>Selective backups</strong>: Can export specific
                  tables, schemas or entire databases.
                </li>
                <li>
                  <strong>Consistent snapshots</strong>: Uses PostgreSQL&apos;s
                  MVCC to create consistent backups without blocking writes.
                </li>
                <li>
                  <strong>Widely supported</strong>: Available on every
                  PostgreSQL installation, well-documented and battle-tested.
                </li>
                <li>
                  <strong>Flexible output formats</strong>: Plain SQL, custom,
                  directory or tar formats.
                </li>
              </ul>

              <h3 id="pgdump-limitations">pg_dump limitations</h3>

              <p>
                While <code>pg_dump</code> is powerful, using it in production
                typically requires additional scripting:
              </p>

              <ul>
                <li>
                  <strong>No built-in scheduling</strong>: Requires cron jobs or
                  external schedulers.
                </li>
                <li>
                  <strong>Local storage only</strong>: Outputs to local
                  filesystem; cloud uploads require additional scripts.
                </li>
                <li>
                  <strong>No encryption</strong>: Backup files are unencrypted
                  by default; requires piping through gpg or similar tools.
                </li>
                <li>
                  <strong>No notifications</strong>: No way to alert on backup
                  success or failure without custom scripting.
                </li>
                <li>
                  <strong>No retention management</strong>: Old backups must be
                  cleaned up manually or via scripts.
                </li>
                <li>
                  <strong>Command-line only</strong>: No visual interface for
                  monitoring or management.
                </li>
              </ul>

              <h2 id="how-databasus-extends">How Databasus extends pg_dump</h2>

              <p>
                Databasus uses <code>pg_dump</code> as its backup engine,
                preserving all the benefits of logical backups while adding
                enterprise features on top.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Under the hood:</strong>{" "}
                  When you trigger a backup in Databasus, it executes{" "}
                  <code className="bg-[#374151] text-gray-200">pg_dump</code>{" "}
                  with optimized parameters, then handles compression,
                  encryption and upload to your configured storage destination.
                </p>
              </div>

              <h3 id="web-interface">Web interface</h3>

              <p>
                Instead of remembering <code>pg_dump</code> command-line
                options, Databasus provides a web UI where you can:
              </p>

              <ul>
                <li>Add databases with a guided connection wizard</li>
                <li>Configure backup schedules with visual controls</li>
                <li>Monitor backup history and status at a glance</li>
                <li>Download or restore backups with one click</li>
                <li>View database health and availability charts</li>
              </ul>

              <h3 id="optimized-compression">Optimized compression</h3>

              <p>
                Databasus uses zstd compression (level 5) by default, which
                provides:
              </p>

              <ul>
                <li>
                  <strong>4-8x size reduction</strong> compared to uncompressed
                  dumps
                </li>
                <li>
                  <strong>~20% runtime overhead</strong> — much faster than gzip
                </li>
                <li>
                  <strong>Automatic handling</strong> — no need to pipe through
                  compression tools
                </li>
              </ul>

              <h2 id="beyond-pgdump">
                Beyond pg_dump: Physical backups and PITR
              </h2>

              <p>
                While Databasus builds on <code>pg_dump</code> for logical
                backups, it also goes beyond what <code>pg_dump</code> can
                offer:
              </p>

              <ul>
                <li>
                  <strong>Physical backups</strong>: File-level copies of the
                  entire database cluster via <code>pg_basebackup</code>. Faster
                  backup and restore for large databases.
                </li>
                <li>
                  <strong>Incremental and WAL backups</strong>: Block-level
                  incremental backups via <code>pg_basebackup --incremental</code>{" "}
                  (driven by server-side WAL summaries) plus continuous WAL
                  streaming via <code>pg_receivewal</code>, enabling
                  Point-in-Time Recovery — restore to any second between backups.
                </li>
                <li>
                  <strong>Disaster recovery</strong>: Designed for near-zero
                  data loss requirements with physical base backups and
                  continuous WAL streaming.
                </li>
              </ul>

              <p>
                These backups are built on PostgreSQL 17&apos;s native backup
                mechanism, so Databasus reuses PostgreSQL&apos;s own
                battle-tested tooling instead of re-inventing it. They require
                PostgreSQL 17 or newer; on older versions only logical{" "}
                <code>pg_dump</code> backups are available. Everything runs remotely
                from the Databasus host over the replication protocol, so nothing
                is installed on the database server. Closed networks are reached
                through an SSH tunnel to an internal host or a bastion, so the
                database never has to be exposed publicly.{" "}
                <a
                  href="/faq#pitr"
                  className="text-blue-400 hover:text-blue-600"
                >
                  Read how physical and PITR backups work
                </a>
                .
              </p>

              <h2 id="backup-automation">Backup automation</h2>

              <p>
                One of the most common challenges with <code>pg_dump</code> is
                setting up reliable automated backups.
              </p>

              <h3 id="automation-pgdump">Traditional pg_dump automation</h3>

              <p>
                A typical <code>pg_dump</code> automation script might look
                like:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
                </pre>
              </div>

              <p>
                This script needs to be maintained, tested and monitored. Each
                database requires its own cron entry.
              </p>

              <h3 id="automation-databasus">Databasus automation</h3>

              <p>With Databasus, the same functionality is built-in:</p>

              <ul>
                <li>
                  <strong>Visual scheduler</strong>: Set hourly, daily, weekly,
                  monthly or cron backups with specific times.
                </li>
                <li>
                  <strong>Automatic compression</strong>: zstd compression
                  applied automatically.
                </li>
                <li>
                  <strong>Built-in encryption</strong>: AES-256-GCM encryption
                  with unique keys per backup.
                </li>
                <li>
                  <strong>Cloud upload</strong>: Direct upload to S3, Google
                  Drive, Cloudflare R2, Azure or other destinations.
                </li>
                <li>
                  <strong>Retention policies</strong>: Automatic cleanup of old
                  backups based on your retention settings.
                </li>
                <li>
                  <strong>Notifications</strong>: Alerts to Slack, Teams,
                  Telegram, Email on success or failure.
                </li>
              </ul>

              <h2 id="storage-options">Storage options</h2>

              <p>
                <code>pg_dump</code> writes to the local filesystem. Getting
                backups to cloud storage requires additional tools and scripts.
              </p>

              <h3 id="storage-databasus">Databasus storage destinations</h3>

              <p>
                Databasus supports multiple storage destinations out of the box:
              </p>

              <ul>
                <li>Local storage</li>
                <li>Amazon S3 and S3-compatible services</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>

              <p>
                Each database can have its own storage destination and you can
                configure multiple destinations for redundancy.
              </p>

              <p>
                <a
                  href="/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all storage options →
                </a>
              </p>

              <h2 id="notifications">Notifications</h2>

              <p>
                Knowing when backups succeed or fail is critical for data
                protection.
              </p>

              <h3 id="notifications-pgdump">pg_dump notifications</h3>

              <p>
                <code>pg_dump</code> has no notification system. You need to:
              </p>

              <ul>
                <li>Write wrapper scripts that check exit codes</li>
                <li>Integrate with external monitoring tools</li>
                <li>Set up custom alerting pipelines</li>
              </ul>

              <h3 id="notifications-databasus">Databasus notifications</h3>

              <p>Databasus includes built-in notifications to:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Webhooks (for custom integrations)</li>
              </ul>

              <p>
                Configure which events trigger notifications: backup success,
                backup failure or both.
              </p>

              <p>
                <a
                  href="/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all notification channels →
                </a>
              </p>

              <h2 id="team-features">Team features</h2>

              <p>
                <code>pg_dump</code> is a single-user command-line tool.
                Databasus adds collaboration features for teams:
              </p>

              <h3 id="team-databasus">Databasus team capabilities</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: Organize databases, notifiers,
                  and storages by project or team. Users only see workspaces
                  they&apos;re invited to.
                </li>
                <li>
                  <strong>Role-based access control</strong>: Assign viewer,
                  editor or admin permissions to control what each team member
                  can do.
                </li>
                <li>
                  <strong>Audit logs</strong>: Track all system activities and
                  changes. Essential for security compliance and accountability.
                </li>
                <li>
                  <strong>Shared notifications</strong>: Team channels receive
                  backup status updates automatically.
                </li>
              </ul>

              <p>
                <a
                  href="/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about access management →
                </a>
              </p>

              <h2 id="security">Security</h2>

              <p>
                Security is where Databasus adds significant value over raw{" "}
                <code>pg_dump</code> usage.
              </p>

              <h3 id="security-pgdump">pg_dump security</h3>

              <p>
                <code>pg_dump</code> creates unencrypted backup files. Securing
                them requires:
              </p>

              <ul>
                <li>Piping output through encryption tools (gpg, openssl)</li>
                <li>Managing encryption keys separately</li>
                <li>Ensuring secure key storage and rotation</li>
                <li>Setting up proper file permissions</li>
              </ul>

              <h3 id="security-databasus">Databasus security</h3>

              <p>Databasus implements security at multiple levels:</p>

              <ul>
                <li>
                  <strong>AES-256-GCM encryption</strong>: All passwords, tokens
                  and credentials are encrypted. The encryption key is stored
                  separately from the database.
                </li>
                <li>
                  <strong>Unique backup encryption</strong>: Each backup file is
                  encrypted with a unique key derived from master key, backup ID
                  and random salt.
                </li>
                <li>
                  <strong>Read-only database access</strong>: Enforces SELECT
                  permissions only, preventing data corruption even if
                  compromised.
                </li>
              </ul>

              <p>
                <a
                  href="/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about Databasus security →
                </a>
              </p>

              <h2 id="restore-process">Restore process</h2>

              <p>
                Both tools support restoring backups, but with different
                workflows.
              </p>

              <h3 id="restore-pgdump">Restoring pg_dump backups</h3>

              <p>
                Restoring a <code>pg_dump</code> backup requires:
              </p>

              <ol>
                <li>Locating the backup file</li>
                <li>Decrypting if encrypted</li>
                <li>Decompressing if compressed</li>
                <li>
                  Running <code>pg_restore</code> or <code>psql</code> with
                  correct parameters
                </li>
              </ol>

              <h3 id="restore-databasus">Restoring Databasus backups</h3>

              <p>Databasus simplifies restoration:</p>

              <ul>
                <li>
                  <strong>One-click download</strong>: Download any backup
                  directly from the web interface.
                </li>
                <li>
                  <strong>Automatic decryption</strong>: Backups are decrypted
                  automatically when downloaded.
                </li>
                <li>
                  <strong>Restore commands provided</strong>: Databasus shows
                  the exact <code>pg_restore</code> command for each backup.
                </li>
                <li>
                  <strong>Parallel restore support</strong>: Utilize multiple
                  CPU cores for faster restoration of large databases.
                </li>
              </ul>

              <h2 id="installation">Installation</h2>

              <h3 id="install-pgdump">pg_dump installation</h3>

              <p>
                <code>pg_dump</code> comes with PostgreSQL. If you have
                PostgreSQL installed, you have <code>pg_dump</code>.
              </p>

              <h3 id="install-databasus">Databasus installation</h3>

              <p>Databasus offers multiple installation methods:</p>

              <ul>
                <li>
                  <strong>One-line script</strong>: Installs Docker (if needed),
                  sets up Databasus and configures automatic startup.
                </li>
                <li>
                  <strong>Docker run</strong>: Single command to start with
                  embedded PostgreSQL.
                </li>
                <li>
                  <strong>Docker Compose</strong>: For more control over
                  deployment.
                </li>
              </ul>

              <p>
                <a
                  href="/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View installation guide →
                </a>
              </p>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                <code>pg_dump</code> is PostgreSQL&apos;s proven backup utility,
                and Databasus builds directly on top of it. The choice between
                using <code>pg_dump</code> directly or through Databasus depends
                on your needs.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Use pg_dump directly if:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>You need one-off or ad-hoc database exports</li>
                  <li>
                    You&apos;re comfortable writing and maintaining shell
                    scripts
                  </li>
                  <li>
                    You have existing automation infrastructure (Ansible,
                    Terraform, etc.)
                  </li>
                  <li>You only need local backups without cloud storage</li>
                  <li>You&apos;re a single developer with simple needs</li>
                </ul>
              </div>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Use Databasus if:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    You want automated, scheduled backups without writing
                    scripts
                  </li>
                  <li>
                    You need to store backups in cloud storage (S3, Google
                    Drive, etc.)
                  </li>
                  <li>
                    You want built-in encryption without managing keys manually
                  </li>
                  <li>You need notifications when backups succeed or fail</li>
                  <li>
                    You&apos;re working in a team and need collaboration
                    features
                  </li>
                  <li>You prefer a visual interface over command-line tools</li>
                  <li>You want automatic retention policies and cleanup</li>
                  <li>
                    You need physical backups, incremental backups or
                    Point-in-Time Recovery for disaster recovery
                  </li>
                </ul>
              </div>

              <p>
                Databasus builds on <code>pg_dump</code> for logical backups and
                extends it with automation, security and team features. Beyond
                that, Databasus also supports physical backups, incremental
                backups with WAL archiving and Point-in-Time Recovery —
                capabilities that <code>pg_dump</code> simply cannot provide.
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
