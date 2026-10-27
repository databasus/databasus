import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Security - How Databasus Protects Your Data | Databasus",
  description:
    "Learn how Databasus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
  keywords: [
    "Databasus security",
    "PostgreSQL backup security",
    "AES-256-GCM encryption",
    "database encryption",
    "backup encryption",
    "read-only database access",
    "enterprise security",
    "data protection",
    "secure backups",
  ],
  openGraph: {
    title: "Security - How Databasus Protects Your Data | Databasus",
    description:
      "Learn how Databasus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/security",
  },
  twitter: {
    card: "summary",
    title: "Security - How Databasus Protects Your Data | Databasus",
    description:
      "Learn how Databasus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
  },
  alternates: {
    canonical: "https://databasus.com/security",
    languages: getLanguageAlternates("security"),
  },
  robots: "index, follow",
};

export default function SecurityPage() {
  const encryptionPipeline = `PostgreSQL pg_dump → Compression → Encryption → Cloud Storage`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Security - How Databasus Protects Your Data",
            description:
              "Learn how Databasus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
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
              <h1 id="security">How Databasus enforces security?</h1>

              <p className="text-lg text-gray-400">
                Databasus is responsible for sensitive data:
              </p>

              <ul>
                <li>it accesses your DB;</li>
                <li>it backs it up (meaning makes a copy of data);</li>
                <li>
                  it keeps credentials to be able to access your DB on a regular
                  basis;
                </li>
                <li>
                  it saves backups in your S3 or other cloud storages (if you enable it);
                </li>
              </ul>

              <p>
                Therefore,{" "}
                <strong>
                  there is a main priority for Databasus to be enterprise-level
                  secure and reliable
                </strong>
                .
              </p>

              <p>To make sure:</p>

              <ul>
                <li>sensitive data is never exposed and always encrypted;</li>
                <li>
                  backups are encrypted and useless even if someone sees them in
                  the cloud storage;
                </li>
                <li>
                  Databasus doesn&apos;t even receive access to DB with write
                  or update access;
                </li>
                <li>all actions are logged and can be audited;</li>
              </ul>

              <p>
                All these steps protect your data. As you know, there is no 100%
                secure system, but we do our best to make it as secure as
                possible. Even in case of hacking, nobody will be able to
                corrupt your data.
              </p>

              <p>Databasus enforces security on three levels:</p>

              <ol>
                <li>Sensitive data encryption;</li>
                <li>Backups encryption;</li>
                <li>Read-only access to DB.</li>
              </ol>

              <h2 id="level-1-sensitive-data-encryption">
                Level 1: sensitive data encryption
              </h2>

              <p>
                Internally, Databasus uses PostgreSQL DB to store connection
                details, configs, settings of notifiers and storages (S3, Google
                Drive, Dropbox, etc.).
              </p>

              <p>Any sensitive data is encrypted. For example:</p>

              <ul>
                <li>passwords</li>
                <li>tokens</li>
                <li>webhooks with secrets</li>
              </ul>

              <p>
                So in DB Databasus keeps only hashes or encoded values. For
                encryption is used <strong>AES-256-GCM</strong> algorithm. Also,
                despite the encryption, those values are never exposed via API
                or UI.
              </p>

              <p>
                The secret key used for encryption is stored on local storage (
                <code>./databasus-data/secret.key</code> by default) and is not
                present in the DB itself. So DB compromise doesn&apos;t give
                access to sensitive data.
              </p>

              <h2 id="level-2-backups-encryption">
                Level 2: backups encryption
              </h2>

              <p>
                Each backup file is encrypted on the fly during backup creation.
                Databasus uses <strong>AES-256-GCM</strong> encryption
                algorithm, which ensures that backup data cannot be read without
                the encryption key and any tampering is detected during
                decryption.
              </p>

              <p>Backups flow through this pipeline:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{encryptionPipeline}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={encryptionPipeline} />
                </div>
              </div>

              <p>
                Each backup gets its own unique encryption key derived from:
              </p>

              <ul>
                <li>
                  Master key (stored in{" "}
                  <code>./databasus-data/secret.key</code>)
                </li>
                <li>Backup ID</li>
                <li>Random salt (unique per backup)</li>
              </ul>

              <p>
                <strong>Result</strong>: Even if someone gains access to your
                cloud storage (S3, Google Drive, etc.), they cannot read the
                backups without your master key.
              </p>

              <h2 id="level-3-read-only-access">
                Level 3: read-only access to DB
              </h2>

              <p>
                Databasus enforces the principle of least privilege - it only
                needs read access to create backups, never write access. This
                protects your database from accidental or malicious data
                corruption through the backup tool.
              </p>

              <p>
                Before accepting database credentials, Databasus performs
                checks across three levels:
              </p>

              <ol>
                <li>
                  <strong>Role-level</strong>: Verifies the user is NOT a
                  superuser and cannot create roles or databases
                </li>
                <li>
                  <strong>Database-level</strong>: Ensures no CREATE or TEMP
                  privileges
                </li>
                <li>
                  <strong>Table-level</strong>: Confirms zero write permissions
                  (INSERT, UPDATE, DELETE, TRUNCATE, etc.)
                </li>
              </ol>

              <p>
                The database user must pass all three checks to be considered
                read-only. If any write privilege is detected, Databasus will
                warn you.
              </p>

              <p>
                Databasus suggests creating read-only users for you with proper
                permissions:
              </p>

              <ul>
                <li>Grants SELECT on all current and future tables</li>
                <li>Grants USAGE on schemas (but not CREATE)</li>
                <li>Explicitly revokes all write privileges</li>
              </ul>

              <p>
                <strong>Result</strong>: Even if Databasus is compromised,
                server is hacked, secret key is stolen and credentials are
                decrypted, attackers cannot corrupt your database.
              </p>

              <h2 id="two-factor-authentication">
                Two-factor authentication at sign-in
              </h2>

              <p>
                A password alone opens every database credential, storage key
                and backup the instance holds, so an administrator can require a
                second factor for password sign-in. With it on, a correct
                password sends a six-digit code to the account&apos;s email
                address, and the sign-in finishes only once that code is
                entered. The code stops working ten minutes after it was sent,
                is discarded after five wrong guesses, and requesting another
                one replaces it. Turning the setting on guards the next sign-in
                and leaves sessions that are already open working.
              </p>

              <p>
                The setting covers password sign-in. Signing in through Google
                or GitHub still issues a token directly, because those providers
                run their own multi-factor checks - an instance that wants the
                second factor to cover everybody does not configure them.
              </p>

              <p>
                It can only be turned on when the instance has a mail server and
                every active administrator holds a real email address, so nobody
                switches it on and locks the instance out. If the mail server
                later stops delivering, nobody gets in with a password, which is
                deliberate: a broken mail server must not quietly downgrade
                authentication back to one factor. An operator with shell access
                on the server switches it off again:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{disableTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={disableTwoFactorCommand} />
                </div>
              </div>

              <p>
                The command reports what it changed and records it in the audit
                log, so an instance that stopped asking for codes says why. It
                is documented beside password recovery on the{" "}
                <a
                  href="/password#disable-two-factor"
                  className="text-blue-400 hover:text-blue-600"
                >
                  password page
                </a>
                .
              </p>

              <h2 id="security-and-reliability-engineering">
                🛡️ Security &amp; reliability engineering
              </h2>

              <p>
                Databasus works with sensitive data, so preventing
                vulnerabilities, unauthorised access and data leaks is a primary
                concern. We invest in this on both sides of the system: in the
                code itself (permission checks, encryption, careful handling of
                secrets) and in the infrastructure around it (dependency
                analysis, CVE response, DevSecOps practices). The pipeline below
                runs automatically on every commit and PR — no single layer is
                enough on its own, but together they reduce the chance of
                vulnerable code, unsafe dependencies, broken images, or
                non-restorable backups reaching a release.
              </p>

              <h3 id="static-analysis">Static analysis</h3>

              <p>
                Static analysis runs in several independent passes. CodeQL
                scans the full codebase for security issues. On every PR,{" "}
                <strong>gitleaks</strong> scans for leaked secrets and{" "}
                <strong>semgrep</strong> checks the code against security rules.
                Dockerfiles and CI workflows get extra rules of their own
                (pinned action references, least-privilege permissions,
                suspicious base images), so insecure patterns are flagged
                before they ever merge.
              </p>

              <p>
                On top of these per-PR checks,{" "}
                <strong>Codex Security</strong> from OpenAI runs regular,
                deeper audits of the whole codebase. It is a separate program
                that catches architectural and cross-cutting issues which
                narrow PR-time scans miss.
              </p>

              <h3 id="dependency-management">Dependency management</h3>

              <p>
                Dependabot watches all of our dependencies against the GitHub
                Advisory Database and surfaces CVEs within minutes of
                publication. Updates run through a cooldown so newly-published
                versions get a chance to mature before we adopt them — a
                deliberate defence against compromised-package incidents like
                supply-chain attacks.
              </p>

              <p>
                The <strong>Dependency Review Action</strong> blocks any PR
                that introduces a new <strong>HIGH</strong> or{" "}
                <strong>CRITICAL</strong> CVE outright.
              </p>

              <h3 id="container-and-ci-hardening">
                Container &amp; CI hardening
              </h3>

              <ul>
                <li>
                  Container images are scanned with <strong>Trivy</strong> on
                  every build.
                </li>
                <li>
                  A separate Trivy pass on the Dockerfile catches
                  misconfigurations before they make it into an image.
                </li>
                <li>
                  All GitHub Actions are pinned to full commit SHAs rather than
                  floating tags like <code>@v4</code> or <code>@main</code>,
                  which have been an active attack vector in 2025.
                </li>
                <li>
                  Workflows default to least-privilege permissions and only
                  elevate per-job when genuinely needed.
                </li>
              </ul>

              <h3 id="testing-and-verification">Testing &amp; verification</h3>

              <p>
                Critical paths are covered by both unit and integration tests,
                run against real database containers for every supported engine
                and major version.
              </p>

              <p>
                Restore is the path that matters most for a backup tool, so we
                test it explicitly: every PR runs full backup-then-restore
                cycles against those same real containers, verifying that
                backups can actually be restored end-to-end — not just written
                successfully.
              </p>

              <p>
                The rest of the CI/CD pipeline runs lint, type-check, the full
                test suite, image smoke tests and multi-architecture builds on
                every PR. A release only ships if all of it passes.
              </p>

              <h3 id="reporting-a-vulnerability">
                Reporting a vulnerability
              </h3>

              <p>
                Found a vulnerability? Report it via the GitHub Security tab —
                see{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . Security reports are the highest-priority work queue.
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
