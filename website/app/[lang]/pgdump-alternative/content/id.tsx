import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Alternatif pg_dump - Databasus, alat backup PostgreSQL",
  description:
    "Databasus dibangun di atas pg_dump dan melengkapinya dengan pengelolaan backup, antarmuka web, penjadwalan otomatis, penyimpanan cloud, notifikasi, kolaborasi tim, dan enkripsi.",
  keywords: [
    "alternatif pg_dump",
    "pg_dump GUI",
    "otomatisasi pg_dump",
    "antarmuka web pg_dump",
    "alat backup PostgreSQL",
    "penjadwal pg_dump",
    "penyimpanan cloud pg_dump",
    "enkripsi pg_dump",
    "otomatisasi backup PostgreSQL",
    "wrapper pg_dump",
  ],
  openGraph: {
    title: "Alternatif pg_dump - Databasus, alat backup PostgreSQL",
    description:
      "Databasus dibangun di atas pg_dump dan melengkapinya dengan pengelolaan backup, antarmuka web, penjadwalan otomatis, penyimpanan cloud, notifikasi, kolaborasi tim, dan enkripsi.",
    type: "article",
    url: getLocalizedUrl("id", "pgdump-alternative"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Alternatif pg_dump - Databasus, alat backup PostgreSQL",
    description:
      "Databasus dibangun di atas pg_dump dan melengkapinya dengan pengelolaan backup, antarmuka web, penjadwalan otomatis, penyimpanan cloud, notifikasi, kolaborasi tim, dan enkripsi.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "pgdump-alternative"),
    languages: getLanguageAlternates("pgdump-alternative"),
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Alternatif pg_dump - Databasus, alat backup PostgreSQL",
            description:
              "Panduan lengkap Databasus sebagai alternatif pg_dump, menjelaskan penggunaan pg_dump sebagai fondasi dan tambahan kemampuan otomatisasi, penyimpanan cloud, notifikasi, serta fitur tim.",
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
        {/* Sidebar */}
        <DocsSidebarComponent lang="id" />
        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="pgdump-alternative">Alternatif pg_dump</h1>
              <p className="text-lg text-gray-400">
                Untuk backup logis, Databasus dibangun di atas <code>pg_dump</code>. Alih-alih
                menggantikan <code>pg_dump</code>, Databasus melengkapinya dengan pengelolaan
                backup, antarmuka web, penjadwalan otomatis, integrasi penyimpanan cloud,
                notifikasi, fitur kolaborasi tim, dan enkripsi bawaan. Selain backup logis,
                Databasus juga mendukung backup fisik, backup inkremental dengan pengarsipan WAL,
                dan Point-in-Time Recovery.
              </p>
              <h2 id="quick-comparison">Perbandingan singkat</h2>
              <p>
                Berikut ringkasan kemampuan yang ditambahkan Databasus pada fungsi inti{" "}
                <code>pg_dump</code>:
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Fitur</th>
                    <th>pg_dump</th>
                    <th>Databasus</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Mesin backup</td>
                    <td data-label="pg_dump">pg_dump</td>
                    <td data-label="Databasus">Dibangun di atas pg_dump</td>
                  </tr>
                  <tr>
                    <td>Pengelolaan backup</td>
                    <td data-label="pg_dump">❌ Tidak</td>
                    <td data-label="Databasus">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Dukungan database lain</td>
                    <td data-label="pg_dump">Hanya PostgreSQL</td>
                    <td data-label="Databasus">PostgreSQL, MySQL, MariaDB, MongoDB</td>
                  </tr>
                  <tr>
                    <td>Antarmuka</td>
                    <td data-label="pg_dump">Command-line</td>
                    <td data-label="Databasus">Antarmuka web + API</td>
                  </tr>
                  <tr>
                    <td>Penjadwalan</td>
                    <td data-label="pg_dump">Manual atau script cron</td>
                    <td data-label="Databasus">✅ Penjadwal bawaan</td>
                  </tr>
                  <tr>
                    <td>Tujuan penyimpanan</td>
                    <td data-label="pg_dump">Filesystem lokal</td>
                    <td data-label="Databasus">Lokal, S3, Google Drive, R2, Azure, NAS, Dropbox</td>
                  </tr>
                  <tr>
                    <td>Kompresi</td>
                    <td data-label="pg_dump">gzip, LZ4, zstd (manual)</td>
                    <td data-label="Databasus">zstd (otomatis, dioptimalkan)</td>
                  </tr>
                  <tr>
                    <td>Enkripsi</td>
                    <td data-label="pg_dump">Memerlukan alat eksternal</td>
                    <td data-label="Databasus">✅ AES-256-GCM bawaan</td>
                  </tr>
                  <tr>
                    <td>Notifikasi</td>
                    <td data-label="pg_dump">❌ Tidak ada</td>
                    <td data-label="Databasus">✅ Slack, Teams, Telegram, Email, Webhooks</td>
                  </tr>
                  <tr>
                    <td>Fitur tim</td>
                    <td data-label="pg_dump">❌ Tidak ada</td>
                    <td data-label="Databasus">✅ Ruang kerja, RBAC, log audit</td>
                  </tr>
                  <tr>
                    <td>Kebijakan retensi</td>
                    <td data-label="pg_dump">Script pembersihan manual</td>
                    <td data-label="Databasus">✅ Retensi otomatis</td>
                  </tr>
                  <tr>
                    <td>Pemantauan ketersediaan</td>
                    <td data-label="pg_dump">❌ Tidak ada</td>
                    <td data-label="Databasus">✅ Pemeriksaan ketersediaan bawaan</td>
                  </tr>
                  <tr>
                    <td>Backup fisik</td>
                    <td data-label="pg_dump">❌ Tidak</td>
                    <td data-label="Databasus">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Backup inkremental</td>
                    <td data-label="pg_dump">❌ Tidak</td>
                    <td data-label="Databasus">✅ Tingkat blok (PG 17+)</td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="pg_dump">❌ Tidak</td>
                    <td data-label="Databasus">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Backup jarak jauh</td>
                    <td data-label="pg_dump">✅ Ya (CLI)</td>
                    <td data-label="Databasus">✅ Ya</td>
                  </tr>
                </tbody>
              </table>
              <h2 id="what-is-pgdump">Apa itu pg_dump?</h2>
              <p>
                <code>pg_dump</code> adalah utilitas bawaan PostgreSQL untuk membuat backup logis.
                Alat ini sudah menjadi bagian dari PostgreSQL sejak awal dan merupakan alat standar
                untuk mengekspor database.
              </p>
              <h3 id="pgdump-strengths">Keunggulan pg_dump</h3>
              <ul>
                <li>
                  <strong>Backup portabel</strong>: Membuat dump SQL atau format custom yang dapat
                  di-restore ke versi PostgreSQL berbeda.
                </li>
                <li>
                  <strong>Backup selektif</strong>: Dapat mengekspor tabel tertentu, skema, atau
                  seluruh database.
                </li>
                <li>
                  <strong>Snapshot konsisten</strong>: Menggunakan MVCC PostgreSQL untuk membuat
                  backup konsisten tanpa memblokir penulisan.
                </li>
                <li>
                  <strong>Dukungan luas</strong>: Tersedia di setiap instalasi PostgreSQL,
                  terdokumentasi dengan baik, dan sudah teruji.
                </li>
                <li>
                  <strong>Format keluaran fleksibel</strong>: Format SQL biasa, custom, directory,
                  atau tar.
                </li>
              </ul>
              <h3 id="pgdump-limitations">Keterbatasan pg_dump</h3>
              <p>
                Meski <code>pg_dump</code> memiliki banyak kemampuan, penggunaannya di production
                biasanya memerlukan script tambahan:
              </p>
              <ul>
                <li>
                  <strong>Tanpa penjadwalan bawaan</strong>: Memerlukan cron job atau penjadwal
                  eksternal.
                </li>
                <li>
                  <strong>Hanya penyimpanan lokal</strong>: Keluaran ditulis ke filesystem lokal;
                  upload ke cloud memerlukan script tambahan.
                </li>
                <li>
                  <strong>Tanpa enkripsi</strong>: Secara default file backup tidak dienkripsi;
                  perlu meneruskan keluaran melalui gpg atau alat serupa.
                </li>
                <li>
                  <strong>Tanpa notifikasi</strong>: Tidak dapat memberi peringatan backup berhasil
                  atau gagal tanpa script khusus.
                </li>
                <li>
                  <strong>Tanpa pengelolaan retensi</strong>: Backup lama harus dibersihkan secara
                  manual atau melalui script.
                </li>
                <li>
                  <strong>Hanya command-line</strong>: Tidak ada antarmuka visual untuk pemantauan
                  atau pengelolaan.
                </li>
              </ul>
              <h2 id="how-databasus-extends">Cara Databasus melengkapi pg_dump</h2>
              <p>
                Databasus menggunakan <code>pg_dump</code> sebagai mesin backup, mempertahankan
                semua manfaat backup logis sambil menambahkan fitur perusahaan.
              </p>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Di balik layar:</strong> Saat kamu memulai
                  backup di Databasus, aplikasi menjalankan{" "}
                  <code className="bg-[#374151] text-gray-200">pg_dump</code> dengan parameter yang
                  dioptimalkan, lalu menangani kompresi, enkripsi, dan upload ke tujuan penyimpanan
                  yang dikonfigurasi.
                </p>
              </div>
              <h3 id="web-interface">Antarmuka web</h3>
              <p>
                Daripada mengingat opsi command-line <code>pg_dump</code>, Databasus menyediakan
                antarmuka web untuk:
              </p>
              <ul>
                <li>Menambahkan database dengan panduan koneksi bertahap</li>
                <li>Mengatur jadwal backup dengan kontrol visual</li>
                <li>Memantau riwayat dan status backup sekilas</li>
                <li>Mengunduh atau me-restore backup dengan satu klik</li>
                <li>Melihat grafik kesehatan dan ketersediaan database</li>
              </ul>
              <h3 id="optimized-compression">Kompresi yang dioptimalkan</h3>
              <p>Databasus menggunakan kompresi zstd (level 5) secara default, dengan hasil:</p>
              <ul>
                <li>
                  <strong>Ukuran 4-8x lebih kecil</strong> dibandingkan dump tanpa kompresi
                </li>
                <li>
                  <strong>Tambahan waktu proses ~20%</strong>, jauh lebih cepat daripada gzip
                </li>
                <li>
                  <strong>Penanganan otomatis</strong>, tidak perlu meneruskan keluaran melalui alat
                  kompresi
                </li>
              </ul>
              <h2 id="beyond-pgdump">Melampaui pg_dump: Backup fisik dan PITR</h2>
              <p>
                Databasus menggunakan <code>pg_dump</code> untuk backup logis, tetapi juga
                menawarkan kemampuan di luar jangkauan <code>pg_dump</code>:
              </p>
              <ul>
                <li>
                  <strong>Backup fisik</strong>: Salinan tingkat file dari seluruh cluster database
                  melalui <code>pg_basebackup</code>. Backup dan restore lebih cepat untuk database
                  besar.
                </li>
                <li>
                  <strong>Backup inkremental dan WAL</strong>: Backup inkremental tingkat blok
                  melalui <code>pg_basebackup --incremental</code> (menggunakan ringkasan WAL dari
                  server), ditambah streaming WAL berkelanjutan melalui <code>pg_receivewal</code>,
                  memungkinkan Point-in-Time Recovery, yaitu restore ke detik mana pun di antara
                  backup.
                </li>
                <li>
                  <strong>Pemulihan bencana</strong>: Dirancang untuk kebutuhan kehilangan data
                  mendekati nol dengan backup dasar fisik dan streaming WAL berkelanjutan.
                </li>
              </ul>
              <p>
                Backup ini menggunakan mekanisme backup bawaan PostgreSQL 17, jadi Databasus
                menggunakan kembali alat PostgreSQL yang sudah teruji alih-alih membuat ulang.
                Memerlukan PostgreSQL 17 atau lebih baru; pada versi lama hanya tersedia backup
                logis <code>pg_dump</code>. Semuanya berjalan dari jarak jauh dari host Databasus
                melalui protokol replikasi, sehingga tidak perlu instalasi di server database.
                Jaringan tertutup dijangkau melalui SSH tunnel ke host internal atau bastion, jadi
                database tidak perlu dibuka ke publik.{" "}
                <a
                  href={`${getLocalizedHref("id", "faq")}#pitr`}
                  className="text-blue-400 hover:text-blue-600"
                >
                  Baca cara kerja backup fisik dan PITR
                </a>
                .
              </p>
              <h2 id="backup-automation">Otomatisasi backup</h2>
              <p>
                Salah satu tantangan paling umum dengan <code>pg_dump</code> adalah menyiapkan
                backup otomatis yang andal.
              </p>
              <h3 id="automation-pgdump">Otomatisasi pg_dump konvensional</h3>
              <p>
                Script otomatisasi <code>pg_dump</code> umumnya seperti ini:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
                </pre>
              </div>
              <p>
                Script ini perlu dipelihara, diuji, dan dipantau. Setiap database memerlukan entri
                cron sendiri.
              </p>
              <h3 id="automation-databasus">Otomatisasi Databasus</h3>
              <p>Di Databasus, fungsi yang sama sudah tersedia sebagai fitur bawaan:</p>
              <ul>
                <li>
                  <strong>Penjadwal visual</strong>: Atur backup per jam, harian, mingguan, bulanan,
                  atau cron dengan waktu tertentu.
                </li>
                <li>
                  <strong>Kompresi otomatis</strong>: Kompresi zstd diterapkan secara otomatis.
                </li>
                <li>
                  <strong>Enkripsi bawaan</strong>: Enkripsi AES-256-GCM dengan kunci unik untuk
                  setiap backup.
                </li>
                <li>
                  <strong>Upload ke cloud</strong>: Upload langsung ke S3, Google Drive, Cloudflare
                  R2, Azure, atau tujuan lain.
                </li>
                <li>
                  <strong>Kebijakan retensi</strong>: Pembersihan otomatis backup lama berdasarkan
                  pengaturan retensi.
                </li>
                <li>
                  <strong>Notifikasi</strong>: Peringatan ke Slack, Teams, Telegram, Email saat
                  backup berhasil atau gagal.
                </li>
              </ul>
              <h2 id="storage-options">Pilihan penyimpanan</h2>
              <p>
                <code>pg_dump</code> menulis ke filesystem lokal. Mengirim backup ke penyimpanan
                cloud memerlukan alat dan script tambahan.
              </p>
              <h3 id="storage-databasus">Tujuan penyimpanan Databasus</h3>
              <p>Databasus langsung mendukung banyak tujuan penyimpanan:</p>
              <ul>
                <li>Penyimpanan lokal</li>
                <li>Amazon S3 dan layanan kompatibel S3</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>
              <p>
                Setiap database dapat memiliki tujuan penyimpanan sendiri, dan beberapa tujuan dapat
                dikonfigurasi untuk redundansi.
              </p>
              <p>
                <a
                  href={getLocalizedHref("id", "storages")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua pilihan penyimpanan →
                </a>
              </p>
              <h2 id="notifications">Notifikasi</h2>
              <p>
                Mengetahui kapan backup berhasil atau gagal sangat penting untuk perlindungan data.
              </p>
              <h3 id="notifications-pgdump">Notifikasi pg_dump</h3>
              <p>
                <code>pg_dump</code> tidak memiliki sistem notifikasi. Kamu perlu:
              </p>
              <ul>
                <li>Menulis script pembungkus yang memeriksa exit code</li>
                <li>Mengintegrasikan dengan alat pemantauan eksternal</li>
                <li>Menyiapkan alur peringatan khusus</li>
              </ul>
              <h3 id="notifications-databasus">Notifikasi Databasus</h3>
              <p>Databasus menyediakan notifikasi bawaan ke:</p>
              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Webhooks (untuk integrasi khusus)</li>
              </ul>
              <p>
                Tentukan kejadian yang memicu notifikasi: backup berhasil, backup gagal, atau
                keduanya.
              </p>
              <p>
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua kanal notifikasi →
                </a>
              </p>
              <h2 id="team-features">Fitur tim</h2>
              <p>
                <code>pg_dump</code> adalah alat command-line untuk satu pengguna. Databasus
                menambahkan fitur kolaborasi untuk tim:
              </p>
              <h3 id="team-databasus">Kemampuan Databasus untuk tim</h3>
              <ul>
                <li>
                  <strong>Ruang kerja</strong>: Atur database, kanal notifikasi, dan penyimpanan
                  berdasarkan proyek atau tim. Pengguna hanya melihat ruang kerja yang
                  mengundangnya.
                </li>
                <li>
                  <strong>Kontrol akses berbasis peran</strong>: Tetapkan izin viewer, editor, atau
                  admin untuk mengontrol tindakan setiap anggota tim.
                </li>
                <li>
                  <strong>Log audit</strong>: Lacak semua aktivitas dan perubahan sistem. Penting
                  untuk kepatuhan keamanan dan pertanggungjawaban.
                </li>
                <li>
                  <strong>Notifikasi bersama</strong>: Kanal tim menerima pembaruan status backup
                  secara otomatis.
                </li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "access-management")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Pelajari pengelolaan akses →
                </a>
              </p>
              <h2 id="security">Keamanan</h2>
              <p>
                Keamanan adalah salah satu manfaat besar Databasus dibandingkan penggunaan{" "}
                <code>pg_dump</code> secara langsung.
              </p>
              <h3 id="security-pgdump">Keamanan pg_dump</h3>
              <p>
                <code>pg_dump</code> membuat file backup tanpa enkripsi. Untuk mengamankannya,
                perlu:
              </p>
              <ul>
                <li>Meneruskan keluaran melalui alat enkripsi (gpg, openssl)</li>
                <li>Mengelola kunci enkripsi secara terpisah</li>
                <li>Memastikan penyimpanan dan rotasi kunci yang aman</li>
                <li>Mengatur izin file yang tepat</li>
              </ul>
              <h3 id="security-databasus">Keamanan Databasus</h3>
              <p>Databasus menerapkan keamanan pada beberapa tingkat:</p>
              <ul>
                <li>
                  <strong>Enkripsi AES-256-GCM</strong>: Semua password, token, dan kredensial
                  dienkripsi. Kunci enkripsi disimpan terpisah dari database.
                </li>
                <li>
                  <strong>Enkripsi backup unik</strong>: Setiap file backup dienkripsi dengan kunci
                  unik yang diturunkan dari master key, ID backup, dan salt acak.
                </li>
                <li>
                  <strong>Akses database read-only</strong>: Membatasi izin hanya pada SELECT,
                  mencegah kerusakan data sekalipun sistem disusupi.
                </li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "security")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Pelajari keamanan Databasus →
                </a>
              </p>
              <h2 id="restore-process">Proses restore</h2>
              <p>Kedua alat mendukung restore backup, tetapi dengan alur kerja berbeda.</p>
              <h3 id="restore-pgdump">Restore backup pg_dump</h3>
              <p>
                Restore backup <code>pg_dump</code> memerlukan:
              </p>
              <ol>
                <li>Menemukan file backup</li>
                <li>Mendekripsi jika terenkripsi</li>
                <li>Mendekompresi jika terkompresi</li>
                <li>
                  Menjalankan <code>pg_restore</code> atau <code>psql</code> dengan parameter yang
                  benar
                </li>
              </ol>
              <h3 id="restore-databasus">Restore backup Databasus</h3>
              <p>Databasus mempermudah restore:</p>
              <ul>
                <li>
                  <strong>Unduh satu klik</strong>: Unduh backup mana pun langsung dari antarmuka
                  web.
                </li>
                <li>
                  <strong>Dekripsi otomatis</strong>: Backup didekripsi secara otomatis saat
                  diunduh.
                </li>
                <li>
                  <strong>Perintah restore disediakan</strong>: Databasus menampilkan perintah{" "}
                  <code>pg_restore</code> yang tepat untuk setiap backup.
                </li>
                <li>
                  <strong>Dukungan restore paralel</strong>: Gunakan beberapa core CPU untuk
                  mempercepat restore database besar.
                </li>
              </ul>
              <h2 id="installation">Instalasi</h2>
              <h3 id="install-pgdump">Instalasi pg_dump</h3>
              <p>
                <code>pg_dump</code> disertakan bersama PostgreSQL. Jika PostgreSQL sudah terpasang,{" "}
                <code>pg_dump</code> juga tersedia.
              </p>
              <h3 id="install-databasus">Instalasi Databasus</h3>
              <p>Databasus menawarkan beberapa metode instalasi:</p>
              <ul>
                <li>
                  <strong>Script satu baris</strong>: Memasang Docker (jika diperlukan), menyiapkan
                  Databasus, dan mengatur startup otomatis.
                </li>
                <li>
                  <strong>Docker run</strong>: Satu perintah untuk menjalankan aplikasi dengan
                  PostgreSQL bawaan.
                </li>
                <li>
                  <strong>Docker Compose</strong>: Untuk kontrol deployment yang lebih besar.
                </li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "installation")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat panduan instalasi →
                </a>
              </p>
              <h2 id="conclusion">Kesimpulan</h2>
              <p>
                <code>pg_dump</code> adalah utilitas backup PostgreSQL yang sudah teruji, dan
                Databasus dibangun langsung di atasnya. Pilihan antara menggunakan{" "}
                <code>pg_dump</code> langsung atau melalui Databasus bergantung pada kebutuhan kamu.
              </p>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Gunakan pg_dump langsung jika:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>Membutuhkan ekspor database sekali jalan atau sesuai kebutuhan</li>
                  <li>Terbiasa menulis dan memelihara shell script</li>
                  <li>
                    Sudah memiliki infrastruktur otomatisasi (Ansible, Terraform, dan lainnya)
                  </li>
                  <li>Hanya membutuhkan backup lokal tanpa penyimpanan cloud</li>
                  <li>Bekerja sebagai developer tunggal dengan kebutuhan sederhana</li>
                </ul>
              </div>
              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Gunakan Databasus jika:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>Menginginkan backup otomatis terjadwal tanpa menulis script</li>
                  <li>
                    Perlu menyimpan backup di penyimpanan cloud (S3, Google Drive, dan lainnya)
                  </li>
                  <li>Menginginkan enkripsi bawaan tanpa mengelola kunci secara manual</li>
                  <li>Membutuhkan notifikasi saat backup berhasil atau gagal</li>
                  <li>Bekerja dalam tim dan membutuhkan fitur kolaborasi</li>
                  <li>Lebih suka antarmuka visual daripada alat command-line</li>
                  <li>Menginginkan kebijakan retensi dan pembersihan otomatis</li>
                  <li>
                    Membutuhkan backup fisik, backup inkremental, atau Point-in-Time Recovery untuk
                    pemulihan bencana
                  </li>
                </ul>
              </div>
              <p>
                Databasus menggunakan <code>pg_dump</code> untuk backup logis dan melengkapinya
                dengan otomatisasi, keamanan, dan fitur tim. Selain itu, Databasus juga mendukung
                backup fisik, backup inkremental dengan pengarsipan WAL, dan Point-in-Time Recovery,
                kemampuan yang memang tidak dapat disediakan <code>pg_dump</code>.
              </p>
            </article>
          </div>
        </main>
        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
