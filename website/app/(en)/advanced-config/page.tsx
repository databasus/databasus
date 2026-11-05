import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Advanced config - Databasus Documentation",
  description:
    "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, OpenTelemetry log export and a custom analytics script. Not needed for a default install.",
  keywords: [
    "Databasus environment variables",
    "Databasus advanced configuration",
    "self-hosted configuration",
    "GitHub OAuth",
    "Google OAuth",
    "SMTP email setup",
    "Cloudflare Turnstile",
    "Docker environment variables",
    "OpenTelemetry logs",
  ],
  openGraph: {
    title: "Advanced config - Databasus Documentation",
    description:
      "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, OpenTelemetry log export and a custom analytics script. Not needed for a default install.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/advanced-config",
  },
  twitter: {
    card: "summary",
    title: "Advanced config - Databasus Documentation",
    description:
      "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, OpenTelemetry log export and a custom analytics script. Not needed for a default install.",
  },
  alternates: {
    canonical: "https://databasus.com/advanced-config",
    languages: getLanguageAlternates("advanced-config"),
  },
  robots: "index, follow",
};

export default function AdvancedConfigPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Advanced config - Databasus Documentation",
            description:
              "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, OpenTelemetry log export and a custom analytics script. Not needed for a default install.",
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
              <h1 id="advanced-config">Advanced config</h1>

              <p className="text-lg text-gray-400">
                Databasus runs with sensible defaults out of the box — a
                standard single-container install needs no configuration at all.
                Every variable on this page is <strong>optional</strong> and not
                needed in 99% of production setups
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                By default Databasus uses email and password sign-in. You can
                additionally let people sign in with their Google or GitHub
                account. A provider&apos;s button appears as soon as its client
                ID is set, but sign-in only completes when <strong>both</strong>{" "}
                the client ID and the client secret are present.
              </p>

              <p>
                When you register the OAuth application, set its redirect
                (callback) URL to{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. Because
                of that redirect, OAuth sign-in needs your instance served over
                HTTPS on a public domain — see the note below.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    <strong>HTTPS is required for sign-in and email.</strong>{" "}
                    OAuth sign-in and email both need your instance reachable
                    over HTTPS on a public domain — OAuth providers redirect the
                    browser back to{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, and
                    links inside emails must open for whoever receives them. A
                    localhost-only or plain-HTTP instance cannot use these
                    features. The simplest way to get HTTPS is the{" "}
                    <a
                      href="/installation/#caddy-reverse-proxy"
                      className="text-blue-400 hover:text-blue-300"
                    >
                      Caddy reverse proxy
                    </a>{" "}
                    setup.
                  </p>
                </div>
              </div>

              <h3 id="oauth-google">Google</h3>

              <p>
                Create an OAuth client in the{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, application type <em>Web application</em>) and add{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code> as an
                authorized redirect URI.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID of your Google OAuth client. Setting it shows
                      the &quot;Sign in with Google&quot; button.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret of your Google OAuth client. Required
                      together with the ID for sign-in to work.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Create an OAuth app under{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  GitHub Developer settings
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) and
                set the authorization callback URL to{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID of your GitHub OAuth app. Setting it shows the
                      &quot;Sign in with GitHub&quot; button.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret of your GitHub OAuth app. Required together
                      with the ID for sign-in to work.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">Email (SMTP)</h2>

              <p>
                Connect a mail server so Databasus can send workspace invitations, password reset codes and sign-in codes. The mail server counts as configured <strong>as soon as <code>SMTP_HOST</code> is set</strong>. Without it, the settings screen reports the mail server as not configured, and the sign-in screen hides the password reset link.
              </p>

              <p>
                This is the instance mail server. Email notifiers, which report backup events, have SMTP settings of their own in each notifier&apos;s form, so a notifier that delivers mail tells you nothing about this server. To check the instance mail server, open <strong>Databasus settings → Mail server</strong> and press <strong>Send test email</strong>. The message goes to your own address, and a failed delivery shows the mail server&apos;s response.
              </p>

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
                    <strong>If your relay does not offer STARTTLS, set <code>SMTP_SECURITY=none</code> before upgrading.</strong> On any port other than 465 the default mode is <code>starttls</code>, which refuses to continue without encryption, so such a relay receives nothing until the variable is set. If the instance requires the emailed sign-in code, nobody gets in with a password while mail cannot be delivered. To get back in, switch the second factor off with <code>docker exec -it databasus ./main --disable-2fa</code> (see <a href="/password/#disable-two-factor" className="text-blue-400 hover:text-blue-300">Stop requiring a sign-in code</a>).
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Description">
                      Mail server host name (e.g. <code>smtp.gmail.com</code>). Setting it turns instance mail on.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Description">
                      Mail server port (e.g. <code>587</code>). Must be a positive integer when <code>SMTP_HOST</code> is set.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_SECURITY</code>
                    </td>
                    <td data-label="Description">
                      Connection security: <code>tls</code>, <code>starttls</code> or <code>none</code>, described below. Defaults to <code>tls</code> on port 465 and to <code>starttls</code> on every other port. The instance refuses to start with any other value.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Description">
                      Username for SMTP authentication.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Description">
                      Password for SMTP authentication. For Gmail, use an App Password, not your account password.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Description">
                      Sender address, either bare (<code>noreply@example.com</code>) or with a name (<code>Acme Backups &lt;noreply@example.com&gt;</code>). Without a name, recipients see <code>Databasus</code>. When it is not set, <code>SMTP_USER</code> is used if it is an email address, and otherwise <code>noreply@</code> followed by <code>SMTP_HOST</code>, so a username such as <code>apikey</code> never becomes the sender. The instance refuses to start when the value cannot be read as an address.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_HELO_NAME</code>
                    </td>
                    <td data-label="Description">
                      The name the instance greets the mail server with. Defaults to the host of <code>DATABASUS_URL</code>, then to the machine&apos;s host name. Must be a host name or an address literal such as <code>[192.0.2.1]</code> or <code>[IPv6:2001:db8::1]</code>, or the instance refuses to start.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Description">
                      Set to <code>true</code> to skip TLS certificate verification in the <code>tls</code> and <code>starttls</code> modes. Defaults to <code>false</code>. Use it only for servers with a self-signed certificate on a trusted network, because it disables protection against man-in-the-middle attacks.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Description">
                      Public base URL of your instance (e.g. <code>https://backup.example.com</code>). Optional: it adds a link to invitation emails and supplies the default greeting name. Mail works without it.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                <code>SMTP_SECURITY</code> selects one of three modes. Email notifiers offer the same choice under <strong>Advanced settings</strong>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Mode</th>
                    <th>Behavior</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>tls</code>
                    </td>
                    <td data-label="Behavior">
                      The connection is encrypted from the first byte. Usual on port 465.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>starttls</code>
                    </td>
                    <td data-label="Behavior">
                      The connection starts unencrypted and is upgraded before the password or any message is sent. If the server does not offer the upgrade, the delivery fails instead of continuing unencrypted. Usual on port 587.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>none</code>
                    </td>
                    <td data-label="Behavior">
                      The connection is never encrypted. Use it for a relay that does not offer STARTTLS, such as a local Postfix on port 25.
                    </td>
                  </tr>
                </tbody>
              </table>

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
                    <code>none</code> sends the SMTP password and every message unencrypted, and anyone between Databasus and the relay can read them. Use it only for a relay on the same host or on a trusted private network.
                  </p>
                </div>
              </div>

              <h2 id="signup-captcha">
                Sign up captcha (Cloudflare Turnstile)
              </h2>

              <p>
                If your instance is reachable from the public internet, you can
                put a{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                challenge on the sign-up and sign-in forms to keep bots out.
                Both keys come from the Turnstile dashboard, and the challenge
                activates only when both are set.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    To stop external sign-ups entirely rather than just
                    challenging them, you do not need a captcha at all — open{" "}
                    <strong>Databasus settings → Allow sign up</strong> in the
                    UI and turn it off. That closes the sign-up form completely.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Description">
                      Public Turnstile site key, used to render the widget in
                      the browser.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Description">
                      Secret Turnstile key, used by the backend to validate
                      challenge responses.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="docker-storage-permissions">Docker storage permissions</h2>

              <p>
                Databasus normally selects its numeric user and group IDs from an
                existing pgdata directory, then from the backups mount or data
                root, and finally uses <code>999</code>. Set <code>PUID</code> or{" "}
                <code>PGID</code> only when that automatic choice does not fit your
                bind mount, CIFS share, or NFS export. Values must be base-10
                integers from <code>1</code> through <code>4294967294</code>.
              </p>

              <table>
                <thead><tr><th>Variable</th><th>Automatic value</th><th>Account</th></tr></thead>
                <tbody>
                  <tr><td><code>PUID</code></td><td>Mounted owner or <code>999</code></td><td><code>databasus</code> user</td></tr>
                  <tr><td><code>PGID</code></td><td>Mounted group or <code>999</code></td><td><code>databasus</code> primary group</td></tr>
                </tbody>
              </table>

              <p>
                Databasus and PostgreSQL use the same non-root operating-system
                account named <code>databasus</code>.
              </p>

              <p>
                The entrypoint starts as root inside the container to select the
                IDs and attempt <code>chown</code> and <code>chmod</code>. It then
                checks the actual file lifecycle as <code>databasus</code>. A mount
                can work even if it rejects metadata changes. An arbitrary Docker{" "}
                <code>user:</code> override is not supported.
              </p>

              <pre><code>{`ERROR: Databasus cannot write to local storage paths /databasus-data/temp and /databasus-data/backups as UID 999 and GID 999.
Required operation: save a file through local storage.
Set PUID and PGID or fix the mounted directory permissions: https://databasus.com/advanced-config/#docker-storage-permissions
Details: permission denied`}</code></pre>

              <p>
                The previous four service-specific identity variables were
                deliberately removed. Remove them from your configuration when
                upgrading. If the automatic IDs cannot access a mount, startup
                stops with the failed operation and this documentation link.
              </p>

              <h2 id="telemetry">Telemetry</h2>

              <p>
                Databasus sends anonymous, non-identifying usage telemetry by
                default. It carries no personal data and helps us understand how
                the project is used. You can read exactly what is collected in
                the{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  privacy policy
                </a>
                , and you can turn it off completely.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Default</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Default">
                      <code>false</code>
                    </td>
                    <td data-label="Description">
                      Set to <code>true</code> to disable anonymous usage
                      telemetry.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">Logging</h2>

              <p>
                Databasus writes its logs to stdout and mirrors them as JSON to{" "}
                <code>databasus.log</code> on the data volume. Set{" "}
                <code>OPEN_TELEMETRY_URL</code> and it also exports them over
                Open Telemetry to a backend such as VictoriaLogs, Graylog, SigNoz, Grafana
                Loki, Datadog or Honeycomb, or to an OpenTelemetry Collector,
                which is itself an OTLP receiver.
              </p>

              <ul>
                <li>
                  <strong>Transport</strong> follows the scheme.{" "}
                  <code>http://</code> and <code>https://</code> send OTLP/HTTP
                  and use the URL verbatim, path included; <code>grpc://</code>{" "}
                  and <code>grpcs://</code> send OTLP/gRPC and use only the host
                  and port.
                </li>
                <li>
                  <strong>Authentication</strong> goes into{" "}
                  <code>OPEN_TELEMETRY_HEADERS</code> or into the URL as{" "}
                  <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Secrets</strong> (passwords, tokens, credentials)
                  inside URLs — are redacted before a record leaves the process.
                </li>
                <li>
                  <strong>Audit entries</strong> ship with the application logs
                  tagged <code>log_type=audit</code> and ignore{" "}
                  <code>LOG_LEVEL</code>, so raising the level never drops the
                  audit trail.
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Default</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      Full OTLP endpoint URL, including the path. Leave unset to
                      keep logs in the container. A query string, a missing host
                      or an unknown scheme stops the container at startup
                      instead of exporting nowhere.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      Comma-separated <code>key=value</code> pairs sent with
                      every export, usually an API key. Values are
                      percent-decoded, matching the standard{" "}
                      <code>OTEL_EXPORTER_OTLP_HEADERS</code> format.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="Default">
                      <code>info</code>
                    </td>
                    <td data-label="Description">
                      One of <code>debug</code>, <code>info</code>,{" "}
                      <code>warn</code> or <code>error</code>. An unrecognised
                      value falls back to <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="Default">
                      <code>true</code>
                    </td>
                    <td data-label="Description">
                      Writes <code>databasus.log</code> next to the rest of the
                      data, rotating at 5 MB and keeping 3 older files. Set to{" "}
                      <code>false</code> if your platform already collects
                      stdout.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Values for common backends, each with the header that
                authenticates it. Replace hosts, regions and keys with your own:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Backend</th>
                    <th>
                      <code>OPEN_TELEMETRY_URL</code>
                    </th>
                    <th>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>VictoriaLogs</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        http://victoria-logs:9428/insert/opentelemetry/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code> —
                      the credentials your <code>vmauth</code> or reverse proxy
                      expects, since VictoriaLogs itself has no auth on the
                      ingest path.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> — matches a{" "}
                      <code>bearertokenauth</code> or{" "}
                      <code>basicauth</code> extension on the receiver. A
                      Collector reachable only inside your network usually needs
                      none.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> — the token
                      set on the OpenTelemetry (gRPC) input. The input also
                      accepts mTLS instead.
                    </td>
                  </tr>
                  <tr>
                    <td>SigNoz Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpcs://ingest.eu.signoz.cloud:443</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>signoz-ingestion-key=your-ingestion-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Grafana Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        https://otlp-gateway-prod-eu-west-0.grafana.net/otlp/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20&lt;base64&gt;</code> — base64
                      of <code>instance-id:api-token</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Honeycomb</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>https://api.honeycomb.io/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>x-honeycomb-team=your-api-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Datadog Agent</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://datadog-agent:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      None — the Agent holds the API key and forwards on your
                      behalf.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Header values are percent-decoded, so the space after{" "}
                <code>Basic</code> or <code>Bearer</code> is written as{" "}
                <code>%20</code> and a comma inside a value as <code>%2C</code>.
                Basic auth can also go straight into the URL as{" "}
                <code>https://user:password@host/path</code> — Databasus turns
                it into the same header and keeps it out of the logs. Over{" "}
                <code>http://</code> and <code>grpc://</code> keys and passwords
                travel in clear, so use <code>https://</code> or{" "}
                <code>grpcs://</code> outside a trusted network.
              </p>

              <h2 id="analytics-script">Analytics script</h2>

              <p>
                Databasus can inject your own analytics or tracking snippet —
                Google Analytics, Plausible, Umami and similar into the app.
                When <code>ANALYTICS_SCRIPT</code> is set, its value is inserted
                into the page <code>&lt;head&gt;</code> at startup.
              </p>

              <p>
                <strong>Security warning:</strong> the value is injected
                verbatim as raw HTML and JavaScript and runs with full access to
                the Databasus UI in every visitor&apos;s browser. Only ever set
                it to a snippet you fully control and trust.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Description">
                      Custom <code>&lt;script&gt;</code> markup injected before
                      the closing <code>&lt;/head&gt;</code> tag. Leave unset to
                      add no analytics.
                    </td>
                  </tr>
                </tbody>
              </table>

            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
