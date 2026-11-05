import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs WAL-G - Perbandingan Tool Backup PostgreSQL",
  description:
    "Bandingkan tool backup PostgreSQL Databasus dan WAL-G. Pelajari perbedaan pendekatan backup, dukungan berbagai database, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing tool.",
  keywords: [
    "Databasus vs WAL-G",
    "perbandingan backup PostgreSQL",
    "alternatif WAL-G",
    "tool backup PostgreSQL",
    "perbandingan backup database",
    "pg_dump vs pengarsipan WAL",
    "backup self-hosted",
    "PITR PostgreSQL",
    "pengarsipan WAL",
    "backup berbagai database",
  ],
  openGraph: {
    title: "Databasus vs WAL-G - Perbandingan Tool Backup PostgreSQL",
    description:
      "Bandingkan tool backup PostgreSQL Databasus dan WAL-G. Pelajari perbedaan pendekatan backup, dukungan berbagai database, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing tool.",
    type: "article",
    url: getLocalizedUrl("id", "databasus-vs-wal-g"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs WAL-G - Perbandingan Tool Backup PostgreSQL",
    description:
      "Bandingkan tool backup PostgreSQL Databasus dan WAL-G. Pelajari perbedaan pendekatan backup, dukungan berbagai database, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing tool.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "databasus-vs-wal-g"),
    languages: getLanguageAlternates("databasus-vs-wal-g"),
  },
  robots: "index, follow",
};

export default function DatabasusVsWalGPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Databasus vs WAL-G - Perbandingan Tool Backup PostgreSQL",
            description:
              "Perbandingan menyeluruh tool backup PostgreSQL Databasus dan WAL-G, meliputi pendekatan backup, dukungan berbagai database, kemudahan penggunaan, fitur tim, dan kapan memilih masing-masing tool.",
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
              <h1 id="databasus-vs-wal-g">Databasus vs WAL-G</h1>
              <p className="text-lg text-gray-400">
                Databasus dan WAL-G sama-sama dirancang untuk pemulihan bencana dengan RTO dan RPO
                minimal. Keduanya mendukung backup physical PostgreSQL, pengarsipan WAL, dan
                Point-in-Time Recovery. Databasus menjalankan backup dari jarak jauh menggunakan
                tool native PostgreSQL 17 yang sudah teruji, bukan membuat ulang mekanismenya,
                melalui antarmuka web yang mudah digunakan. Databasus bisa digunakan untuk database
                dengan ukuran dan kompleksitas apa pun. Backup physical memerlukan PostgreSQL 17
                atau yang lebih baru; pada versi sebelumnya, hanya backup logical{" "}
                <code>pg_dump</code> yang tersedia. WAL-G adalah tool command-line dengan engine
                sendiri, sehingga mendukung backup physical pada versi PostgreSQL yang jauh lebih
                lama. WAL-G menggunakan protokol streaming khusus untuk performa yang sedikit lebih
                baik, mendukung backup delta (hanya page yang berubah), dan mendukung lebih banyak
                engine database, termasuk MS SQL, FoundationDB, dan Greenplum.
              </p>
              <h2 id="quick-comparison">Perbandingan singkat</h2>
              <p>Berikut ringkasan perbedaan utama Databasus dan WAL-G:</p>
              <table>
                <thead>
                  <tr>
                    <th>Fitur</th>
                    <th>Databasus</th>
                    <th>WAL-G</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Pengelolaan backup</td>
                    <td data-label="Databasus">✅ Ya (berbagai database)</td>
                    <td data-label="WAL-G">❌ Tidak (hanya satu database)</td>
                  </tr>
                  <tr>
                    <td>Dukungan database lain</td>
                    <td data-label="Databasus">✅ PostgreSQL, MySQL, MariaDB, MongoDB</td>
                    <td data-label="WAL-G">✅ PostgreSQL, MySQL, MS SQL</td>
                  </tr>
                  <tr>
                    <td>Antarmuka</td>
                    <td data-label="Databasus">Antarmuka web</td>
                    <td data-label="WAL-G">Hanya command-line</td>
                  </tr>
                  <tr>
                    <td>Jenis backup</td>
                    <td data-label="Databasus">Logical + Physical</td>
                    <td data-label="WAL-G">Physical (pengarsipan WAL)</td>
                  </tr>
                  <tr>
                    <td>Versi PostgreSQL untuk backup physical</td>
                    <td data-label="Databasus">17+ (native)</td>
                    <td data-label="WAL-G">9.x+ (engine sendiri)</td>
                  </tr>
                  <tr>
                    <td>Penjadwalan backup</td>
                    <td data-label="Databasus">✅ Penjadwal bawaan</td>
                    <td data-label="WAL-G">Memerlukan penjadwal eksternal (cron)</td>
                  </tr>
                  <tr>
                    <td>Opsi pemulihan</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="WAL-G">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Backup incremental</td>
                    <td data-label="Databasus">✅ Tingkat blok (PG 17+)</td>
                    <td data-label="WAL-G">Backup delta (hanya page yang berubah)</td>
                  </tr>
                  <tr>
                    <td>Backup jarak jauh</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="WAL-G">❌ Tidak (berjalan secara lokal)</td>
                  </tr>
                  <tr>
                    <td>Fitur tim</td>
                    <td data-label="Databasus">✅ Ruang kerja, RBAC, log audit</td>
                    <td data-label="WAL-G">❌ Hanya izin tingkat OS</td>
                  </tr>
                  <tr>
                    <td>Notifikasi</td>
                    <td data-label="Databasus">✅ Slack, Teams, Telegram, Email</td>
                    <td data-label="WAL-G">❌ Memerlukan skrip sendiri</td>
                  </tr>
                  <tr>
                    <td>Enkripsi</td>
                    <td data-label="Databasus">AES-256-GCM bawaan</td>
                    <td data-label="WAL-G">GPG atau libsodium</td>
                  </tr>
                  <tr>
                    <td>Proses belajar</td>
                    <td data-label="Databasus">Minimal</td>
                    <td data-label="WAL-G">Memerlukan kemahiran CLI</td>
                  </tr>
                  <tr>
                    <td>Instalasi</td>
                    <td data-label="Databasus">Skrip satu baris atau Docker</td>
                    <td data-label="WAL-G">Unduh binary + konfigurasi</td>
                  </tr>
                  <tr>
                    <td>Cocok untuk database self-hosted</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="WAL-G">✅ Ya</td>
                  </tr>
                  <tr>
                    <td>Cocok untuk database cloud</td>
                    <td data-label="Databasus">✅ Ya (RDS, Cloud SQL, Azure)</td>
                    <td data-label="WAL-G">❌ Hanya backup (tidak bisa restore ke cloud)</td>
                  </tr>
                </tbody>
              </table>
              <h2 id="database-focus">Cakupan database</h2>
              <p>
                Salah satu perbedaan terbesar kedua tool ini adalah cakupan database yang didukung:
              </p>
              <h3 id="focus-databasus">Databasus: Pengelolaan backup menyeluruh</h3>
              <p>
                Databasus dirancang untuk mengelola backup berbagai sistem database secara
                menyeluruh, dengan mengutamakan kemudahan penggunaan:
              </p>
              <ul>
                <li>
                  <strong>Dukungan berbagai database</strong>: Kelola backup PostgreSQL, MySQL,
                  MariaDB, dan MongoDB dari satu antarmuka.
                </li>
                <li>
                  <strong>Penggunaan yang konsisten</strong>: Antarmuka, alur kerja, dan fitur
                  bekerja dengan cara yang sama pada semua database yang didukung.
                </li>
                <li>
                  <strong>Dukungan versi</strong>: Mendukung PostgreSQL versi 12 hingga 18, dengan
                  optimasi khusus tiap versi.
                </li>
                <li>
                  <strong>Pengelolaan yang praktis</strong>: Seluruh upaya pengembangan difokuskan
                  untuk mempermudah pengelolaan backup.
                </li>
              </ul>
              <h3 id="focus-wal-g">WAL-G: Dukungan berbagai database</h3>
              <p>
                WAL-G awalnya merupakan tool backup PostgreSQL, lalu berkembang untuk mendukung
                berbagai sistem database:
              </p>
              <ul>
                <li>
                  <strong>PostgreSQL</strong>: Implementasi awal dan yang paling matang.
                </li>
                <li>
                  <strong>MySQL/MariaDB</strong>: Mendukung backup berbasis binlog.
                </li>
                <li>
                  <strong>MS SQL Server</strong>: Backup SQL Server berbasis Windows.
                </li>
                <li>
                  <strong>MongoDB</strong>: Dukungan backup database dokumen.
                </li>
                <li>
                  <strong>FoundationDB</strong>: Dukungan database terdistribusi.
                </li>
                <li>
                  <strong>Greenplum</strong>: Dukungan backup data warehouse.
                </li>
              </ul>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">
                    Saat pengelolaan menyeluruh dibutuhkan:
                  </strong>{" "}
                  Jika perlu mengelola backup berbagai database melalui satu antarmuka, Databasus
                  mempermudahnya. Pengelolaan backup terpusat dan fitur tim tersedia tanpa harus
                  menggunakan tool berbeda untuk setiap database.
                </p>
              </div>
              <h2 id="target-audience">Pengguna yang dituju</h2>
              <p>Kedua tool melayani kelompok pengguna yang berbeda sesuai pendekatan desainnya:</p>
              <h3 id="audience-databasus">Pengguna Databasus</h3>
              <p>
                Databasus dirancang untuk beragam pengguna, dari developer individu hingga
                perusahaan besar:
              </p>
              <ul>
                <li>
                  <strong>Developer individu</strong>: Setup sederhana dan UI yang mudah dipahami
                  membantu melindungi proyek pribadi tanpa keahlian PostgreSQL yang mendalam.
                </li>
                <li>
                  <strong>Tim pengembangan</strong>: Ruang kerja, kontrol akses berbasis role, dan
                  log audit mendukung kolaborasi aman antaranggota tim.
                </li>
                <li>
                  <strong>Perusahaan</strong>: Bisa menyesuaikan kebutuhan perusahaan dengan
                  keamanan menyeluruh, berbagai tujuan penyimpanan, dan kanal notifikasi.
                </li>
                <li>
                  <strong>Lingkungan dengan berbagai database</strong>: Organisasi yang menjalankan
                  PostgreSQL, MySQL, MariaDB, atau MongoDB bisa menggunakan pengelolaan backup
                  terpusat.
                </li>
                <li>
                  <strong>DBA dan pemulihan bencana</strong>: Backup physical, pengarsipan WAL, dan
                  PITR untuk sistem kritis yang mensyaratkan kehilangan data mendekati nol.
                </li>
                <li>
                  <strong>Engineer DevOps</strong>: Mode agen terintegrasi dengan infrastruktur yang
                  ada, sementara UI web dan API memberikan pemantauan dan kontrol tanpa skrip
                  sendiri.
                </li>
              </ul>
              <h3 id="audience-wal-g">Pengguna WAL-G</h3>
              <p>WAL-G dirancang untuk pengguna yang terbiasa dengan tool command-line:</p>
              <ul>
                <li>
                  <strong>Engineer DevOps</strong>: Pengguna yang memilih infrastructure-as-code dan
                  alur kerja berbasis CLI.
                </li>
                <li>
                  <strong>Lingkungan dengan berbagai database</strong>: Organisasi yang menjalankan
                  PostgreSQL bersama MySQL, MongoDB, atau database lain yang didukung.
                </li>
                <li>
                  <strong>Deployment cloud-native</strong>: Tim yang menggunakan Kubernetes atau
                  lingkungan container yang mudah diintegrasikan dengan tool CLI.
                </li>
                <li>
                  <strong>Dukungan database tambahan</strong>: Tim yang memerlukan backup MS SQL,
                  FoundationDB, atau Greenplum selain PostgreSQL.
                </li>
              </ul>
              <h2 id="backup-approach">Pendekatan backup</h2>
              <p>
                Kedua tool menggunakan strategi backup yang berbeda secara mendasar, masing-masing
                dengan keunggulannya:
              </p>
              <h3 id="backup-databasus">Databasus: Backup logical + physical</h3>
              <p>Databasus mendukung strategi backup logical maupun physical:</p>
              <ul>
                <li>
                  <strong>Backup physical, incremental, dan WAL</strong>: Berjalan dari jarak jauh
                  melalui protokol replikasi PostgreSQL dengan tool native PostgreSQL 17:{" "}
                  <code>pg_basebackup</code>, <code>pg_basebackup --incremental</code> tingkat blok
                  yang menggunakan ringkasan WAL di sisi server, <code>pg_receivewal</code>, dan{" "}
                  <code>pg_combinebackup</code>. Databasus menggunakan tool PostgreSQL yang sudah
                  teruji, bukan membuat ulang mekanismenya. Memerlukan PostgreSQL 17 atau yang lebih
                  baru.
                </li>
                <li>
                  <strong>Backup logical</strong>: Menggunakan <code>pg_dump</code> untuk backup
                  portabel yang bisa di-restore ke versi PostgreSQL lain. Ini juga satu-satunya
                  jenis backup pada PostgreSQL sebelum versi 17 dan pendekatan untuk MySQL, MariaDB,
                  serta MongoDB.
                </li>
                <li>
                  <strong>Tidak perlu instalasi pada database</strong>: Backup terhubung dari jarak
                  jauh. Jaringan tertutup dijangkau melalui SSH tunnel ke host internal atau
                  bastion, sehingga database tidak perlu dibuka ke publik.
                </li>
                <li>
                  <strong>Kompresi efisien</strong>: Menggunakan zstd (level 5) untuk kedua jenis
                  backup, mengurangi ukuran hingga 4-8x.
                </li>
                <li>
                  <strong>Akses read-only</strong>: Backup logical hanya memerlukan izin SELECT,
                  sehingga mengurangi risiko keamanan.
                </li>
              </ul>
              <h3 id="backup-wal-g">WAL-G: Backup physical dengan pengarsipan WAL</h3>
              <p>
                WAL-G membuat backup tingkat file (physical) dengan pengarsipan WAL berkelanjutan:
              </p>
              <ul>
                <li>
                  <strong>Backup dasar</strong>: Salinan lengkap direktori data PostgreSQL pada
                  tingkat file.
                </li>
                <li>
                  <strong>Backup delta</strong>: Hanya page yang berubah yang di-backup, sehingga
                  mengurangi penyimpanan dan waktu transfer.
                </li>
                <li>
                  <strong>Pengarsipan WAL</strong>: Pengarsipan Write-Ahead Logs berkelanjutan
                  memungkinkan Point-in-Time Recovery.
                </li>
                <li>
                  <strong>Optimasi copy-on-write</strong>: Penanganan efisien untuk blok data yang
                  tidak berubah.
                </li>
              </ul>
              <h2 id="recovery-options">Opsi pemulihan</h2>
              <p>
                Kedua tool menyediakan kemampuan pemulihan dengan tingkat perincian yang berbeda:
              </p>
              <h3 id="recovery-databasus">Pemulihan Databasus</h3>
              <ul>
                <li>
                  <strong>Point-in-Time Recovery</strong>: Restore ke detik tertentu menggunakan WAL
                  replay.
                </li>
                <li>
                  <strong>Restore cluster penuh</strong>: Restore seluruh cluster database ke titik
                  waktu tertentu dari backup physical.
                </li>
                <li>
                  <strong>Restore logical</strong>: Restore dari backup logical terjadwal ke titik
                  backup mana pun.
                </li>
                <li>
                  <strong>Restore satu klik</strong>: Unduh dan restore backup logical langsung dari
                  antarmuka web.
                </li>
                <li>
                  <strong>Kompatibilitas lintas versi</strong>: Backup logical bisa di-restore ke
                  versi PostgreSQL yang berbeda.
                </li>
              </ul>
              <h3 id="recovery-wal-g">Pemulihan WAL-G</h3>
              <ul>
                <li>
                  <strong>Point-in-Time Recovery (PITR)</strong>: Restore ke detik tertentu
                  menggunakan WAL replay untuk meminimalkan kehilangan data.
                </li>
                <li>
                  <strong>Restore cluster penuh</strong>: Restore seluruh cluster database ke titik
                  waktu tertentu.
                </li>
                <li>
                  <strong>Restore delta</strong>: Pemulihan lebih cepat dengan hanya mengambil page
                  yang berubah.
                </li>
                <li>
                  <strong>Pembuatan standby</strong>: Buat replica PostgreSQL dari backup untuk
                  konfigurasi high availability.
                </li>
              </ul>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Catatan:</strong> Kedua tool mendukung PITR.
                  WAL-G juga menawarkan restore delta (hanya mengambil page yang berubah) dan
                  menggunakan protokol streaming khusus untuk performa yang sedikit lebih baik pada
                  skala besar.{" "}
                  <a
                    href={`${getLocalizedHref("id", "faq")}#pitr`}
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Pelajari dukungan PITR di Databasus →
                  </a>
                </p>
              </div>
              <h2 id="ease-of-use">Kemudahan penggunaan</h2>
              <p>Kedua tool memiliki pendekatan penggunaan yang sangat berbeda:</p>
              <h3 id="ease-databasus">Penggunaan Databasus</h3>
              <ul>
                <li>
                  <strong>Antarmuka web</strong>: Konfigurasi semua pengaturan backup dengan klik,
                  tanpa command-line.
                </li>
                <li>
                  <strong>Instalasi 2 menit</strong>: Jalankan skrip cURL satu baris atau perintah
                  Docker sederhana untuk langsung mulai.
                </li>
                <li>
                  <strong>Pemantauan visual</strong>: Dashboard menampilkan status backup,
                  pemeriksaan ketersediaan, dan riwayat secara sekilas.
                </li>
                <li>
                  <strong>Notifikasi bawaan</strong>: Atur pemberitahuan Slack, Teams, Telegram,
                  Email, atau webhook langsung di UI.
                </li>
                <li>
                  <strong>Tidak memerlukan keahlian PostgreSQL</strong>: Dirancang untuk developer
                  yang memerlukan backup andal tanpa harus menjadi ahli database.
                </li>
              </ul>
              <h3 id="ease-wal-g">Penggunaan WAL-G</h3>
              <ul>
                <li>
                  <strong>Antarmuka command-line</strong>: Semua operasi dilakukan melalui perintah
                  terminal seperti <code>wal-g backup-push</code>, <code>wal-g backup-fetch</code>.
                </li>
                <li>
                  <strong>Environment variable</strong>: Konfigurasi terutama menggunakan
                  environment variable, bukan file konfigurasi.
                </li>
                <li>
                  <strong>Penjadwalan eksternal</strong>: Memerlukan cron job atau orkestrasi
                  eksternal untuk backup otomatis.
                </li>
                <li>
                  <strong>Setup pengarsipan WAL</strong>: Harus mengonfigurasi{" "}
                  <code>archive_command</code> PostgreSQL untuk integrasi dengan WAL-G.
                </li>
                <li>
                  <strong>Memerlukan kemahiran CLI</strong>: Dokumentasi mengasumsikan pengguna
                  terbiasa dengan tool command-line dan skrip shell.
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
              <h3 id="team-databasus">Kemampuan tim Databasus</h3>
              <ul>
                <li>
                  <strong>Ruang kerja</strong>: Kelompokkan database, kanal notifikasi, dan
                  penyimpanan berdasarkan proyek atau tim. Pengguna hanya melihat ruang kerja yang
                  mengundang mereka.
                </li>
                <li>
                  <strong>Kontrol akses berbasis role</strong>: Tetapkan izin viewer, editor, atau
                  admin untuk mengatur tindakan setiap anggota tim.
                </li>
                <li>
                  <strong>Log audit</strong>: Lacak semua aktivitas sistem dan perubahan. Penting
                  untuk kepatuhan keamanan dan akuntabilitas.
                </li>
                <li>
                  <strong>Notifikasi bersama</strong>: Kanal tim menerima pembaruan status backup
                  secara otomatis.
                </li>
              </ul>
              <h3 id="team-wal-g">Kemampuan tim WAL-G</h3>
              <p>WAL-G adalah tool command-line tanpa fitur tim bawaan:</p>
              <ul>
                <li>Tidak ada pengelolaan pengguna atau kontrol akses</li>
                <li>Tidak ada pencatatan audit operasi</li>
                <li>Koordinasi tim memerlukan tool dan proses eksternal</li>
                <li>Akses diatur melalui izin tingkat OS dan kebijakan IAM cloud</li>
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
              <p>Kedua tool menyediakan fitur keamanan dengan pendekatan berbeda:</p>
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
                  <strong>Akses database read-only</strong>: Hanya mengizinkan SELECT, sehingga
                  mencegah kerusakan data meskipun sistem disusupi.
                </li>
              </ul>
              <h3 id="security-wal-g">Keamanan WAL-G</h3>
              <ul>
                <li>
                  <strong>Enkripsi GPG</strong>: Mendukung enkripsi berbasis GPG untuk file backup.
                </li>
                <li>
                  <strong>Enkripsi libsodium</strong>: Alternatif enkripsi menggunakan library
                  libsodium.
                </li>
                <li>
                  <strong>Integrasi IAM cloud</strong>: Menggunakan IAM penyedia cloud untuk
                  mengontrol akses penyimpanan.
                </li>
                <li>
                  <strong>Tanpa pengelolaan kredensial bawaan</strong>: Bergantung pada environment
                  variable atau pengelolaan secret eksternal.
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
              <h2 id="storage-options">Opsi penyimpanan</h2>
              <p>Kedua tool mendukung penyimpanan cloud dengan fokus berbeda:</p>
              <h3 id="storage-databasus">Penyimpanan Databasus</h3>
              <p>Opsi yang mudah digunakan untuk beragam kebutuhan:</p>
              <ul>
                <li>Penyimpanan lokal</li>
                <li>Amazon S3 dan layanan kompatibel S3</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>
              <h3 id="storage-wal-g">Penyimpanan WAL-G</h3>
              <p>Opsi penyimpanan cloud-native:</p>
              <ul>
                <li>Amazon S3</li>
                <li>Google Cloud Storage (GCS)</li>
                <li>Azure Blob Storage</li>
                <li>Swift (OpenStack)</li>
                <li>Filesystem lokal</li>
                <li>SSH/SFTP</li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "storages")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua opsi penyimpanan Databasus →
                </a>
              </p>
              <h2 id="notifications">Notifikasi</h2>
              <p>Untuk mengetahui status backup:</p>
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
              <h3 id="notifications-wal-g">Notifikasi WAL-G</h3>
              <p>WAL-G tidak memiliki dukungan notifikasi bawaan. Notifikasi memerlukan:</p>
              <ul>
                <li>Skrip sendiri yang membungkus perintah backup</li>
                <li>Integrasi dengan tool pemantauan eksternal</li>
                <li>Parsing log dan pengaturan pemberitahuan secara manual</li>
                <li>Integrasi dengan tool seperti Prometheus, Grafana, atau solusi sendiri</li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua kanal notifikasi Databasus →
                </a>
              </p>
              <h2 id="compression">Kompresi</h2>
              <p>Kedua tool menawarkan kompresi untuk mengurangi ukuran backup:</p>
              <h3 id="compression-databasus">Kompresi Databasus</h3>
              <ul>
                <li>
                  <strong>Kompresi zstd</strong>: Menggunakan zstd level 5 untuk menyeimbangkan
                  kecepatan dan rasio kompresi.
                </li>
                <li>
                  <strong>Pengurangan ukuran 4-8x</strong>: Rasio kompresi yang umum dicapai dengan
                  tambahan waktu pemrosesan hanya ~20%.
                </li>
                <li>
                  <strong>Otomatis</strong>: Kompresi aktif secara default tanpa konfigurasi.
                </li>
              </ul>
              <h3 id="compression-wal-g">Kompresi WAL-G</h3>
              <ul>
                <li>
                  <strong>Berbagai algoritma</strong>: Mendukung LZ4, LZMA, Brotli, dan zstd.
                </li>
                <li>
                  <strong>Level yang bisa diatur</strong>: Sesuaikan keseimbangan rasio kompresi dan
                  kecepatan secara terperinci.
                </li>
                <li>
                  <strong>Kompresi per file</strong>: File WAL dan backup dasar bisa menggunakan
                  pengaturan berbeda.
                </li>
              </ul>
              <h2 id="conclusion">Kesimpulan</h2>
              <p>
                Databasus dan WAL-G memenuhi kebutuhan berbeda dalam ekosistem backup PostgreSQL.
                Pilihan yang tepat bergantung pada lingkungan database, struktur tim, dan preferensi
                operasional.
              </p>
              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Pilih Databasus jika:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>Memerlukan pengelolaan backup PostgreSQL menyeluruh dari satu antarmuka</li>
                  <li>Lebih memilih antarmuka web daripada tool command-line</li>
                  <li>Memerlukan fitur kolaborasi tim (ruang kerja, RBAC, log audit)</li>
                  <li>Menginginkan notifikasi bawaan ke Slack, Teams, Telegram, dan lainnya</li>
                  <li>Menginginkan penjadwalan bawaan tanpa setup cron eksternal</li>
                  <li>
                    Ingin mengelola backup berbagai database dari satu dashboard dengan penjadwalan,
                    notifikasi, dan fitur tim
                  </li>
                  <li>Menginginkan setup cepat dengan keahlian database minimal</li>
                  <li>Menganggap enkripsi backup bawaan penting</li>
                  <li>
                    Menggunakan database yang dikelola cloud (AWS RDS, Google Cloud SQL, Azure) atau
                    database self-hosted
                  </li>
                </ul>
              </div>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Pilih WAL-G jika:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Memerlukan backup physical atau incremental pada PostgreSQL sebelum versi 17
                    (WAL-G memiliki engine backup sendiri)
                  </li>
                  <li>
                    Memerlukan backup delta (hanya page yang berubah) untuk mengurangi penyimpanan
                    dan waktu transfer
                  </li>
                  <li>Memerlukan dukungan MS SQL, FoundationDB, atau Greenplum</li>
                  <li>Lebih memilih tool command-line dan alur kerja infrastructure-as-code</li>
                  <li>
                    Menginginkan berbagai algoritma kompresi (LZ4, LZMA, Brotli, zstd) dengan
                    kontrol terperinci
                  </li>
                  <li>Tim memiliki keahlian DevOps untuk mengelola tool berbasis CLI</li>
                </ul>
              </div>
              <p>
                Kedua tool mendukung backup physical, pengarsipan WAL, dan PITR, serta dirancang
                untuk pemulihan bencana dengan RTO dan RPO minimal. Databasus bisa digunakan untuk
                database dengan ukuran dan kompleksitas apa pun, menyediakan antarmuka web, fitur
                tim, serta backup logical dan physical untuk database self-hosted maupun yang
                dikelola cloud.
                <br />
                <br />
                WAL-G tetap menjadi pilihan yang sangat baik untuk tim yang memilih alur kerja
                berbasis CLI dan memerlukan keunggulan khususnya: backup delta (hanya page yang
                berubah), protokol streaming khusus untuk performa yang sedikit lebih baik, serta
                dukungan engine database tambahan selain PostgreSQL.
              </p>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
