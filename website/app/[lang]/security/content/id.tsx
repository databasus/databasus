import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Keamanan - Cara Databasus melindungi data | Databasus",
  description:
    "Pelajari cara Databasus menerapkan keamanan tingkat enterprise dengan enkripsi AES-256-GCM untuk data sensitif dan backup, akses database hanya-baca, serta pencatatan audit menyeluruh.",
  keywords: [
    "keamanan Databasus",
    "keamanan backup PostgreSQL",
    "enkripsi AES-256-GCM",
    "enkripsi database",
    "enkripsi backup",
    "akses database hanya-baca",
    "keamanan enterprise",
    "perlindungan data",
    "backup aman",
  ],
  openGraph: {
    title: "Keamanan - Cara Databasus melindungi data | Databasus",
    description:
      "Pelajari cara Databasus menerapkan keamanan tingkat enterprise dengan enkripsi AES-256-GCM untuk data sensitif dan backup, akses database hanya-baca, serta pencatatan audit menyeluruh.",
    type: "article",
    url: getLocalizedUrl("id", "security"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Keamanan - Cara Databasus melindungi data | Databasus",
    description:
      "Pelajari cara Databasus menerapkan keamanan tingkat enterprise dengan enkripsi AES-256-GCM untuk data sensitif dan backup, akses database hanya-baca, serta pencatatan audit menyeluruh.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "security"),
    languages: getLanguageAlternates("security"),
  },
  robots: "index, follow",
};

export default function SecurityPage() {
  const encryptionPipeline = `PostgreSQL pg_dump → Compression → Encryption → Cloud Storage`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Keamanan - Cara Databasus melindungi data",
            description:
              "Pelajari cara Databasus menerapkan keamanan tingkat enterprise dengan enkripsi AES-256-GCM untuk data sensitif dan backup, akses database hanya-baca, serta pencatatan audit menyeluruh.",
            author: { "@type": "Organization", name: "Databasus" },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: { "@type": "ImageObject", url: "https://databasus.com/logo.svg" },
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
              <h1 id="security">Bagaimana Databasus menerapkan keamanan?</h1>
              <p className="text-lg text-gray-400">
                Databasus bertanggung jawab atas data sensitif:
              </p>
              <ul>
                <li>mengakses database kamu;</li>
                <li>membuat backup database (artinya menyalin data);</li>
                <li>menyimpan kredensial agar bisa mengakses database secara berkala;</li>
                <li>
                  menyimpan backup di S3 atau penyimpanan cloud lain milik kamu (jika diaktifkan);
                </li>
              </ul>
              <p>
                Karena itu,{" "}
                <strong>
                  keamanan dan keandalan tingkat enterprise menjadi prioritas utama Databasus
                </strong>
                .
              </p>
              <p>Tujuannya untuk memastikan:</p>
              <ul>
                <li>data sensitif tidak pernah terekspos dan selalu dienkripsi;</li>
                <li>
                  backup terenkripsi dan tidak berguna bagi orang yang menemukannya di penyimpanan
                  cloud;
                </li>
                <li>
                  Databasus bahkan tidak mendapat akses database dengan izin menulis atau
                  memperbarui data;
                </li>
                <li>semua tindakan dicatat dan bisa diaudit;</li>
              </ul>
              <p>
                Semua langkah ini melindungi data kamu. Tidak ada sistem yang 100% aman, tetapi kami
                berusaha membuatnya seaman mungkin. Bahkan jika terjadi peretasan, tidak ada yang
                bisa merusak data kamu.
              </p>
              <p>Databasus menerapkan keamanan pada tiga lapisan:</p>
              <ol>
                <li>Enkripsi data sensitif;</li>
                <li>Enkripsi backup;</li>
                <li>Akses database hanya-baca.</li>
              </ol>
              <h2 id="level-1-sensitive-data-encryption">Lapisan 1: enkripsi data sensitif</h2>
              <p>
                Secara internal, Databasus memakai database PostgreSQL untuk menyimpan detail
                koneksi, konfigurasi, serta pengaturan kanal notifikasi dan penyimpanan (S3, Azure Blob Storage, FTP, dll.).
              </p>
              <p>Semua data sensitif dienkripsi. Contohnya:</p>
              <ul>
                <li>kata sandi</li>
                <li>token</li>
                <li>webhook yang berisi secret</li>
              </ul>
              <p>
                Jadi, di database, Databasus hanya menyimpan hash atau nilai yang sudah dikodekan.
                Enkripsi memakai algoritma <strong>AES-256-GCM</strong>. Selain dienkripsi,
                nilai-nilai ini juga tidak pernah ditampilkan lewat API atau antarmuka.
              </p>
              <p>
                Kunci rahasia untuk enkripsi disimpan di penyimpanan lokal (secara default{" "}
                <code>./databasus-data/secret.key</code>), bukan di dalam database. Jadi, database
                yang disusupi tidak memberikan akses ke data sensitif.
              </p>
              <h2 id="level-2-backups-encryption">Lapisan 2: enkripsi backup</h2>
              <p>
                Setiap file backup langsung dienkripsi selama proses pembuatannya. Databasus memakai
                algoritma enkripsi <strong>AES-256-GCM</strong>, sehingga data backup tidak bisa
                dibaca tanpa kunci enkripsi dan setiap manipulasi terdeteksi saat dekripsi.
              </p>
              <p>Backup melewati alur berikut:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{encryptionPipeline}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={encryptionPipeline} lang="id" />
                </div>
              </div>
              <p>Setiap backup mendapat kunci enkripsi unik yang diturunkan dari:</p>
              <ul>
                <li>
                  Kunci utama (disimpan di <code>./databasus-data/secret.key</code>)
                </li>
                <li>ID backup</li>
                <li>Salt acak (unik untuk setiap backup)</li>
              </ul>
              <p>
                <strong>Hasilnya</strong>: Sekalipun seseorang mendapat akses ke penyimpanan cloud
                kamu (S3, Azure Blob Storage, dll.), mereka tidak bisa membaca backup tanpa kunci utama.
              </p>
              <h2 id="level-3-read-only-access">Lapisan 3: akses database hanya-baca</h2>
              <p>
                Databasus menerapkan prinsip hak akses minimum. Untuk membuat backup, Databasus
                hanya memerlukan akses baca, tidak pernah akses tulis. Ini melindungi database dari
                kerusakan data yang disengaja maupun tidak disengaja melalui alat backup.
              </p>
              <p>
                Sebelum menerima kredensial database, Databasus melakukan pemeriksaan pada tiga
                tingkat:
              </p>
              <ol>
                <li>
                  <strong>Tingkat role</strong>: Memastikan pengguna BUKAN superuser dan tidak bisa
                  membuat role atau database
                </li>
                <li>
                  <strong>Tingkat database</strong>: Memastikan tidak ada hak akses CREATE atau TEMP
                </li>
                <li>
                  <strong>Tingkat tabel</strong>: Memastikan tidak ada izin tulis (INSERT, UPDATE,
                  DELETE, TRUNCATE, dll.)
                </li>
              </ol>
              <p>
                Pengguna database harus lolos ketiga pemeriksaan agar dianggap hanya-baca. Jika izin
                tulis terdeteksi, Databasus akan memberi peringatan.
              </p>
              <p>Databasus menawarkan pembuatan pengguna hanya-baca dengan izin yang sesuai:</p>
              <ul>
                <li>Memberikan SELECT pada semua tabel yang sudah ada maupun yang akan dibuat</li>
                <li>Memberikan USAGE pada schema (tetapi bukan CREATE)</li>
                <li>Secara eksplisit mencabut semua hak akses tulis</li>
              </ul>
              <p>
                <strong>Hasilnya</strong>: Sekalipun Databasus disusupi, server diretas, kunci
                rahasia dicuri, dan kredensial didekripsi, penyerang tidak bisa merusak database
                kamu.
              </p>
              <h2 id="two-factor-authentication">Autentikasi dua faktor saat masuk</h2>
              <p>
                Kata sandi saja membuka akses ke semua kredensial database, kunci penyimpanan, dan
                backup yang disimpan instance. Karena itu, administrator bisa mewajibkan faktor
                kedua saat masuk dengan kata sandi. Jika aktif, memasukkan kata sandi yang benar
                akan mengirim kode enam digit ke alamat email akun. Proses masuk baru selesai
                setelah kode dimasukkan. Kode kedaluwarsa sepuluh menit setelah dikirim, dibuang
                setelah lima tebakan salah, dan diganti jika kode baru diminta. Mengaktifkan
                pengaturan ini melindungi proses masuk berikutnya tanpa menghentikan sesi yang sudah
                terbuka.
              </p>
              <p>
                Pengaturan ini berlaku untuk proses masuk dengan kata sandi. Masuk lewat Google atau
                GitHub tetap langsung menerbitkan token karena penyedia tersebut menjalankan
                pemeriksaan multi-faktor sendiri. Instance yang ingin faktor kedua berlaku bagi
                semua pengguna tidak mengonfigurasi kedua penyedia itu.
              </p>
              <p>
                Pengaturan ini hanya bisa diaktifkan jika instance memiliki server email dan setiap
                administrator aktif memiliki alamat email asli, agar tidak ada yang mengaktifkannya
                lalu membuat instance tidak bisa diakses. Jika server email kemudian berhenti
                mengirimkan email, tidak ada yang bisa masuk dengan kata sandi. Ini disengaja:
                kegagalan server email tidak boleh diam-diam menurunkan autentikasi menjadi satu
                faktor. Operator dengan akses shell di server bisa menonaktifkannya kembali:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{disableTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={disableTwoFactorCommand} lang="id" />
                </div>
              </div>
              <p>
                Perintah melaporkan perubahan dan mencatatnya di log audit, sehingga alasan instance
                berhenti meminta kode tetap tercatat. Perintah ini didokumentasikan bersama
                pemulihan kata sandi di{" "}
                <a
                  href={`${getLocalizedHref("id", "password")}#disable-two-factor`}
                  className="text-blue-400 hover:text-blue-600"
                >
                  halaman kata sandi
                </a>
                .
              </p>
              <h2 id="security-and-reliability-engineering">🛡️ Rekayasa keamanan dan keandalan</h2>

              <p>
                Databasus menangani data sensitif. Sebelum setiap rilis, pemeriksaan
                otomatis mencari masalah keamanan dan memastikan backup bisa di-restore.
                Pemeriksaan ini mengurangi risiko, tetapi tidak menjamin perangkat lunak
                bebas kerentanan.
              </p>

              <h3 id="static-analysis">Analisis statis</h3>

              <p>
                CodeQL memeriksa Go, JavaScript/TypeScript, dan workflow GitHub Actions.
                gitleaks mencari secret yang bocor dalam riwayat Git untuk rilis tersebut.
                semgrep memeriksa kode sumber, Dockerfile, dan workflow CI berdasarkan
                aturan keamanan. Pemindaian ini berjalan sebelum setiap rilis dan setiap
                minggu.
              </p>

              <p>
                Temuan CodeQL tingkat <strong>CRITICAL</strong> yang masih terbuka pada
                analisis rilis memblokir publikasi. Temuan <strong>HIGH</strong>{" "}
                menghasilkan peringatan.
              </p>

              <p>
                Codex Security dari OpenAI juga melakukan audit berkala yang lebih mendalam.
                Audit ini dapat menemukan masalah arsitektur yang terlewat oleh pemindaian
                yang lebih khusus. Audit dilakukan terpisah dari pemeriksaan rilis otomatis.
              </p>

              <h3 id="dependency-management">Pengelolaan dependency</h3>

              <p>
                Pembaruan biasa menunggu tujuh hari penuh untuk paket, alat pengembangan,
                dan GitHub Actions. Image, klien database, dan ekstensi verifikasi mandiri
                menunggu empat belas hari. Pengelola memeriksa tanggal publikasi dan
                kompatibilitas, termasuk versi yang ditetapkan secara pasti dan pembaruan
                keamanan yang bisa melewati filter umur rilis. Setiap pengecualian
                dilaporkan beserta alasannya.
              </p>

              <p>
                Dependency Review Action membandingkan calon rilis dengan rilis sebelumnya
                dan memblokir kerentanan baru tingkat <strong>HIGH</strong> atau{" "}
                <strong>CRITICAL</strong>. govulncheck menggagalkan pemeriksaan Go jika kode
                kami dapat memanggil fungsi dengan kerentanan yang diketahui. Perintah{" "}
                <code>go mod verify</code> membandingkan modul Go yang diunduh dengan
                checksum yang tercatat.
              </p>

              <p>
                Sebelum publikasi rilis dan image pengembangan,{" "}
                <code>mise run verify-deps</code> memeriksa versi tetap alat, image, dan
                GitHub Actions. Perintah ini juga memeriksa kecocokan versi lingkungan
                runtime yang terkait, serta hash dan izin eksekusi klien database dalam
                repositori. Pemeriksaan ini mencakup daftar deklarasi dan file lokal yang
                telah ditentukan. Pengelola paket secara terpisah memeriksa versi dalam lock
                file dan checksum paket yang diunduh.
              </p>

              <p>
                Pemeriksaan dependency lokal tidak menjamin keamanan kode pihak ketiga atau
                menerapkan batas umur publikasi untuk setiap dependency. Pemilihan dan
                peninjauan rilis tetap menjadi tanggung jawab pengelola.
              </p>

              <h3 id="container-and-ci-hardening">Keamanan container dan CI</h3>

              <ul>
                <li>
                  Trivy memindai image aplikasi pada amd64 dan arm64. Kerentanan{" "}
                  <strong>CRITICAL</strong> dengan perbaikan yang tersedia menggagalkan
                  pemeriksaan. Kerentanan <strong>HIGH</strong> dengan perbaikan dilaporkan
                  tanpa memblokirnya. Kerentanan yang belum memiliki perbaikan dikecualikan
                  dari pemeriksaan ini.
                </li>
                <li>
                  Pemindaian konfigurasi Trivy yang terpisah memblokir temuan{" "}
                  <strong>HIGH</strong> dan <strong>CRITICAL</strong> pada Dockerfile
                  aplikasi, agen verifikasi, dan image pengujian tertentu.
                </li>
                <li>
                  GitHub Actions pihak ketiga dipatok ke SHA commit lengkap. Perubahan tag
                  tidak dapat diam-diam mengganti kode action yang dijalankan.
                </li>
                <li>
                  Workflow secara default hanya memiliki akses baca ke repositori. Setiap job
                  mendapat izin tambahan hanya saat diperlukan.
                </li>
              </ul>

              <h3 id="testing-and-verification">Pengujian dan verifikasi</h3>

              <p>
                Unit test dan integration test mencakup perilaku kritis. Sebelum setiap
                rilis, siklus backup dan restore lengkap berjalan pada container database
                sungguhan untuk setiap engine dan versi mayor yang didukung. Pengujian
                memastikan data bisa di-restore, bukan sekadar file backup berhasil dibuat.
              </p>

              <p>
                Pemeriksaan rilis juga mencakup lint, pemeriksaan tipe, pengujian browser,
                smoke test image, serta build dan pengujian native pada amd64 dan arm64.
                Publikasi menunggu semua job wajib lolos, termasuk pemeriksaan keamanan di
                atas.
              </p>

              <h3 id="reporting-a-vulnerability">Melaporkan kerentanan</h3>

              <p>
                Laporkan kerentanan secara privat melalui pilihan kontak dalam{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . Jangan membuka issue publik. Laporan keamanan ditangani lebih dahulu.
              </p>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
