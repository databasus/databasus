import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "安装 - Databasus 文档",
  description:
    "了解如何安装 Databasus：自动化脚本、Docker run、Docker Compose、面向 Kubernetes 的 Helm，或使用 Caddy 反向代理。为你的自托管 PostgreSQL 备份系统提供零配置的简单安装。",
  keywords: [
    "Databasus 安装",
    "Docker 安装",
    "PostgreSQL 备份配置",
    "自托管备份",
    "Docker Compose",
    "数据库备份安装",
    "pg_dump 配置",
    "Kubernetes",
    "Helm chart",
    "K8s 部署",
    "Caddy 反向代理",
    "HTTPS 配置",
    "健康检查",
    "监控",
    "liveness probe",
  ],
  openGraph: {
    title: "安装 - Databasus 文档",
    description:
      "了解如何安装 Databasus：自动化脚本、Docker run、Docker Compose、面向 Kubernetes 的 Helm，或使用 Caddy 反向代理。为你的自托管 PostgreSQL 备份系统提供零配置的简单安装。",
    type: "article",
    url: getLocalizedUrl("zh", "installation"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "安装 - Databasus 文档",
    description:
      "了解如何安装 Databasus：自动化脚本、Docker run、Docker Compose、面向 Kubernetes 的 Helm，或使用 Caddy 反向代理。为你的自托管 PostgreSQL 备份系统提供零配置的简单安装。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "installation"),
    languages: getLanguageAlternates("installation"),
  },
  robots: "index, follow",
};

export default function InstallationPage() {
  const installScript = `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh | sudo bash`;

  const dockerRun = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`;

  const dockerCompose = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`;

  const helmInstallClusterIP = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace`;

  const helmPortForward = `kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005`;

  const helmInstallLoadBalancer = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set service.type=LoadBalancer`;

  const helmGetSvc = `kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005`;

  const helmInstallIngress = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set ingress.enabled=true \\
  --set ingress.hosts[0].host=backup.example.com`;

  const helmUpgrade = `helm upgrade databasus oci://ghcr.io/databasus/charts/databasus -n databasus`;

  const dockerComposeCaddy = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
    # No port exposed - Caddy handles external access

  caddy:
    container_name: caddy
    image: caddy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - ./caddy-data:/data
      - ./caddy-config:/config
    restart: unless-stopped
    depends_on:
      - databasus`;

  const caddyfile = `backup.example.com {
    reverse_proxy databasus:4005
}`;

  const healthEndpoint = `GET http://<host>:4005/api/v1/system/health`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "安装 - Databasus 文档",
            description:
              "了解如何安装 Databasus：自动化脚本、Docker run、Docker Compose、面向 Kubernetes 的 Helm，或使用 Caddy 反向代理。为你的自托管 PostgreSQL 备份系统提供零配置的简单安装。",
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
            name: "如何安装 Databasus",
            description: "Databasus PostgreSQL 备份工具的分步安装指南",
            step: [
              {
                "@type": "HowToStep",
                name: "自动化安装脚本",
                text: "运行自动化安装脚本：它会安装 Docker 并配置 Databasus 开机自启。",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "执行 curl 命令下载并运行安装脚本",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Docker Run",
                text: "使用 docker run 命令快速启动 Databasus 容器并持久化数据。",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "创建 docker-compose.yml 文件，通过 Docker Compose 管理部署。",
              },
              {
                "@type": "HowToStep",
                name: "Kubernetes 与 Helm",
                text: "使用官方 Helm chart 在 Kubernetes 上部署 Databasus：StatefulSet、持久化存储和可选的 ingress。",
              },
              {
                "@type": "HowToStep",
                name: "使用 Caddy 反向代理运行",
                text: "生产环境使用 Docker Compose 搭配 Caddy，自动获取 HTTPS 证书。",
              },
            ],
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
              <h1 id="installation">安装</h1>

              <p className="text-lg text-gray-400">
                你可以通过多种方式安装 Databasus：自动化脚本（推荐）、简单的
                docker run、Docker Compose、面向 Kubernetes 的 Helm，或在生产
                环境使用 Docker Compose 搭配 Caddy。
              </p>

              <h2 id="system-requirements">系统要求</h2>

              <p>Databasus 正常运行所需的最低系统资源：</p>

              <ul>
                <li>
                  <strong>CPU</strong>：至少 1 个 CPU 核心
                </li>
                <li>
                  <strong>RAM</strong>：至少 500 MB 内存
                </li>
                <li>
                  <strong>存储</strong>：安装占用 5 GB，另加备份所需的空间
                </li>
                <li>
                  <strong>Docker</strong>：Docker Engine 20.10+ 和 Docker
                  Compose v2.0+
                </li>
              </ul>

              <h2 id="option-1-automated-script">
                方式 1：安装脚本（推荐，仅限 Linux）
              </h2>

              <p>安装脚本会：</p>

              <ul>
                <li>✅ 安装 Docker 和 Docker Compose（如果尚未安装）</li>
                <li>✅ 配置 Databasus</li>
                <li>✅ 设置系统重启后自动启动</li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{installScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={installScript} />
                </div>
              </div>

              <p>
                这种情况下，Databasus 会安装到 <code>/opt/databasus</code>{" "}
                目录。
              </p>

              <h2 id="option-2-docker-run">方式 2：简单的 docker run</h2>

              <p>运行 Databasus 最简单的方法：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRun}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerRun} />
                </div>
              </div>

              <p>这一条命令会：</p>

              <ul>
                <li>✅ 启动 Databasus</li>
                <li>
                  ✅ 把所有数据保存在 <code>./databasus-data</code> 目录中
                </li>
                <li>✅ 系统重启后自动重启容器</li>
              </ul>

              <h2 id="option-3-docker-compose">方式 3：Docker Compose 部署</h2>

              <p>
                创建 <code>docker-compose.yml</code> 文件，写入以下配置：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerCompose}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerCompose} />
                </div>
              </div>

              <p>然后运行：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text="docker compose up -d" />
                </div>
              </div>

              <p>请注意，启动可能需要最多约 2 分钟。</p>

              <h2 id="option-4-helm">方式 4：Kubernetes 与 Helm</h2>

              <p>
                在 Kubernetes 上部署时，可直接从 OCI 仓库安装。根据你的环境
                选择合适的访问方式。
              </p>

              <h3 id="helm-clusterip">ClusterIP + port-forward（开发环境）</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallClusterIP}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={helmInstallClusterIP} />
                </div>
              </div>

              <p>通过 port-forward 访问：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmPortForward}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={helmPortForward} />
                </div>
              </div>

              <h3 id="helm-loadbalancer">LoadBalancer（云环境）</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallLoadBalancer}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={helmInstallLoadBalancer} />
                </div>
              </div>

              <p>获取外部 IP 并访问 Databasus：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmGetSvc}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={helmGetSvc} />
                </div>
              </div>

              <h3 id="helm-ingress">Ingress（通过域名访问）</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallIngress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={helmInstallIngress} />
                </div>
              </div>

              <p>
                更多选项（NodePort、TLS、Gateway API 的 HTTPRoute）请参阅{" "}
                <a
                  href="https://github.com/databasus/databasus/tree/main/deploy/helm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Helm chart 文档
                </a>
                。
              </p>

              <h2 id="caddy-reverse-proxy">使用 Caddy 反向代理运行</h2>

              <p>
                生产环境可以使用{" "}
                <a
                  href="https://caddyserver.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Caddy
                </a>{" "}
                作为反向代理，自动获取 HTTPS 证书并安全地访问 Databasus。
                下面是搭配 Caddy 的完整 Docker Compose 配置。
              </p>

              <h3 id="caddy-docker-compose">Docker Compose 搭配 Caddy</h3>

              <p>
                创建 <code>docker-compose.yml</code> 文件：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeCaddy}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerComposeCaddy} />
                </div>
              </div>

              <p>
                在同一目录下创建 <code>Caddyfile</code>：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{caddyfile}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={caddyfile} />
                </div>
              </div>

              <p>然后启动服务：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text="docker compose up -d" />
                </div>
              </div>

              <p>这套配置提供：</p>

              <ul>
                <li>✅ 使用 Let&apos;s Encrypt 证书自动启用 HTTPS</li>
                <li>✅ HTTP 自动重定向到 HTTPS</li>
                <li>✅ 反向代理到 Databasus</li>
                <li>✅ Caddy 和 Databasus 的数据都持久化保存</li>
              </ul>

              <p>
                把 <code>backup.example.com</code> 替换成你的实际域名。启动
                服务前，请确认域名的 DNS 已指向服务器的 IP 地址。
              </p>

              <h2 id="getting-started">快速上手</h2>

              <p>安装完成后：</p>

              <ol>
                <li>
                  <strong>启动并打开 Databasus</strong>：启动 Databasus 并访问{" "}
                  <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>创建第一个备份任务</strong>：点击 &quot;New
                  Backup&quot;，配置 PostgreSQL 数据库连接
                </li>
                <li>
                  <strong>配置备份计划</strong>：设置备份周期（每小时、每天、
                  每周、每月或 cron）
                </li>
                <li>
                  <strong>选择存储位置</strong>：指定备份保存的位置（本地、
                  S3、Google Drive 等）
                </li>
                <li>
                  <strong>设置通知</strong>：添加通知渠道（Slack、Telegram、
                  Discord），随时了解备份状态
                </li>
                <li>
                  <strong>开始备份</strong>：保存配置，看着第一个备份跑起来！
                </li>
              </ol>

              <h2 id="health-checks">健康检查</h2>

              <h3 id="docker-health-check">Docker 健康检查</h3>

              <p>
                <code>docker run</code> 和 Docker Compose
                会自动启用内置的健康检查。当 Databasus
                开始响应请求后（经过短暂的启动宽限期），容器会被标记为{" "}
                <code>healthy</code>。健康检查只确认应用是否响应，因此磁盘
                空间不足这类非致命状况不会触发容器重启。
              </p>

              <h3 id="monitoring-endpoint">监控 / 状态端点</h3>

              <p>用于可用性监控和状态面板：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{healthEndpoint}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={healthEndpoint} />
                </div>
              </div>

              <p>
                一切正常时返回 <code>200</code>；有情况需要关注时返回{" "}
                <code>503</code> 并附带原因：内部数据库、磁盘使用率（超过
                95%）、数据库客户端工具、后台任务以及验证代理的存活状态。
                无需认证，CORS 开放，方便浏览器端的监控工具使用。
              </p>

              <p>
                <strong>⚠️ 重要：</strong>该端点仅用于监控和告警，不要用作
                容器或 Kubernetes 的 liveness probe。在「性能下降但仍在服务」
                的状态下（例如磁盘快满了）它也会返回 <code>503</code>，
                这会导致正常运行的容器被重启。
              </p>

              <h3 id="kubernetes-health-check">Kubernetes</h3>

              <p>
                liveness/readiness probe 请使用{" "}
                <code>databasus healthcheck</code> 命令；把{" "}
                <a
                  href="#monitoring-endpoint"
                  className="text-blue-400 hover:text-blue-300"
                >
                  /api/v1/system/health
                </a>{" "}
                端点留给外部监控。
              </p>

              <h2 id="how-to-update">如何更新 Databasus？</h2>

              <h3 id="update-docker">更新 Docker 安装</h3>

              <p>
                要更新通过 Docker 运行的 Databasus，需要先停止它、清理 Docker
                缓存，再重启容器。
              </p>

              <ol>
                <li>
                  进入 Databasus 的安装目录（通常是 <code>/opt/databasus</code>
                  ）
                </li>
                <li>
                  停止容器：<code>docker compose stop</code>
                </li>
                <li>
                  清理 Docker 缓存：<code>docker system prune -a</code>
                </li>
                <li>
                  重启容器：<code>docker compose up -d</code>
                </li>
              </ol>

              <p>
                Docker 会从 Docker Hub 拉取最新版本的 Databasus（前提是你没有 在{" "}
                <code>docker-compose.yml</code> 文件中固定版本）。
              </p>

              <h3 id="update-helm">更新 Helm 安装</h3>

              <p>
                要更新通过 Helm 部署在 Kubernetes 上的 Databasus，执行 upgrade
                命令：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmUpgrade}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={helmUpgrade} />
                </div>
              </div>

              <p>
                如果你有自定义配置，添加 <code>-f values.yaml</code> 或使用{" "}
                <code>--set</code> 参数来保留它们。Helm 会以滚动更新的方式
                升级到新版本。
              </p>

              <h2 id="postgresus-migration">从 Postgresus 迁移</h2>

              <p>
                Databasus 是 Postgresus 的新名字。如果你正在使用
                Postgresus，可以继续使用，也可以迁移到 Databasus。
              </p>

              <p>
                <strong>重要：</strong>只改 Docker 镜像名是不够的，因为
                Postgresus 和 Databasus 使用不同的数据目录和不同的内部
                数据库命名。
              </p>

              <p>迁移步骤：</p>

              <ol>
                <li>
                  停止 Postgresus 容器：<code>docker compose stop</code>
                </li>
                <li>
                  用上面任意一种方式安装 Databasus（使用不同的卷路径{" "}
                  <code>./databasus-data</code>）
                </li>
                <li>在 Databasus 中手动重建数据库、存储和通知渠道</li>
              </ol>

              <p>
                迁移期间，可以用不同的端口和卷路径让 Postgresus 和 Databasus
                并行运行。
              </p>

              <h2 id="troubleshooting">故障排查</h2>

              <h3 id="container-wont-start">容器无法启动</h3>

              <p>如果容器启动失败，查看日志：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs databasus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text="docker logs databasus" />
                </div>
              </div>

              <h3 id="port-already-in-use">端口已被占用</h3>

              <p>如果 4005 端口被占用，可以在 docker-compose.yml 中修改：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    ports:
                    {"\n  "}- &quot;8080:4005&quot; # Change 8080 to any
                    available port
                  </code>
                </pre>
              </div>

              <h3 id="permission-denied">权限错误</h3>

              <p>如果数据目录出现权限问题：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    sudo chown -R $USER:$USER ./databasus-data
                    {"\n"}
                    chmod -R 755 ./databasus-data
                  </code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton
                    lang="zh"
                    text={`sudo chown -R $USER:$USER ./databasus-data\nchmod -R 755 ./databasus-data`}
                  />
                </div>
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
