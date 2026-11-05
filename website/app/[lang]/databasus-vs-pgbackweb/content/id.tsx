import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs PgBackWeb - Perbandingan alat backup PostgreSQL",
  description:
    "Bandingkan alat backup PostgreSQL Databasus dan PgBackWeb. Lihat perbedaan fitur, keamanan, dukungan tim, pilihan penyimpanan, notifikasi, dan kemudahan penggunaan.",
  keywords: [
    "Databasus vs PgBackWeb",
    "perbandingan backup PostgreSQL",
    "alternatif PgBackWeb",
    "alat backup PostgreSQL",
    "perbandingan backup database",
    "pg_dump GUI",
    "backup self-hosted",
    "keamanan backup PostgreSQL",
  ],
  openGraph: {
    title: "Databasus vs PgBackWeb - Perbandingan alat backup PostgreSQL",
    description:
      "Bandingkan alat backup PostgreSQL Databasus dan PgBackWeb. Lihat perbedaan fitur, keamanan, dukungan tim, pilihan penyimpanan, notifikasi, dan kemudahan penggunaan.",
    type: "article",
    url: getLocalizedUrl("id", "databasus-vs-pgbackweb"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs PgBackWeb - Perbandingan alat backup PostgreSQL",
    description:
      "Bandingkan alat backup PostgreSQL Databasus dan PgBackWeb. Lihat perbedaan fitur, keamanan, dukungan tim, pilihan penyimpanan, notifikasi, dan kemudahan penggunaan.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "databasus-vs-pgbackweb"),
    languages: getLanguageAlternates("databasus-vs-pgbackweb"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackWebPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Databasus vs PgBackWeb - Perbandingan alat backup PostgreSQL",
            description:
              "Perbandingan lengkap alat backup PostgreSQL Databasus dan PgBackWeb, meliputi fitur, keamanan, dukungan tim, pilihan penyimpanan, dan kemudahan penggunaan.",
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
              <h1 id="databasus-vs-pgbackweb">Databasus vs PgBackWeb</h1>
              <p className="text-lg text-gray-400">
                Databasus dan PgBackWeb adalah alat open source yang dirancang untuk mempermudah
                pengelolaan backup PostgreSQL melalui antarmuka web. Keduanya bertujuan membuat
                backup lebih mudah digunakan, tetapi sangat berbeda dalam fitur, keamanan, dukungan
                tim, dan kemudahan penggunaan.
              </p>
              <h2 id="quick-comparison">Perbandingan singkat</h2>
              <p>Berikut ringkasan perbedaan utama Databasus dan PgBackWeb:</p>
              <table>
                <thead>
                  <tr>
                    <th>Fitur</th>
                    <th>Databasus</th>
                    <th>PgBackWeb</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Lisensi</td>
                    <td data-label="Databasus">Apache 2.0</td>
                    <td data-label="PgBackWeb">AGPL-3.0</td>
                  </tr>
                  <tr>
                    <td>Pengelolaan backup</td>
                    <td data-label="Databasus">✅ Banyak database</td>
                    <td data-label="PgBackWeb">✅ Banyak database</td>
                  </tr>
                  <tr>
                    <td>Dukungan database lain</td>
                    <td data-label="Databasus">✅ PostgreSQL, MySQL, MariaDB, MongoDB</td>
                    <td data-label="PgBackWeb">❌ Hanya PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Pilihan penyimpanan</td>
                    <td data-label="Databasus">
                      Lokal, S3, Google Drive, Cloudflare R2, Azure, NAS, Dropbox
                    </td>
                    <td data-label="PgBackWeb">Hanya lokal dan kompatibel S3</td>
                  </tr>
                  <tr>
                    <td>Notifikasi</td>
                    <td data-label="Databasus">Slack, Discord, Telegram, Teams, Email, Webhooks</td>
                    <td data-label="PgBackWeb">Hanya webhook</td>
                  </tr>
                  <tr>
                    <td>Keamanan</td>
                    <td data-label="Databasus">
                      ✅ AES-256-GCM, kunci backup unik, pembatasan read-only
                    </td>
                    <td data-label="PgBackWeb">✅ Enkripsi PGP</td>
                  </tr>
                  <tr>
                    <td>Fitur tim</td>
                    <td data-label="Databasus">✅ Ruang kerja, akses berbasis peran, log audit</td>
                    <td data-label="PgBackWeb">❌ Tidak tersedia</td>
                  </tr>
                  <tr>
                    <td>Pemantauan ketersediaan</td>
                    <td data-label="Databasus">✅ Bawaan</td>
                    <td data-label="PgBackWeb">❌ Tidak tersedia</td>
                  </tr>
                  <tr>
                    <td>Instalasi</td>
                    <td data-label="Databasus">Script satu baris, Docker, atau Helm</td>
                    <td data-label="PgBackWeb">Penyiapan Docker manual</td>
                  </tr>
                  <tr>
                    <td>Backup fisik</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="PgBackWeb">❌ Tidak tersedia</td>
                  </tr>
                  <tr>
                    <td>Backup inkremental</td>
                    <td data-label="Databasus">✅ Tingkat blok (PG 17+)</td>
                    <td data-label="PgBackWeb">❌ Tidak tersedia</td>
                  </tr>
                  <tr>
                    <td>Pengarsipan WAL</td>
                    <td data-label="Databasus">✅ Streaming berkelanjutan</td>
                    <td data-label="PgBackWeb">❌ Tidak tersedia</td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="Databasus">✅ Ya</td>
                    <td data-label="PgBackWeb">❌ Tidak tersedia</td>
                  </tr>
                </tbody>
              </table>
              <h2 id="backup-features">Fitur backup</h2>
              <p>Kedua alat mendukung backup terjadwal dengan waktu yang fleksibel:</p>
              <ul>
                <li>
                  <strong>Databasus</strong>: Mendukung jadwal per jam, harian, mingguan, bulanan,
                  atau cron dengan waktu yang presisi (misalnya pukul 4 pagi). Menggunakan{" "}
                  <strong>kompresi seimbang dengan zstd (level 5)</strong>, memperkecil ukuran
                  backup 4-8x dengan tambahan waktu proses hanya ~20%. Ini jauh lebih efisien
                  daripada gzip.
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Mendukung penjadwalan berbasis cron untuk menjalankan
                  backup. Menggunakan kompresi gzip untuk backup, yang lebih lambat dan kurang
                  efisien dibandingkan zstd.
                </li>
              </ul>
              <p>
                Selain backup logis, Databasus juga mendukung backup fisik, inkremental, dan WAL.
                Backup ini menggunakan perangkat backup bawaan PostgreSQL 17 dan dijalankan dari
                jarak jauh, sehingga tidak perlu instalasi di server database dan jaringan tertutup
                bisa dijangkau lewat SSH tunnel. Ini menyediakan backup inkremental tingkat blok,
                streaming WAL berkelanjutan, dan Point-in-Time Recovery untuk pemulihan bencana
                dengan kehilangan data mendekati nol, serta restore ke detik mana pun di antara
                backup. PgBackWeb tidak menawarkan kemampuan tersebut.
              </p>
              <h2 id="storage-options">Pilihan penyimpanan</h2>
              <p>
                Fleksibilitas penyimpanan penting untuk strategi backup. Berikut perbandingan kedua
                alat:
              </p>
              <ul>
                <li>
                  <strong>Databasus</strong>: Mendukung beragam tujuan penyimpanan:
                  <ul>
                    <li>Penyimpanan lokal</li>
                    <li>Amazon S3 dan layanan kompatibel S3</li>
                    <li>Google Drive</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS (Network-attached storage)</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Terbatas pada penyimpanan lokal dan penyimpanan
                  kompatibel S3.
                </li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "storages")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua pilihan penyimpanan Databasus →
                </a>
              </p>
              <h2 id="security">Keamanan</h2>
              <p>
                Keamanan merupakan aspek penting pengelolaan backup. Databasus menerapkan keamanan
                kelas perusahaan pada tiga tingkat:
              </p>
              <h3 id="security-databasus">Model keamanan Databasus</h3>
              <ol>
                <li>
                  <strong>Enkripsi data sensitif</strong>: Semua password, token, dan kredensial
                  dienkripsi dengan AES-256-GCM. Kunci enkripsi disimpan terpisah dari database,
                  sehingga data sensitif tetap terlindungi sekalipun database disusupi.
                </li>
                <li>
                  <strong>Enkripsi backup</strong>: Setiap file backup dienkripsi dengan kunci unik
                  yang diturunkan dari master key, ID backup, dan salt acak. Sekalipun seseorang
                  mendapatkan akses ke penyimpanan cloud, mereka tidak bisa membaca backup tanpa
                  kunci enkripsi kamu.
                </li>
                <li>
                  <strong>Akses database read-only</strong>: Databasus menerapkan akses read-only
                  dengan memeriksa izin pada tingkat peran, database, dan tabel. Databasus hanya
                  memerlukan izin SELECT dan akan memberi peringatan jika mendeteksi izin tulis. Ini
                  mencegah kerusakan data sekalipun Databasus disusupi.
                </li>
              </ol>
              <h3 id="security-pgbackweb">Model keamanan PgBackWeb</h3>
              <ul>
                <li>
                  <strong>Enkripsi PGP</strong>: PgBackWeb menawarkan enkripsi PGP untuk file
                  backup.
                </li>
                <li>
                  <strong>Tanpa pembatasan read-only</strong>: PgBackWeb tidak menerapkan atau
                  memverifikasi akses database read-only, sehingga backup dapat dibuat menggunakan
                  pengguna yang memiliki izin tulis.
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
              <h2 id="notifications">Notifikasi</h2>
              <p>Mengetahui status backup penting untuk operasional yang andal:</p>
              <ul>
                <li>
                  <strong>Databasus</strong>: Menyediakan notifikasi real-time melalui berbagai
                  kanal:
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>Email</li>
                    <li>Webhooks</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Hanya mendukung webhook untuk notifikasi. Untuk
                  menerima peringatan melalui Slack, Telegram, atau platform lain, perlu menyiapkan
                  middleware atau layanan tambahan.
                </li>
              </ul>
              <p>
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Lihat semua kanal notifikasi Databasus →
                </a>
              </p>
              <h2 id="team-features">Fitur tim</h2>
              <p>
                Fitur kolaborasi penting untuk organisasi dan tim DevOps. Di sinilah Databasus jauh
                mengungguli PgBackWeb:
              </p>
              <h3 id="team-databasus">Kemampuan Databasus untuk tim</h3>
              <ul>
                <li>
                  <strong>Ruang kerja</strong>: Kelompokkan database, kanal notifikasi, dan
                  penyimpanan untuk proyek atau tim berbeda. Pengguna hanya melihat ruang kerja yang
                  mengundangnya.
                </li>
                <li>
                  <strong>Kontrol akses berbasis peran</strong>: Tingkat izin untuk mengontrol
                  tindakan setiap anggota tim di dalam ruang kerja.
                </li>
                <li>
                  <strong>Log audit</strong>: Lacak semua aktivitas sistem dan perubahan oleh
                  pengguna. Penting untuk kepatuhan keamanan dan pertanggungjawaban tim.
                </li>
              </ul>
              <h3 id="team-pgbackweb">Kemampuan PgBackWeb untuk tim</h3>
              <p>
                PgBackWeb tidak memiliki pengelolaan pengguna, ruang kerja, atau log audit bawaan.
                Alat ini terutama dirancang untuk penggunaan oleh satu pengguna.
              </p>
              <p>
                <a
                  href={getLocalizedHref("id", "access-management")}
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Pelajari pengelolaan akses Databasus →
                </a>
              </p>
              <h2 id="ease-of-use">Kemudahan penggunaan</h2>
              <p>
                <strong>Databasus dirancang agar jauh lebih mudah digunakan</strong> daripada
                PgBackWeb, dengan fokus pada UX intuitif dan waktu penyiapan minimal:
              </p>
              <h3 id="ease-databasus">Pengalaman menggunakan Databasus</h3>
              <ul>
                <li>
                  <strong>Instalasi mudah</strong>: Gunakan Docker langsung atau jalankan script
                  satu baris yang memasang Docker (jika diperlukan), menyiapkan Databasus, dan
                  mengatur startup otomatis. Total waktu: ~2 menit.
                </li>
                <li>
                  <strong>Antarmuka web intuitif</strong>: UI yang dipoles desainer memandu
                  konfigurasi backup langkah demi langkah. Tidak perlu keahlian PostgreSQL.
                </li>
                <li>
                  <strong>Tema gelap dan terang</strong>: Pilih tampilan yang sesuai dengan alur
                  kerja.
                </li>
                <li>
                  <strong>Adaptif di perangkat mobile</strong>: Periksa backup dari mana saja dengan
                  perangkat apa pun.
                </li>
                <li>
                  <strong>Pemantauan ketersediaan bawaan</strong>: Pemeriksaan ketersediaan yang
                  dapat dikonfigurasi dengan grafik ketersediaan visual.
                </li>
                <li>
                  <strong>Restore satu klik</strong>: Unduh dan restore dari backup mana pun dengan
                  satu klik.
                </li>
              </ul>
              <h3 id="ease-pgbackweb">Pengalaman menggunakan PgBackWeb</h3>
              <ul>
                <li>
                  <strong>Penyiapan Docker manual</strong>: Memerlukan konfigurasi environment
                  variable dan penyiapan database PostgreSQL eksternal untuk menyimpan konfigurasi.
                </li>
                <li>
                  <strong>Antarmuka web dasar</strong>: UI berfungsi, tetapi kurang dipoles
                  dibandingkan Databasus. Tema gelap tersedia.
                </li>
                <li>
                  <strong>Tanpa pemantauan ketersediaan</strong>: Pemantauan ketersediaan database
                  harus disiapkan terpisah.
                </li>
              </ul>
              <h2 id="installation">Instalasi dan deployment</h2>
              <h3 id="install-databasus">Memasang Databasus</h3>
              <p>
                Databasus menawarkan tiga metode instalasi, dengan script otomatis sebagai yang
                tercepat:
              </p>
              <ul>
                <li>
                  <strong>Script otomatis (disarankan)</strong>: Perintah cURL satu baris yang
                  memasang Docker, menyiapkan Databasus, dan mengatur startup otomatis.
                </li>
                <li>
                  <strong>Docker run</strong>: Satu perintah untuk menjalankan Databasus dengan
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
                  Lihat panduan instalasi Databasus →
                </a>
              </p>
              <h3 id="install-pgbackweb">Memasang PgBackWeb</h3>
              <p>
                PgBackWeb memerlukan Docker dan konfigurasi environment variable secara manual.
                Perlu juga menyiapkan database PostgreSQL eksternal untuk menyimpan konfigurasi
                PgBackWeb.
              </p>
              <h2 id="licensing">Lisensi</h2>
              <p>
                Model lisensi dapat sangat memengaruhi cara menggunakan dan memodifikasi perangkat
                lunak:
              </p>
              <ul>
                <li>
                  <strong>Databasus (Apache 2.0)</strong>: Lisensi permisif yang mengizinkan
                  penggunaan komersial, modifikasi, dan distribusi tanpa pembatasan. Databasus dapat
                  digunakan dalam proyek proprietary tanpa kekhawatiran lisensi.
                </li>
                <li>
                  <strong>PgBackWeb (AGPL-3.0)</strong>: Lisensi copyleft yang mewajibkan karya
                  turunan atau modifikasi juga menjadi open source di bawah AGPL-3.0. Jika kamu
                  memodifikasi PgBackWeb dan menyediakannya sebagai layanan, modifikasi tersebut
                  harus dipublikasikan.
                </li>
              </ul>
              <h2 id="conclusion">Kesimpulan</h2>
              <p>
                Databasus dan PgBackWeb sama-sama mampu menangani backup PostgreSQL, tetapi memenuhi
                kebutuhan berbeda:
              </p>
              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Pilih Databasus jika kamu membutuhkan:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>Keamanan kelas perusahaan dengan perlindungan 3 tingkat</li>
                  <li>Kolaborasi tim dengan ruang kerja dan log audit</li>
                  <li>Banyak tujuan penyimpanan (Google Drive, Azure, dan lainnya)</li>
                  <li>Notifikasi bawaan ke Slack, Teams, Telegram, dan lainnya</li>
                  <li>Instalasi cepat dengan script satu baris atau Docker</li>
                  <li>UI modern yang intuitif dan mudah dipelajari</li>
                  <li>Lisensi Apache 2.0 yang permisif untuk penggunaan komersial</li>
                  <li>
                    Backup fisik, backup inkremental, pengarsipan WAL, dan PITR untuk pemulihan
                    bencana
                  </li>
                </ul>
              </div>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Pilih PgBackWeb jika:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>Membutuhkan solusi backup sederhana untuk satu pengguna</li>
                  <li>Hanya membutuhkan penyimpanan lokal atau S3</li>
                  <li>Notifikasi melalui webhook saja sudah cukup</li>
                  <li>Lisensi AGPL-3.0 dapat diterima untuk kebutuhan kamu</li>
                </ul>
              </div>
              <p>
                Untuk sebagian besar pengguna, terutama tim dan organisasi yang membutuhkan keamanan
                kuat, banyak pilihan penyimpanan, dan kanal notifikasi lengkap,{" "}
                <strong>Databasus adalah pilihan yang disarankan</strong>.
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
