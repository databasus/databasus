import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
  description:
    "Compare Databasus and PgBackWeb PostgreSQL backup tools. See differences in features, security, team support, storage options, notifications and ease of use.",
  keywords: [
    "Databasus vs PgBackWeb",
    "PostgreSQL backup comparison",
    "PgBackWeb alternative",
    "PostgreSQL backup tools",
    "database backup comparison",
    "pg_dump GUI",
    "self-hosted backup",
    "PostgreSQL backup security",
  ],
  openGraph: {
    title: "Databasus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Databasus and PgBackWeb PostgreSQL backup tools. See differences in features, security, team support, storage options, notifications and ease of use.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/databasus-vs-pgbackweb",
  },
  twitter: {
    card: "summary",
    title: "Databasus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Databasus and PgBackWeb PostgreSQL backup tools. See differences in features, security, team support, storage options, notifications and ease of use.",
  },
  alternates: {
    canonical: "https://databasus.com/databasus-vs-pgbackweb",
    languages: getLanguageAlternates("databasus-vs-pgbackweb"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackWebPage() {
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
              "Databasus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
            description:
              "A comprehensive comparison of Databasus and PgBackWeb PostgreSQL backup tools, covering features, security, team support, storage options and ease of use.",
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
              <h1 id="databasus-vs-pgbackweb">Databasus vs PgBackWeb</h1>

              <p className="text-lg text-gray-400">
                Both Databasus and PgBackWeb are open-source tools designed to
                simplify PostgreSQL backup management through web interfaces.
                While they share the common goal of making backups more
                accessible, they differ significantly in features, security,
                team support and ease of use.
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s a quick overview of the key differences between
                Databasus and PgBackWeb:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Feature</th>
                    <th>Databasus</th>
                    <th>PgBackWeb</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>License</td>
                    <td data-label="Databasus">Apache 2.0</td>
                    <td data-label="PgBackWeb">AGPL-3.0</td>
                  </tr>
                  <tr>
                    <td>Backups management</td>
                    <td data-label="Databasus">✅ Multiple DBs</td>
                    <td data-label="PgBackWeb">✅ Multiple DBs</td>
                  </tr>
                  <tr>
                    <td>Support of other DBs</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="PgBackWeb">❌ PostgreSQL only</td>
                  </tr>
                  <tr>
                    <td>Storage options</td>
                    <td data-label="Databasus">
                      Local, S3, Google Drive, Cloudflare R2, Azure, NAS,
                      Dropbox
                    </td>
                    <td data-label="PgBackWeb">Local, S3-compatible only</td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="Databasus">
                      Slack, Discord, Telegram, Teams, Email, Webhooks
                    </td>
                    <td data-label="PgBackWeb">Webhooks only</td>
                  </tr>
                  <tr>
                    <td>Security</td>
                    <td data-label="Databasus">
                      ✅ AES-256-GCM, unique backup keys, read-only enforcement
                    </td>
                    <td data-label="PgBackWeb">✅ PGP encryption</td>
                  </tr>
                  <tr>
                    <td>Team features</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, role-based access, audit logs
                    </td>
                    <td data-label="PgBackWeb">❌ Not available</td>
                  </tr>
                  <tr>
                    <td>Health monitoring</td>
                    <td data-label="Databasus">✅ Built-in</td>
                    <td data-label="PgBackWeb">❌ Not available</td>
                  </tr>
                  <tr>
                    <td>Installation</td>
                    <td data-label="Databasus">
                      One-line script, Docker or Helm
                    </td>
                    <td data-label="PgBackWeb">Manual Docker setup</td>
                  </tr>
                  <tr>
                    <td>Physical backups</td>
                    <td data-label="Databasus">✅ Yes</td>
                    <td data-label="PgBackWeb">❌ Not available</td>
                  </tr>
                  <tr>
                    <td>Incremental backups</td>
                    <td data-label="Databasus">✅ Block-level (PG 17+)</td>
                    <td data-label="PgBackWeb">❌ Not available</td>
                  </tr>
                  <tr>
                    <td>WAL archiving</td>
                    <td data-label="Databasus">✅ Continuous streaming</td>
                    <td data-label="PgBackWeb">❌ Not available</td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="Databasus">✅ Yes</td>
                    <td data-label="PgBackWeb">❌ Not available</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="backup-features">Backup features</h2>

              <p>Both tools support scheduled backups with flexible timing:</p>

              <ul>
                <li>
                  <strong>Databasus</strong>: Supports hourly, daily, weekly
                  monthly or cron schedules with precise timing (e.g. 4 AM).
                  Implements{" "}
                  <strong>balanced compression using zstd (level 5)</strong>,
                  reducing backup sizes by 4-8x with only ~20% runtime overhead.
                  This is significantly more efficient than gzip.
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Supports cron-based scheduling for
                  backup execution. Uses gzip compression for backups which is
                  slower and less efficient than zstd.
                </li>
              </ul>

              <p>
                Beyond logical backups, Databasus also supports physical,
                incremental and WAL backups. These are built on PostgreSQL
                17&apos;s native backup stack and run remotely, so nothing is
                installed on the database server and closed networks can be
                reached through an SSH tunnel. This gives you block-level
                incremental backups, continuous WAL streaming and Point-in-Time
                Recovery for disaster recovery with near-zero data loss,
                restoring to any second between backups. PgBackWeb does not offer
                any of this.
              </p>

              <h2 id="storage-options">Storage options</h2>

              <p>
                Storage flexibility is crucial for backup strategies.
                Here&apos;s how the two tools compare:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: Supports a wide range of storage
                  destinations:
                  <ul>
                    <li>Local storage</li>
                    <li>Amazon S3 and S3-compatible services</li>
                    <li>Google Drive</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS (Network-attached storage)</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Limited to local storage and
                  S3-compatible storage only.
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

              <h2 id="security">Security</h2>

              <p>
                Security is a critical aspect of backup management. Databasus
                implements enterprise-grade security on three levels:
              </p>

              <h3 id="security-databasus">Databasus security model</h3>

              <ol>
                <li>
                  <strong>Sensitive data encryption</strong>: All passwords,
                  tokens and credentials are encrypted with AES-256-GCM. The
                  encryption key is stored separately from the database, so even
                  if the database is compromised, sensitive data remains
                  protected.
                </li>
                <li>
                  <strong>Backup encryption</strong>: Each backup file is
                  encrypted with a unique key derived from the master key,
                  backup ID and random salt. Even if someone gains access to
                  your cloud storage, they cannot read the backups without your
                  encryption key.
                </li>
                <li>
                  <strong>Read-only database access</strong>: Databasus
                  enforces read-only access by checking role-level,
                  database-level and table-level permissions. It only requires
                  SELECT permissions and will warn you if write privileges are
                  detected. This prevents data corruption even if Databasus is
                  compromised.
                </li>
              </ol>

              <h3 id="security-pgbackweb">PgBackWeb security model</h3>

              <ul>
                <li>
                  <strong>PGP encryption</strong>: PgBackWeb offers PGP
                  encryption for backup files.
                </li>
                <li>
                  <strong>No read-only enforcement</strong>: PgBackWeb does not
                  enforce or verify read-only database access which means
                  backups may be created with users that have write permissions.
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

              <h2 id="notifications">Notifications</h2>

              <p>
                Staying informed about backup status is essential for reliable
                operations:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: Provides real-time notifications
                  through multiple channels:
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>Email</li>
                    <li>Webhooks</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Supports webhooks only for
                  notifications. To receive alerts via Slack, Telegram or other
                  platforms you need to set up additional middleware or
                  services.
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

              <h2 id="team-features">Team features</h2>

              <p>
                For organizations and DevOps teams, collaboration features are
                essential. This is where Databasus significantly outshines
                PgBackWeb:
              </p>

              <h3 id="team-databasus">Databasus team capabilities</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: Group databases, notifiers and
                  storages for different projects or teams. Users only see
                  workspaces they&apos;re invited to.
                </li>
                <li>
                  <strong>Role-based access control</strong>: Permission levels
                  to control what each team member can do within workspaces.
                </li>
                <li>
                  <strong>Audit logs</strong>: Track all system activities and
                  changes made by users. Essential for security compliance and
                  team accountability.
                </li>
              </ul>

              <h3 id="team-pgbackweb">PgBackWeb team capabilities</h3>

              <p>
                PgBackWeb does not have built-in user management, workspaces or
                audit logs. It&apos;s designed primarily for single-user
                scenarios.
              </p>

              <p>
                <a
                  href="/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about Databasus access management →
                </a>
              </p>

              <h2 id="ease-of-use">Ease of use</h2>

              <p>
                <strong>
                  Databasus is designed to be significantly easier to use
                </strong>{" "}
                than PgBackWeb, with a focus on intuitive UX and minimal setup
                time:
              </p>

              <h3 id="ease-databasus">Databasus user experience</h3>

              <ul>
                <li>
                  <strong>Easy installation</strong>: Use Docker directly or run
                  a one-line script that installs Docker (if needed), sets up
                  Databasus and configures automatic startup. Total time: ~2
                  minutes.
                </li>
                <li>
                  <strong>Intuitive web interface</strong>: Designer-polished UI
                  that guides you through backup configuration step by step. No
                  PostgreSQL expertise required.
                </li>
                <li>
                  <strong>Dark and light themes</strong>: Choose the look that
                  suits your workflow.
                </li>
                <li>
                  <strong>Mobile adaptive</strong>: Check your backups from
                  anywhere on any device.
                </li>
                <li>
                  <strong>Built-in health monitoring</strong>: Configurable
                  health checks with visual availability charts.
                </li>
                <li>
                  <strong>One-click restore</strong>: Download and restore from
                  any backup with a single click.
                </li>
              </ul>

              <h3 id="ease-pgbackweb">PgBackWeb user experience</h3>

              <ul>
                <li>
                  <strong>Manual Docker setup</strong>: Requires configuring
                  environment variables and setting up an external PostgreSQL
                  database for configuration storage.
                </li>
                <li>
                  <strong>Basic web interface</strong>: Functional but less
                  polished UI compared to Databasus. Dark theme available.
                </li>
                <li>
                  <strong>No health monitoring</strong>: Database availability
                  monitoring must be set up separately.
                </li>
              </ul>

              <h2 id="installation">Installation and deployment</h2>

              <h3 id="install-databasus">Installing Databasus</h3>

              <p>
                Databasus offers three installation methods, with the automated
                script being the quickest:
              </p>

              <ul>
                <li>
                  <strong>Automated script (recommended)</strong>: One-line cURL
                  command that installs Docker, sets up Databasus and
                  configures automatic startup.
                </li>
                <li>
                  <strong>Docker run</strong>: Single command to start
                  Databasus with embedded PostgreSQL.
                </li>
                <li>
                  <strong>Docker Compose</strong>: For more control over the
                  deployment.
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

              <h3 id="install-pgbackweb">Installing PgBackWeb</h3>

              <p>
                PgBackWeb requires Docker and manual configuration of
                environment variables. You also need to set up an external
                PostgreSQL database for storing PgBackWeb&apos;s configuration.
              </p>

              <h2 id="licensing">Licensing</h2>

              <p>
                The licensing model can significantly impact how you can use and
                modify the software:
              </p>

              <ul>
                <li>
                  <strong>Databasus (Apache 2.0)</strong>: Permissive license
                  that allows unrestricted commercial use, modification and
                  distribution. You can use Databasus in proprietary projects
                  without any licensing concerns.
                </li>
                <li>
                  <strong>PgBackWeb (AGPL-3.0)</strong>: Copyleft license that
                  requires any derivative works or modifications to also be
                  open-source under AGPL-3.0. If you modify PgBackWeb and
                  provide it as a service you must release your modifications.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Both Databasus and PgBackWeb are capable PostgreSQL backup
                tools, but they serve different needs:
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Choose Databasus if you need:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>Enterprise-grade security with 3-level protection</li>
                  <li>Team collaboration with workspaces and audit logs</li>
                  <li>
                    Multiple storage destinations (Google Drive, Azure etc.)
                  </li>
                  <li>Built-in notifications to Slack, Teams, Telegram etc.</li>
                  <li>Quick installation with one-line script or Docker</li>
                  <li>Intuitive modern UI with minimal learning curve</li>
                  <li>Permissive Apache 2.0 license for commercial use</li>
                  <li>
                    Physical backups, incremental backups, WAL archiving and
                    PITR for disaster recovery
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Choose PgBackWeb if you need:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>Simple backup solution for single-user scenarios</li>
                  <li>Only local or S3 storage</li>
                  <li>Webhook-only notifications are sufficient</li>
                  <li>AGPL-3.0 license is acceptable for your use case</li>
                </ul>
              </div>

              <p>
                For most users, especially teams and organizations requiring
                robust security, multiple storage options and comprehensive
                notification channels,{" "}
                <strong>Databasus is the recommended choice</strong>.
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
