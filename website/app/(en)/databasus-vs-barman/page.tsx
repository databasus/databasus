import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs Barman - PostgreSQL Backup Tools Comparison",
  description:
    "Compare Databasus and Barman PostgreSQL backup tools. See differences in backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
  keywords: [
    "Databasus vs Barman",
    "PostgreSQL backup comparison",
    "Barman alternative",
    "PostgreSQL backup tools",
    "database backup comparison",
    "pg_dump vs physical backup",
    "self-hosted backup",
    "PostgreSQL PITR",
    "WAL archiving",
    "disaster recovery PostgreSQL",
  ],
  openGraph: {
    title: "Databasus vs Barman - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Databasus and Barman PostgreSQL backup tools. See differences in backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/databasus-vs-barman",
  },
  twitter: {
    card: "summary",
    title: "Databasus vs Barman - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Databasus and Barman PostgreSQL backup tools. See differences in backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
  },
  alternates: {
    canonical: "https://databasus.com/databasus-vs-barman",
    languages: getLanguageAlternates("databasus-vs-barman"),
  },
  robots: "index, follow",
};

export default function DatabasusVsBarmanPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline:
              "Databasus vs Barman - PostgreSQL Backup Tools Comparison",
            description:
              "A comprehensive comparison of Databasus and Barman PostgreSQL backup tools, covering backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
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
              <h1 id="databasus-vs-barman">Databasus vs Barman</h1>

              <p className="text-lg text-gray-400">
                Databasus and Barman are both built for disaster recovery with
                minimal RTO and RPO, and both support physical backups, WAL
                archiving and Point-in-Time Recovery. Databasus runs these
                backups remotely on PostgreSQL 17&apos;s native stack, so it
                reuses PostgreSQL&apos;s own battle-tested tooling instead of
                re-inventing it, all behind an intuitive web interface with team
                features and support for multiple database engines. It works for
                databases of any size and complexity. Physical backups require
                PostgreSQL 17 or newer; on older versions only logical{" "}
                <code>pg_dump</code> backups are available. Barman (Backup and Recovery
                Manager) ships its own backup engine, so it covers physical
                backups on much older PostgreSQL versions and adds advanced
                features like rsync-based incremental backups, streaming
                replication integration and Barman-to-Barman geo-redundancy.
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s a quick overview of the key differences between
                Databasus and Barman:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Feature</th>
                    <th>Databasus</th>
                    <th>Barman</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Target audience</td>
                    <td data-label="Databasus">
                      Individuals, teams, DBAs, enterprises
                    </td>
                    <td data-label="Barman">
                      DBAs, enterprises
                    </td>
                  </tr>
                  <tr>
                    <td>Support of other DBs</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="Barman">❌ PostgreSQL only</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Databasus">Web UI</td>
                    <td data-label="Barman">Command-line only</td>
                  </tr>
                  <tr>
                    <td>Backup type</td>
                    <td data-label="Databasus">Logical + Physical</td>
                    <td data-label="Barman">Physical (file-level)</td>
                  </tr>
                  <tr>
                    <td>PostgreSQL version for physical backups</td>
                    <td data-label="Databasus">17+ (native)</td>
                    <td data-label="Barman">9.x+ (own engine)</td>
                  </tr>
                  <tr>
                    <td>Recovery options</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="Barman">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Incremental backups</td>
                    <td data-label="Databasus">✅ Block-level (PG 17+)</td>
                    <td data-label="Barman">rsync-based incremental</td>
                  </tr>
                  <tr>
                    <td>Remote backups</td>
                    <td data-label="Databasus">✅ Yes</td>
                    <td data-label="Barman">
                      ❌ No (requires filesystem access)
                    </td>
                  </tr>
                  <tr>
                    <td>Multi-server management</td>
                    <td data-label="Databasus">Per-database scheduling</td>
                    <td data-label="Barman">Centralized backup server</td>
                  </tr>
                  <tr>
                    <td>Team features</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, RBAC, audit logs
                    </td>
                    <td data-label="Barman">❌ OS-level permissions only</td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, Email
                    </td>
                    <td data-label="Barman">❌ Requires custom scripting</td>
                  </tr>
                  <tr>
                    <td>Learning curve</td>
                    <td data-label="Databasus">Minimal</td>
                    <td data-label="Barman">DBA expertise required</td>
                  </tr>
                  <tr>
                    <td>Installation</td>
                    <td data-label="Databasus">One-line script or Docker</td>
                    <td data-label="Barman">Manual configuration required</td>
                  </tr>
                  <tr>
                    <td>Backups management</td>
                    <td data-label="Databasus">✅ Yes</td>
                    <td data-label="Barman">❌ No</td>
                  </tr>
                  <tr>
                    <td>Suitable for self-hosted DBs</td>
                    <td data-label="Databasus">✅ Yes</td>
                    <td data-label="Barman">✅ Yes</td>
                  </tr>
                  <tr>
                    <td>Suitable for cloud DBs</td>
                    <td data-label="Databasus">
                      ✅ Yes (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="Barman">
                      ❌ No (requires filesystem access)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="target-audience">Target audience</h2>

              <p>
                The most significant difference between these tools is who they
                are designed for:
              </p>

              <h3 id="audience-databasus">Databasus audience</h3>

              <p>
                Databasus is built for a broad audience, from individual
                developers to large enterprises:
              </p>

              <ul>
                <li>
                  <strong>Individual developers</strong>: Simple setup and
                  intuitive UI make it easy to protect personal projects without
                  deep PostgreSQL expertise.
                </li>
                <li>
                  <strong>Development teams</strong>: Workspaces, role-based
                  access control and audit logs enable secure collaboration
                  across team members.
                </li>
                <li>
                  <strong>Enterprises</strong>: Scales to meet enterprise needs
                  with comprehensive security, multiple storage destinations and
                  notification channels.
                </li>
                <li>
                  <strong>DBAs and disaster recovery</strong>: Physical
                  backups, WAL archiving and PITR for mission-critical systems
                  with near-zero data loss requirements.
                </li>
              </ul>

              <h3 id="audience-barman">Barman audience</h3>

              <p>
                Barman is specifically designed for Database Administrators
                (DBAs) managing enterprise PostgreSQL infrastructure:
              </p>

              <ul>
                <li>
                  <strong>Enterprise DBAs</strong>: Professionals who need
                  centralized backup management for multiple PostgreSQL servers
                  from a dedicated backup server.
                </li>
                <li>
                  <strong>Teams needing rsync-based incremental</strong>:
                  File-level diffing reduces backup time and network usage for
                  large clusters.
                </li>
                <li>
                  <strong>Geo-redundancy requirements</strong>: Barman-to-Barman
                  replication for geographical redundancy across data centers.
                </li>
              </ul>

              <h2 id="backup-approach">Backup approach</h2>

              <p>
                The tools use fundamentally different backup strategies, each
                with distinct advantages:
              </p>

              <h3 id="backup-databasus">
                Databasus: Logical + Physical backups
              </h3>

              <p>
                Databasus supports both logical and physical backup
                strategies:
              </p>

              <ul>
                <li>
                  <strong>Physical, incremental and WAL backups</strong>: Run
                  remotely over the PostgreSQL replication protocol on
                  PostgreSQL 17&apos;s native stack — <code>pg_basebackup</code>,
                  block-level <code>pg_basebackup --incremental</code> driven by
                  server-side WAL summaries, <code>pg_receivewal</code> and{" "}
                  <code>pg_combinebackup</code>. Databasus reuses
                  PostgreSQL&apos;s own battle-tested tooling instead of
                  re-inventing it. Requires PostgreSQL 17 or newer.
                </li>
                <li>
                  <strong>Logical backups</strong>: Uses <code>pg_dump</code> for
                  portable backups that can be restored to different PostgreSQL
                  versions. This is also the only backup type on PostgreSQL older
                  than 17 and the path for MySQL, MariaDB and MongoDB.
                </li>
                <li>
                  <strong>Nothing installed on the database</strong>: Backups
                  connect remotely; closed networks are reached through an SSH
                  tunnel to an internal host or a bastion, so the database never
                  has to be exposed publicly.
                </li>
                <li>
                  <strong>Efficient compression</strong>: Uses zstd (level 5)
                  compression for both logical and physical backups.
                </li>
                <li>
                  <strong>Read-only access</strong>: Logical backups only
                  require SELECT permissions, minimizing security risks.
                </li>
              </ul>

              <h3 id="backup-barman">Barman: Physical backups</h3>

              <p>
                Barman performs file-level (physical) backups of the PostgreSQL
                data directory:
              </p>

              <ul>
                <li>
                  <strong>Full cluster backup</strong>: Captures the entire
                  database cluster at the file system level using rsync or
                  pg_basebackup.
                </li>
                <li>
                  <strong>WAL archiving</strong>: Continuously archives
                  Write-Ahead Logs for Point-in-Time Recovery.
                </li>
                <li>
                  <strong>Incremental with rsync</strong>: Uses rsync to
                  transfer only changed files, reducing backup time and network
                  usage.
                </li>
                <li>
                  <strong>Streaming replication integration</strong>: Can
                  receive WAL files via streaming replication protocol for
                  real-time archiving.
                </li>
              </ul>

              <h2 id="recovery-options">Recovery options</h2>

              <p>
                Both tools offer flexible recovery options, but with different
                granularity:
              </p>

              <h3 id="recovery-databasus">Databasus recovery</h3>

              <ul>
                <li>
                  <strong>Point-in-Time Recovery</strong>: Restore to any
                  specific second using WAL replay.
                </li>
                <li>
                  <strong>Full cluster restore</strong>: Restore the entire
                  database cluster to a specific point in time from physical
                  backups.
                </li>
                <li>
                  <strong>Logical restore</strong>: Restore from scheduled
                  logical backups to any backup point.
                </li>
                <li>
                  <strong>One-click restore</strong>: Download and restore
                  logical backups directly from the web interface.
                </li>
                <li>
                  <strong>Cross-version compatibility</strong>: Logical
                  backups can be restored to different PostgreSQL versions.
                </li>
              </ul>

              <h3 id="recovery-barman">Barman recovery</h3>

              <ul>
                <li>
                  <strong>Point-in-Time Recovery (PITR)</strong>: Restore to any
                  specific second using WAL replay, minimizing data loss.
                </li>
                <li>
                  <strong>Full cluster restore</strong>: Restore the entire
                  database cluster to a specific point in time.
                </li>
                <li>
                  <strong>Remote recovery</strong>: Recover databases to remote
                  servers over SSH.
                </li>
                <li>
                  <strong>Standby creation</strong>: Create PostgreSQL replicas
                  from backups for high availability setups.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Note:</strong> Both
                  tools support PITR. Barman additionally offers standby
                  creation from backups and SSH-based remote recovery to other
                  servers, which can be valuable for high availability setups.{" "}
                  <a
                    href="/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Learn how Databasus supports PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">Ease of use</h2>

              <p>
                The tools differ dramatically in their approach to user
                experience:
              </p>

              <h3 id="ease-databasus">Databasus user experience</h3>

              <ul>
                <li>
                  <strong>Web interface</strong>: Point-and-click configuration
                  for all backup settings. No command-line required.
                </li>
                <li>
                  <strong>2-minute installation</strong>: One-line cURL script
                  or simple Docker command gets you running immediately.
                </li>
                <li>
                  <strong>Visual monitoring</strong>: Dashboard shows backup
                  status, health checks and history at a glance.
                </li>
                <li>
                  <strong>Built-in notifications</strong>: Configure Slack,
                  Teams, Telegram, Email or webhook alerts directly in the UI.
                </li>
                <li>
                  <strong>No PostgreSQL expertise required</strong>: Designed
                  for developers who want reliable backups without becoming
                  database experts.
                </li>
              </ul>

              <h3 id="ease-barman">Barman user experience</h3>

              <ul>
                <li>
                  <strong>Command-line interface</strong>: All operations
                  performed via terminal commands like{" "}
                  <code>barman backup</code>, <code>barman recover</code>.
                </li>
                <li>
                  <strong>Configuration files</strong>: Requires manual editing
                  of INI-style configuration files for each server.
                </li>
                <li>
                  <strong>WAL archiving setup</strong>: Must configure
                  PostgreSQL&apos;s <code>archive_command</code> or streaming
                  replication settings.
                </li>
                <li>
                  <strong>SSH key management</strong>: Requires setting up SSH
                  keys between Barman server and PostgreSQL servers.
                </li>
                <li>
                  <strong>DBA expertise expected</strong>: Documentation assumes
                  familiarity with PostgreSQL internals and WAL mechanics.
                </li>
              </ul>

              <p>
                <a
                  href="/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View Databasus installation guide →
                </a>
              </p>

              <h2 id="team-features">Team features</h2>

              <p>
                For organizations with multiple team members managing backups:
              </p>

              <h3 id="team-databasus">Databasus team capabilities</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: Organize databases, notifiers and
                  storages by project or team. Users only see workspaces
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

              <h3 id="team-barman">Barman team capabilities</h3>

              <p>
                Barman is a command-line tool without built-in team features:
              </p>

              <ul>
                <li>No user management or access control</li>
                <li>No audit logging of operations</li>
                <li>Team coordination requires external tools and processes</li>
                <li>Access controlled via OS-level permissions and SSH keys</li>
              </ul>

              <p>
                <a
                  href="/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about Databasus access management →
                </a>
              </p>

              <h2 id="security">Security</h2>

              <p>
                Both tools provide security features, but with different
                approaches:
              </p>

              <h3 id="security-databasus">Databasus security</h3>

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

              <h3 id="security-barman">Barman security</h3>

              <ul>
                <li>
                  <strong>SSH-based communication</strong>: Uses SSH for secure
                  communication between Barman server and PostgreSQL servers.
                </li>
                <li>
                  <strong>No built-in encryption</strong>: Barman does not
                  provide built-in backup encryption. External tools or
                  encrypted storage must be used.
                </li>
                <li>
                  <strong>OS-level security</strong>: Relies on file system
                  permissions and SSH key management for access control.
                </li>
                <li>
                  <strong>Checksum verification</strong>: Validates backup
                  integrity using checksums.
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

              <h2 id="storage-options">Storage options</h2>

              <p>The tools support different storage destinations:</p>

              <h3 id="storage-databasus">Databasus storage</h3>

              <p>Consumer-friendly options for various use cases:</p>

              <ul>
                <li>Local storage</li>
                <li>Amazon S3 and S3-compatible services</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>

              <h3 id="storage-barman">Barman storage</h3>

              <p>Enterprise-focused storage options:</p>

              <ul>
                <li>Local storage (POSIX file systems)</li>
                <li>Amazon S3 and S3-compatible object storage</li>
                <li>
                  Geographical redundancy via Barman-to-Barman replication
                </li>
              </ul>

              <p>
                <a
                  href="/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all Databasus storage options →
                </a>
              </p>

              <h2 id="notifications">Notifications</h2>

              <p>Staying informed about backup status:</p>

              <h3 id="notifications-databasus">Databasus notifications</h3>

              <p>Built-in support for multiple notification channels:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Webhooks</li>
              </ul>

              <h3 id="notifications-barman">Barman notifications</h3>

              <p>
                Barman does not have built-in notification support.
                Notifications require:
              </p>

              <ul>
                <li>Custom scripting around backup commands</li>
                <li>External monitoring tools integration</li>
                <li>Manual log parsing and alerting setup</li>
                <li>
                  Integration with tools like Nagios, Zabbix or custom solutions
                </li>
              </ul>

              <p>
                <a
                  href="/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all Databasus notification channels →
                </a>
              </p>

              <h2 id="multi-server-management">Multi-server management</h2>

              <p>
                Both tools can manage backups for multiple PostgreSQL servers,
                but with different approaches:
              </p>

              <h3 id="multi-databasus">Databasus approach</h3>

              <ul>
                <li>
                  <strong>Per-database scheduling</strong>: Each database can
                  have its own backup schedule and storage destination.
                </li>
                <li>
                  <strong>Workspace organization</strong>: Group related
                  databases into workspaces for easier management.
                </li>
                <li>
                  <strong>Unified dashboard</strong>: View all database backups
                  and their status in a single web interface.
                </li>
              </ul>

              <h3 id="multi-barman">Barman approach</h3>

              <ul>
                <li>
                  <strong>Centralized backup server</strong>: A dedicated Barman
                  server manages backups for multiple PostgreSQL instances.
                </li>
                <li>
                  <strong>Configuration per server</strong>: Each PostgreSQL
                  server requires its own configuration file on the Barman
                  server.
                </li>
                <li>
                  <strong>Geo-redundancy</strong>: Barman servers can replicate
                  to other Barman servers for geographical redundancy.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Databasus and Barman serve different needs in the PostgreSQL
                backup ecosystem. The right choice depends on your recovery
                requirements, team structure and technical expertise.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Choose Databasus if:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    You&apos;re an individual developer, team or enterprise
                    looking for an intuitive backup solution
                  </li>
                  <li>You prefer a web interface over command-line tools</li>
                  <li>
                    You need team collaboration features (workspaces, RBAC,
                    audit logs)
                  </li>
                  <li>
                    You want built-in notifications to Slack, Teams, Telegram
                    etc.
                  </li>
                  <li>
                    You want to manage backups for multiple databases from a
                    single dashboard with scheduling, notifications and team
                    features
                  </li>
                  <li>
                    You want quick setup with minimal PostgreSQL expertise
                  </li>
                  <li>Built-in backup encryption is important to you</li>
                  <li>
                    You use cloud-managed databases (AWS RDS, Google Cloud SQL,
                    Azure) or self-hosted PostgreSQL
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Choose Barman if:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    You need physical or incremental backups on PostgreSQL older
                    than 17 (Barman ships its own backup engine)
                  </li>
                  <li>
                    You need rsync-based incremental backups (file-level
                    diffing) for reduced transfer time
                  </li>
                  <li>
                    You need streaming replication integration for real-time
                    WAL archiving
                  </li>
                  <li>
                    You need Barman-to-Barman geographical redundancy
                  </li>
                  <li>
                    You need standby creation from backups for high
                    availability setups
                  </li>
                  <li>
                    You&apos;re comfortable with command-line tools and
                    PostgreSQL internals
                  </li>
                  <li>
                    Your organization has dedicated DBA expertise available
                  </li>
                </ul>
              </div>

              <p>
                Both tools support physical backups, WAL archiving and PITR, and
                both are built for disaster recovery with minimal RTO and RPO.
                Databasus works for databases of any size and complexity, and it
                gives you a web interface, team features and both logical and
                physical backups across self-hosted and cloud-managed databases.
                Barman is the better fit when you need rsync-based incremental
                backups, streaming replication integration, Barman-to-Barman
                geo-redundancy or standby creation from backups.
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
