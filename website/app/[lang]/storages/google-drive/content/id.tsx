import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Cara menghubungkan Google Drive ke Databasus | Databasus",
  description:
    "Panduan langkah demi langkah mengatur penyimpanan Google Drive untuk backup PostgreSQL dengan Databasus. Pelajari cara menyiapkan project Google Cloud dan OAuth.",
  keywords: [
    "Databasus",
    "Google Drive",
    "backup PostgreSQL",
    "Google Cloud",
    "OAuth",
    "penyimpanan cloud",
    "backup database",
  ],
  openGraph: {
    title: "Cara menghubungkan Google Drive ke Databasus | Databasus",
    description:
      "Panduan langkah demi langkah mengatur penyimpanan Google Drive untuk backup PostgreSQL dengan Databasus. Pelajari cara menyiapkan project Google Cloud dan OAuth.",
    type: "article",
    url: getLocalizedUrl("id", "storages/google-drive"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara menghubungkan Google Drive ke Databasus | Databasus",
    description:
      "Panduan langkah demi langkah mengatur penyimpanan Google Drive untuk backup PostgreSQL dengan Databasus. Pelajari cara menyiapkan project Google Cloud dan OAuth.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "storages/google-drive"),
    languages: getLanguageAlternates("storages/google-drive"),
  },
  robots: "index, follow",
};

export default function GoogleDrivePage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara menghubungkan Google Drive ke Databasus",
            description:
              "Panduan langkah demi langkah mengatur penyimpanan Google Drive untuk backup PostgreSQL dengan Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Buat project baru",
                text: "Buka Google Cloud Console dan buat project baru.",
              },
              {
                "@type": "HowToStep",
                name: "Aktifkan Google Drive API",
                text: "Buka tab API & Services, lalu API library, dan aktifkan Google Drive API.",
              },
              {
                "@type": "HowToStep",
                name: "Atur layar persetujuan",
                text: "Buka Credentials → Create credentials → Configure consent screen dan isi data yang diperlukan.",
              },
              {
                "@type": "HowToStep",
                name: "Buat OAuth client ID",
                text: "Buka Credentials → Create credentials → OAuth client ID.",
              },
              {
                "@type": "HowToStep",
                name: "Atur pengaturan aplikasi",
                text: "Pilih Web application sebagai jenis aplikasi, lalu atur authorized origins dan redirect URIs.",
              },
              {
                "@type": "HowToStep",
                name: "Tambahkan scope",
                text: 'Buka Data Access dan tambahkan scope "/auth/drive.file".',
              },
              {
                "@type": "HowToStep",
                name: "Publikasikan aplikasi",
                text: "Buka Audience dan publikasikan aplikasi.",
              },
              {
                "@type": "HowToStep",
                name: "Masuk melalui akun Google",
                text: "Isi data kredensial dan masuk dengan akun Google.",
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
              <h1 id="google-drive">Penyimpanan Google Drive</h1>
              <p className="text-lg text-gray-400">
                Untuk menyimpan backup di Google Drive, buat project Google Cloud agar bisa
                mengakses Google Drive API, lalu masuk melalui akun Google.
              </p>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Penting:</strong> Databasus harus berjalan
                  melalui HTTPS agar bisa terhubung ke Google Drive. Untuk mengetahui cara
                  menyiapkan HTTPS, lihat{" "}
                  <a
                    href={`${getLocalizedHref("id", "installation")}#caddy-reverse-proxy`}
                    className="text-blue-400 hover:text-blue-300"
                  >
                    panduan pengaturan reverse proxy Caddy →
                  </a>
                  <br />
                  Panduan ini mengasumsikan Databasus berjalan di{" "}
                  <code>databasus.yourdomain.com</code>.
                </p>
              </div>
              <h2 id="create-google-cloud-project">Buat project Google Cloud</h2>
              <h3 id="create-new-project">1. Buat project baru</h3>
              <p>
                Buka{" "}
                <a
                  href="https://console.cloud.google.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://console.cloud.google.com/
                </a>{" "}
                dan pilih <strong>&quot;new project&quot;</strong> (kiri atas).
              </p>
              <h3 id="enable-google-drive-api">2. Aktifkan Google Drive API</h3>
              <p>
                Buka tab <strong>&quot;API &amp; Services&quot;</strong>, lalu{" "}
                <strong>&quot;API library&quot;</strong>. Pilih <strong>Google Drive API</strong>{" "}
                dan aktifkan:
              </p>
              <Image
                src="/images/google-drive-storage/image-1.webp"
                alt="Mengaktifkan Google Drive API"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="configure-consent-screen">3. Atur layar persetujuan</h3>
              <p>
                Buka <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;Configure consent screen&quot;</strong> dan isi datanya sesuai
                kebutuhan:
              </p>
              <Image
                src="/images/google-drive-storage/image-2.webp"
                alt="Mengatur layar persetujuan"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="create-oauth-client-id">4. Buat OAuth client ID</h3>
              <p>
                Buka <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;OAuth client ID&quot;</strong>:
              </p>
              <Image
                src="/images/google-drive-storage/image-3.webp"
                alt="Membuat OAuth client ID"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="configure-application-settings">5. Atur pengaturan aplikasi</h3>
              <p>Isi data berikut:</p>
              <ul>
                <li>
                  <strong>Application type:</strong> Web application
                </li>
                <li>
                  <strong>Authorized JavaScript origins:</strong>{" "}
                  <code>https://databasus.yourdomain.com</code>
                </li>
                <li>
                  <strong>Authorized redirect URIs:</strong>{" "}
                  <code>https://databasus.yourdomain.com/storages/google-oauth</code>
                </li>
              </ul>
              <p>
                <strong>Catatan:</strong> Ganti <code>databasus.yourdomain.com</code> dengan domain
                tempat Databasus dihosting (misalnya <code>backup.example.com</code>).
              </p>
              <p>Lalu salin kredensialnya:</p>
              <Image
                src="/images/google-drive-storage/image-4.png"
                alt="Mengatur pengaturan aplikasi - bagian 1"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <Image
                src="/images/google-drive-storage/image-5.png"
                alt="Mengatur pengaturan aplikasi - bagian 2"
                width={450}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="add-scope">6. Tambahkan scope</h3>
              <p>
                Buka <strong>&quot;Data Access&quot;</strong> dan tambahkan scope{" "}
                <code>&quot;/auth/drive.file&quot;</code>:
              </p>
              <Image
                src="/images/google-drive-storage/image-6.png"
                alt="Menambahkan scope"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="publish-app">7. Publikasikan aplikasi</h3>
              <p>
                Buka <strong>&quot;Audience&quot;</strong> dan publikasikan aplikasi:
              </p>
              <Image
                src="/images/google-drive-storage/image-7.png"
                alt="Memublikasikan aplikasi"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h2 id="sign-in-google-account">Masuk melalui akun Google</h2>
              <h3 id="fill-credentials">1. Isi data kredensial</h3>
              <p>Masukkan kredensial dari langkah sebelumnya ke Databasus:</p>
              <Image
                src="/images/google-drive-storage/image-8.png"
                alt="Mengisi data kredensial"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="choose-account">2. Pilih akun</h3>
              <p>Pilih akun Google yang akan digunakan untuk masuk.</p>
              <h3 id="handle-security-warning">3. Tangani peringatan keamanan</h3>
              <p>
                Jika muncul peringatan, klik <strong>&quot;Advanced&quot;</strong> (sudut kiri
                bawah), lalu pilih <strong>&quot;Proceed anyway&quot;</strong>.
              </p>
              <p>
                <strong>Catatan:</strong> Peringatan ini muncul karena aplikasi belum diverifikasi
                oleh Google. Aman untuk melanjutkan jika ini aplikasi milikmu sendiri.
              </p>
              <p>
                Google Drive sekarang terhubung ke Databasus dan siap menyimpan backup PostgreSQL.
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
