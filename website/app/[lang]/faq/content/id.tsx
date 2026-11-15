import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "FAQ - Pertanyaan umum | Databasus",
  description:
    "Pertanyaan umum tentang Databasus, alat backup PostgreSQL yang mendukung MySQL, MariaDB, dan MongoDB. Pelajari backup database localhost, format backup, metode kompresi, dan lainnya.",
  keywords: [
    "FAQ Databasus",
    "pertanyaan backup PostgreSQL",
    "backup database localhost",
    "format backup",
    "kompresi pg_dump",
    "kompresi zstd",
    "bantuan backup PostgreSQL",
    "panduan backup database",
  ],
  openGraph: {
    title: "FAQ - Pertanyaan umum | Databasus",
    description:
      "Pertanyaan umum tentang Databasus, alat backup PostgreSQL yang mendukung MySQL, MariaDB, dan MongoDB. Pelajari backup database localhost, format backup, metode kompresi, dan lainnya.",
    type: "article",
    url: getLocalizedUrl("id", "faq"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "FAQ - Pertanyaan umum | Databasus",
    description:
      "Pertanyaan umum tentang Databasus, alat backup PostgreSQL yang mendukung MySQL, MariaDB, dan MongoDB. Pelajari backup database localhost, format backup, metode kompresi, dan lainnya.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "faq"),
    languages: getLanguageAlternates("faq"),
  },
  robots: "index, follow",
};

export default function FAQPage() {
  const startupRecoveryCommand = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  -e RECOVERY_DUMP_PATH=/databasus-data/recovery/backup.dbbackup \\
  -e RECOVERY_MASTER_KEY_PATH=/databasus-data/recovery/master.key \\
  --restart unless-stopped \\
  databasus/databasus:<version>`;

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Mengapa Databasus tidak memakai format dump SQL mentah untuk backup logis PostgreSQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Untuk backup logis, Databasus memakai format custom pg_dump dengan kompresi zstd karena pengujian menyeluruh menunjukkan format ini paling efisien untuk kecepatan backup dan restore. Format custom dengan kompresi zstd level 5 menawarkan keseimbangan optimal antara kecepatan pembuatan backup, kecepatan restore, dan ukuran file.",
                },
              },
              {
                "@type": "Question",
                name: "Di mana Databasus diinstal?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus diinstal di /opt/databasus/",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana backup fisik dan PITR (Point-in-Time Recovery) bekerja?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus menjalankan backup fisik dari host-nya sendiri secara remote, terhubung ke PostgreSQL melalui protokol replikasi standar, sehingga tidak perlu menginstal apa pun di server database. Database dalam jaringan tertutup bisa dijangkau melalui tunnel SSH. Backup fisik memakai stack native PostgreSQL 17: backup penuh melalui pg_basebackup, backup inkremental tingkat blok melalui pg_basebackup --incremental dengan ringkasan WAL di sisi server (summarize_wal = on), dan streaming WAL terus-menerus melalui pg_receivewal. Backup fisik memerlukan PostgreSQL 17 atau lebih baru; pada versi lebih lama, gunakan backup logis pg_dump. Untuk restore ke waktu tertentu, pg_combinebackup menyusun kembali direktori data yang bisa dijalankan dari backup penuh dan rangkaian backup inkrementalnya. PostgreSQL kemudian memutar ulang WAL hingga waktu target pilihan kamu, sehingga pemulihan bisa dilakukan ke detik mana pun di antara backup. Antarmuka Databasus memberi petunjuk langkah demi langkah untuk restore ke host atau database Docker, baik melalui script siap pakai yang membuat restore cukup dengan satu perintah, maupun dengan mengunduh backup dan menyusun sendiri rangkaian backup penuh, inkremental, dan WAL. Backup inkremental dan WAL bersifat opsional: kamu bisa hanya membuat backup penuh, dan WAL tidak wajib. Kami memakai backup native PostgreSQL 17 karena memanfaatkan mekanisme backup PostgreSQL yang sudah teruji, bukan membuat ulang, serta menjaga kehilangan data tetap mendekati nol.",
                },
              },
              {
                "@type": "Question",
                name: "Mengapa Databasus meninggalkan backup berbasis agen?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Versi Databasus sebelumnya menyediakan agen backup: binary yang berjalan di host database untuk melakukan streaming WAL dan membuat backup fisik secara lokal. Implementasi pertama itu ternyata keliru dan sudah dihapus. Implementasinya terlalu sederhana, hanya menyalin WAL di atas backup penuh, sehingga RTO menjadi lama. Pengguna harus mengatur Databasus sekaligus agen terpisah, padahal menjalankan semuanya secara remote dari satu tempat jauh lebih sederhana. Karena agen berada di luar sistem utama, semua kasus pengujian sulit dicakup. Sebenarnya hanya ada satu masalah yang diselesaikan agen: menjangkau database yang tidak bisa diakses dari luar. Untuk 99% pengguna, hal itu sudah bisa ditangani dengan menjalankan Databasus di jaringan privat atau terhubung melalui SSH, sehingga agen mengulang solusi yang sudah ada dan membuat masalah sederhana jauh lebih rumit dari yang diperlukan. Agen juga tidak bisa berjalan di database terkelola seperti RDS dan Cloud SQL, yang melarang instalasi di host tetapi sudah menyediakan protokol replikasi, sehingga jalur remote tetap diperlukan. Selain itu, ada banyak kasus khusus seputar koneksi terputus, pengelolaan pembaruan agen, dan pengumpulan log dari proses terpisah. Semakin sedikit komponen yang perlu dikelola, semakin andal sistem dalam penggunaan sehari-hari. Backup fisik kini berjalan secara remote dari host Databasus. Backup yang sudah ada tetap aman: jika kamu memperbarui dari versi yang masih memiliki backup agen, Databasus tidak melakukannya diam-diam, tetapi memperingatkan perubahan dan memberi pilihan untuk tetap memakai versi 3.42.0 yang didukung atau menghapus sendiri backup agen lama sebelum memperbarui. Implementasi berbasis agen tetap tersedia hingga versi 3.42.0 dan akan terus bekerja untuk waktu yang lama.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana cara membuat backup Databasus itu sendiri?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus dirancang agar tetap bisa dipulihkan meski server-nya sendiri hilang atau rusak. Databasus bisa membuat backup database internalnya sendiri: akun, pengaturan, kredensial terenkripsi untuk database, penyimpanan, dan kanal notifikasi, jadwal, serta katalog backup. Aktifkan fitur ini di Databasus settings → Databasus backups dengan memilih satu atau beberapa penyimpanan yang sudah ada. Secara default, backup Databasus berjalan setiap jam dan disimpan selama 30 hari. File dienkripsi dengan kunci utama dan tidak berisi kunci itu maupun file backup database kamu, jadi simpan kunci utama secara terpisah. Databasus meminta kamu mengunduhnya, dan kunci ini juga tersimpan sebagai . /databasus-data/secret. key di host. Pemulihan memerlukan file backup, kunci utamanya, dan versi Databasus yang persis sama, termasuk patch. Caranya ada dua. Pertama, taruh kedua file di direktori data yang di-mount, lalu jalankan Databasus dengan RECOVERY_DUMP_PATH dan RECOVERY_MASTER_KEY_PATH di server yang belum memiliki akun dan pengaturannya belum diubah. Kedua, unggah file dan tempel kunci di Databasus settings → Databasus backups; cara ini mengganti instalasi saat ini.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana program open-source Anthropic dan OpenAI mendukung Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Pada Maret 2026, Databasus diterima dalam Claude for Open Source dari Anthropic dan Codex for Open Source dari OpenAI. Dukungan program ini menjadi penanda keandalan: proyek telah dievaluasi secara independen dan diakui oleh para pemimpin industri sebagai infrastruktur open-source penting yang layak didukung.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana AI digunakan dalam pengembangan Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "AI membantu kami mengembangkan Databasus, dari perencanaan dan implementasi hingga pengujian dan dokumentasi. Manusia menentukan arah dan mengambil keputusan akhir. Pengembang membaca setiap baris kode sebelum perubahan digabungkan. Kami menggunakan pengembangan berbasis spesifikasi (SDD) dengan OpenSpec untuk menyepakati perilaku sebelum implementasi. Review independen pada setiap tahap Hook otomatis memulai sesi peninjau terpisah dengan akses hanya baca ke repositori. Setiap agen memeriksa kode terkait dan memberikan temuannya sendiri. Agen pelaksana tidak dapat menyetujui pekerjaannya sendiri. Review dimulai sebelum implementasi. Agen perencanaan terpisah mencari solusi yang lebih sederhana, memeriksa nama yang diusulkan, dan menilai desain. Mereka juga memeriksa kesesuaian persyaratan dengan tugas verifikasi. Peninjau kode: Mencari bug, kasus batas yang terlewat, penanganan error yang salah, kondisi balapan, kebocoran sumber daya, dan masalah keamanan. Peninjau persyaratan: Menelusuri implementasi dan pengujian untuk setiap persyaratan. Memeriksa apakah pengujian benar-benar membuktikan perilaku yang dijanjikan. Peninjau arsitektur: Memeriksa batas modul, arah dependensi, kepemilikan data, migrasi, dan aturan API. Peninjau struktur kode: Memeriksa nama, tanggung jawab file dan fungsi, komentar, serta keterbacaan. Peninjau pengujian: Menilai apakah pengujian dapat mendeteksi perilaku yang hilang atau regresi. Juga memeriksa penantian kondisi, pembersihan, dan bukti verifikasi. Peninjau dokumentasi dan tugas: Memeriksa kesesuaian dokumentasi dengan kode, penyelesaian tugas, dan bukti pemeriksaan yang dilaporkan. Review implementasi lengkap berjalan paralel. Setiap spesifikasi yang berubah memiliki peninjau persyaratannya sendiri. Arsitektur, pengujian, dan dokumentasi memiliki peninjau terpisah. Jadi, perubahan pada dua spesifikasi mendapat lima agen review lengkap. Temuan yang menghalangi penerimaan menghentikan persetujuan pekerjaan. Agen pelaksana memperbaikinya atau menjelaskan mengapa suatu temuan keliru. Peninjau lanjutan memeriksa penyelesaian itu dan perubahan berikutnya. Review yang tidak dapat berjalan tidak dianggap lulus. Pemeriksaan otomatis dan restore sungguhan Review independen didampingi pemeriksaan otomatis yang ketat. golangci-lint dan pemeriksa arsitektur kami memeriksa dependensi backend, batas modul, dan aturan migrasi. Pemeriksaan API memvalidasi kontrak OpenAPI dan kode hasil generasi. Frontend memeriksa TypeScript, React, impor antarlapisan, gaya, dan pelokalan. Ruff dan ty memeriksa format, aturan kode, dan tipe Python. ShellCheck memeriksa skrip shell, sedangkan actionlint memeriksa workflow GitHub Actions. Hook sebelum commit memeriksa file backend, frontend, dan Python yang terkait tanpa menulis ulang file. CI juga menjalankan pemeriksaan lint. Pengujian memeriksa perilaku nyata. Backend diuji melalui API aplikasi, sedangkan mesin database diperiksa dengan database sungguhan dalam container. Skenario menyeluruh membuat backup, melakukan restore, lalu memeriksa data hasil restore. Pengujian lain mencakup upgrade, layar frontend, dan alur browser dengan perbandingan visual. Pemeriksaan dipilih sesuai perubahan. Rangkaian pengujian wajib untuk rilis berjalan pada amd64 dan arm64. Agen pelaksana menjalankan pemeriksaan lint dan pengujian terkait sebelum review. Peninjau memeriksa buktinya tanpa menjalankan ulang perintah tersebut. Sebelum publikasi, CodeQL dan Semgrep menganalisis kode, Gitleaks mencari rahasia, dan dependensi diperiksa untuk menemukan kerentanan. Trivy memindai image container dan Dockerfile. Pemeriksaan CI wajib harus lulus sebelum rilis.",
                },
              },
              {
                "@type": "Question",
                name: "Apakah Databasus punya komunitas?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Ya. Komunitas Databasus berdiskusi di obrolan Telegram (https://t.me/databasus_community) dan Discord (https://databasus.com/discord). Di sana bisa bertanya, berbagi cara memakai Databasus, membahas fitur, dan lebih dulu tahu tentang rilis baru. Cukup pilih aplikasi yang sudah dipakai.",
                },
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
              <h1 id="faq">Pertanyaan umum</h1>
              <p className="text-lg text-gray-400">
                Temukan jawaban atas pertanyaan yang paling sering diajukan tentang Databasus,
                termasuk instalasi, konfigurasi, dan strategi backup.
              </p>
              <h2 id="why-no-raw-sql-dump">
                Mengapa Databasus tidak memakai format dump SQL mentah untuk backup logis
                PostgreSQL?
              </h2>
              <p>
                Untuk backup logis, Databasus memakai <code>pg_dump</code> dalam{" "}
                <strong>format custom</strong> dengan <strong>kompresi zstd level 5</strong>, bukan
                format SQL biasa, karena memberikan keseimbangan paling efisien antara:
              </p>
              <ul>
                <li>Kecepatan pembuatan backup</li>
                <li>Kecepatan restore</li>
                <li>Kompresi ukuran file (hingga 20x lebih kecil daripada format SQL biasa)</li>
              </ul>
              <p>
                Keputusan ini diambil setelah pengujian dan benchmark menyeluruh atas berbagai
                format backup dan metode kompresi PostgreSQL. Baca lebih lanjut tentang pengujiannya
                di{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Backup PostgreSQL: perbandingan kecepatan pg_dump dalam berbagai format dan
                  kompresi
                </a>
                .
              </p>
              <p>Databasus tidak akan menyediakan format dump SQL mentah karena:</p>
              <ul>
                <li>terlalu banyak pilihan memperburuk pengalaman penggunaan;</li>
                <li>membuat kode lebih sulit dipelihara;</li>
                <li>format dump saat ini cocok untuk 99% kasus</li>
              </ul>
              <h2 id="installation-directory">
                Di mana Databasus diinstal jika memakai script .sh?
              </h2>
              <p>
                Databasus diinstal di direktori <code>/opt/databasus/</code>.
              </p>
              <h2 id="pitr">Bagaimana backup fisik dan PITR (Point-in-Time Recovery) bekerja?</h2>
              <p>
                Databasus menjalankan backup fisik{" "}
                <strong>secara remote dari host-nya sendiri</strong>, terhubung ke PostgreSQL
                melalui <strong>protokol replikasi</strong> standar, sehingga tidak perlu menginstal
                apa pun di server database. Jika database berada di jaringan tertutup, Databasus
                bisa menjangkaunya melalui tunnel SSH ke host internal atau bastion, jadi database
                tidak perlu dibuka ke publik.
              </p>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Mengapa ini sekarang bisa dilakukan:</strong> selama bertahun-tahun,
                    alat seperti pgBackRest dan WAL-G harus membuat engine backup inkremental
                    tingkat blok sendiri karena PostgreSQL belum menyediakannya secara native. Ini
                    berubah di PostgreSQL 17, ketika fitur tersebut dikembangkan oleh{" "}
                    <strong>Robert Haas</strong> dengan bantuan <strong>David Steele</strong>,
                    pembuat pgBackRest. PostgreSQL kini menyediakan backup inkremental tingkat blok
                    native di sisi server (<code>pg_basebackup --incremental</code> dan{" "}
                    <code>summarize_wal</code>), jadi Databasus memanfaatkannya, bukan membuat
                    ulang.
                  </p>
                </div>
              </div>
              <p>
                <strong>Cara kerja backup:</strong>
              </p>
              <ul>
                <li>
                  Backup penuh dibuat dengan <code>pg_basebackup</code> dan dialirkan langsung ke
                  Databasus
                </li>
                <li>
                  Backup inkremental tingkat blok memakai <code>pg_basebackup --incremental</code>.
                  Ringkasan WAL di sisi server PostgreSQL 17 (<code>summarize_wal = on</code>)
                  melacak perubahan sehingga hanya blok yang berubah yang ditransfer
                </li>
                <li>
                  WAL dialirkan terus-menerus melalui <code>pg_receivewal</code> untuk menjaga
                  rangkaian pemulihan tetap lengkap di antara backup
                </li>
                <li>
                  Backup fisik memerlukan <strong>PostgreSQL 17 atau lebih baru</strong>; pada versi
                  lebih lama, gunakan backup logis <code>pg_dump</code>
                </li>
              </ul>
              <p>
                <strong>Cara kerja restore:</strong>
              </p>
              <ul>
                <li>
                  <code>pg_combinebackup</code> menyusun kembali direktori data yang bisa dijalankan
                  dari backup penuh dan rangkaian backup inkrementalnya
                </li>
                <li>
                  PostgreSQL kemudian memutar ulang WAL hingga waktu target pilihan kamu, sehingga
                  pemulihan bisa dilakukan ke detik mana pun di antara backup
                </li>
                <li>
                  Setelah PostgreSQL dijalankan, proses pemulihan diselesaikan, server beralih
                  menjadi primary, dan operasi normal dilanjutkan
                </li>
              </ul>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Kamu tidak perlu melakukannya secara manual.</strong> Antarmuka
                    Databasus memberi petunjuk langkah demi langkah untuk restore ke host atau
                    database Docker, baik melalui script siap pakai maupun dengan mengunduh backup
                    secara manual. Kami menyiapkan script agar restore cukup dengan satu perintah,
                    tetapi kamu juga bisa menyusun sendiri rangkaian backup penuh, inkremental, dan
                    WAL. Backup inkremental dan WAL juga opsional: kamu bisa hanya membuat backup
                    penuh tanpa backup inkremental, dan WAL tidak wajib.
                  </p>
                </div>
              </div>
              <p>
                <strong>Alasan kami memakai backup native PG 17:</strong>
              </p>
              <ul>
                <li>
                  Memanfaatkan mekanisme backup PostgreSQL sendiri alih-alih membuat ulang, sehingga
                  bagian internalnya sudah teruji dengan ribuan pengujian dan kasus khusus
                </li>
                <li>
                  Menjaga kehilangan data tetap mendekati nol, dengan restore ke detik mana pun di
                  antara backup
                </li>
              </ul>
              <h2 id="why-no-agent">Mengapa Databasus meninggalkan backup berbasis agen?</h2>
              <p>
                Versi Databasus sebelumnya menyediakan <strong>agen</strong> backup: binary yang
                berjalan di host database untuk melakukan streaming WAL dan membuat backup fisik
                secara lokal. Implementasi pertama itu ternyata keliru, dan kami menghapusnya.
                Backup fisik kini berjalan secara remote dari host Databasus, seperti dijelaskan di
                atas.
              </p>
              <p>
                <strong>Mengapa pendekatan agen tidak tepat:</strong>
              </p>
              <ul>
                <li>
                  Implementasinya terlalu sederhana, hanya menyalin WAL di atas backup penuh,
                  sehingga RTO menjadi lama
                </li>
                <li>
                  Pengguna harus mengatur Databasus sekaligus agen terpisah, padahal menjalankan
                  semuanya secara remote dari satu tempat jauh lebih sederhana
                </li>
                <li>
                  Karena agen berada di luar sistem utama, semua kasus pengujian sulit dicakup
                </li>
                <li>
                  Sebenarnya hanya satu masalah yang diselesaikan agen: menjangkau database yang
                  tidak bisa diakses dari luar. Untuk 99% pengguna, hal itu sudah ditangani dengan
                  menjalankan Databasus di jaringan privat atau terhubung melalui SSH. Agen jadi
                  mengulang solusi yang sudah ada dan membuat masalah sederhana jauh lebih rumit
                  dari yang diperlukan
                </li>
                <li>
                  Agen tidak bisa berjalan di database terkelola seperti RDS dan Cloud SQL, yang
                  melarang instalasi di host tetapi sudah menyediakan protokol replikasi, sehingga
                  jalur remote tetap diperlukan
                </li>
                <li>
                  Ada banyak kasus khusus yang menyulitkan: koneksi terputus, pengelolaan pembaruan
                  agen, dan pengumpulan log dari proses terpisah. Semakin sedikit komponen yang
                  perlu dikelola, semakin andal sistem dalam penggunaan sehari-hari
                </li>
              </ul>
              <p>
                <strong>Kami memastikan backup yang sudah ada tetap aman.</strong> Jika kamu
                memperbarui dari versi yang masih memiliki backup agen, Databasus tidak melakukannya
                diam-diam. Databasus memperingatkan perubahan dan memberi pilihan untuk tetap
                memakai <strong>versi 3.42.0</strong> yang didukung atau menghapus sendiri backup
                agen lama sebelum memperbarui. Implementasi berbasis agen tetap tersedia hingga
                versi 3.42.0 dan akan terus bekerja untuk waktu yang lama, jadi tidak ada yang
                rusak.
              </p>
              <h2 id="ai-usage">Bagaimana AI digunakan dalam pengembangan Databasus?</h2>

              <p>
                AI membantu kami mengembangkan Databasus, dari perencanaan dan implementasi hingga pengujian dan
                dokumentasi. Manusia menentukan arah dan mengambil keputusan akhir.{" "}
                <strong>Pengembang membaca setiap baris kode sebelum perubahan digabungkan.</strong>
              </p>

              <p>
                Kami menggunakan pengembangan berbasis spesifikasi (SDD) dengan{" "}
                <a href="https://github.com/databasus/databasus/tree/main/openspec">OpenSpec</a> untuk
                menyepakati perilaku sebelum implementasi.
              </p>

              <h3 id="ai-review-hooks">Review independen pada setiap tahap</h3>

              <p>
                Hook otomatis memulai sesi peninjau terpisah dengan akses hanya baca ke repositori. Setiap agen
                memeriksa kode terkait dan memberikan temuannya sendiri. Agen pelaksana tidak dapat menyetujui
                pekerjaannya sendiri.
              </p>

              <p>
                Review dimulai sebelum implementasi. Agen perencanaan terpisah mencari solusi yang lebih
                sederhana, memeriksa nama yang diusulkan, dan menilai desain. Mereka juga memeriksa kesesuaian
                persyaratan dengan tugas verifikasi.
              </p>

              <ul>
                <li>
                  <strong>Peninjau kode:</strong> Mencari bug, kasus batas yang terlewat, penanganan error yang
                  salah, kondisi balapan, kebocoran sumber daya, dan masalah keamanan.
                </li>
                <li>
                  <strong>Peninjau persyaratan:</strong> Menelusuri implementasi dan pengujian untuk setiap
                  persyaratan. Memeriksa apakah pengujian benar-benar membuktikan perilaku yang dijanjikan.
                </li>
                <li>
                  <strong>Peninjau arsitektur:</strong> Memeriksa batas modul, arah dependensi, kepemilikan
                  data, migrasi, dan aturan API.
                </li>
                <li>
                  <strong>Peninjau struktur kode:</strong> Memeriksa nama, tanggung jawab file dan fungsi,
                  komentar, serta keterbacaan.
                </li>
                <li>
                  <strong>Peninjau pengujian:</strong> Menilai apakah pengujian dapat mendeteksi perilaku yang
                  hilang atau regresi. Juga memeriksa penantian kondisi, pembersihan, dan bukti verifikasi.
                </li>
                <li>
                  <strong>Peninjau dokumentasi dan tugas:</strong> Memeriksa kesesuaian dokumentasi dengan kode,
                  penyelesaian tugas, dan bukti pemeriksaan yang dilaporkan.
                </li>
              </ul>

              <p>
                Review implementasi lengkap berjalan paralel. Setiap spesifikasi yang berubah memiliki peninjau
                persyaratannya sendiri. Arsitektur, pengujian, dan dokumentasi memiliki peninjau terpisah. Jadi,
                perubahan pada dua spesifikasi mendapat lima agen review lengkap.
              </p>

              <p>
                <strong>Temuan yang menghalangi penerimaan menghentikan persetujuan pekerjaan.</strong> Agen
                pelaksana memperbaikinya atau menjelaskan mengapa suatu temuan keliru. Peninjau lanjutan
                memeriksa penyelesaian itu dan perubahan berikutnya. Review yang tidak dapat berjalan tidak
                dianggap lulus.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/.agent-hooks/review_gate.py">
                  Implementasi hook review
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/tree/main/.agent-hooks">Instruksi peninjau</a>
              </p>

              <h3 id="ai-linting">Pemeriksaan otomatis dan restore sungguhan</h3>

              <p>
                Review independen didampingi pemeriksaan otomatis yang ketat. golangci-lint dan pemeriksa
                arsitektur kami memeriksa dependensi backend, batas modul, dan aturan migrasi. Pemeriksaan API
                memvalidasi kontrak OpenAPI dan kode hasil generasi. Frontend memeriksa TypeScript, React, impor
                antarlapisan, gaya, dan pelokalan.
              </p>

              <p>
                Ruff dan ty memeriksa format, aturan kode, dan tipe Python. ShellCheck memeriksa skrip shell,
                sedangkan actionlint memeriksa workflow GitHub Actions. Hook sebelum commit memeriksa file
                backend, frontend, dan Python yang terkait tanpa menulis ulang file. CI juga menjalankan
                pemeriksaan lint.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/mise-tasks/backend/lint">
                  Pemeriksaan lint backend
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/frontend/lint/check.sh">
                  Pemeriksaan lint frontend
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/scripts/python_quality.py">
                  Pemeriksaan Python
                </a>
              </p>

              <p>
                Pengujian memeriksa perilaku nyata. Backend diuji melalui API aplikasi, sedangkan mesin database
                diperiksa dengan database sungguhan dalam container. Skenario menyeluruh membuat backup,
                melakukan restore, lalu memeriksa data hasil restore. Pengujian lain mencakup upgrade, layar
                frontend, dan alur browser dengan perbandingan visual.
              </p>

              <p>
                Pemeriksaan dipilih sesuai perubahan. Rangkaian pengujian wajib untuk rilis berjalan pada amd64
                dan arm64. Agen pelaksana menjalankan pemeriksaan lint dan pengujian terkait sebelum review.
                Peninjau memeriksa buktinya tanpa menjalankan ulang perintah tersebut.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/docs/testing.md">
                  Strategi pengujian dan perintah
                </a>
              </p>

              <p>
                Sebelum publikasi, CodeQL dan Semgrep menganalisis kode, Gitleaks mencari rahasia, dan
                dependensi diperiksa untuk menemukan kerentanan. Trivy memindai image container dan Dockerfile.
              </p>

              <p>
                <strong>Pemeriksaan CI wajib harus lulus sebelum rilis.</strong>{" "}
                <a href={`${getLocalizedHref("id", "security")}#security-and-reliability-engineering`}>
                  Keamanan dan keandalan
                </a>
                .
              </p>

              <h2 id="backup-databasus">Bagaimana cara membuat backup Databasus itu sendiri?</h2>

              <p>
                Databasus dirancang agar tetap bisa dipulihkan meski server-nya sendiri hilang atau rusak.
                Databasus bisa membuat backup database internalnya sendiri:
                akun, pengaturan,
                kredensial terenkripsi untuk database, penyimpanan, dan kanal
                notifikasi kamu,
                jadwal, serta katalog backup. Aktifkan fitur ini di{" "}
                <strong>Databasus settings → Databasus backups</strong> dengan
                memilih satu atau
                beberapa penyimpanan yang sudah ada. Secara default, backup
                Databasus berjalan setiap
                jam dan disimpan selama 30 hari. Setiap file dienkripsi dengan
                kunci utama.
              </p>

              <p>
                Backup Databasus tidak berisi kunci utama maupun file backup
                database kamu, karena
                file tersebut tetap berada di penyimpanannya masing-masing.{" "}
                <strong>Simpan kunci utama secara terpisah.</strong> Databasus
                meminta kamu
                mengunduhnya pada kunjungan pertama, dan kunci ini juga
                tersimpan sebagai file{" "}
                <code>./databasus-data/secret.key</code> di host.
              </p>

              <ul>
                <li>
                  <strong>Pilih penyimpanan remote.</strong> Penyimpanan lokal
                  di server yang sama akan hilang bersama server itu.
                </li>
                <li>
                  <strong>Peringatan.</strong> Databasus menampilkan
                  pemberitahuan saat backup Databasus gagal, hanya tersimpan di
                  sebagian penyimpanan, berhenti, atau terlambat. Terlambat
                  berarti belum ada salinan yang berhasil 15 menit setelah
                  jadwal berjalan. Peringatan yang sama juga dikirim lewat email
                  melalui{" "}
                  <a href={`${getLocalizedHref("id", "advanced-config")}#email-smtp`}>
                  mail server instance
                  </a>{" "}
                  jika sudah dikonfigurasi.
                </li>
              </ul>

              <h3 id="recover-databasus">Bagaimana cara memulihkan Databasus dari backup Databasus?</h3>

              <p>
                Diperlukan dua hal: file backup Databasus
                (<code>.dbbackup</code>) yang diunduh dari riwayat atau diambil
                langsung dari penyimpanan, dan kunci utama instalasi yang
                membuatnya.{" "}
                <strong>
                Pemulihan hanya berjalan pada versi Databasus yang persis sama,
                termasuk patch.
                </strong>{" "}
                Untuk versi lain, Databasus menolak dan menyebutkan versi yang
                harus dijalankan: pulihkan di versi itu, lalu upgrade seperti
                biasa.
              </p>

              <p>
                <strong>Cara 1: file saat startup.</strong> Cara ini untuk
                server baru. Taruh kedua file di direktori data yang di-mount,
                misalnya{" "}
                <code>./databasus-data/recovery/</code>, lalu jalankan image
                versi yang membuat backup dengan dua pengaturan tambahan.
                Ganti <code>&lt;version&gt;</code> dengan versi tersebut:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startupRecoveryCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startupRecoveryCommand} lang="id" />
                </div>
              </div>

              <ul>
                <li>
                  Jika Databasus sudah pernah dijalankan secara normal di server
                  ini, tetapi belum ada akun yang dibuat dan pengaturan belum
                  diubah, datanya tidak perlu dihapus. Hentikan Databasus,
                  tambahkan kedua file dan kedua pengaturan, lalu jalankan lagi.
                </li>
                <li>
                  Pemulihan saat startup tidak pernah mengganti instalasi yang
                  memiliki akun (bahkan hanya satu akun), data pengguna lain,
                  atau pengaturan yang sudah diubah. Dalam kasus itu Databasus
                  tidak berjalan. Log-nya menjelaskan bahwa pemulihan tidak
                  dijalankan, alasannya, bahwa instalasi saat ini tidak berubah,
                  dan langkah berikutnya. Untuk menjalankan instalasi saat ini,
                  hapus kedua pengaturan tersebut, atau gunakan unggahan di
                  antarmuka.
                </li>
                <li>
                  Setelah pemulihan berhasil, file dan pengaturan boleh
                  dibiarkan. Databasus mengingat SHA-256 file yang sudah
                  diterapkan, sehingga saat restart dan upgrade berikutnya file
                  itu dilewati dan log mencatatnya, meskipun nama file sudah
                  diubah. File lain di path yang sama tidak dilewati.
                </li>
              </ul>

              <p>
                <strong>Cara 2: unggah di antarmuka.</strong> Cara ini untuk
                instalasi yang sudah berjalan. Pada instalasi baru, daftar dulu.
                Buka <strong>Databasus settings → Databasus backups</strong>,
                pilih file backup, lalu tempel atau ketik kunci utama instalasi
                yang membuatnya.
                Seluruh isi instalasi saat ini diganti: akun, pengaturan dan
                katalog backup. Setelahnya, masuk dengan akun dari backup.
              </p>

              <h2 id="oss-programs">
                Bagaimana program open-source Anthropic dan OpenAI mendukung Databasus?
              </h2>
              <p>
                Pada Maret 2026, Databasus diterima dalam{" "}
                <strong>
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Claude for Open Source
                  </a>
                </strong>{" "}
                dari Anthropic dan{" "}
                <strong>
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Codex for Open Source
                  </a>
                </strong>{" "}
                dari OpenAI. Pengakuan dari dua perusahaan AI terkemuka dunia bahwa proyek ini
                merupakan perangkat lunak open-source penting bagi industri sangat berarti bagi
                kami, terutama mengingat persyaratan kelayakan kedua program yang tinggi.
              </p>
              <p>
                Apa artinya bagi pengguna? Ini menjadi satu lagi penanda keandalan bahwa proyek
                telah dievaluasi secara independen dan diakui oleh para pemimpin industri sebagai
                infrastruktur penting yang layak didukung. Akses tanpa batas ke AI terbaru membantu
                kami meningkatkan kualitas kode, mempercepat review keamanan, dan terus
                mengembangkan proyek secara aktif.
              </p>
              <img
                src="/images/faq/anthropic-email.png"
                alt="Databasus diterima dalam program Claude for Open Source dari Anthropic"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
              <img
                src="/images/faq/openai-email.png"
                alt="Databasus diterima dalam program Codex for Open Source dari OpenAI"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h2 id="community">Apakah Databasus punya komunitas?</h2>

              <p>
                Ya. Komunitas Databasus berdiskusi di obrolan{" "}
                <a
                  href="https://t.me/databasus_community"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Telegram
                </a>{" "}
                dan{" "}
                <a
                  href="/discord"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Discord
                </a>
                . Di sana bisa bertanya, berbagi cara memakai Databasus, membahas fitur, dan lebih dulu tahu tentang rilis baru.
              </p>

              <p>Cukup pilih aplikasi yang sudah dipakai.</p>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
