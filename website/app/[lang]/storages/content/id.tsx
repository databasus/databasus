import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Penyimpanan - Dokumentasi Databasus",
  description:
    "Daftar tujuan penyimpanan yang didukung untuk backup Databasus, termasuk penyimpanan lokal, S3, Cloudflare R2, Google Drive, Azure Blob Storage, NAS, FTP, SFTP, rclone, dan Dropbox.",
  keywords: [
    "penyimpanan Databasus",
    "penyimpanan backup",
    "penyimpanan S3",
    "backup Google Drive",
    "Cloudflare R2",
    "backup NAS",
    "backup Dropbox",
    "penyimpanan lokal",
    "Azure Blob Storage",
    "backup FTP",
    "backup SFTP",
    "backup rclone",
  ],
  openGraph: {
    title: "Penyimpanan - Dokumentasi Databasus",
    description:
      "Daftar tujuan penyimpanan yang didukung untuk backup Databasus, termasuk penyimpanan lokal, S3, Cloudflare R2, Google Drive, Azure Blob Storage, NAS, FTP, SFTP, rclone, dan Dropbox.",
    type: "article",
    url: getLocalizedUrl("id", "storages"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Penyimpanan - Dokumentasi Databasus",
    description:
      "Daftar tujuan penyimpanan yang didukung untuk backup Databasus, termasuk penyimpanan lokal, S3, Cloudflare R2, Google Drive, Azure Blob Storage, NAS, FTP, SFTP, rclone, dan Dropbox.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "storages"),
    languages: getLanguageAlternates("storages"),
  },
  robots: "index, follow",
};

export default function StoragesPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Penyimpanan - Dokumentasi Databasus",
            description:
              "Daftar tujuan penyimpanan yang didukung untuk backup Databasus, termasuk penyimpanan lokal, S3, Cloudflare R2, Google Drive, Azure Blob Storage, NAS, FTP, SFTP, rclone, dan Dropbox.",
            author: { "@type": "Organization", name: "Databasus" },
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
      <DocsNavbarComponent lang="id" />
      <div className="flex min-h-screen bg-[#0F1115]">
        <DocsSidebarComponent lang="id" />
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="storages">Penyimpanan</h1>
              <p className="text-lg text-gray-400">
                Databasus mendukung berbagai tujuan penyimpanan untuk backup PostgreSQL. Pilih
                tempat menyimpan file backup sesuai infrastruktur dan kebutuhanmu.
              </p>
              <h2 id="supported-storages">Penyimpanan yang didukung</h2>
              <ul>
                <li>
                  <strong>Penyimpanan lokal</strong> - Simpan backup langsung di server atau VPS
                </li>
                <li>
                  <strong>S3</strong> - Amazon S3 dan layanan penyimpanan yang kompatibel dengan S3
                </li>
                <li>
                  <a
                    href={getLocalizedHref("id", "storages/cloudflare-r2")}
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Cloudflare R2
                  </a>{" "}
                  - Penyimpanan objek dari Cloudflare yang kompatibel dengan S3
                </li>
                <li>
                  <a
                    href={getLocalizedHref("id", "storages/google-drive")}
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Google Drive
                  </a>{" "}
                  - Penyimpanan cloud dari Google
                </li>
                <li>
                  <strong>Azure Blob Storage</strong> - Penyimpanan cloud dari Microsoft Azure
                </li>
                <li>
                  <strong>NAS</strong> - Perangkat penyimpanan yang terhubung ke jaringan
                </li>
                <li>
                  <strong>FTP</strong> - Server File Transfer Protocol
                </li>
                <li>
                  <strong>SFTP</strong> - Server SSH File Transfer Protocol
                </li>
                <li>
                  <strong>rclone</strong> - Hubungkan lebih dari 70 penyedia penyimpanan cloud
                  melalui rclone
                </li>
              </ul>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
