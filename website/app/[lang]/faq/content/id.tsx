import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

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
                  text: "Databasus menjalankan backup fisik dari host-nya sendiri secara remote, terhubung ke PostgreSQL melalui protokol replikasi standar, sehingga tidak perlu menginstal apa pun di server database. Database dalam jaringan tertutup bisa dijangkau melalui tunnel SSH. Backup fisik memakai stack native PostgreSQL 17: backup penuh melalui pg_basebackup, backup inkremental tingkat blok melalui pg_basebackup --incremental dengan ringkasan WAL di sisi server (summarize_wal = on), dan streaming WAL terus-menerus melalui pg_receivewal. Backup fisik memerlukan PostgreSQL 17 atau lebih baru; pada versi lebih lama, gunakan backup logis pg_dump. Untuk restore ke waktu tertentu, pg_combinebackup menyusun kembali direktori data yang bisa dijalankan dari backup penuh dan rangkaian backup inkrementalnya. PostgreSQL kemudian memutar ulang WAL hingga waktu target pilihan kamu, sehingga pemulihan bisa dilakukan ke detik mana pun di antara backup. Antarmuka Databasus memberi petunjuk langkah demi langkah untuk restore ke host atau database Docker, baik melalui script siap pakai yang membuat restore cukup dengan satu perintah, maupun dengan mengunduh backup dan menyusun sendiri rangkaian backup penuh, inkremental, dan WAL. Backup inkremental dan WAL bersifat opsional: kamu bisa hanya membuat backup penuh, dan WAL tidak wajib. Kami memakai backup native PostgreSQL 17 karena memanfaatkan mekanisme backup PostgreSQL yang sudah teruji, bukan membuat ulang, bekerja dengan database remote termasuk layanan terkelola seperti RDS dan Cloud SQL, serta memberi kehilangan data yang mendekati nol.",
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
                  text: "Untuk membuat backup Databasus, buka /opt/databasus (atau folder instalasinya), lalu masuk ke direktori databasus-data. Kamu perlu membuat backup file secret.key (kunci enkripsi kredensial) dan folder /pgdata (database internal yang berisi konfigurasi dan metadata backup). Ada dua skenario pemulihan: 1) Backup database bisa dipulihkan hanya dengan secret.key tanpa antarmuka Databasus (lihat panduan pemulihan manual), 2) Untuk restore antarmuka Databasus beserta seluruh konfigurasi dan riwayat, diperlukan secret.key dan folder /pgdata. Untuk restore, buat kembali struktur folder ini di server lain.",
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
                  Bekerja dengan database remote, termasuk layanan terkelola seperti Amazon RDS dan
                  Google Cloud SQL yang menyediakan protokol replikasi tetapi melarang instalasi
                  perangkat lunak di host
                </li>
                <li>
                  Memberi kehilangan data yang mendekati nol, dengan restore ke detik mana pun di
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
              <p>
                Alasan lengkapnya ada dalam catatan keputusan arsitektur:{" "}
                <a
                  href="https://github.com/databasus/databasus/blob/main/adr/0008-why-pg17-native-backups-with-mandatory-wal-summary.md"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  ADR-0008: backup native PG17 dengan ringkasan WAL wajib
                </a>{" "}
                dan{" "}
                <a
                  href="https://github.com/databasus/databasus/blob/main/adr/0009-why-remote-physical-backups-instead-of-agents.md"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  ADR-0009: backup fisik remote, bukan agen
                </a>
                .
              </p>
              <h2 id="backup-databasus">Bagaimana cara membuat backup Databasus itu sendiri?</h2>
              <p>
                Untuk membuat backup instance Databasus (termasuk semua konfigurasi, database, dan
                kredensial), ikuti langkah berikut:
              </p>
              <ol>
                <li>
                  Buka <code>/opt/databasus</code> (atau folder tempat Databasus diinstal)
                </li>
                <li>
                  Masuk ke direktori <code>databasus-data</code>
                </li>
              </ol>
              <p>
                <strong>Yang perlu di-backup:</strong>
              </p>
              <ul>
                <li>
                  <code>secret.key</code>: kunci enkripsi kredensial
                </li>
                <li>
                  <code>/pgdata</code>: database PostgreSQL internal Databasus yang berisi semua
                  konfigurasi dan metadata backup
                </li>
              </ul>
              <p>
                Jika memakai penyimpanan lokal untuk backup, kamu juga bisa membuat backup folder{" "}
                <code>backups</code>.
              </p>
              <p>
                <strong>Penting:</strong> Ada dua skenario pemulihan yang berbeda:
              </p>
              <ul>
                <li>
                  <strong>Memulihkan backup tanpa antarmuka Databasus:</strong> Backup database bisa
                  dipulihkan hanya dengan file <code>secret.key</code>, tanpa memerlukan Databasus
                  atau data internalnya. Lihat{" "}
                  <a href={getLocalizedHref("id", "how-to-recover-without-databasus")}>
                    panduan pemulihan manual
                  </a>{" "}
                  untuk petunjuk lengkap.
                </li>
                <li>
                  <strong>Restore antarmuka Databasus dan seluruh konfigurasi:</strong> Untuk
                  restore antarmuka Databasus beserta seluruh konfigurasi, backup terjadwal, dan
                  riwayat backup, kamu perlu membuat backup <code>secret.key</code> sekaligus folder{" "}
                  <code>/pgdata</code> (yang berisi metadata enkripsi dan seluruh konfigurasi
                  Databasus).
                </li>
              </ul>
              <p>
                <strong>Untuk restore Databasus di server lain:</strong> buat kembali struktur
                folder <code>databasus-data</code> dengan file hasil backup, lalu jalankan
                Databasus.
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
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
