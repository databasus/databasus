import type { Metadata } from "next";
import { getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

export const metadata: Metadata = {
  title: "How to backup Neon PostgreSQL: step-by-step guide | Databasus",
  description:
    "Step-by-step guide with screenshots: back up a Neon PostgreSQL database with Databasus on a schedule, store backups in your own storage and restore them into Neon.",
  keywords: [
    "Databasus",
    "Neon backup",
    "Neon PostgreSQL backup",
    "backup Neon database",
    "Neon restore",
    "Neon read-only user",
    "PostgreSQL backup",
    "database backup",
  ],
  openGraph: {
    title: "How to backup Neon PostgreSQL: step-by-step guide | Databasus",
    description:
      "Step-by-step guide with screenshots: back up a Neon PostgreSQL database with Databasus on a schedule, store backups in your own storage and restore them into Neon.",
    type: "article",
    locale: "en_US",
    url: getLocalizedUrl("en", "faq/neon"),
  },
  twitter: {
    card: "summary",
    title: "How to backup Neon PostgreSQL: step-by-step guide | Databasus",
    description:
      "Step-by-step guide with screenshots: back up a Neon PostgreSQL database with Databasus on a schedule, store backups in your own storage and restore them into Neon.",
  },
  alternates: {
    canonical: getLocalizedUrl("en", "faq/neon"),
    languages: getLanguageAlternates("faq/neon"),
  },
  robots: "index, follow",
};

export default function NeonPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to backup Neon PostgreSQL with Databasus",
            description:
              "Step-by-step guide to back up a Neon PostgreSQL database with Databasus and restore it into Neon",
            step: [
              {
                "@type": "HowToStep",
                name: "Copy the connection string from Neon",
                text: "In the Neon console, open your project, click Connect and click Copy snippet.",
                url: `${getLocalizedUrl("en", "faq/neon")}#copy-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Add the Neon database to Databasus",
                text: "Click Add database, choose Neon, paste the connection string with Paste from clipboard and click Test connection.",
                url: `${getLocalizedUrl("en", "faq/neon")}#paste-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Create a read-only user",
                text: "Click Yes, create read-only user, so Databasus stores credentials that cannot change your data.",
                url: `${getLocalizedUrl("en", "faq/neon")}#read-only-user`,
              },
              {
                "@type": "HowToStep",
                name: "Set the schedule, storage and retention",
                text: "Choose how often to back up, where to store backups and how long to keep them.",
                url: `${getLocalizedUrl("en", "faq/neon")}#schedule-storage-retention`,
              },
              {
                "@type": "HowToStep",
                name: "Choose notifiers",
                text: "Pick the channels that receive backup notifications and click Complete. Databasus starts the first backup.",
                url: `${getLocalizedUrl("en", "faq/neon")}#notifiers`,
              },
              {
                "@type": "HowToStep",
                name: "Restore a backup into Neon",
                text: "Click the restore icon next to a backup, paste the connection string of the target Neon database and click Restore to this DB.",
                url: `${getLocalizedUrl("en", "faq/neon")}#restore`,
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
              <h1 id="neon-backup">How to backup Neon PostgreSQL</h1>

              <p className="text-lg text-gray-400">
                Neon is a serverless managed PostgreSQL. To back it up with
                Databasus, copy the connection string from the Neon console,
                add the database in Databasus with the Neon type and choose
                where to store the backups. Databasus then backs it up on your schedule and can restore
                the backups into any Neon database.
              </p>

              <p>
                All you need is a running Databasus. If you don&apos;t have one yet, follow the{" "}
                <a href={getLocalizedHref("en", "installation")}>installation guide</a>
                . You can create a storage and notifiers along the way, right in the setup wizard.
              </p>

              <h2 id="copy-connection-string">
                Step 1. Copy the connection string from Neon
              </h2>

              <p>
                Open your project in the Neon console and click{" "}
                <strong>Connect</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-connect-button.png"
                alt="Connect button on the Computes page of a Neon project"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
              />

              <p>
                In the <strong>Connection details</strong> window, pick the
                branch, database and role to back up, then click{" "}
                <strong>Copy snippet</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-copy-connection-string.png"
                alt="Neon connection details window with the Copy snippet button"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="add-database">Step 2. Add the Neon database to Databasus</h2>

              <h3 id="paste-connection-string">
                Choose Neon and paste the connection string
              </h3>

              <p>
                In Databasus, click <strong>Add database</strong>, give the
                database a name, choose <strong>Neon</strong> and click{" "}
                <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-neon.png"
                alt="Add database window in Databasus with the Neon type selected"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Click <strong>Paste from clipboard</strong>. Databasus reads
                the connection string you copied in Neon and fills in the host,
                port, username, password, database name and SSL mode.
              </p>

              <img
                src="/images/faq/neon/databasus-paste-connection-string.png"
                alt="Neon connection form in Databasus with the Paste from clipboard button"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Click <strong>Test connection</strong>. When the test passes,
                click <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-test-connection.png"
                alt="Filled Neon connection form in Databasus after a passed connection test"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="read-only-user">Create a read-only user</h3>

              <p>
                Databasus offers to create a{" "}
                <a
                  href={`${getLocalizedHref("en", "security")}#level-3-read-only-access`}
                >
                  read-only user
                </a>{" "}
                for backups. Click <strong>Yes, create read-only user</strong>.
                Databasus then stores only credentials that can read your data
                and cannot change it.
              </p>

              <img
                src="/images/faq/neon/databasus-create-read-only-user.png"
                alt="Databasus offers to create a read-only user for Neon backups"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="schedule-storage-retention">
                Set the backup schedule, storage and retention
              </h3>

              <p>Choose:</p>

              <ul>
                <li>
                  <strong>Backup interval</strong> and time: how often
                  Databasus makes a backup, for example daily at 03:00.
                </li>
                <li>
                  <strong>Storage</strong>: where the backups go. You can{" "}
                  <a href={getLocalizedHref("en", "storages")}>
                    add a storage
                  </a>{" "}
                  right from this list.
                </li>
                <li>
                  <strong>Encryption</strong>: whether Databasus{" "}
                  <a
                    href={`${getLocalizedHref("en", "security")}#level-2-backups-encryption`}
                  >
                    encrypts the backup files
                  </a>{" "}
                  before it uploads them.
                </li>
                <li>
                  <strong>Retention policy</strong>: how long to keep old
                  backups, for example 3 months.
                </li>
                <li>
                  <strong>Notifications</strong>: which events to report, a
                  failed backup or a successful one too.
                </li>
              </ul>

              <img
                src="/images/faq/neon/databasus-schedule-storage-retention.png"
                alt="Backup schedule, storage, encryption and retention settings in Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="notifiers">Choose notifiers</h3>

              <p>
                Pick the{" "}
                <a href={getLocalizedHref("en", "notifiers")}>notifiers</a>{" "}
                that receive these notifications, for example{" "}
                <a href={getLocalizedHref("en", "notifiers/slack")}>Slack</a>{" "}
                or{" "}
                <a href={getLocalizedHref("en", "notifiers/teams")}>
                  Microsoft Teams
                </a>
                . You can pick several, or none and add them later. Click{" "}
                <strong>Complete</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-notifiers.png"
                alt="Notifier selection in Databasus before completing the database setup"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="first-backup">Step 3. Check the first backup</h2>

              <p>
                Databasus starts the first backup as soon as you click{" "}
                <strong>Complete</strong>. It appears on the{" "}
                <strong>Backups</strong> tab with its size and duration. The
                next backups follow the schedule, and{" "}
                <strong>Make backup right now</strong> starts one at any time.
              </p>

              <img
                src="/images/faq/neon/databasus-first-neon-backup.png"
                alt="First successful Neon backup on the Backups tab in Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                A successful backup is not yet a proven restore. To check that
                each backup really restores, set up{" "}
                <a href={getLocalizedHref("en", "restore-verification")}>
                  restore verification
                </a>
                .
              </p>

              <h2 id="restore">How to restore a Neon backup</h2>

              <p>
                Click the restore icon (a cloud with an arrow) next to the
                backup. Copy the connection string of the target database in
                the Neon console the same way as in{" "}
                <a href="#copy-connection-string">step 1</a>, click{" "}
                <strong>Paste from clipboard</strong> and then{" "}
                <strong>Restore to this DB</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-restore-neon-backup.png"
                alt="Restore from backup window in Databasus with the target Neon database"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Databasus clears the target database before the restore. The
                target can be:
              </p>

              <ul>
                <li>a new database in the same Neon project</li>
                <li>a database on a Neon branch</li>
                <li>an existing database, including the source itself</li>
              </ul>

              <p>
                The safest choice is a new Neon branch. You check the restored
                data there, and the production database stays untouched until
                you decide to switch.
              </p>

              <p>When the restore finishes, its status turns Successful.</p>

              <img
                src="/images/faq/neon/databasus-restore-done.png"
                alt="Finished restore of a Neon backup in Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="health-check">
                Will health checks keep my Neon compute awake?
              </h2>

              <p>
                Not by default. Neon puts a compute to sleep when nobody is
                connected and does not bill it while it sleeps. A health check
                connects every minute, which would keep the compute awake and
                billed around the clock. So Databasus switches health checks
                off for Neon databases. If you turn them on, Databasus asks you
                to confirm first. A scheduled backup wakes the compute only
                while it runs, and then Neon puts it back to sleep.
              </p>

              <p>
                Backing up Supabase as well? See{" "}
                <a href={getLocalizedHref("en", "faq/supabase")}>
                  how to backup Supabase
                </a>
                .
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("en", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Back to FAQ
                </a>
              </div>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
