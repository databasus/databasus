import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "How to backup localhost databases | Databasus",
  description:
    "Learn how to backup PostgreSQL databases running on localhost using Databasus. Configure Docker host network mode for local database backups.",
  keywords: [
    "Databasus",
    "localhost backup",
    "local PostgreSQL backup",
    "backup local database",
    "Docker host network",
    "PostgreSQL backup",
    "database backup",
    "localhost database",
  ],
  openGraph: {
    title: "How to backup localhost databases | Databasus",
    description:
      "Learn how to backup PostgreSQL databases running on localhost using Databasus. Configure Docker host network mode for local database backups.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/faq/localhost",
  },
  twitter: {
    card: "summary",
    title: "How to backup localhost databases | Databasus",
    description:
      "Learn how to backup PostgreSQL databases running on localhost using Databasus. Configure Docker host network mode for local database backups.",
  },
  alternates: {
    canonical: "https://databasus.com/faq/localhost",
    languages: getLanguageAlternates("faq/localhost"),
  },
  robots: "index, follow",
};

export default function LocalhostPage() {
  const dockerComposeHost = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    network_mode: host
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`;

  const dockerRunHost = `docker run -d \\
  --name databasus \\
  --network host \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to backup localhost databases with Databasus",
            description:
              "Step-by-step guide to backup PostgreSQL databases running on localhost using Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Configure Docker host network mode",
                text: "Update your Docker configuration to use host network mode so the container can access services on localhost.",
              },
              {
                "@type": "HowToStep",
                name: "Use Docker Compose or Docker run",
                text: "Apply the network_mode: host setting in Docker Compose or use --network host flag with Docker run.",
              },
              {
                "@type": "HowToStep",
                name: "Connect to localhost database",
                text: "Use 127.0.0.1 or localhost as the database host in your Databasus backup configuration.",
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
              <h1 id="localhost-backup">How to backup localhost databases</h1>

              <p className="text-lg text-gray-400">
                Learn how to configure Databasus to backup PostgreSQL databases
                running on your host machine (localhost) when using Docker.
              </p>

              <h2 id="the-problem">The problem</h2>

              <p>
                If you&apos;re running Databasus in Docker and want to back up
                databases running on your host machine (localhost), you need to
                configure Docker to use <strong>host network mode</strong>.
              </p>

              <p>
                By default, Docker containers run in an isolated network and
                cannot access services on <code>localhost</code>. The host
                network mode allows the container to share the host&apos;s
                network namespace.
              </p>

              <h2 id="docker-compose-solution">Solution for Docker Compose</h2>

              <p>
                Update your <code>docker-compose.yml</code> file to use{" "}
                <code>network_mode: host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerComposeHost} />
                </div>
              </div>

              <h2 id="docker-run-solution">Solution for Docker run</h2>

              <p>
                Use the <code>--network host</code> flag:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerRunHost} />
                </div>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Note:</strong> When
                  using host network mode, you can connect to your localhost
                  database using{" "}
                  <code className="bg-[#374151] text-gray-200">127.0.0.1</code>{" "}
                  or{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  as the host in your Databasus backup configuration.
                  You&apos;ll also access the Databasus UI directly at{" "}
                  <code className="bg-[#374151] text-gray-200">
                    http://localhost:4005
                  </code>{" "}
                  without port mapping.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ⚠️ Important for Windows and macOS users:
                  </strong>{" "}
                  The <code className="bg-[#374151] text-red-400">host</code>{" "}
                  network mode only works natively on Linux. On Windows and
                  macOS, Docker runs inside a Linux VM, so{" "}
                  <code className="bg-[#374151] text-gray-200">
                    host.docker.internal
                  </code>{" "}
                  should be used instead of{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  as the database host address in your backup configuration.
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/faq"
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
