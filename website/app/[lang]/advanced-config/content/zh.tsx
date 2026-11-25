import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "高级配置 - Databasus 文档",
  description:
    "自托管 Databasus 的可选环境变量：Google 和 GitHub 登录、SMTP 邮件、Cloudflare Turnstile 验证码、遥测、OpenTelemetry 日志导出和自定义分析脚本。默认安装无需配置。",
  keywords: [
    "Databasus 环境变量",
    "Databasus 高级配置",
    "自托管配置",
    "GitHub OAuth",
    "Google OAuth",
    "SMTP 邮件配置",
    "Cloudflare Turnstile",
    "Docker 环境变量",
    "OpenTelemetry 日志",
  ],
  openGraph: {
    title: "高级配置 - Databasus 文档",
    description:
      "自托管 Databasus 的可选环境变量：Google 和 GitHub 登录、SMTP 邮件、Cloudflare Turnstile 验证码、遥测、OpenTelemetry 日志导出和自定义分析脚本。默认安装无需配置。",
    type: "article",
    url: getLocalizedUrl("zh", "advanced-config"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "高级配置 - Databasus 文档",
    description:
      "自托管 Databasus 的可选环境变量：Google 和 GitHub 登录、SMTP 邮件、Cloudflare Turnstile 验证码、遥测、OpenTelemetry 日志导出和自定义分析脚本。默认安装无需配置。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "advanced-config"),
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
            headline: "高级配置 - Databasus 文档",
            description:
              "自托管 Databasus 的可选环境变量：Google 和 GitHub 登录、SMTP 邮件、Cloudflare Turnstile 验证码、遥测、OpenTelemetry 日志导出和自定义分析脚本。默认安装无需配置。",
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

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="advanced-config">高级配置</h1>

              <p className="text-lg text-gray-400">
                Databasus
                开箱即用，默认设置就很合理：标准的单容器安装完全不需要任何配置。本页的每个变量都是
                <strong>可选的</strong>，99% 的生产环境用不到。
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                Databasus 默认使用邮箱加密码登录。你还可以额外允许用户用 Google
                或 GitHub 账号登录。只要设置了某个提供商的 client
                ID，对应的按钮就会显示，但只有 client ID 和 client secret{" "}
                <strong>都</strong>存在时，登录才能完成。
              </p>

              <p>
                注册 OAuth 应用时，把重定向（回调）URL 设置为{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>
                。由于这个重定向，OAuth 登录要求你的实例通过公网域名以 HTTPS
                提供服务，见下面的说明。
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
                    <strong>登录和邮件功能需要 HTTPS。</strong> OAuth
                    登录和邮件都要求你的实例能通过公网域名以 HTTPS 访问：OAuth
                    提供商会把浏览器重定向回{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>
                    ，而邮件里的链接必须对收件人可打开。只在 localhost 或纯 HTTP
                    上运行的实例无法使用这些功能。获得 HTTPS 最简单的方式是{" "}
                    <a
                      href="/zh/installation/#caddy-reverse-proxy"
                      className="text-blue-400 hover:text-blue-300"
                    >
                      Caddy 反向代理
                    </a>{" "}
                    方案。
                  </p>
                </div>
              </div>

              <h3 id="oauth-google">Google</h3>

              <p>
                在{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                中创建 OAuth client（APIs &amp; Services → Credentials → Create
                credentials → OAuth client ID，应用类型选{" "}
                <em>Web application</em>），并把{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>{" "}
                添加为授权重定向 URI。
              </p>

              <table>
                <thead>
                  <tr>
                    <th>变量</th>
                    <th>说明</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      你的 Google OAuth client 的 client
                      ID。设置后会显示&quot;Sign in with Google&quot;按钮。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      你的 Google OAuth client 的 client secret。必须与 ID
                      一起设置，登录才能生效。
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                在{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  GitHub Developer settings
                </a>{" "}
                中创建 OAuth 应用（Settings → Developer settings → OAuth Apps →
                New OAuth App），把 authorization callback URL 设置为{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>。
              </p>

              <table>
                <thead>
                  <tr>
                    <th>变量</th>
                    <th>说明</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      你的 GitHub OAuth 应用的 client ID。设置后会显示&quot;Sign
                      in with GitHub&quot;按钮。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      你的 GitHub OAuth 应用的 client secret。必须与 ID
                      一起设置，登录才能生效。
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">邮件 (SMTP)</h2>

              <p>
                连接邮件服务器后，Databasus 就能发送工作区邀请、密码重置码和登录验证码。<strong>只要设置了 <code>SMTP_HOST</code></strong>，邮件服务器就算已配置。没有它时，设置页面会显示邮件服务器未配置，登录页面也不会显示重置密码的链接。
              </p>

              <p>
                这是实例的邮件服务器。邮件通知渠道用来报告备份情况，它们在各自的表单里有自己的 SMTP 设置，所以某个通知渠道能收发邮件，并不能说明这台服务器可用。要检查实例的邮件服务器，请打开 <strong>Databasus settings → Mail server</strong> 并点击 <strong>Send test email</strong>。测试邮件会发到你自己的地址，发送失败时会显示邮件服务器的回复。
              </p>

              <table>
                <thead>
                  <tr>
                    <th>变量</th>
                    <th>说明</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Description">
                      邮件服务器主机名（例如 <code>smtp.gmail.com</code>）。设置后即启用实例邮件。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Description">
                      邮件服务器端口（例如 <code>587</code>）。设置了 <code>SMTP_HOST</code> 时必须是正整数。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_SECURITY</code>
                    </td>
                    <td data-label="Description">
                      连接安全：<code>tls</code>、<code>starttls</code> 或 <code>none</code>，说明见下文。465 端口默认 <code>tls</code>，其他端口默认 <code>starttls</code>。取其他值时实例拒绝启动。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Description">
                      SMTP 认证用户名。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Description">
                      SMTP 认证密码。Gmail 请使用 App Password，而不是账号密码。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Description">
                      发件人地址，可以只写地址（<code>noreply@example.com</code>），也可以带名称（<code>Acme Backups &lt;noreply@example.com&gt;</code>）。不带名称时，收件人看到的是 <code>Databasus</code>。未设置时，如果 <code>SMTP_USER</code> 是邮箱地址就使用它，否则使用 <code>noreply@</code> 加 <code>SMTP_HOST</code>，因此 <code>apikey</code> 这样的用户名不会成为发件人。值无法解析为地址时，实例拒绝启动。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_HELO_NAME</code>
                    </td>
                    <td data-label="Description">
                      实例向邮件服务器问候时使用的名称。默认取 <code>DATABASUS_URL</code> 中的主机，没有时取本机主机名。必须是主机名，或 <code>[192.0.2.1]</code>、<code>[IPv6:2001:db8::1]</code> 这样的地址字面量，否则实例拒绝启动。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Description">
                      设为 <code>true</code> 时，在 <code>tls</code> 和 <code>starttls</code> 模式下跳过 TLS 证书校验。默认 <code>false</code>。只在可信网络内的自签名证书服务器上使用：它会关闭对中间人攻击的防护。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Description">
                      实例的公开基础 URL（例如 <code>https://backup.example.com</code>）。可选：它会在邀请邮件里加上链接，并提供默认的问候名称。没有它邮件也能正常发送。
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                <code>SMTP_SECURITY</code> 从三种模式中选一种。邮件通知渠道在 <strong>Advanced settings</strong> 里提供同样的选择。
              </p>

              <table>
                <thead>
                  <tr>
                    <th>模式</th>
                    <th>行为</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>tls</code>
                    </td>
                    <td data-label="Description">
                      从第一个字节开始加密连接。通常用于 465 端口。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>starttls</code>
                    </td>
                    <td data-label="Description">
                      连接先以明文建立，在发送密码或任何邮件之前升级为加密连接。服务器不支持时发送失败，而不是继续以明文发送。通常用于 587 端口。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>none</code>
                    </td>
                    <td data-label="Description">
                      连接始终不加密。适用于不支持 STARTTLS 的中继，例如 25 端口上的本地 Postfix。
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
                    <code>none</code> 会以明文发送 SMTP 密码和所有邮件，Databasus 与中继之间的任何人都能读取。只应在同一主机或可信私有网络中的中继上使用。
                  </p>
                </div>
              </div>

              <h2 id="signup-captcha">注册验证码（Cloudflare Turnstile）</h2>

              <p>
                如果你的实例能从公网访问，可以在注册和登录表单上加一层{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                挑战来挡机器人。两个密钥都来自 Turnstile
                控制台，只有两者都设置了，挑战才会生效。
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
                    如果你想彻底禁止外部注册，而不只是加一道挑战，那根本不需要验证码：在界面中打开{" "}
                    <strong>Databasus settings → Allow sign up</strong>{" "}
                    并把它关掉。这会完全关闭注册表单。
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>变量</th>
                    <th>说明</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Description">
                      公开的 Turnstile site key，用于在浏览器中渲染小组件。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Description">
                      Turnstile 私密密钥，后端用它校验挑战响应。
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="docker-storage-permissions">Docker 存储权限</h2>

              <p>
                Databasus 通常先使用现有 pgdata 目录的用户和组 ID，再检查备份 mount 或数据根目录，最后使用{" "}
                <code>999</code>。只有自动选择不适合你的 bind mount、CIFS 共享或 NFS 导出时，才需要设置{" "}
                <code>PUID</code> 或 <code>PGID</code>。值必须是从 <code>1</code> 到{" "}
                <code>4294967294</code> 的十进制整数。
              </p>

              <table>
                <thead><tr><th>变量</th><th>自动值</th><th>账户</th></tr></thead>
                <tbody>
                  <tr><td><code>PUID</code></td><td>mount 所有者或 <code>999</code></td><td><code>databasus</code> 用户</td></tr>
                  <tr><td><code>PGID</code></td><td>mount 组或 <code>999</code></td><td><code>databasus</code> 主组</td></tr>
                </tbody>
              </table>

              <p>
                Databasus 和 PostgreSQL 使用同一个名为 <code>databasus</code> 的无 root 权限操作系统账户。
              </p>

              <p>
                Entrypoint 先以容器内的 root 身份启动，用于选择 ID 并尝试执行 <code>chown</code> 和{" "}
                <code>chmod</code>。随后，它会以 <code>databasus</code> 身份检查完整的文件生命周期。即使 mount
                拒绝更改所有者或模式，只要实际文件操作成功，容器仍可运行。不支持任意 Docker <code>user:</code> 覆盖。
              </p>

              <pre><code>{`ERROR: Databasus cannot write to local storage paths /databasus-data/temp and /databasus-data/backups as UID 999 and GID 999.
Required operation: save a file through local storage.
Set PUID and PGID or fix the mounted directory permissions: https://databasus.com/advanced-config/#docker-storage-permissions
Details: permission denied`}</code></pre>

              <p>
                之前用于不同服务的四个变量已被有意删除。升级时请从配置中移除这些变量。如果自动选择的 ID
                无法访问 mount，启动会停止，并显示失败的操作以及本文档链接。
              </p>

              <h2 id="telemetry">遥测</h2>

              <p>
                Databasus
                默认发送匿名、无法识别身份的使用遥测。它不包含任何个人数据，帮助我们了解项目的使用情况。具体收集哪些内容可以在
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  隐私政策
                </a>
                中查看，你也可以完全关闭它。
              </p>

              <table>
                <thead>
                  <tr>
                    <th>变量</th>
                    <th>默认值</th>
                    <th>说明</th>
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
                      设为 <code>true</code> 可禁用匿名使用遥测。
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">日志</h2>

              <p>
                Databasus 把日志写到 stdout，并以 JSON 形式镜像到数据卷上的{" "}
                <code>databasus.log</code>。设置 <code>OPEN_TELEMETRY_URL</code>{" "}
                后，日志还会通过 Open Telemetry 导出到
                VictoriaLogs、Graylog、SigNoz、Grafana Loki、Datadog、Honeycomb
                等后端，或导出到 OpenTelemetry Collector（它本身就是一个 OTLP
                接收端）。
              </p>

              <ul>
                <li>
                  <strong>传输方式</strong>由 URL 协议决定。{" "}
                  <code>http://</code> 和 <code>https://</code> 发送
                  OTLP/HTTP，按原样使用整个 URL（包含路径）；{" "}
                  <code>grpc://</code> 和 <code>grpcs://</code> 发送
                  OTLP/gRPC，只使用主机和端口。
                </li>
                <li>
                  <strong>认证</strong>写在 <code>OPEN_TELEMETRY_HEADERS</code>{" "}
                  里，或以 <code>user:password@host</code> 的形式写进 URL。
                </li>
                <li>
                  <strong>敏感信息</strong>（密码、令牌、凭据）出现在 URL
                  中时，会在日志记录离开进程之前被脱敏。
                </li>
                <li>
                  <strong>审计条目</strong>随应用日志一起发送，带{" "}
                  <code>log_type=audit</code> 标签，且不受{" "}
                  <code>LOG_LEVEL</code>{" "}
                  影响，因此调高日志级别不会丢失审计记录。
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>变量</th>
                    <th>默认值</th>
                    <th>说明</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      完整的 OTLP 端点
                      URL，含路径。不设置则日志只留在容器内。URL
                      带查询字符串、缺少主机或使用未知协议时，容器会在启动时直接停止，而不是把日志导出到不存在的地方。
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      逗号分隔的 <code>key=value</code>{" "}
                      对，随每次导出发送，通常是 API
                      密钥。值会做百分号解码，与标准的{" "}
                      <code>OTEL_EXPORTER_OTLP_HEADERS</code> 格式一致。
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
                      取值为 <code>debug</code>、<code>info</code>、{" "}
                      <code>warn</code> 或 <code>error</code>
                      之一。无法识别的值回退到 <code>info</code>。
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
                      在数据目录旁写入 <code>databasus.log</code>，5 MB
                      轮转并保留 3 个旧文件。如果你的平台已经收集 stdout，可设为{" "}
                      <code>false</code>。
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                常见后端的取值，以及各自用于认证的
                header。把主机、区域和密钥替换成你自己的：
              </p>

              <table>
                <thead>
                  <tr>
                    <th>后端</th>
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
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code>{" "}
                      &mdash; 你的 <code>vmauth</code>{" "}
                      或反向代理所要求的凭据，因为 VictoriaLogs
                      本身在写入路径上没有认证。
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> &mdash;
                      对应接收端的 <code>bearertokenauth</code> 或{" "}
                      <code>basicauth</code> 扩展。只在内网可达的 Collector
                      通常不需要。
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> &mdash; 在
                      OpenTelemetry (gRPC) 输入上设置的令牌。该输入也接受 mTLS。
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
                      <code>Authorization=Basic%20&lt;base64&gt;</code> &mdash;{" "}
                      <code>instance-id:api-token</code> 的 base64 编码
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
                      无 &mdash; Agent 持有 API 密钥并代为转发。
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                header 值会做百分号解码，所以 <code>Basic</code> 或{" "}
                <code>Bearer</code> 后面的空格写作 <code>%20</code>
                ，值里的逗号写作 <code>%2C</code>。Basic 认证也可以直接放进
                URL：<code>https://user:password@host/path</code>，Databasus
                会把它转换成同样的 header 并使其不出现在日志中。通过{" "}
                <code>http://</code> 和 <code>grpc://</code>{" "}
                传输时密钥和密码是明文的，所以在可信网络之外请使用{" "}
                <code>https://</code> 或 <code>grpcs://</code>。
              </p>

              <h2 id="analytics-script">分析脚本</h2>

              <p>
                Databasus 可以把你自己的分析或跟踪代码片段注入应用，例如 Google
                Analytics、Plausible、Umami 等。设置了{" "}
                <code>ANALYTICS_SCRIPT</code> 后，它的值会在启动时插入到页面的{" "}
                <code>&lt;head&gt;</code> 中。
              </p>

              <p>
                <strong>安全警告：</strong>该值会被原样注入为 HTML 和
                JavaScript，并在每个访问者的浏览器中以对 Databasus
                界面的完全访问权限运行。只设置你完全掌控且信任的代码片段。
              </p>

              <table>
                <thead>
                  <tr>
                    <th>变量</th>
                    <th>说明</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Description">
                      自定义的 <code>&lt;script&gt;</code> 标记，插入到{" "}
                      <code>&lt;/head&gt;</code>{" "}
                      闭合标签之前。不设置则不添加任何分析代码。
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
