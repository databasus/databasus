import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs pgBackRest - Perbandingan alat backup PostgreSQL",
  description:
    "Bandingkan alat backup PostgreSQL Databasus dan pgBackRest. Lihat perbedaan pendekatan backup, sasaran pengguna, kemudahan penggunaan, opsi pemulihan, dan kapan memilih masing-masing alat.",
  keywords: [
    "Databasus vs pgBackRest",
    "perbandingan backup PostgreSQL",
    "alternatif pgBackRest",
    "alat backup PostgreSQL",
    "perbandingan backup database",
    "pg_dump vs backup fisik",
    "backup self-hosted",
    "PITR PostgreSQL",
    "backup database besar",
    "alat backup DBA",
  ],
  openGraph: {
    title: "Databasus vs pgBackRest - Perbandingan alat backup PostgreSQL",
    description:
      "Bandingkan alat backup PostgreSQL Databasus dan pgBackRest. Lihat perbedaan pendekatan backup, sasaran pengguna, kemudahan penggunaan, opsi pemulihan, dan kapan memilih masing-masing alat.",
    type: "article",
    url: getLocalizedUrl("id", "databasus-vs-pgbackrest"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs pgBackRest - Perbandingan alat backup PostgreSQL",
    description:
      "Bandingkan alat backup PostgreSQL Databasus dan pgBackRest. Lihat perbedaan pendekatan backup, sasaran pengguna, kemudahan penggunaan, opsi pemulihan, dan kapan memilih masing-masing alat.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "databasus-vs-pgbackrest"),
    languages: getLanguageAlternates("databasus-vs-pgbackrest"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackRestPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Databasus vs pgBackRest - Perbandingan alat backup PostgreSQL",
            description:
              "Perbandingan lengkap alat backup PostgreSQL Databasus dan pgBackRest, meliputi pendekatan backup, sasaran pengguna, kemudahan penggunaan, opsi pemulihan, dan kapan memilih masing-masing alat.",
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
              <h1 id="databasus-vs-pgbackrest">Databasus vs pgBackRest</h1>
              <p className="text-lg text-gray-400">
                Databasus dan pgBackRest sama-sama dirancang untuk pemulihan bencana dengan RTO dan
                RPO minimal. Keduanya mendukung backup fisik, pengarsipan WAL, dan Point-in-Time
                Recovery. Databasus menjalankan backup ini dari jarak jauh dengan perangkat bawaan
                PostgreSQL 17, menggunakan kembali alat PostgreSQL yang sudah teruji alih-alih
                membuat ulang, melalui antarmuka web yang intuitif. Databasus dapat digunakan untuk
                database dengan ukuran dan kompleksitas apa pun. Backup fisik memerlukan PostgreSQL
                17 atau lebih baru; pada versi lama hanya tersedia backup logis <code>pg_dump</code>
                . pgBackRest memiliki mesin backup sendiri sehingga juga mendukung backup fisik dan
                inkremental pada versi PostgreSQL yang jauh lebih lama, serta menyediakan fitur
                lanjutan seperti backup diferensial dan delta restore.
              </p>
              <h2 id="quick-comparison">Perbandingan singkat</h2>
              <p>Berikut ringkasan perbedaan utama Databasus dan pgBackRest:</p>
              <table>
                <thead>
                  <tr>
                    <th>Fitur</th>
                    <th>Databasus</th>
                    <th>pgBackRest</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Sasaran pengguna</td>
                    <td data-label="Databasus">Individu, tim, DBA, perusahaan</td>
                    <td data-label="pgBackRest">DBA, alur kerja command-line dan IaC</td>
                  </tr>
                  <tr>
                    <td>Pengelolaan backup</td>
                    <td data-label="Databasus">✅ Banyak database</td>
                    <td data-label="pgBackRest">❌ Hanya satu database</td>
                  </tr>
                  <tr>
                    <td>Dukungan database lain</td>
                    <td data-label="Databasus">✅ PostgreSQL, MySQL, MariaDB, MongoDB</td>
                    <td data-label="pgBackRest">❌ Hanya PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Antarmuka</td>
                    <td data-label="Databasus">Antarmuka web</td>
                    <td data-label="pgBackRest">Command-line, file konfigurasi</td>
                  </tr>
                  <tr>
                    <td>Jenis backup</td>
                    <td data-label="Databasus">Logis + Fisik</td>
                    <td data-label="pgBackRest">Fisik (tingkat file)</td>
                  </tr>
                  <tr>
                    <td>Versi PostgreSQL untuk backup fisik</td>
                    <td data-label="Databasus">17+ (bawaan)</td>
                    <td data-label="pgBackRest">9.4+ (mesin sendiri)</td>
                  </tr>
                  <tr>
                    <td>Opsi pemulihan</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="pgBackRest">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Operasi paralel</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="pgBackRest">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Backup inkremental</td>
                    <td data-label="Databasus">✅ Tingkat blok (PG 17+)</td>
                    <td data-label="pgBackRest">Inkremental tingkat blok</td>
                  </tr>
                  <tr>
                    <td>Backup diferensial</td>
                    <td data-label="Databasus">❌ Tidak</td>
                    <td data-label="pgBackRest">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Delta restore</td>
                    <td data-label="Databasus">❌ Tidak</td>
                    <td data-label="pgBackRest">✅ Ya (hanya file yang berubah)</td>
                  </tr>
                  <tr>
                    <td>Backup jarak jauh</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="pgBackRest">❌ Tidak (memerlukan akses filesystem)</td>
                  </tr>
                  <tr>
                    <td>Fitur tim</td>
                    <td data-label="Databasus">✅ Ruang kerja, RBAC, log audit</td>
                    <td data-label="pgBackRest">❌ Satu pengguna</td>
                  </tr>
                  <tr>
                    <td>Kurva belajar</td>
                    <td data-label="Databasus">Minimal</td>
                    <td data-label="pgBackRest">Memerlukan keahlian DBA</td>
                  </tr>
                  <tr>
                    <td>Instalasi</td>
                    <td data-label="Databasus">Script satu baris atau Docker</td>
                    <td data-label="pgBackRest">Memerlukan konfigurasi manual</td>
                  </tr>
                  <tr>
                    <td>Cocok untuk database self-hosted</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="pgBackRest">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Cocok untuk database cloud</td>
                    <td data-label="Databasus">✅ Ya (RDS, Cloud SQL, Azure)</td>
                    <td data-label="pgBackRest">❌ Tidak (memerlukan akses filesystem)</td>
                  </tr>
                </tbody>
              </table>
              <h2 id="target-audience">Sasaran pengguna</h2>
              <p>Perbedaan paling besar antara kedua alat ini adalah siapa yang dituju:</p>
              <h3 id="audience-databasus">Pengguna Databasus</h3>
              <p>
                Databasus dibuat untuk beragam pengguna, dari developer individu hingga perusahaan
                besar:
              </p>
              <ul>
                <li>
                  <strong>Developer individu</strong>: Penyiapan sederhana dan UI intuitif
                  memudahkan perlindungan proyek pribadi tanpa keahlian PostgreSQL mendalam.
                </li>
                <li>
                  <strong>Tim pengembangan</strong>: Ruang kerja, kontrol akses berbasis peran, dan
                  log audit memungkinkan kolaborasi yang aman antaranggota tim.
                </li>
                <li>
                  <strong>Perusahaan</strong>: Dapat ditingkatkan untuk memenuhi kebutuhan
                  perusahaan dengan keamanan menyeluruh, banyak tujuan penyimpanan, dan kanal
                  notifikasi.
                </li>
                <li>
                  <strong>DBA dan pemulihan bencana</strong>: Backup fisik, pengarsipan WAL, dan
                  PITR untuk sistem kritis yang menuntut kehilangan data mendekati nol.
                </li>
              </ul>
              <h3 id="audience-pgbackrest">Pengguna pgBackRest</h3>
              <p>
                pgBackRest adalah alat command-line untuk tim yang ingin menjalankan sendiri mesin
                backup:
              </p>
              <ul>
                <li>
                  <strong>Alur kerja command-line dan IaC</strong>: Tim yang lebih suka mengatur
                  backup melalui file dan script daripada antarmuka web.
                </li>
                <li>
                  <strong>Versi PostgreSQL lama</strong>: pgBackRest memiliki mesin sendiri sehingga
                  dapat menjalankan backup fisik dan inkremental pada versi PostgreSQL sebelum 17.
                </li>
                <li>
                  <strong>Fitur lanjutan untuk skala besar</strong>: Saat manfaat backup
                  diferensial, delta restore, dan pembuatan standby sepadan dengan tambahan
                  penyiapannya.
                </li>
              </ul>
              <h2 id="backup-approach">Pendekatan backup</h2>
              <p>
                Kedua alat menggunakan strategi backup yang berbeda secara mendasar, dengan
                keunggulan masing-masing:
              </p>
              <h3 id="backup-databasus">Databasus: Backup logis + fisik</h3>
              <p>Databasus mendukung strategi backup logis dan fisik:</p>
              <ul>
                <li>
                  <strong>Backup fisik, inkremental, dan WAL</strong>: Dijalankan dari jarak jauh
                  melalui protokol replikasi PostgreSQL dengan perangkat bawaan PostgreSQL 17:{" "}
                  <code>pg_basebackup</code>, <code>pg_basebackup --incremental</code> tingkat blok
                  yang menggunakan ringkasan WAL dari server, <code>pg_receivewal</code>, dan{" "}
                  <code>pg_combinebackup</code>. Databasus menggunakan kembali alat PostgreSQL yang
                  sudah teruji alih-alih membuat ulang. Memerlukan PostgreSQL 17 atau lebih baru.
                </li>
                <li>
                  <strong>Backup logis</strong>: Menggunakan <code>pg_dump</code> untuk backup
                  portabel yang bisa di-restore ke versi PostgreSQL berbeda, dengan restore selektif
                  untuk tabel atau skema tertentu. Ini juga satu-satunya jenis backup pada
                  PostgreSQL sebelum 17, serta metode untuk MySQL, MariaDB, dan MongoDB.
                </li>
                <li>
                  <strong>Tanpa instalasi di database</strong>: Backup terhubung dari jarak jauh;
                  jaringan tertutup dijangkau melalui SSH tunnel ke host internal atau bastion, jadi
                  database tidak perlu dibuka ke publik.
                </li>
                <li>
                  <strong>Kompresi efisien</strong>: Menggunakan zstd (level 5) untuk kedua jenis
                  backup, memperkecil ukuran 4-8x.
                </li>
                <li>
                  <strong>Akses read-only</strong>: Backup logis hanya memerlukan izin SELECT
                  sehingga risiko keamanan lebih kecil.
                </li>
              </ul>
              <h3 id="backup-pgbackrest">pgBackRest: Backup fisik</h3>
              <p>pgBackRest membuat backup tingkat file (fisik) dari direktori data PostgreSQL:</p>
              <ul>
                <li>
                  <strong>Inkremental tingkat blok</strong>: Hanya blok yang berubah yang di-backup,
                  mengurangi waktu backup dan kebutuhan penyimpanan untuk database sangat besar.
                </li>
                <li>
                  <strong>Pengarsipan WAL</strong>: Pengarsipan Write-Ahead Logs secara
                  berkelanjutan memungkinkan Point-in-Time Recovery yang presisi.
                </li>
                <li>
                  <strong>Penuh, diferensial, inkremental</strong>: Beragam strategi backup untuk
                  berbagai skenario pemulihan.
                </li>
                <li>
                  <strong>Dioptimalkan untuk skala besar</strong>: Dirancang untuk database yang
                  backup logisnya akan memakan waktu terlalu lama.
                </li>
              </ul>
              <h2 id="recovery-options">Opsi pemulihan</h2>
              <p>
                Kedua alat menawarkan opsi pemulihan yang fleksibel, tetapi tingkat rinciannya
                berbeda:
              </p>
              <h3 id="recovery-databasus">Pemulihan Databasus</h3>
              <ul>
                <li>
                  <strong>Point-in-Time Recovery</strong>: Restore ke detik tertentu dengan memutar
                  ulang WAL.
                </li>
                <li>
                  <strong>Restore seluruh cluster</strong>: Restore seluruh cluster database ke
                  waktu tertentu dari backup fisik.
                </li>
                <li>
                  <strong>Restore logis</strong>: Restore dari backup logis terjadwal ke titik
                  backup mana pun.
                </li>
                <li>
                  <strong>Restore satu klik</strong>: Unduh dan restore backup logis langsung dari
                  antarmuka web.
                </li>
                <li>
                  <strong>Kompatibilitas lintas versi</strong>: Backup logis bisa di-restore ke
                  versi PostgreSQL berbeda.
                </li>
              </ul>
              <h3 id="recovery-pgbackrest">Pemulihan pgBackRest</h3>
              <ul>
                <li>
                  <strong>Point-in-Time Recovery (PITR)</strong>: Restore ke detik tertentu dengan
                  memutar ulang WAL.
                </li>
                <li>
                  <strong>Restore seluruh cluster</strong>: Restore seluruh cluster database ke
                  waktu tertentu dari backup fisik.
                </li>
                <li>
                  <strong>Restore paralel</strong>: Restore multi-thread untuk mempercepat pemulihan
                  database besar.
                </li>
                <li>
                  <strong>Delta restore</strong>: Hanya me-restore file yang berubah untuk
                  mengurangi waktu pemulihan.
                </li>
                <li>
                  <strong>Pembuatan standby</strong>: Membuat replika PostgreSQL dari backup.
                </li>
              </ul>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Catatan:</strong> Kedua alat mendukung PITR.
                  pgBackRest juga menawarkan delta restore (hanya mengambil file yang berubah),
                  backup diferensial, dan pembuatan standby dari backup.{" "}
                  <a
                    href={`${getLocalizedHref("id", "faq")}#pitr`}
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Pelajari dukungan PITR di Databasus →
                  </a>
                </p>
              </div>
              <h2 id="ease-of-use">Kemudahan penggunaan</h2>
              <p>
                Kedua alat memiliki pendekatan yang sangat berbeda terhadap pengalaman pengguna:
              </p>
              <h3 id="ease-databasus">Pengalaman menggunakan Databasus</h3>
              <ul>
                <li>
                  <strong>Antarmuka web</strong>: Semua pengaturan backup dapat dikonfigurasi dengan
                  klik. Tidak perlu command-line.
                </li>
                <li>
                  <strong>Instalasi 2 menit</strong>: Script cURL satu baris atau perintah Docker
                  sederhana langsung menjalankan aplikasi.
                </li>
                <li>
                  <strong>Pemantauan visual</strong>: Dashboard menampilkan status backup,
                  pemeriksaan ketersediaan, dan riwayat sekilas.
                </li>
                <li>
                  <strong>Notifikasi bawaan</strong>: Atur peringatan Slack, Teams, Telegram, Email,
                  atau webhook langsung di UI.
                </li>
                <li>
                  <strong>Tidak perlu keahlian PostgreSQL</strong>: Dirancang untuk developer yang
                  menginginkan backup andal tanpa harus menjadi ahli database.
                </li>
              </ul>
              <h3 id="ease-pgbackrest">Pengalaman menggunakan pgBackRest</h3>
              <ul>
                <li>
                  <strong>Antarmuka command-line</strong>: Semua operasi dijalankan lewat perintah
                  terminal.
                </li>
                <li>
                  <strong>File konfigurasi</strong>: Memerlukan pengeditan manual file konfigurasi
                  bergaya INI.
                </li>
                <li>
                  <strong>Penyiapan pengarsipan WAL</strong>: Harus mengatur{" "}
                  <code>archive_command</code> PostgreSQL dan pengaturan terkait.
                </li>
                <li>
                  <strong>Kurva belajar curam</strong>: Memerlukan pemahaman internal PostgreSQL,
                  mekanisme WAL, dan strategi backup.
                </li>
                <li>
                  <strong>Keahlian DBA diharapkan</strong>: Dokumentasi mengasumsikan pembaca sudah
                  memahami konsep administrasi database.
                </li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "installation")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat panduan instalasi Databasus →
                </a>
              </p>
              <h2 id="team-features">Fitur tim</h2>
              <p>Untuk organisasi dengan beberapa anggota tim yang mengelola backup:</p>
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
              <h3 id="team-pgbackrest">Kemampuan pgBackRest untuk tim</h3>
              <p>pgBackRest adalah alat command-line tanpa fitur tim bawaan:</p>
              <ul>
                <li>Tidak ada pengelolaan pengguna atau kontrol akses</li>
                <li>Tidak ada pencatatan audit operasi</li>
                <li>Koordinasi tim memerlukan alat dan proses eksternal</li>
                <li>Akses dikontrol melalui izin tingkat OS pada file konfigurasi</li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "access-management")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Pelajari pengelolaan akses Databasus →
                </a>
              </p>
              <h2 id="security">Keamanan</h2>
              <p>Kedua alat menyediakan fitur keamanan yang kuat:</p>
              <h3 id="security-databasus">Keamanan Databasus</h3>
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
              <h3 id="security-pgbackrest">Keamanan pgBackRest</h3>
              <ul>
                <li>
                  <strong>Enkripsi repositori</strong>: Repositori backup dapat dienkripsi dengan
                  AES-256.
                </li>
                <li>
                  <strong>Transport TLS/SSH</strong>: Komunikasi aman untuk operasi jarak jauh.
                </li>
                <li>
                  <strong>Verifikasi checksum</strong>: Memvalidasi integritas backup saat pembuatan
                  dan restore.
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
              <h2 id="storage-options">Pilihan penyimpanan</h2>
              <p>
                Kedua alat mendukung banyak tujuan penyimpanan, dengan Databasus menawarkan lebih
                banyak pilihan yang ramah pengguna:
              </p>
              <h3 id="storage-databasus">Penyimpanan Databasus</h3>
              <ul>
                <li>Penyimpanan lokal</li>
                <li>Amazon S3 dan layanan kompatibel S3</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>
              <h3 id="storage-pgbackrest">Penyimpanan pgBackRest</h3>
              <ul>
                <li>Penyimpanan lokal (POSIX, CIFS)</li>
                <li>Amazon S3 dan layanan kompatibel S3</li>
                <li>Cloudflare R2 (kompatibel S3)</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Google Cloud Storage</li>
                <li>SFTP</li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "storages")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua pilihan penyimpanan Databasus →
                </a>
              </p>
              <h2 id="notifications">Notifikasi</h2>
              <p>Cara mengikuti status backup:</p>
              <h3 id="notifications-databasus">Notifikasi Databasus</h3>
              <p>Dukungan bawaan untuk berbagai kanal notifikasi:</p>
              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Webhooks</li>
              </ul>
              <h3 id="notifications-pgbackrest">Notifikasi pgBackRest</h3>
              <p>pgBackRest tidak memiliki dukungan notifikasi bawaan. Notifikasi memerlukan:</p>
              <ul>
                <li>Script khusus yang membungkus perintah backup</li>
                <li>Integrasi dengan alat pemantauan eksternal</li>
                <li>Parsing log dan penyiapan peringatan secara manual</li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua kanal notifikasi Databasus →
                </a>
              </p>
              <h2 id="conclusion">Kesimpulan</h2>
              <p>
                Databasus dan pgBackRest memenuhi kebutuhan berbeda dalam ekosistem backup
                PostgreSQL. Pilihan yang tepat bergantung pada ukuran database, struktur tim, dan
                kebutuhan teknis.
              </p>
              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Pilih Databasus jika:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Kamu developer individu, tim, atau perusahaan yang mencari solusi backup
                    intuitif
                  </li>
                  <li>Lebih suka antarmuka web daripada alat command-line</li>
                  <li>Membutuhkan fitur kolaborasi tim (ruang kerja, RBAC, log audit)</li>
                  <li>Menginginkan notifikasi bawaan ke Slack, Teams, Telegram, dan lainnya</li>
                  <li>
                    Ingin mengelola backup banyak database dari satu dashboard dengan penjadwalan,
                    notifikasi, dan fitur tim
                  </li>
                  <li>Menginginkan penyiapan cepat dengan pengetahuan PostgreSQL minimal</li>
                  <li>
                    Menggunakan database cloud terkelola (AWS RDS, Google Cloud SQL, Azure) atau
                    PostgreSQL self-hosted
                  </li>
                </ul>
              </div>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Pilih pgBackRest jika:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Membutuhkan backup fisik atau inkremental pada PostgreSQL sebelum 17 (pgBackRest
                    memiliki mesin backup sendiri)
                  </li>
                  <li>
                    Membutuhkan backup diferensial atau delta restore (hanya file yang berubah)
                  </li>
                  <li>Membutuhkan pembuatan standby dari backup untuk ketersediaan tinggi</li>
                  <li>Lebih suka alat command-line dan alur kerja infrastructure-as-code</li>
                  <li>Tim memiliki keahlian PostgreSQL untuk menjalankan dan menyesuaikannya</li>
                </ul>
              </div>
              <p>
                Kedua alat mendukung backup fisik, pengarsipan WAL, dan PITR, serta dirancang untuk
                pemulihan bencana dengan RTO dan RPO minimal. Databasus dapat digunakan untuk
                database dengan ukuran dan kompleksitas apa pun, serta menyediakan antarmuka web,
                fitur tim, dan backup logis maupun fisik untuk database self-hosted dan cloud
                terkelola.
                <br />
                <br />
                pgBackRest lebih cocok jika kamu ingin menjalankan sendiri mesin backup, memerlukan
                backup fisik pada PostgreSQL sebelum 17, atau mengandalkan backup diferensial dan
                delta restore.
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
