import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs Barman - Perbandingan alat backup PostgreSQL",
  description:
    "Bandingkan alat backup PostgreSQL Databasus dan Barman. Lihat perbedaan pendekatan backup, kemampuan PITR, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing alat.",
  keywords: [
    "Databasus vs Barman",
    "perbandingan backup PostgreSQL",
    "alternatif Barman",
    "alat backup PostgreSQL",
    "perbandingan backup database",
    "pg_dump vs backup fisik",
    "backup self-hosted",
    "PITR PostgreSQL",
    "pengarsipan WAL",
    "pemulihan bencana PostgreSQL",
  ],
  openGraph: {
    title: "Databasus vs Barman - Perbandingan alat backup PostgreSQL",
    description:
      "Bandingkan alat backup PostgreSQL Databasus dan Barman. Lihat perbedaan pendekatan backup, kemampuan PITR, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing alat.",
    type: "article",
    url: getLocalizedUrl("id", "databasus-vs-barman"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs Barman - Perbandingan alat backup PostgreSQL",
    description:
      "Bandingkan alat backup PostgreSQL Databasus dan Barman. Lihat perbedaan pendekatan backup, kemampuan PITR, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing alat.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "databasus-vs-barman"),
    languages: getLanguageAlternates("databasus-vs-barman"),
  },
  robots: "index, follow",
};

export default function DatabasusVsBarmanPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Databasus vs Barman - Perbandingan alat backup PostgreSQL",
            description:
              "Perbandingan lengkap alat backup PostgreSQL Databasus dan Barman, meliputi pendekatan backup, kemampuan PITR, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing alat.",
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
              <h1 id="databasus-vs-barman">Databasus vs Barman</h1>
              <p className="text-lg text-gray-400">
                Databasus dan Barman sama-sama dirancang untuk pemulihan bencana dengan RTO dan RPO
                minimal. Keduanya mendukung backup fisik, pengarsipan WAL, dan Point-in-Time
                Recovery. Databasus menjalankan backup ini dari jarak jauh dengan perangkat bawaan
                PostgreSQL 17, menggunakan kembali alat PostgreSQL yang sudah teruji alih-alih
                membuat ulang, melalui antarmuka web intuitif dengan fitur tim dan dukungan banyak
                mesin database. Databasus dapat digunakan untuk database dengan ukuran dan
                kompleksitas apa pun. Backup fisik memerlukan PostgreSQL 17 atau lebih baru; pada
                versi lama hanya tersedia backup logis <code>pg_dump</code>. Barman (Backup and
                Recovery Manager) memiliki mesin backup sendiri sehingga mendukung backup fisik pada
                versi PostgreSQL yang jauh lebih lama, serta menyediakan fitur lanjutan seperti
                backup inkremental berbasis rsync, integrasi streaming replication, dan redundansi
                geografis antarsesama Barman.
              </p>
              <h2 id="quick-comparison">Perbandingan singkat</h2>
              <p>Berikut ringkasan perbedaan utama Databasus dan Barman:</p>
              <table>
                <thead>
                  <tr>
                    <th>Fitur</th>
                    <th>Databasus</th>
                    <th>Barman</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Sasaran pengguna</td>
                    <td data-label="Databasus">Individu, tim, DBA, perusahaan</td>
                    <td data-label="Barman">DBA, perusahaan</td>
                  </tr>
                  <tr>
                    <td>Dukungan database lain</td>
                    <td data-label="Databasus">✅ PostgreSQL, MySQL, MariaDB, MongoDB</td>
                    <td data-label="Barman">❌ Hanya PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Antarmuka</td>
                    <td data-label="Databasus">Antarmuka web</td>
                    <td data-label="Barman">Hanya command-line</td>
                  </tr>
                  <tr>
                    <td>Jenis backup</td>
                    <td data-label="Databasus">Logis + Fisik</td>
                    <td data-label="Barman">Fisik (tingkat file)</td>
                  </tr>
                  <tr>
                    <td>Versi PostgreSQL untuk backup fisik</td>
                    <td data-label="Databasus">17+ (bawaan)</td>
                    <td data-label="Barman">9.x+ (mesin sendiri)</td>
                  </tr>
                  <tr>
                    <td>Opsi pemulihan</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="Barman">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Backup inkremental</td>
                    <td data-label="Databasus">✅ Tingkat blok (PG 17+)</td>
                    <td data-label="Barman">Inkremental berbasis rsync</td>
                  </tr>
                  <tr>
                    <td>Backup jarak jauh</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="Barman">❌ Tidak (memerlukan akses filesystem)</td>
                  </tr>
                  <tr>
                    <td>Pengelolaan banyak server</td>
                    <td data-label="Databasus">Penjadwalan per database</td>
                    <td data-label="Barman">Server backup terpusat</td>
                  </tr>
                  <tr>
                    <td>Fitur tim</td>
                    <td data-label="Databasus">✅ Ruang kerja, RBAC, log audit</td>
                    <td data-label="Barman">❌ Hanya izin tingkat OS</td>
                  </tr>
                  <tr>
                    <td>Notifikasi</td>
                    <td data-label="Databasus">✅ Slack, Teams, Telegram, Email</td>
                    <td data-label="Barman">❌ Memerlukan script khusus</td>
                  </tr>
                  <tr>
                    <td>Kurva belajar</td>
                    <td data-label="Databasus">Minimal</td>
                    <td data-label="Barman">Memerlukan keahlian DBA</td>
                  </tr>
                  <tr>
                    <td>Instalasi</td>
                    <td data-label="Databasus">Script satu baris atau Docker</td>
                    <td data-label="Barman">Memerlukan konfigurasi manual</td>
                  </tr>
                  <tr>
                    <td>Pengelolaan backup</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="Barman">❌ Tidak</td>
                  </tr>
                  <tr>
                    <td>Cocok untuk database self-hosted</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="Barman">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Cocok untuk database cloud</td>
                    <td data-label="Databasus">✅ Ya (RDS, Cloud SQL, Azure)</td>
                    <td data-label="Barman">❌ Tidak (memerlukan akses filesystem)</td>
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
              <h3 id="audience-barman">Pengguna Barman</h3>
              <p>
                Barman dirancang khusus untuk Database Administrator (DBA) yang mengelola
                infrastruktur PostgreSQL perusahaan:
              </p>
              <ul>
                <li>
                  <strong>DBA perusahaan</strong>: Profesional yang memerlukan pengelolaan backup
                  terpusat untuk banyak server PostgreSQL dari server backup khusus.
                </li>
                <li>
                  <strong>Tim yang memerlukan inkremental berbasis rsync</strong>: Perbandingan
                  perubahan tingkat file mengurangi waktu backup dan penggunaan jaringan untuk
                  cluster besar.
                </li>
                <li>
                  <strong>Kebutuhan redundansi geografis</strong>: Replikasi Barman-to-Barman untuk
                  redundansi geografis antarpusat data.
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
                  portabel yang bisa di-restore ke versi PostgreSQL berbeda. Ini juga satu-satunya
                  jenis backup pada PostgreSQL sebelum 17, serta metode untuk MySQL, MariaDB, dan
                  MongoDB.
                </li>
                <li>
                  <strong>Tanpa instalasi di database</strong>: Backup terhubung dari jarak jauh;
                  jaringan tertutup dijangkau melalui SSH tunnel ke host internal atau bastion, jadi
                  database tidak perlu dibuka ke publik.
                </li>
                <li>
                  <strong>Kompresi efisien</strong>: Menggunakan kompresi zstd (level 5) untuk
                  backup logis maupun fisik.
                </li>
                <li>
                  <strong>Akses read-only</strong>: Backup logis hanya memerlukan izin SELECT
                  sehingga risiko keamanan lebih kecil.
                </li>
              </ul>
              <h3 id="backup-barman">Barman: Backup fisik</h3>
              <p>Barman membuat backup tingkat file (fisik) dari direktori data PostgreSQL:</p>
              <ul>
                <li>
                  <strong>Backup seluruh cluster</strong>: Menyalin seluruh cluster database pada
                  tingkat filesystem menggunakan rsync atau pg_basebackup.
                </li>
                <li>
                  <strong>Pengarsipan WAL</strong>: Mengarsipkan Write-Ahead Logs secara
                  berkelanjutan untuk Point-in-Time Recovery.
                </li>
                <li>
                  <strong>Inkremental dengan rsync</strong>: Menggunakan rsync untuk mentransfer
                  hanya file yang berubah, mengurangi waktu backup dan penggunaan jaringan.
                </li>
                <li>
                  <strong>Integrasi streaming replication</strong>: Dapat menerima file WAL melalui
                  protokol streaming replication untuk pengarsipan real-time.
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
              <h3 id="recovery-barman">Pemulihan Barman</h3>
              <ul>
                <li>
                  <strong>Point-in-Time Recovery (PITR)</strong>: Restore ke detik tertentu dengan
                  memutar ulang WAL, meminimalkan kehilangan data.
                </li>
                <li>
                  <strong>Restore seluruh cluster</strong>: Restore seluruh cluster database ke
                  waktu tertentu.
                </li>
                <li>
                  <strong>Pemulihan jarak jauh</strong>: Pulihkan database ke server jarak jauh
                  melalui SSH.
                </li>
                <li>
                  <strong>Pembuatan standby</strong>: Membuat replika PostgreSQL dari backup untuk
                  konfigurasi ketersediaan tinggi.
                </li>
              </ul>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Catatan:</strong> Kedua alat mendukung PITR.
                  Barman juga menawarkan pembuatan standby dari backup dan pemulihan jarak jauh
                  berbasis SSH ke server lain, yang bisa berguna untuk konfigurasi ketersediaan
                  tinggi.{" "}
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
              <h3 id="ease-barman">Pengalaman menggunakan Barman</h3>
              <ul>
                <li>
                  <strong>Antarmuka command-line</strong>: Semua operasi dijalankan lewat perintah
                  terminal seperti <code>barman backup</code>, <code>barman recover</code>.
                </li>
                <li>
                  <strong>File konfigurasi</strong>: Memerlukan pengeditan manual file konfigurasi
                  bergaya INI untuk setiap server.
                </li>
                <li>
                  <strong>Penyiapan pengarsipan WAL</strong>: Harus mengatur{" "}
                  <code>archive_command</code> PostgreSQL atau pengaturan streaming replication.
                </li>
                <li>
                  <strong>Pengelolaan kunci SSH</strong>: Memerlukan penyiapan kunci SSH antara
                  server Barman dan server PostgreSQL.
                </li>
                <li>
                  <strong>Keahlian DBA diharapkan</strong>: Dokumentasi mengasumsikan pembaca sudah
                  memahami internal PostgreSQL dan mekanisme WAL.
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
              <h3 id="team-barman">Kemampuan Barman untuk tim</h3>
              <p>Barman adalah alat command-line tanpa fitur tim bawaan:</p>
              <ul>
                <li>Tidak ada pengelolaan pengguna atau kontrol akses</li>
                <li>Tidak ada pencatatan audit operasi</li>
                <li>Koordinasi tim memerlukan alat dan proses eksternal</li>
                <li>Akses dikontrol melalui izin tingkat OS dan kunci SSH</li>
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
              <p>Kedua alat menyediakan fitur keamanan, tetapi dengan pendekatan berbeda:</p>
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
              <h3 id="security-barman">Keamanan Barman</h3>
              <ul>
                <li>
                  <strong>Komunikasi berbasis SSH</strong>: Menggunakan SSH untuk komunikasi aman
                  antara server Barman dan server PostgreSQL.
                </li>
                <li>
                  <strong>Tanpa enkripsi bawaan</strong>: Barman tidak menyediakan enkripsi backup
                  bawaan. Perlu menggunakan alat eksternal atau penyimpanan terenkripsi.
                </li>
                <li>
                  <strong>Keamanan tingkat OS</strong>: Mengandalkan izin filesystem dan pengelolaan
                  kunci SSH untuk kontrol akses.
                </li>
                <li>
                  <strong>Verifikasi checksum</strong>: Memvalidasi integritas backup menggunakan
                  checksum.
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
              <p>Kedua alat mendukung tujuan penyimpanan yang berbeda:</p>
              <h3 id="storage-databasus">Penyimpanan Databasus</h3>
              <p>Pilihan yang ramah pengguna untuk berbagai kebutuhan:</p>
              <ul>
                <li>Penyimpanan lokal</li>
                <li>Amazon S3 dan layanan kompatibel S3</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>
              <h3 id="storage-barman">Penyimpanan Barman</h3>
              <p>Pilihan penyimpanan yang berfokus pada perusahaan:</p>
              <ul>
                <li>Penyimpanan lokal (filesystem POSIX)</li>
                <li>Amazon S3 dan object storage kompatibel S3</li>
                <li>Redundansi geografis melalui replikasi Barman-to-Barman</li>
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
              <h3 id="notifications-barman">Notifikasi Barman</h3>
              <p>Barman tidak memiliki dukungan notifikasi bawaan. Notifikasi memerlukan:</p>
              <ul>
                <li>Script khusus yang membungkus perintah backup</li>
                <li>Integrasi dengan alat pemantauan eksternal</li>
                <li>Parsing log dan penyiapan peringatan secara manual</li>
                <li>Integrasi dengan alat seperti Nagios, Zabbix, atau solusi khusus</li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua kanal notifikasi Databasus →
                </a>
              </p>
              <h2 id="multi-server-management">Pengelolaan banyak server</h2>
              <p>
                Kedua alat dapat mengelola backup untuk banyak server PostgreSQL, tetapi
                pendekatannya berbeda:
              </p>
              <h3 id="multi-databasus">Pendekatan Databasus</h3>
              <ul>
                <li>
                  <strong>Penjadwalan per database</strong>: Setiap database dapat memiliki jadwal
                  backup dan tujuan penyimpanan sendiri.
                </li>
                <li>
                  <strong>Pengaturan ruang kerja</strong>: Kelompokkan database terkait ke dalam
                  ruang kerja agar lebih mudah dikelola.
                </li>
                <li>
                  <strong>Dashboard terpadu</strong>: Lihat semua backup database beserta statusnya
                  dalam satu antarmuka web.
                </li>
              </ul>
              <h3 id="multi-barman">Pendekatan Barman</h3>
              <ul>
                <li>
                  <strong>Server backup terpusat</strong>: Server Barman khusus mengelola backup
                  untuk banyak instance PostgreSQL.
                </li>
                <li>
                  <strong>Konfigurasi per server</strong>: Setiap server PostgreSQL memerlukan file
                  konfigurasi sendiri di server Barman.
                </li>
                <li>
                  <strong>Redundansi geografis</strong>: Server Barman dapat bereplikasi ke server
                  Barman lain untuk redundansi geografis.
                </li>
              </ul>
              <h2 id="conclusion">Kesimpulan</h2>
              <p>
                Databasus dan Barman memenuhi kebutuhan berbeda dalam ekosistem backup PostgreSQL.
                Pilihan yang tepat bergantung pada kebutuhan pemulihan, struktur tim, dan keahlian
                teknis.
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
                  <li>Enkripsi backup bawaan penting bagi kamu</li>
                  <li>
                    Menggunakan database cloud terkelola (AWS RDS, Google Cloud SQL, Azure) atau
                    PostgreSQL self-hosted
                  </li>
                </ul>
              </div>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Pilih Barman jika:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Membutuhkan backup fisik atau inkremental pada PostgreSQL sebelum 17 (Barman
                    memiliki mesin backup sendiri)
                  </li>
                  <li>
                    Membutuhkan backup inkremental berbasis rsync (perbandingan perubahan tingkat
                    file) untuk mengurangi waktu transfer
                  </li>
                  <li>
                    Membutuhkan integrasi streaming replication untuk pengarsipan WAL real-time
                  </li>
                  <li>Membutuhkan redundansi geografis Barman-to-Barman</li>
                  <li>
                    Membutuhkan pembuatan standby dari backup untuk konfigurasi ketersediaan tinggi
                  </li>
                  <li>Terbiasa dengan alat command-line dan internal PostgreSQL</li>
                  <li>Organisasi memiliki tenaga dengan keahlian DBA khusus</li>
                </ul>
              </div>
              <p>
                Kedua alat mendukung backup fisik, pengarsipan WAL, dan PITR, serta dirancang untuk
                pemulihan bencana dengan RTO dan RPO minimal. Databasus dapat digunakan untuk
                database dengan ukuran dan kompleksitas apa pun, serta menyediakan antarmuka web,
                fitur tim, dan backup logis maupun fisik untuk database self-hosted dan cloud
                terkelola. Barman lebih cocok jika kamu memerlukan backup inkremental berbasis
                rsync, integrasi streaming replication, redundansi geografis Barman-to-Barman, atau
                pembuatan standby dari backup.
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
