import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Хранилища - документация Databasus",
  description:
    "Список поддерживаемых хранилищ для бекапов Databasus: локальное хранилище, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  keywords: [
    "хранилища Databasus",
    "хранилище бекапов",
    "хранилище S3",
    "Cloudflare R2",
    "бекап на NAS",
    "локальное хранилище",
    "Azure Blob Storage",
    "бекап на FTP",
    "бекап на SFTP",
  ],
  openGraph: {
    title: "Хранилища - документация Databasus",
    description:
      "Список поддерживаемых хранилищ для бекапов Databasus: локальное хранилище, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
    type: "article",
    url: getLocalizedUrl("ru", "storages"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Хранилища - документация Databasus",
    description:
      "Список поддерживаемых хранилищ для бекапов Databasus: локальное хранилище, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "storages"),
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
            headline: "Хранилища - документация Databasus",
            description:
              "Список поддерживаемых хранилищ для бекапов Databasus: локальное хранилище, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
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

      <DocsNavbarComponent lang="ru" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="ru" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="storages">Хранилища</h1>

              <p className="text-lg text-gray-400">
                Databasus поддерживает несколько типов хранилищ для бекапов
                PostgreSQL. Выберите, где хранить файлы, исходя из своей
                инфраструктуры и требований.
              </p>

              <h2 id="supported-storages">Поддерживаемые хранилища</h2>

              <ul>
                <li>
                  <strong>Local Storage</strong> - хранение бекапов прямо на
                  вашем сервере или VPS
                </li>
                <li>
                  <strong>S3</strong> - Amazon S3 и S3-совместимые хранилища
                </li>
                <li>
                  <a
                    href="/ru/storages/cloudflare-r2"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Cloudflare R2
                  </a>{" "}
                  - S3-совместимое объектное хранилище от Cloudflare
                </li>
                <li>
                  <strong>Azure Blob Storage</strong> - облачное хранилище от
                  Microsoft Azure
                </li>
                <li>
                  <strong>NAS</strong> - сетевые хранилища
                </li>
                <li>
                  <strong>FTP</strong> - серверы File Transfer Protocol
                </li>
                <li>
                  <strong>SFTP</strong> - серверы SSH File Transfer Protocol
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
