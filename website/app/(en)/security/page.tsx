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
                details, configs, settings of notifiers and storages (S3, Azure Blob Storage, FTP, etc.).
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
                cloud storage (S3, Azure Blob Storage, etc.), they cannot read the
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
                Databasus handles sensitive data. Before each release, automated checks look
                for security issues and verify that backups can be restored. These checks
                reduce risk, but cannot guarantee that software is free of vulnerabilities.
              </p>

              <h3 id="static-analysis">Static analysis</h3>

              <p>
                CodeQL checks Go, JavaScript/TypeScript and GitHub Actions workflows.
                gitleaks scans the release&#x27;s Git history for leaked secrets. semgrep
                checks source code, Dockerfiles and CI workflows against security rules.
                These scans run before each release and on a weekly schedule.
              </p>

              <p>
                Open <strong>CRITICAL</strong> CodeQL findings in the release analysis block
                publication. <strong>HIGH</strong> findings produce warnings.
              </p>

              <p>
                OpenAI&#x27;s Codex Security also provides regular, deeper audits. These can
                find architectural issues that narrower scans may miss. They are separate
                from the automated release checks.
              </p>

              <h3 id="dependency-management">Dependency management</h3>

              <p>
                Ordinary updates wait seven complete days for packages, development tools
                and GitHub Actions, or fourteen days for images, database clients and
                standalone verification extensions. Maintainers check publication dates and
                compatibility, including exact version selections and security updates that
                can bypass age filters. Exceptions are reported with a reason.
              </p>

              <p>
                Dependency Review Action compares the candidate release with the previous
                release and blocks newly introduced vulnerabilities rated{" "}
                <strong>HIGH</strong> or <strong>CRITICAL</strong>. govulncheck blocks the
                Go check when our code can call a function with a known vulnerability. The{" "}
                <code>go mod verify</code> command checks downloaded Go modules against
                their recorded checksums.
              </p>

              <p>
                Before release and development-image publication,{" "}
                <code>mise run verify-deps</code> checks fixed versions of tools, images and
                GitHub Actions. It also checks that related runtime versions agree and
                verifies hashes and executable permissions for database clients committed to
                the repository. These checks cover a defined list of local declarations and
                files. Package managers separately verify locked versions and the checksums
                of downloaded packages.
              </p>

              <p>
                The local dependency check does not certify the safety of third-party code
                or enforce a minimum publication age for every dependency. Release selection
                and review remain the maintainer&#x27;s responsibility.
              </p>

              <h3 id="container-and-ci-hardening">Container and CI security</h3>

              <ul>
                <li>
                  Trivy scans the application image on amd64 and arm64.{" "}
                  <strong>CRITICAL</strong> vulnerabilities with an available fix fail the
                  check. <strong>HIGH</strong> vulnerabilities with an available fix are
                  reported without blocking it; vulnerabilities without a fix are excluded
                  from this check.
                </li>
                <li>
                  A separate Trivy configuration scan blocks <strong>HIGH</strong> and{" "}
                  <strong>CRITICAL</strong> findings in the application, verification-agent
                  and selected test Dockerfiles.
                </li>
                <li>
                  Third-party GitHub Actions are fixed to full commit SHAs. This prevents a
                  moved tag from silently changing the action code we run.
                </li>
                <li>
                  Workflows default to read-only repository access. Individual jobs receive
                  additional permissions only when needed.
                </li>
              </ul>

              <h3 id="testing-and-verification">Testing and verification</h3>

              <p>
                Unit and integration tests cover critical behavior. Before each release,
                full backup-and-restore tests run against real database containers for every
                supported engine and major version. They check that data can be restored,
                not just that a backup file was created.
              </p>

              <p>
                Release checks also include lint, type checks, browser tests, image smoke
                tests and native amd64 and arm64 builds and tests. Publication waits for all
                required jobs to pass, including the security checks described above.
              </p>

              <h3 id="reporting-a-vulnerability">Reporting a vulnerability</h3>

              <p>
                Report vulnerabilities privately using the contact options in{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . Please do not open a public issue. Security reports take priority.
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
