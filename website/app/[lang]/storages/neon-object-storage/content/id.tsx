import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

const PAGE_URL = getLocalizedUrl("id", "storages/neon-object-storage");

export const metadata: Metadata = {
  title: "Cara backup PostgreSQL ke Neon Object Storage: panduan langkah demi langkah | Databasus",
  description:
    "Panduan langkah demi langkah dengan screenshot: buat bucket dan kredensial di Neon Object Storage, tambahkan ke Databasus sebagai penyimpanan S3, lalu simpan backup database di Neon sesuai jadwal.",
  keywords: [
    "Databasus",
    "backup Neon Object Storage",
    "Neon Object Storage",
    "Neon S3",
    "bucket Neon",
    "backup PostgreSQL ke S3",
    "penyimpanan kompatibel S3",
    "backup database",
  ],
  openGraph: {
    title: "Cara backup PostgreSQL ke Neon Object Storage: panduan langkah demi langkah | Databasus",
    description:
      "Panduan langkah demi langkah dengan screenshot: buat bucket dan kredensial di Neon Object Storage, tambahkan ke Databasus sebagai penyimpanan S3, lalu simpan backup database di Neon sesuai jadwal.",
    type: "article",
    url: PAGE_URL,
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara backup PostgreSQL ke Neon Object Storage: panduan langkah demi langkah | Databasus",
    description:
      "Panduan langkah demi langkah dengan screenshot: buat bucket dan kredensial di Neon Object Storage, tambahkan ke Databasus sebagai penyimpanan S3, lalu simpan backup database di Neon sesuai jadwal.",
  },
  alternates: {
    canonical: PAGE_URL,
    languages: getLanguageAlternates("storages/neon-object-storage"),
  },
  robots: "index, follow",
};

export default function NeonObjectStoragePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara backup ke Neon Object Storage dengan Databasus",
            description:
              "Panduan langkah demi langkah untuk membuat bucket dan kredensial di Neon Object Storage lalu menambahkannya ke Databasus sebagai penyimpanan backup",
            step: [
              {
                "@type": "HowToStep",
                name: "Buat bucket di Neon",
                text: "Di Neon Console, buka proyek untuk backup, klik Object storage di sidebar, lalu klik Create bucket.",
                url: `${PAGE_URL}#create-bucket`,
              },
              {
                "@type": "HowToStep",
                name: "Salin kredensial penyimpanan",
                text: "Klik Connect, buka tab Storage, pilih Parameters only, lalu klik Copy credentials.",
                url: `${PAGE_URL}#copy-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Pilih Neon Object Storage di Databasus",
                text: "Di Databasus, buka Storages, klik Add storage, lalu pilih Neon Object Storage di daftar tipe.",
                url: `${PAGE_URL}#choose-neon-object-storage`,
              },
              {
                "@type": "HowToStep",
                name: "Tempel kredensial dan ketik nama bucket",
                text: "Klik Paste from clipboard untuk mengisi endpoint, region, access key, dan secret key, lalu ketik nama bucket.",
                url: `${PAGE_URL}#paste-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Uji koneksi dan simpan",
                text: "Klik Test connection, lalu klik Save setelah pengujian berhasil.",
                url: `${PAGE_URL}#test-connection`,
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
              <h1 id="neon-object-storage-backup">Cara backup ke Neon Object Storage</h1>

              <p className="text-lg text-gray-400">
                <a href="https://neon.com/docs/storage/overview">Neon Object Storage</a> adalah
                penyimpanan objek kompatibel S3 yang tersedia di setiap proyek Neon. Untuk
                menyimpan backup di sana, buat bucket dan kredensial di Neon Console, lalu
                tambahkan bucket itu ke Databasus sebagai penyimpanan. Databasus akan mengunggah
                setiap backup dari database yang memakai penyimpanan ini, baik database itu
                berjalan di Neon maupun di tempat lain.
              </p>

              <p>
                Yang dibutuhkan hanya Databasus yang sudah berjalan. Jika belum ada,
                ikuti <a href={getLocalizedHref("id", "installation")}>panduan instalasi</a>.{" "}
                <a href="https://neon.com/docs/storage/s3-compatibility">Halaman
                kompatibilitas S3</a> di dokumentasi Neon menjelaskan apa saja yang didukung.
              </p>

              <h2 id="create-bucket">Langkah 1. Buat bucket di Neon</h2>

              <p>
                Buka proyek yang akan menyimpan backup di Neon Console. Sebaiknya gunakan proyek
                yang berbeda dari proyek database yang di-backup, karena menghapus proyek juga
                menghapus bucket-nya. Klik <strong>Object storage</strong> di sidebar,
                lalu <strong>Create bucket</strong>, dan masukkan nama bucket. Bucket baru
                bersifat privat dan menjadi milik branch proyek saat ini.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-object-storage-buckets.png"
                alt="Halaman Object storage proyek Neon dengan sebuah bucket dan tombol Create bucket"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
              />

              <h2 id="copy-credentials">Langkah 2. Salin kredensial penyimpanan</h2>

              <p>
                Klik <strong>Connect</strong> dan buka tab <strong>Storage</strong>.
                Dengan <strong>Parameters only</strong> terpilih, Neon menampilkan endpoint,
                access key, secret key, dan region. Jika blok itu berisi placeholder alih-alih
                kunci, buat kredensial lebih dulu seperti yang diminta baris pertamanya. Lalu
                klik <strong>Copy credentials</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-connect-storage-credentials.png"
                alt="Tab Storage di jendela Connect Neon dengan endpoint, kunci, region, dan tombol Copy credentials"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Bloknya terlihat seperti ini. Nama bucket tidak ada di dalamnya, jadi ingat nama
                dari langkah 1.
              </p>

              <pre>
                <code>{`AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.c-9.us-east-2.aws.neon.tech"
AWS_ACCESS_KEY_ID="nak_live_..."
AWS_SECRET_ACCESS_KEY="..."
AWS_REGION="us-east-2"`}</code>
              </pre>

              <h2 id="add-storage">Langkah 3. Tambahkan penyimpanan ke Databasus</h2>

              <h3 id="choose-neon-object-storage">Pilih Neon Object Storage di Databasus</h3>

              <p>
                Di Databasus, buka <strong>Storages</strong> dan klik <strong>Add
                storage</strong>. Beri nama penyimpanan dan pilih <strong>Neon Object
                Storage</strong> di daftar tipe. Databasus menyimpannya sebagai penyimpanan S3
                biasa, sehingga daftar penyimpanan menampilkan tipenya sebagai S3.
              </p>

              <h3 id="paste-credentials">Tempel kredensial dan ketik nama bucket</h3>

              <p>
                Klik <strong>Paste from clipboard</strong>. Databasus membaca blok yang disalin
                dari Neon dan mengisi endpoint, region, access key, dan secret key. Blok itu
                tetap di browser: kunci baru keluar dari browser saat penyimpanan diuji atau
                disimpan. Ketik sendiri nama bucket di <strong>S3 Bucket</strong>. Semua kolom
                juga bisa diisi manual di bawah pemisah <strong>or enter the details
                manually</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/databasus-neon-object-storage-form.png"
                alt="Formulir Neon Object Storage di Databasus dengan endpoint, region dan kunci yang sudah ditempel serta nama bucket yang diketik"
                width={1435}
                height={897}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="test-connection">Uji koneksi dan simpan</h3>

              <p>
                Klik <strong>Test connection</strong>. Setelah pengujian berhasil,
                klik <strong>Save</strong>. Sekarang penyimpanan ini bisa dipilih di pengaturan
                backup database mana pun. Jika database-nya sendiri berjalan di
                Neon, <a href={getLocalizedHref("id", "faq/neon")}>panduan backup
                Neon</a> menjelaskan sisa pengaturannya.
              </p>

              {/* Navigation */}
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

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
