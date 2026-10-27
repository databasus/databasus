import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Cara menggunakan Databasus dengan Cloudflare R2 | Databasus",
  description:
    "Panduan langkah demi langkah mengatur penyimpanan Cloudflare R2 untuk backup PostgreSQL dengan Databasus. Pelajari cara menyiapkan penyimpanan R2 yang kompatibel dengan S3.",
  keywords: [
    "Databasus",
    "Cloudflare R2",
    "backup PostgreSQL",
    "penyimpanan S3",
    "penyimpanan cloud",
    "backup database",
  ],
  openGraph: {
    title: "Cara menggunakan Databasus dengan Cloudflare R2 | Databasus",
    description:
      "Panduan langkah demi langkah mengatur penyimpanan Cloudflare R2 untuk backup PostgreSQL dengan Databasus. Pelajari cara menyiapkan penyimpanan R2 yang kompatibel dengan S3.",
    type: "article",
    url: getLocalizedUrl("id", "storages/cloudflare-r2"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara menggunakan Databasus dengan Cloudflare R2 | Databasus",
    description:
      "Panduan langkah demi langkah mengatur penyimpanan Cloudflare R2 untuk backup PostgreSQL dengan Databasus. Pelajari cara menyiapkan penyimpanan R2 yang kompatibel dengan S3.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "storages/cloudflare-r2"),
    languages: getLanguageAlternates("storages/cloudflare-r2"),
  },
  robots: "index, follow",
};

export default function CloudflareR2Page() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara menggunakan Databasus dengan Cloudflare R2",
            description:
              "Panduan langkah demi langkah mengatur penyimpanan Cloudflare R2 untuk backup PostgreSQL dengan Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Isi nama bucket",
                text: "Masukkan nama bucket R2 di konfigurasi penyimpanan.",
              },
              {
                "@type": "HowToStep",
                name: "Atur region",
                text: 'Isi kolom region dengan "auto"',
              },
              {
                "@type": "HowToStep",
                name: "Buat Access Key ID dan Secret Access Key",
                text: "Di dashboard Cloudflare, buka R2 → API → Manage API Tokens. Buat token dan berikan izin yang diperlukan.",
              },
              {
                "@type": "HowToStep",
                name: "Temukan Account ID",
                text: "Account ID ada di dekat bagian atas setiap halaman R2 di dashboard.",
              },
              {
                "@type": "HowToStep",
                name: "Susun endpoint S3",
                text: "Ganti <ACCOUNT_ID> dengan nilai dari dashboard dalam format: https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
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
              <h1 id="cloudflare-r2">Penyimpanan Cloudflare R2</h1>
              <p className="text-lg text-gray-400">
                Untuk menggunakan Cloudflare R2 sebagai penyimpanan backup PostgreSQL yang
                kompatibel dengan S3, atur kredensial bucket R2 dan endpoint-nya.
              </p>
              <h2 id="configuration-steps">Langkah konfigurasi</h2>
              <h3 id="fill-bucket-name">1. Isi nama bucket</h3>
              <p>Masukkan nama bucket R2 di konfigurasi penyimpanan:</p>
              <Image
                src="/images/cloudflare-r2-storage/image-1.webp"
                alt="Mengisi nama bucket di Cloudflare R2"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="set-region">2. Atur region</h3>
              <p>
                Isi kolom region dengan <code>&quot;auto&quot;</code>
              </p>
              <h3 id="generate-access-key">3. Buat Access Key ID &amp; Secret Access Key</h3>
              <p>
                Di dashboard Cloudflare, buka <strong>R2 → API → Manage API Tokens</strong>. Buat
                token baru dan berikan izin yang diperlukan (misalnya{" "}
                <strong>&quot;Object Read &amp; Write&quot;</strong>).
              </p>
              <p>Setelah token dibuat, akan muncul:</p>
              <ul>
                <li>
                  <strong>Access Key ID</strong> (ID token)
                </li>
                <li>
                  <strong>Secret Access Key</strong> (hash SHA-256 dari nilai token)
                </li>
              </ul>
              <p>Salin kedua nilai tersebut ke Databasus:</p>
              <Image
                src="/images/cloudflare-r2-storage/image-2.gif"
                alt="Membuat Access Key ID dan Secret Access Key"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="find-account-id">4. Temukan Account ID</h3>
              <p>
                Account ID ada di dekat bagian atas setiap halaman R2 di dashboard (atau di
                pengaturan akun):
              </p>
              <Image
                src="/images/cloudflare-r2-storage/image-3.webp"
                alt="Menemukan Account ID di dashboard Cloudflare"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="construct-endpoint">5. Susun endpoint S3</h3>
              <p>Gunakan format berikut untuk endpoint S3:</p>
              <pre>
                <code>https://&lt;ACCOUNT_ID&gt;.r2.cloudflarestorage.com</code>
              </pre>
              <p>
                Ganti <code>&lt;ACCOUNT_ID&gt;</code> dengan nilai dari dashboard, lalu masukkan
                endpoint tersebut ke Databasus.
              </p>
              <p>Konfigurasinya sekarang akan terlihat seperti ini:</p>
              <Image
                src="/images/cloudflare-r2-storage/image-4.png"
                alt="Konfigurasi selesai"
                width={500}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <p>
                Databasus sekarang siap menggunakan Cloudflare R2 sebagai penyimpanan backup
                PostgreSQL.
              </p>
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("id", "storages")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Kembali ke penyimpanan
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
