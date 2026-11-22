import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

export const metadata: Metadata = {
  title: "Cara backup PostgreSQL di Neon: panduan langkah demi langkah | Databasus",
  description:
    "Panduan langkah demi langkah dengan tangkapan layar: jadwalkan backup PostgreSQL di Neon dengan Databasus, simpan backup di penyimpanan milik sendiri, lalu restore kembali ke Neon.",
  keywords: [
    "Databasus",
    "backup Neon",
    "backup PostgreSQL Neon",
    "backup database Neon",
    "restore Neon",
    "pengguna read-only Neon",
    "backup PostgreSQL",
    "backup database",
  ],
  openGraph: {
    title: "Cara backup PostgreSQL di Neon: panduan langkah demi langkah | Databasus",
    description:
      "Panduan langkah demi langkah dengan tangkapan layar: jadwalkan backup PostgreSQL di Neon dengan Databasus, simpan backup di penyimpanan milik sendiri, lalu restore kembali ke Neon.",
    type: "article",
    url: getLocalizedUrl("id", "faq/neon"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara backup PostgreSQL di Neon: panduan langkah demi langkah | Databasus",
    description:
      "Panduan langkah demi langkah dengan tangkapan layar: jadwalkan backup PostgreSQL di Neon dengan Databasus, simpan backup di penyimpanan milik sendiri, lalu restore kembali ke Neon.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "faq/neon"),
    languages: getLanguageAlternates("faq/neon"),
  },
  robots: "index, follow",
};

export default function NeonPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara backup PostgreSQL di Neon dengan Databasus",
            description:
              "Panduan langkah demi langkah untuk backup database PostgreSQL di Neon dengan Databasus dan restore ke Neon",
            step: [
              {
                "@type": "HowToStep",
                name: "Salin connection string dari Neon",
                text: "Di konsol Neon, buka proyek, klik Connect, lalu klik Copy snippet.",
                url: `${getLocalizedUrl("id", "faq/neon")}#copy-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Tambahkan database Neon ke Databasus",
                text: "Klik Add database, pilih Neon, tempel connection string dengan Paste from clipboard, lalu klik Test connection.",
                url: `${getLocalizedUrl("id", "faq/neon")}#paste-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Buat pengguna read-only",
                text: "Klik Yes, create read-only user agar Databasus hanya menyimpan kredensial yang tidak bisa mengubah data.",
                url: `${getLocalizedUrl("id", "faq/neon")}#read-only-user`,
              },
              {
                "@type": "HowToStep",
                name: "Atur jadwal, penyimpanan, dan masa simpan",
                text: "Tentukan frekuensi backup, tempat menyimpannya, dan berapa lama backup dipertahankan.",
                url: `${getLocalizedUrl("id", "faq/neon")}#schedule-storage-retention`,
              },
              {
                "@type": "HowToStep",
                name: "Pilih kanal notifikasi",
                text: "Pilih kanal yang menerima notifikasi backup, lalu klik Complete. Databasus langsung memulai backup pertama.",
                url: `${getLocalizedUrl("id", "faq/neon")}#notifiers`,
              },
              {
                "@type": "HowToStep",
                name: "Restore backup ke Neon",
                text: "Klik ikon restore di samping backup, tempel connection string database Neon tujuan, lalu klik Restore to this DB.",
                url: `${getLocalizedUrl("id", "faq/neon")}#restore`,
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="id" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="id" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="neon-backup">Cara backup PostgreSQL di Neon</h1>

              <p className="text-lg text-gray-400">
                Neon adalah layanan PostgreSQL terkelola yang serverless. Untuk
                membuat backup-nya dengan Databasus, salin connection string
                dari konsol Neon, tambahkan database di Databasus dengan tipe
                Neon, lalu pilih tempat menyimpan backup. Setelah itu Databasus
                membuat backup sesuai jadwal dan bisa
                melakukan restore ke database Neon mana pun.
              </p>

              <p>
                Yang dibutuhkan hanya Databasus yang sudah berjalan. Jika belum ada, ikuti{" "}
                <a href={getLocalizedHref("id", "installation")}>panduan instalasi</a>
                . Penyimpanan dan kanal notifikasi bisa dibuat sambil mengisi pengaturan.
              </p>

              <h2 id="copy-connection-string">
                Langkah 1. Salin connection string dari Neon
              </h2>

              <p>
                Buka proyek di konsol Neon, lalu klik{" "}
                <strong>Connect</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-connect-button.png"
                alt="Tombol Connect di halaman Computes proyek Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
              />

              <p>
                Di jendela <strong>Connection details</strong>, pilih branch,
                database, dan role yang akan di-backup, lalu klik{" "}
                <strong>Copy snippet</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-copy-connection-string.png"
                alt="Jendela Connection details di Neon dengan tombol Copy snippet"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="add-database">
                Langkah 2. Tambahkan database Neon ke Databasus
              </h2>

              <h3 id="paste-connection-string">
                Pilih Neon dan tempel connection string
              </h3>

              <p>
                Di Databasus, klik <strong>Add database</strong>, beri nama
                database, pilih <strong>Neon</strong>, lalu klik{" "}
                <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-neon.png"
                alt="Jendela Add database di Databasus dengan tipe Neon terpilih"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Klik <strong>Paste from clipboard</strong>. Databasus membaca
                connection string yang disalin dari Neon, lalu mengisi host,
                port, nama pengguna, kata sandi, nama database, dan mode SSL.
              </p>

              <img
                src="/images/faq/neon/databasus-paste-connection-string.png"
                alt="Formulir koneksi Neon di Databasus dengan tombol Paste from clipboard"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Klik <strong>Test connection</strong>. Setelah tes berhasil,
                klik <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-test-connection.png"
                alt="Formulir koneksi Neon di Databasus yang sudah terisi setelah tes koneksi berhasil"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="read-only-user">Buat pengguna read-only</h3>

              <p>
                Databasus menawarkan untuk membuat{" "}
                <a
                  href={`${getLocalizedHref("id", "security")}#level-3-read-only-access`}
                >
                  pengguna read-only
                </a>{" "}
                khusus backup. Klik <strong>Yes, create read-only user</strong>.
                Dengan begitu, Databasus hanya menyimpan kredensial yang bisa
                membaca data, tetapi tidak bisa mengubahnya.
              </p>

              <img
                src="/images/faq/neon/databasus-create-read-only-user.png"
                alt="Databasus menawarkan pembuatan pengguna read-only untuk backup Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="schedule-storage-retention">
                Atur jadwal backup, penyimpanan, dan masa simpan
              </h3>

              <p>Tentukan:</p>

              <ul>
                <li>
                  <strong>Backup interval</strong> dan waktunya: seberapa sering
                  Databasus membuat backup, misalnya setiap hari pukul 03:00.
                </li>
                <li>
                  <strong>Storage</strong>: tempat backup disimpan.{" "}
                  <a href={getLocalizedHref("id", "storages")}>
                    Penyimpanan baru
                  </a>{" "}
                  bisa ditambahkan langsung dari daftar ini. Ingin menyimpan backup
                  di Neon juga? Lihat{" "}
                  <a href={getLocalizedHref("id", "storages/neon-object-storage")}>
                    cara backup ke Neon Object Storage
                  </a>
                  .
                </li>
                <li>
                  <strong>Encryption</strong>: apakah Databasus{" "}
                  <a
                    href={`${getLocalizedHref("id", "security")}#level-2-backups-encryption`}
                  >
                    mengenkripsi file backup
                  </a>{" "}
                  sebelum mengunggahnya.
                </li>
                <li>
                  <strong>Retention policy</strong>: berapa lama backup lama
                  disimpan, misalnya 3 bulan.
                </li>
                <li>
                  <strong>Notifications</strong>: peristiwa apa saja yang
                  dilaporkan, hanya backup yang gagal atau juga yang berhasil.
                </li>
              </ul>

              <img
                src="/images/faq/neon/databasus-schedule-storage-retention.png"
                alt="Pengaturan jadwal backup, penyimpanan, enkripsi, dan masa simpan di Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="notifiers">Pilih kanal notifikasi</h3>

              <p>
                Tentukan{" "}
                <a href={getLocalizedHref("id", "notifiers")}>
                  kanal notifikasi
                </a>{" "}
                yang akan menerima pemberitahuan tersebut, misalnya{" "}
                <a href={getLocalizedHref("id", "notifiers/slack")}>Slack</a>{" "}
                atau{" "}
                <a href={getLocalizedHref("id", "notifiers/teams")}>
                  Microsoft Teams
                </a>
                . Kanal bisa dipilih lebih dari satu, atau dilewati dulu dan
                ditambahkan nanti. Klik <strong>Complete</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-notifiers.png"
                alt="Pemilihan kanal notifikasi di Databasus sebelum pengaturan database selesai"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="first-backup">Langkah 3. Periksa backup pertama</h2>

              <p>
                Databasus memulai backup pertama begitu{" "}
                <strong>Complete</strong> diklik. Backup itu muncul di tab{" "}
                <strong>Backups</strong> beserta ukuran dan durasinya. Backup
                berikutnya berjalan sesuai jadwal, dan tombol{" "}
                <strong>Make backup right now</strong> memulai backup kapan
                saja.
              </p>

              <img
                src="/images/faq/neon/databasus-first-neon-backup.png"
                alt="Backup Neon pertama yang berhasil di tab Backups di Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Backup yang berhasil belum membuktikan bahwa restore-nya juga
                akan berhasil. Untuk memeriksa hal ini pada setiap backup,
                aktifkan{" "}
                <a href={getLocalizedHref("id", "restore-verification")}>
                  verifikasi restore
                </a>
                .
              </p>

              <h2 id="restore">Cara restore backup Neon</h2>

              <p>
                Klik ikon restore (awan dengan panah) di samping backup. Salin
                connection string database tujuan di konsol Neon dengan cara
                yang sama seperti pada{" "}
                <a href="#copy-connection-string">langkah 1</a>, klik{" "}
                <strong>Paste from clipboard</strong>, lalu{" "}
                <strong>Restore to this DB</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-restore-neon-backup.png"
                alt="Jendela restore dari backup di Databasus dengan database Neon tujuan"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Databasus mengosongkan database tujuan sebelum restore.
                Database tujuan bisa berupa:
              </p>

              <ul>
                <li>database baru di proyek Neon yang sama</li>
                <li>database di branch Neon</li>
                <li>database yang sudah ada, termasuk database sumbernya sendiri</li>
              </ul>

              <p>
                Pilihan paling aman adalah branch Neon baru. Data hasil restore
                bisa diperiksa di sana, sementara database produksi tetap utuh
                sampai kamu memutuskan untuk beralih.
              </p>

              <p>Setelah restore selesai, statusnya berubah menjadi Successful.</p>

              <img
                src="/images/faq/neon/databasus-restore-done.png"
                alt="Restore backup Neon yang sudah selesai di Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="health-check">
                Apakah pemeriksaan ketersediaan membuat compute Neon tetap aktif?
              </h2>

              <p>
                Secara default tidak. Saat tidak ada yang terhubung, Neon
                menidurkan compute dan tidak menagih biayanya selama compute
                tidur. Pemeriksaan ketersediaan terhubung setiap menit, sehingga
                compute akan terus aktif dan ditagih 24 jam sehari. Karena itu,
                Databasus mematikan pemeriksaan ketersediaan untuk database
                Neon. Jika fitur ini dinyalakan, Databasus meminta konfirmasi
                terlebih dahulu. Backup terjadwal hanya membangunkan compute
                selama backup berjalan, lalu Neon menidurkannya lagi.
              </p>

              <p>
                Perlu backup Supabase juga? Lihat{" "}
                <a href={getLocalizedHref("id", "faq/supabase")}>
                  cara backup Supabase
                </a>
                .
              </p>

              {/* Navigation */}
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

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
