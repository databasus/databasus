import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Stockages - Documentation Databasus",
  description:
    "Liste des destinations de stockage prises en charge pour les sauvegardes Databasus : stockage local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  keywords: [
    "stockages Databasus",
    "stockage de sauvegardes",
    "stockage S3",
    "Cloudflare R2",
    "sauvegarde NAS",
    "stockage local",
    "Azure Blob Storage",
    "sauvegarde FTP",
    "sauvegarde SFTP",
  ],
  openGraph: {
    title: "Stockages - Documentation Databasus",
    description:
      "Liste des destinations de stockage prises en charge pour les sauvegardes Databasus : stockage local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
    type: "article",
    url: getLocalizedUrl("fr", "storages"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Stockages - Documentation Databasus",
    description:
      "Liste des destinations de stockage prises en charge pour les sauvegardes Databasus : stockage local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "storages"),
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
            headline: "Stockages - Documentation Databasus",
            description:
              "Liste des destinations de stockage prises en charge pour les sauvegardes Databasus : stockage local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
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

      <DocsNavbarComponent lang="fr" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="fr" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="storages">Stockages</h1>

              <p className="text-lg text-gray-400">
                Databasus prend en charge plusieurs destinations de stockage
                pour vos sauvegardes PostgreSQL. Choisissez où stocker vos
                fichiers de backup selon votre infrastructure et vos besoins.
              </p>

              <h2 id="supported-storages">Stockages pris en charge</h2>

              <ul>
                <li>
                  <strong>Stockage local</strong> - Stockez les backups
                  directement sur votre serveur ou VPS
                </li>
                <li>
                  <strong>S3</strong> - Amazon S3 et services de stockage
                  compatibles S3
                </li>
                <li>
                  <a
                    href="/fr/storages/cloudflare-r2"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Cloudflare R2
                  </a>{" "}
                  - Stockage objet compatible S3 de Cloudflare
                </li>
                <li>
                  <strong>Azure Blob Storage</strong> - Stockage cloud de
                  Microsoft Azure
                </li>
                <li>
                  <strong>NAS</strong> - Périphériques de stockage en réseau
                </li>
                <li>
                  <strong>FTP</strong> - Serveurs File Transfer Protocol
                </li>
                <li>
                  <strong>SFTP</strong> - Serveurs SSH File Transfer Protocol
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
