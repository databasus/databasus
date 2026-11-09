import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Armazenamentos - Documentação do Databasus",
  description:
    "Lista de destinos de armazenamento suportados para backups do Databasus, incluindo armazenamento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  keywords: [
    "armazenamentos Databasus",
    "armazenamento de backups",
    "armazenamento S3",
    "Cloudflare R2",
    "backup em NAS",
    "armazenamento local",
    "Azure Blob Storage",
    "backup por FTP",
    "backup por SFTP",
  ],
  openGraph: {
    title: "Armazenamentos - Documentação do Databasus",
    description:
      "Lista de destinos de armazenamento suportados para backups do Databasus, incluindo armazenamento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
    type: "article",
    url: getLocalizedUrl("pt", "storages"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Armazenamentos - Documentação do Databasus",
    description:
      "Lista de destinos de armazenamento suportados para backups do Databasus, incluindo armazenamento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "storages"),
    languages: getLanguageAlternates("storages"),
  },
  robots: "index, follow",
};

export default function StoragesPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Armazenamentos - Documentação do Databasus",
            description:
              "Lista de destinos de armazenamento suportados para backups do Databasus, incluindo armazenamento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
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

      <DocsNavbarComponent lang="pt" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="pt" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="storages">Armazenamentos</h1>

              <p className="text-lg text-gray-400">
                O Databasus suporta vários destinos de armazenamento para os
                seus backups PostgreSQL. Escolha onde salvar as suas cópias de
                segurança de acordo com a sua infraestrutura e requisitos.
              </p>

              <h2 id="supported-storages">Armazenamentos suportados</h2>

              <ul>
                <li>
                  <strong>Armazenamento local</strong> - Salve os backups
                  diretamente no seu servidor ou VPS
                </li>
                <li>
                  <strong>S3</strong> - Amazon S3 e serviços de armazenamento
                  compatíveis com S3
                </li>
                <li>
                  <a
                    href="/pt/storages/cloudflare-r2"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Cloudflare R2
                  </a>{" "}
                  - Armazenamento de objetos compatível com S3 da Cloudflare
                </li>
                <li>
                  <strong>Azure Blob Storage</strong> - Armazenamento na nuvem
                  do Microsoft Azure
                </li>
                <li>
                  <strong>NAS</strong> - Dispositivos de armazenamento
                  conectados à rede
                </li>
                <li>
                  <strong>FTP</strong> - Servidores File Transfer Protocol
                </li>
                <li>
                  <strong>SFTP</strong> - Servidores SSH File Transfer Protocol
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
