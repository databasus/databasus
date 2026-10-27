import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Restore verification - Databasus Documentation",
  description:
    "Prove your database backups are actually restorable. Databasus pulls the latest backup, restores it into a throwaway database container, checks the restored database against the source and reports per-table row counts on every run.",
  keywords: [
    "restore verification",
    "database restore",
    "backup verification",
    "disaster recovery",
    "database backup testing",
    "Databasus verification agent",
    "backup integrity",
    "automated restore test",
  ],
  openGraph: {
    title: "Restore verification - Databasus Documentation",
    description:
      "Prove your database backups are actually restorable. Databasus pulls the latest backup, restores it into a throwaway database container, checks the restored database against the source and reports per-table row counts on every run.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/restore-verification",
  },
  twitter: {
    card: "summary",
    title: "Restore verification - Databasus Documentation",
    description:
      "Prove your database backups are actually restorable. Databasus pulls the latest backup, restores it into a throwaway database container, checks the restored database against the source and reports per-table row counts on every run.",
  },
  alternates: {
    canonical: "https://databasus.com/restore-verification",
    languages: getLanguageAlternates("restore-verification"),
  },
  robots: "index, follow",
};

export default function RestoreVerificationPage() {
  const downloadAgent = `curl -L -o verification-agent "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" \\
  && chmod +x verification-agent`;

  const startAgent = `./verification-agent start \\
  --databasus-host=https://your-databasus-host \\
  --agent-id=<AGENT_ID> \\
  --token=<TOKEN> \\
  --max-cpu=2 \\
  --max-ram-mb=2048 \\
  --max-disk-gb=20 \\
  --max-concurrent-jobs=1`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Restore verification - Databasus Documentation",
            description:
              "Prove your database backups are actually restorable. Databasus pulls the latest backup, restores it into a throwaway database container, checks the restored database against the source and reports per-table row counts on every run.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to set up restore verification in Databasus",
            description:
              "Step-by-step guide to register a verification agent, launch it on your server, and configure scheduled restore verification.",
            step: [
              {
                "@type": "HowToStep",
                name: "Create a verification agent in the UI",
                text: "Go to Settings → Verification agents, click Create verification agent, name it, and copy the token and agent ID from the dialog.",
              },
              {
                "@type": "HowToStep",
                name: "Download the agent binary",
                text: "Run the curl command on the host where verification should run, choosing amd64 or arm64 for your architecture.",
              },
              {
                "@type": "HowToStep",
                name: "Launch the agent",
                text: "Start the agent with --agent-id, --token and resource budgets (--max-cpu, --max-ram-mb, --max-disk-gb, --max-concurrent-jobs).",
              },
              {
                "@type": "HowToStep",
                name: "Schedule verifications",
                text: "Open the database's verification settings, enable Scheduled verification, and pick an interval (After backup, Hourly, Daily, Weekly, Monthly or Cron).",
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
              <h1 id="restore-verification">Restore verification</h1>

              <p className="text-lg text-gray-400">
                A backup that finishes without error is not the same as a backup
                you can actually restore. The only real proof is to restore it.
                Databasus does this for you on a schedule:
              </p>

              <ul>
                <li>takes the latest backup</li>
                <li>runs restore into a throwaway database container</li>
                <li>sanity-checks the restored database against the source</li>
                <li>tears the container down</li>
                <li>reports the outcome</li>
              </ul>

              <img
                src="/images/restore-verification/verified-backups.png"
                alt="Verified backups tab"
                loading="lazy"
              />

              <img
                className="mt-5"
                src="/images/restore-verification/verifications.png"
                alt="Verifications tab"
                loading="lazy"
              />

              <h2 id="what-is-verification-agent">
                What is a verification agent?
              </h2>

              <p>
                The verification agent is a small Go binary you run on a machine
                you control — anything with spare CPU, RAM and disk works. The
                agent registers with Databasus, picks up verification jobs from
                a queue, runs them locally and reports results back.
              </p>

              <h3 id="what-you-need">What you need</h3>

              <ul>
                <li>
                  A host with outbound HTTPS access to your Databasus URL.
                </li>
                <li>
                  Docker available on that host — the agent spins up ephemeral
                  database containers of the matching major version for each
                  job.
                </li>
                <li>
                  Disk capacity for each verification job covering the{" "}
                  <strong>backup file size</strong>, the{" "}
                  <strong>database raw size</strong> and a{" "}
                  <strong>safety gap</strong> on top.
                </li>
                <li>
                  At least 1 CPU core and 512 MB of RAM available per concurrent
                  job.
                </li>
              </ul>

              <h3 id="why-not-just-checksums">Why not just checksums?</h3>

              <p>
                Checksums and exit codes catch some failure modes but miss
                others entirely:
              </p>

              <ul>
                <li>
                  <strong>Checksums</strong> catch bit rot on the archive file,
                  but say nothing about whether the dump itself is complete or
                  semantically valid.
                </li>
                <li>
                  <strong>Dump exit code</strong> says the dump command ran. It
                  does not catch a role missing read permissions on certain
                  objects, a missing extension on the source or a tablespace
                  mismatch — all of which can cause objects to be silently
                  skipped or stripped.
                </li>
                <li>
                  <strong>Restore verification</strong> actually runs the
                  archive through the database&apos;s native restore tool and
                  counts rows per table. It is the only check that catches all
                  of the above — if a backup will not restore, you find out
                  before you need it, not during a disaster.
                </li>
              </ul>

              <h2 id="configuration">Configuration</h2>

              <h3 id="create-on-ui">Create an agent in the UI</h3>

              <p>
                Open <strong>Settings → Verification agents</strong> and click{" "}
                <strong>Create verification agent</strong>. Pick a descriptive
                name like <code>staging-verifier</code> or{" "}
                <code>eu-west-host-1</code>. The next dialog shows the
                agent&apos;s <strong>token</strong> and <strong>ID</strong>.
              </p>

              <p>
                The token is shown <strong>exactly once</strong> — copy it
                before closing the dialog. If you lose it later, use the{" "}
                <strong>Rotate token</strong> action on the agent&apos;s row to
                issue a new one; the old token stops working on the agent&apos;s
                next heartbeat. The dialog that follows shows the install
                commands for your server&apos;s architecture — the same commands
                described below.
              </p>

              <h3 id="launch">Launch the agent on your server</h3>

              <p>
                SSH into the machine that will run verifications. First,
                download the agent binary. Replace{" "}
                <code>https://your-databasus-host</code> with your own Databasus
                URL, and swap <code>amd64</code> for <code>arm64</code> if your
                server is ARM:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={downloadAgent} />
                </div>
              </div>

              <p>
                Then launch the agent. The agent ID and token come from the
                dialog in the previous step:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startAgent} />
                </div>
              </div>

              <p>
                <code>start</code> daemonises the agent and writes its flags to{" "}
                <code>databasus-verification.json</code> in the working
                directory, so later restarts can use{" "}
                <code>./verification-agent start</code> with no flags at all.
                Logs are written to <code>databasus-verification.log</code> next
                to the binary.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] mb-3 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    The Databasus host must be <code>https://</code>. Plain HTTP
                    is only allowed if you add{" "}
                    <code>--allow-insecure-http</code>, and it is intended for
                    local testing — never expose a production agent over
                    unencrypted HTTP.
                  </p>
                </div>
              </div>

              <p>
                The four <code>--max-*</code> flags are <strong>budgets</strong>
                , not per-job allocations. The agent reports them to Databasus
                on every heartbeat, and Databasus divides them across the
                concurrent jobs you allow. With{" "}
                <code>
                  --max-cpu=2 --max-ram-mb=2048 --max-concurrent-jobs=1
                </code>{" "}
                the single job gets all 2 CPUs and 2 GB of RAM. With{" "}
                <code>--max-concurrent-jobs=2</code>, each job gets 1 CPU and 1
                GB. The floor is 1 CPU and 512 MB per job — if your budget
                can&apos;t satisfy that floor, the agent advertises lower
                concurrency. The disk budget is the easiest to get wrong: each
                job needs enough space for the <strong>backup file size</strong>
                , the <strong>database raw size</strong> and a{" "}
                <strong>safety gap of up to 5 GB</strong> on top, so set{" "}
                <code>--max-disk-gb</code> comfortably above that for your
                largest database.
              </p>

              <h3 id="manage">Manage the agent</h3>

              <p>The same binary provides four subcommands:</p>

              <ul>
                <li>
                  <code>./verification-agent status</code> — show whether the
                  daemon is running and what jobs it is currently working on.
                </li>
                <li>
                  <code>./verification-agent stop</code> — stop the daemon.
                  In-flight verifications are reported back to Databasus as
                  failed and are re-queued.
                </li>
                <li>
                  <code>./verification-agent start</code> — re-launch the
                  daemon. Flags are remembered from the first start; pass{" "}
                  <code>--token=&lt;NEW&gt;</code> after a rotation to update
                  the stored token.
                </li>
                <li>
                  <code>./verification-agent run</code> — run in the foreground
                  instead of as a daemon. Use this when wrapping the agent in a
                  systemd unit or a Docker container — those supervisors expect
                  the process not to fork off.
                </li>
              </ul>

              <p>
                The Settings page shows three icon actions on each agent&apos;s
                row: view the install commands again (without revealing the
                token), rotate the token, and delete the agent. Deleting is safe
                — any verifications currently assigned to that agent are
                returned to the queue and picked up by another agent if one is
                available.
              </p>

              <h2 id="schedules-and-notifications">
                Schedules and notifications
              </h2>

              <p>
                Restore verification is configured per database. Open the
                database&apos;s verification settings, toggle on{" "}
                <strong>Scheduled verification</strong>, then pick an interval.
              </p>

              <h3 id="interval-options">Interval options</h3>

              <ul>
                <li>
                  <strong>After backup</strong> — strongest guarantee: every
                  successful backup is verified the moment it finishes.
                </li>
                <li>
                  <strong>Hourly, daily, weekly, monthly</strong> — pick a
                  cadence and a time of day.
                </li>
                <li>
                  <strong>Cron</strong> — a UTC cron expression for anything the
                  presets don&apos;t cover. Examples: <code>0 4 * * 0</code>{" "}
                  (every Sunday at 4:00 AM UTC) and <code>0 */6 * * *</code>{" "}
                  (every six hours).
                </li>
              </ul>

              <h3 id="how-the-queue-works">
                How the queue handles &quot;After backup&quot;
              </h3>

              <p>
                A verification is usually slower than the backup that produced
                it, so if backups arrive faster than verifications finish, the
                queue would grow forever. Databasus avoids this by{" "}
                <strong>
                  cancelling any pending verification for the same database
                  whenever a fresh backup arrives
                </strong>{" "}
                — only the most recent backup waits in line. The trade-off is
                intentional: it is better to skip a verification of a stale
                backup than to spend hours verifying something you&apos;d never
                restore from anyway.
              </p>

              <h3 id="manual-runs">Manual runs</h3>

              <p>
                You can also kick off a one-off verification from the
                database&apos;s <strong>Restore verifications</strong> tab
                without changing the schedule. Useful for spot-checking a
                specific backup or smoke-testing a new agent end-to-end before
                you trust it with the scheduled load.
              </p>

              <h3 id="notifications">Notifications</h3>

              <p>
                Success and failure can be sent through any notifier already
                wired up for the database. The two checkboxes —{" "}
                <strong>Verification success</strong> and{" "}
                <strong>Verification failed</strong> — are independent. Most
                teams enable only the failure one to avoid notification fatigue.
                See the{" "}
                <a
                  href="/notifiers"
                  className="text-blue-400 hover:text-blue-300"
                >
                  notifiers documentation
                </a>{" "}
                to wire up Slack, Microsoft Teams, Discord, email, and others.
              </p>

              <h3 id="results">Reading the results</h3>

              <p>
                Each verification attempt shows up as one row in the
                database&apos;s <strong>Restore verifications</strong> tab. The
                status is one of <strong>Pending</strong>,{" "}
                <strong>Running</strong>, <strong>Successful</strong>,{" "}
                <strong>Failed</strong> or <strong>Canceled</strong>. Clicking a
                row opens a drawer with the full timeline, the restore exit
                code, the restored database size, schema and table counts, and a
                per-table row-count breakdown. Failed runs show the failure
                message at the top of the drawer.
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
