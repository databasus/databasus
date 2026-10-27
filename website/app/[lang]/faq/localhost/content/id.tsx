import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Cara backup database localhost | Databasus",
  description:
    "Pelajari cara backup database PostgreSQL yang berjalan di localhost dengan Databasus. Atur mode jaringan host Docker untuk backup database lokal.",
  keywords: [
    "Databasus",
    "backup localhost",
    "backup PostgreSQL lokal",
    "backup database lokal",
    "jaringan host Docker",
    "backup PostgreSQL",
    "backup database",
    "database localhost",
  ],
  openGraph: {
    title: "Cara backup database localhost | Databasus",
    description:
      "Pelajari cara backup database PostgreSQL yang berjalan di localhost dengan Databasus. Atur mode jaringan host Docker untuk backup database lokal.",
    type: "article",
    url: getLocalizedUrl("id", "faq/localhost"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara backup database localhost | Databasus",
    description:
      "Pelajari cara backup database PostgreSQL yang berjalan di localhost dengan Databasus. Atur mode jaringan host Docker untuk backup database lokal.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "faq/localhost"),
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara backup database localhost dengan Databasus",
            description:
              "Panduan langkah demi langkah untuk backup database PostgreSQL yang berjalan di localhost dengan Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Atur mode jaringan host Docker",
                text: "Ubah konfigurasi Docker agar memakai mode jaringan host sehingga container bisa mengakses layanan di localhost.",
              },
              {
                "@type": "HowToStep",
                name: "Gunakan Docker Compose atau Docker run",
                text: "Terapkan pengaturan network_mode: host di Docker Compose atau gunakan flag --network host dengan Docker run.",
              },
              {
                "@type": "HowToStep",
                name: "Hubungkan ke database localhost",
                text: "Gunakan 127.0.0.1 atau localhost sebagai host database dalam konfigurasi backup Databasus.",
              },
            ],
          }),
        }}
      />
      <DocsNavbarComponent lang="id" />
      <div className="flex min-h-screen bg-[#0F1115]">
        <DocsSidebarComponent lang="id" />
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="localhost-backup">Cara backup database localhost</h1>
              <p className="text-lg text-gray-400">
                Pelajari cara mengatur Databasus untuk backup database PostgreSQL yang berjalan di
                mesin host (localhost) saat memakai Docker.
              </p>
              <h2 id="the-problem">Masalahnya</h2>
              <p>
                Jika Databasus berjalan di Docker dan kamu ingin membuat backup database di mesin
                host (localhost), Docker perlu diatur agar memakai{" "}
                <strong>mode jaringan host</strong>.
              </p>
              <p>
                Secara default, container Docker berjalan di jaringan terisolasi dan tidak bisa
                mengakses layanan di <code>localhost</code>. Mode jaringan host memungkinkan
                container berbagi namespace jaringan dengan host.
              </p>
              <h2 id="docker-compose-solution">Solusi untuk Docker Compose</h2>
              <p>
                Ubah file <code>docker-compose.yml</code> agar memakai{" "}
                <code>network_mode: host</code>:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerComposeHost} lang="id" />
                </div>
              </div>
              <h2 id="docker-run-solution">Solusi untuk Docker run</h2>
              <p>
                Gunakan flag <code>--network host</code>:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerRunHost} lang="id" />
                </div>
              </div>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Catatan:</strong> Saat memakai mode jaringan
                  host, kamu bisa terhubung ke database localhost dengan memakai{" "}
                  <code className="bg-[#374151] text-gray-200">127.0.0.1</code> atau{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code> sebagai host dalam
                  konfigurasi backup Databasus. Antarmuka Databasus juga bisa diakses langsung di{" "}
                  <code className="bg-[#374151] text-gray-200">http://localhost:4005</code> tanpa
                  pemetaan port.
                </p>
              </div>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ⚠️ Penting bagi pengguna Windows dan macOS:
                  </strong>{" "}
                  Mode jaringan <code className="bg-[#374151] text-red-400">host</code> hanya
                  bekerja secara native di Linux. Di Windows dan macOS, Docker berjalan di dalam VM
                  Linux, jadi gunakan{" "}
                  <code className="bg-[#374151] text-gray-200">host.docker.internal</code>, bukan{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>, sebagai alamat host
                  database dalam konfigurasi backup.
                </p>
              </div>
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("id", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Kembali ke FAQ
                </a>
              </div>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
