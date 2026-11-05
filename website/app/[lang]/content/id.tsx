import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import HeroStatsComponent from "@/app/components/HeroStatsComponent";
import InstallationComponent from "@/app/components/InstallationComponent";
import LiteYouTubeEmbed from "@/app/components/LiteYouTubeEmbed";
import { VIDEO_PLAY_LABELS } from "@/app/components/copyLabels";

export const metadata: Metadata = {
  title: "Backup PostgreSQL | Databasus",
  description:
    "Alat gratis dan open source untuk backup PostgreSQL terjadwal (juga mendukung MySQL dan MongoDB). Simpan secara lokal atau di cloud. Notifikasi ke Slack, Discord, Telegram, email, webhook, dan lainnya.",
  keywords:
    "PostgreSQL, backup, pemantauan, database, backup terjadwal, Docker, self-hosted, open source, S3, Google Drive, notifikasi Slack, Discord, DevOps, pemantauan database, pg_dump, restore database, enkripsi, AES-256, enkripsi backup",
  robots: "index, follow",
  alternates: { canonical: getLocalizedUrl("id", ""), languages: getLanguageAlternates("") },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("id", ""),
    title: "Backup PostgreSQL | Databasus",
    description:
      "Alat gratis dan open source untuk backup PostgreSQL terjadwal (juga mendukung MySQL dan MongoDB). Simpan secara lokal atau di cloud. Notifikasi ke Slack, Discord, Telegram, email, webhook, dan lainnya.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Antarmuka dashboard Databasus untuk mengelola backup",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary_large_image",
    title: "Backup PostgreSQL | Databasus",
    description:
      "Alat gratis dan open source untuk backup PostgreSQL terjadwal (juga mendukung MySQL dan MongoDB). Simpan secara lokal atau di cloud. Notifikasi ke Slack, Discord, Telegram, email, webhook, dan lainnya.",
    images: ["https://databasus.com/images/index/dashboard.png"],
  },
  applicationName: "Databasus",
  appleWebApp: { title: "Databasus", capable: true },
  icons: {
    icon: [
      { url: "/favicon.ico", type: "image/x-icon" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: "/favicon.svg",
    shortcut: "/favicon.ico",
  },
};

export default function Index() {
  return (
    <div className="overflow-x-hidden">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            name: "Databasus",
            description:
              "Alat gratis dan open source untuk backup PostgreSQL terjadwal (juga mendukung MySQL dan MongoDB). Simpan secara lokal atau di cloud. Notifikasi ke Slack, Discord, Telegram, email, webhook, dan lainnya.",
            url: getLocalizedUrl("id", ""),
            image: "https://databasus.com/images/index/dashboard.png",
            logo: "https://databasus.com/logo.svg",
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: { "@type": "ImageObject", url: "https://databasus.com/logo.svg" },
            },
            featureList: [
              "Backup PostgreSQL terjadwal",
              "Beragam tujuan penyimpanan (S3, Google Drive, Dropbox, SFTP, rclone, dan lainnya)",
              "Notifikasi real-time (Slack, Telegram, Discord, Webhook, email, dan lainnya)",
              "Pemantauan kesehatan database",
              "Self-hosted melalui Docker",
              "Open source dan gratis",
              "Mendukung PostgreSQL 12-18",
              "Kompresi backup dan enkripsi AES-256-GCM",
              "Mendukung PostgreSQL, MySQL, MariaDB, dan MongoDB",
              "Kebijakan retensi: jangka waktu, jumlah, GFS, dan batas ukuran",
              "Point-in-Time Recovery (PITR) dengan pengarsipan WAL",
              "Verifikasi restore: pengujian restore otomatis di container Docker database sungguhan",
            ],
            screenshot: "https://databasus.com/images/index/dashboard.png",
            softwareVersion: "latest",
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "Organization",
            name: "Databasus",
            url: "https://databasus.com/",
            alternateName: ["databasus", "Databasus"],
            logo: "https://databasus.com/logo.svg",
            sameAs: ["https://github.com/databasus/databasus"],
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: "Databasus",
            alternateName: ["databasus", "Databasus"],
            url: getLocalizedUrl("id", ""),
            description: "Alat backup PostgreSQL",
            publisher: { "@type": "Organization", name: "Databasus" },
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Apa itu Databasus dan mengapa perlu menggunakannya alih-alih script buatan sendiri?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus adalah layanan self-hosted berlisensi Apache 2.0 untuk backup PostgreSQL v13 hingga v18. Berbeda dari shell script, Databasus menyediakan antarmuka untuk menjadwalkan tugas, mengompresi dan menyimpan arsip ke berbagai tujuan (disk lokal, S3, Google Drive, Dropbox, SFTP, rclone, dan lainnya), mengatur kebijakan retensi agar backup lama dihapus otomatis, serta memberi tahu tim saat tugas selesai atau gagal, tanpa menulis kode sendiri",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana cara tercepat memasang Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Cara paling langsung adalah menjalankan installer cURL satu baris. Installer mengambil image Docker terbaru dan menjalankan satu container PostgreSQL. Setelah itu, installer membuat docker-compose.yml dan menjalankan layanan agar otomatis aktif kembali setelah reboot. Seluruh proses biasanya kurang dari dua menit pada VPS umum.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana cara kerja verifikasi restore?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus menjalankan agen verifikasi kecil di host yang kamu kendalikan. Pada setiap jadwal, agen mengunduh backup terbaru dan melakukan restore ke container database sementara. Setelah itu, agen memeriksa kewajaran hasil restore dengan membandingkannya terhadap sumber. Hasilnya dilaporkan kembali, termasuk exit code restore dan jumlah baris per tabel. Jadwal bisa dijalankan setelah backup, per jam, harian, mingguan, bulanan, atau menggunakan ekspresi cron UTC. Kegagalan bisa dilaporkan melalui kanal notifikasi mana pun yang terhubung ke database, seperti Slack, Teams, Discord, email, dan lainnya.",
                },
              },
              {
                "@type": "Question",
                name: "Di mana backup disimpan dan berapa ruang yang dibutuhkan?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Arsip bisa disimpan di volume lokal, bucket kompatibel S3, Google Drive, Dropbox, dan tujuan cloud lainnya. Databasus menggunakan kompresi seimbang yang biasanya memperkecil dump 4-8x dengan tambahan waktu proses hanya sekitar 20%, sehingga menghemat penyimpanan dan bandwidth.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana saya tahu backup berhasil atau gagal?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus bisa mengirim notifikasi real-time melalui email, Slack, Telegram, webhook, Mattermost, Discord, dan lainnya. Kamu bisa memilih kanal yang diberi tahu agar tim DevOps langsung mengetahui keberhasilan maupun kegagalan, sehingga rutinitas pemulihan dan audit kepatuhan lebih mudah dilakukan.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana Databasus menjaga keamanan?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus menerapkan keamanan pada tiga tingkat: (1) Enkripsi data sensitif: semua password, token, dan kredensial dienkripsi dengan AES-256-GCM dan disimpan terpisah dari database; (2) Enkripsi backup: setiap file backup dienkripsi dengan key unik yang diturunkan dari master key, ID backup, dan salt acak, sehingga backup tidak berguna tanpa key enkripsi kamu meskipun seseorang mendapat akses penyimpanan; (3) Akses database read-only: Databasus hanya memerlukan izin SELECT dan melakukan pemeriksaan menyeluruh untuk memastikan tidak ada hak tulis, sehingga mencegah kerusakan data meskipun alat ini disusupi. Login juga bisa mewajibkan faktor kedua: jika diaktifkan, password yang benar diikuti kode enam digit yang dikirim ke email akun, sedangkan login melalui Google atau GitHub tetap mengandalkan pemeriksaan penyedia tersebut. Di luar runtime, keamanan dan keandalan diperiksa pada setiap commit dan PR melalui analisis statis CodeQL, pemindaian secret gitleaks, aturan keamanan semgrep, pemantauan CVE Dependabot, pemindaian image dan Dockerfile Trivy, serta audit berkala Codex Security dari OpenAI. Integration test berjalan pada container PostgreSQL, MySQL, MariaDB, dan MongoDB sungguhan serta memverifikasi siklus backup lalu restore lengkap pada setiap PR. GitHub Actions dipatok ke SHA commit dan workflow mengikuti prinsip hak akses minimum. Semua operasi berjalan di container yang kamu kendalikan pada server milik kamu. Karena open source, tim keamanan bisa mengaudit setiap baris kode sebelum deployment.",
                },
              },
              {
                "@type": "Question",
                name: "Apakah Databasus didukung program OSS Anthropic dan OpenAI?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Ya, pada Maret 2026 Databasus diterima dalam Claude for Open Source dari Anthropic dan Codex for Open Source dari OpenAI. Proyek ini telah dievaluasi secara independen dan diakui oleh para pemimpin industri sebagai infrastruktur open source penting yang layak didukung.",
                },
              },
              {
                "@type": "Question",
                name: "Apa perbedaan Databasus dengan PgBackRest, Barman, atau pg_dump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus mengutamakan kesederhanaan. Antarmuka web modern memudahkan pengelolaan backup banyak database sekaligus, dengan penjadwalan, kompresi, berbagai tujuan penyimpanan, pemantauan kesehatan, dan notifikasi real-time bawaan. Berbeda dari pgBackRest dan WAL-G, Databasus membuat backup fisik, incremental, dan WAL menggunakan mekanisme native PostgreSQL 17, bukan membuat mesin backup sendiri. Databasus terhubung ke database dari jarak jauh dan menjangkau jaringan tertutup melalui SSH tunnel ke server atau bastion, sehingga database yang tidak terbuka ke publik tetap bisa di-backup dan dikelola dari satu dashboard.",
                },
              },
              {
                "@type": "Question",
                name: "Database apa saja yang didukung Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus mendukung PostgreSQL, MySQL, MariaDB, dan MongoDB. Namun, Databasus awalnya dibuat khusus untuk PostgreSQL dan tetap berfokus padanya, dengan dukungan 100% dan efisiensi maksimal untuk backup PostgreSQL. Meskipun MySQL, MariaDB, dan MongoDB juga didukung, PostgreSQL tetap menjadi prioritas utama dengan fitur paling optimal dan pengembangan berkelanjutan. Misalnya, Databasus menyediakan dukungan native untuk backup fisik dan WAL bagi pemulihan bencana PostgreSQL. Jadi, Databasus pada dasarnya adalah alat backup PostgreSQL; database lain merupakan perluasan dukungannya.",
                },
              },
              {
                "@type": "Question",
                name: "Seberapa luas Databasus digunakan?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus saat ini merupakan alat backup PostgreSQL open source yang paling banyak digunakan. Per 17 Juni 2026, image Docker-nya telah ditarik lebih dari 1.800.000 kali oleh DBA, engineer DevOps, developer, dan tim di seluruh dunia. Dengan 8.500+ GitHub stars, Databasus melampaui pgBackRest (~4.200 stars, tersedia sejak 2014) dan WAL-G (~4.100 stars, tersedia sejak 2017). Databasus diluncurkan pada 2025 dan melampaui keduanya dalam tahun pertamanya. Tingkat adopsi ini mencerminkan kepercayaan dan pilihan komunitas profesional database.",
                },
              },
              {
                "@type": "Question",
                name: "Jenis backup apa saja yang didukung Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus mendukung backup fisik, penuh, incremental, WAL, dan logis. Backup fisik menyalin seluruh cluster database pada tingkat file, lebih cepat untuk backup dan restore dataset besar dibanding dump logis, serta dibangun di atas mekanisme backup native PostgreSQL 17. Dengan begitu, kami mengandalkan alat PostgreSQL yang telah teruji, bukan membuatnya ulang. Backup penuh adalah salinan cluster lengkap yang mandiri dan menjadi dasar setiap rantai backup. Backup incremental hanya menyimpan perubahan sejak backup sebelumnya, sehingga ukurannya kecil dan prosesnya cepat. WAL streaming menangkap aliran penulisan database secara terus-menerus untuk Point-in-time recovery (PITR), pemulihan bencana, dan kehilangan data mendekati nol. Backup logis berupa dump native database dalam format biner khusus engine-nya, dikompresi dan dialirkan langsung ke penyimpanan tanpa file perantara. Semua jenis backup ini bisa berjalan melalui SSH tunnel jika koneksi harus tetap privat, sehingga database tidak perlu dibuka ke publik. SSH tunneling sudah tersedia secara bawaan.",
                },
              },
              {
                "@type": "Question",
                name: "Saya lupa email atau password admin",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: 'Akun pertama yang dibuat pada instance bertugas mengelolanya. Jika lupa alamatnya, jalankan docker exec -it databasus ./main --list-admins pada server tempat Databasus berjalan. Perintah ini menampilkan setiap akun administrator beserta alamat email, nama tampilan, tanggal pembuatan, dan status aktifnya, tanpa menampilkan informasi password. Untuk mengatur password baru akun tersebut, jalankan docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com". Instance yang dibuat sebelum akun pertama bertugas mengelolanya masih menggunakan alamat placeholder admin sampai pemilik menggantinya. Jika instance juga mewajibkan kode email saat login dan mail server tidak lagi mengirimkannya, jalankan docker exec -it databasus ./main --disable-2fa untuk menonaktifkan kewajiban kode. Ketiga perintah didokumentasikan di halaman password.',
                },
              },
            ],
          }),
        }}
      />
      <LandingHeaderComponent
        lang="id"
        sectionLinks={[{ href: "#how-to-use", label: "Cara penggunaan" }]}
      />
      <main className="relative overflow-hidden pt-[60px] md:pt-[68px]">
        <div className="relative mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px] px-4 md:px-6 lg:px-0 pt-12 md:pt-[100px] pb-12 md:pb-[100px]">
          <div className="relative">
            <div className="absolute left-1/2 -translate-x-1/2 -translate-y-1/4 w-[400px] h-[400px] md:w-[900px] md:h-[900px] bg-[#155dfc]/4 top-0 rounded-full blur-3xl -z-10" />
          </div>
          <div className="text-center mb-8 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Databasus</span>
            </div>
            <h1 className="text-2xl sm:text-4xl sm:max-w-[300px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[650px]">
              Alat backup PostgreSQL dengan Point-in-time-recovery dan verifikasi restore
            </h1>
            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus adalah alat gratis, open source, dan self-hosted untuk backup PostgreSQL.
              Gunakan berbagai penyimpanan (S3, Google Drive, FTP, dan lainnya) serta notifikasi
              perkembangan (Slack, Discord, Telegram, dan lainnya). Berfokus pada Point-in-Time
              Recovery{" "}
              <span className="underline decoration-2 underline-offset-2 decoration-blue-600">
                dengan RPO/RTO rendah
              </span>
            </p>
            <div>
              <div className="flex flex-col items-center justify-center gap-2 max-w-[370px] sm:max-w-[340px] mx-auto">
                <a
                  href="#installation"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-1"
                >
                  Jalankan sendiri melalui Docker
                </a>
                <a
                  href="https://github.com/databasus/databasus"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-2 sm:order-2"
                >
                  <svg
                    aria-hidden={true}
                    width="24"
                    height="24"
                    viewBox="0 0 20 20"
                    fill="none"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <g clipPath="url(#clip0_1_2459)">
                      <path
                        fillRule="evenodd"
                        clipRule="evenodd"
                        d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                        fill="white"
                      />
                    </g>
                    <defs>
                      <clipPath id="clip0_1_2459">
                        <rect width="20" height="20" fill="white" />
                      </clipPath>
                    </defs>
                  </svg>
                  <span>GitHub</span>
                </a>
              </div>
              <div className="mt-2 max-w-[370px] sm:max-w-[340px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
                <HeroStatsComponent />
              </div>
            </div>
          </div>
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Antarmuka dashboard Databasus"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
            </div>
          </div>
          <div className="mt-10 md:mt-15 mb-12 md:mb-20 flex justify-center px-4 md:px-0">
            <div className="flex flex-col md:flex-row items-center">
              <img
                className="h-[45px] md:h-[55px]"
                src="/images/index/ais.svg"
                alt="Dukungan program OSS Anthropic dan OpenAI"
              />
              <div className="flex justify-center text-base md:text-xl mt-4 md:mt-0 md:ml-10">
                <div className="max-w-[370px] text-gray-400 text-center md:text-left">
                  Didukung program open source Anthropic dan OpenAI.{" "}
                  <a
                    href={getLocalizedUrl("id", "faq") + "#oss-programs"}
                    target="_blank"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Selengkapnya&nbsp;→
                  </a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
      <section id="features" className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Gambaran umum</span>
            </div>
            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">Fitur</h2>
            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus menyediakan semua yang kamu butuhkan untuk mengelola backup production
              dengan andal, dari penjadwalan otomatis hingga enkripsi backup. Cocok untuk developer
              individu dengan proyek pribadi, tim DevOps, maupun perusahaan
            </p>
          </div>
        </div>
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 border border-[#ffffff20] rounded-xl">
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                1
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Backup terjadwal
              </h3>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Backup terjadwal"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Backup perlu dilakukan secara rutin pada waktu yang ditentukan. Karena itu, tersedia
                banyak pilihan: per jam, harian, mingguan, bulanan, cron, dan lainnya.
              </p>
            </div>
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Pemeriksaan ketersediaan yang bisa diatur
              </h3>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Pemeriksaan ketersediaan"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base mb-3">
                Setiap menit (atau interval lain), sistem melakukan ping ke database dan menampilkan
                riwayat percobaannya
              </p>
              <p className="text-gray-400 text-sm md:text-base">
                Database bisa dianggap tidak tersedia setelah 3 percobaan gagal atau jumlah lain
                yang ditentukan. Saat database kembali sehat, kamu juga menerima notifikasi
              </p>
            </div>
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Beragam tujuan penyimpanan
              </h3>
              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                File disimpan di VPS, penyimpanan cloud, dan tempat lainnya. Kamu bebas memilih
                penyimpanan. File selalu menjadi milik kamu.{" "}
                <a
                  href={getLocalizedUrl("id", "storages")}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Lihat semua →
                </a>
              </p>
              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="Penyimpanan"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                4
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">Notifikasi</h3>
              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Kamu bisa menerima notifikasi saat proses berhasil atau gagal. Berguna untuk
                developer maupun tim DevOps.{" "}
                <a
                  href={getLocalizedUrl("id", "notifiers")}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Lihat semua →
                </a>
              </p>
              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Notifikasi"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                5
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Self-hosted melalui Docker
              </h3>
              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus berjalan di PC atau VPS kamu. Jadi, semua data tetap menjadi milik kamu
                dan terlindungi. Deployment membutuhkan sekitar 2 menit melalui script, Docker, atau
                k8s
              </p>
              <div className="flex">
                <img src="/images/index/feature-deploy.svg" alt="Docker" loading="lazy" />
              </div>
            </div>
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                6
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Open source dan gratis
              </h3>
              <p className="text-gray-400 text-sm md:text-base mb-4">
                Proyek ini sepenuhnya open source, gratis, dan berlisensi Apache 2.0. Kamu bisa
                menyalin dan melakukan fork kode sendiri. Perusahaan juga bebas menggunakannya
              </p>
              <div>
                <img src="/images/index/feature-github.svg" alt="GitHub" loading="lazy" />
              </div>
            </div>
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Verifikasi restore
                </h3>
                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Backup yang selesai tanpa error belum tentu bisa di-restore. Databasus secara
                  berkala mengambil backup terbaru, melakukan restore ke container database
                  sementara, lalu melaporkan hasilnya.{" "}
                  <a
                    href={getLocalizedUrl("id", "restore-verification")}
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Selengkapnya →
                  </a>
                </p>
                <div>
                  <img src="/images/index/feature-postgresql.svg" alt="PostgreSQL" loading="lazy" />
                </div>
              </div>
              <div className="hidden lg:block p-5 md:p-6">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  10
                </div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">Keamanan</h3>
                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Enkripsi kelas perusahaan melindungi data sensitif dan backup. Akses database
                  read-only mencegah kerusakan data. Semuanya langsung aktif secara otomatis sejak
                  awal tanpa memerlukan pengetahuan khusus.{" "}
                  <a
                    href={getLocalizedUrl("id", "security")}
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Selengkapnya →
                  </a>
                </p>
                <div>
                  <img src="/images/index/feature-encryption.svg" alt="Keamanan" loading="lazy" />
                </div>
              </div>
            </div>
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  8
                </div>
              </div>
              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">Pengelolaan akses</h3>
                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  untuk tim
                </div>
              </div>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Pengelolaan akses"
                  className="w-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Berikan akses kepada pengguna untuk melihat atau mengelola database. Pisahkan tim
                dan proyek. Cocok untuk tim DevOps dan developer.{" "}
                <a
                  href={getLocalizedUrl("id", "access-management") + "#settings"}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Selengkapnya →
                </a>
              </p>
            </div>
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  9
                </div>
              </div>
              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">Log audit</h3>
                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  untuk tim
                </div>
              </div>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="Log audit"
                  className="w-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Lacak semua aktivitas sistem dengan log audit lengkap. Kamu bisa melihat riwayat
                akses dan perubahan setiap pengguna (unduhan backup, perubahan jadwal, pembaruan
                konfigurasi, dan lainnya).{" "}
                <a
                  href={getLocalizedUrl("id", "access-management") + "#audit-logs"}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Selengkapnya →
                </a>
              </p>
            </div>
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">Keamanan</h3>
              <p className="text-gray-400 text-sm md:text-base mb-4">
                Enkripsi kelas perusahaan melindungi data sensitif dan backup. Akses database
                read-only mencegah kerusakan data. Semuanya langsung aktif secara otomatis sejak
                awal tanpa memerlukan pengetahuan khusus.{" "}
                <a
                  href={getLocalizedUrl("id", "security")}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Selengkapnya →
                </a>
              </p>
              <div>
                <img src="/images/index/feature-encryption.svg" alt="Keamanan" loading="lazy" />
              </div>
            </div>
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>
              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Backup logis, fisik, incremental, dan WAL
                </h3>
                <p className="text-gray-400 text-sm md:text-base">
                  Databasus mendukung backup logis dan fisik (penuh serta incremental), dengan
                  WAL-streaming untuk Point-in-Time Recovery. Dengan begitu, Databasus cocok untuk
                  pemulihan bencana dan bisa digunakan pada database self-hosted maupun cloud.
                  Gunakan mode jarak jauh untuk database yang dikelola cloud atau bisa diakses
                  publik. Backup fisik menggunakan backup native PG 17.{" "}
                  <a
                    href={getLocalizedUrl("id", "faq") + "#pitr"}
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Selengkapnya →
                  </a>
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0" id="how-to-use">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            <div className="w-full lg:w-[450px] lg:shrink-0">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Gambaran umum 4 menit</span>
              </div>
              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Bagaimana cara menggunakan Databasus?
              </h2>
              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                Video ini menjelaskan cara menghubungkan database, mengatur jadwal backup, mengunduh
                dan melakukan restore backup, menambahkan anggota tim, serta memahami log audit
                pengguna
              </p>
              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                className="flex items-center gap-3 md:gap-4 hover:opacity-70 transition-colors"
              >
                <img
                  src="/images/index/rostislav.png"
                  alt="Rostislav Dugin"
                  className="w-10 h-10 md:w-12 md:h-12 rounded-full object-cover"
                  loading="lazy"
                />
                <div>
                  <p className="font-medium text-base md:text-lg">Rostislav Dugin</p>
                  <p className="text-sm text-gray-400">Developer Databasus</p>
                </div>
              </a>
            </div>
            <div className="flex-1 relative">
              <div className="rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
                <LiteYouTubeEmbed
                  videoId="1qsAnijJfJE"
                  title="Cara menggunakan Databasus (gambaran umum)"
                  playLabel={VIDEO_PLAY_LABELS.id.replace(
                    "{title}",
                    "Cara menggunakan Databasus (gambaran umum)",
                  )}
                  thumbnailSrc="/images/index/how-to-use-preview.svg"
                />
              </div>
            </div>
          </div>
        </div>
      </section>
      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />
      <section className="pt-12 md:pt-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Database</span>
            </div>
            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Database yang didukung
            </h2>
            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Databasus mendukung PostgreSQL, MySQL, MariaDB, dan MongoDB. Kamu bisa melakukan
              backup dan restore semuanya dengan alat yang sama. Fokus utamanya PostgreSQL, tetapi
              MySQL, MariaDB, dan MongoDB juga didukung
            </p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 border border-[#ffffff20] rounded-xl">
            <div className="border-b md:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-postgresql.svg"
                    alt="PostgreSQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                PostgreSQL
              </h3>
              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                PostgreSQL adalah database utama yang didukung Databasus. Semua versi dari 12 hingga
                18 didukung
              </p>
            </div>
            <div className="border-b lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="MySQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MySQL
              </h3>
              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MySQL adalah database terpopuler kedua di dunia. Kamu bisa melakukan backup dan
                restore database MySQL dengan cara yang sama mudahnya
              </p>
              <div className="text-center mt-auto">
                <a
                  href={getLocalizedUrl("id", "mysql-backup")}
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Selengkapnya →
                </a>
              </div>
            </div>
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mariadb.svg"
                    alt="MariaDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MariaDB
              </h3>
              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MariaDB didukung dengan fitur yang sama seperti MySQL. Kamu bisa melakukan backup
                dan restore database MariaDB dengan mudah
              </p>
              <div className="text-center mt-auto">
                <a
                  href={getLocalizedUrl("id", "mysql-backup")}
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Selengkapnya →
                </a>
              </div>
            </div>
            <div className="p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="MongoDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MongoDB
              </h3>
              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MongoDB adalah database NoSQL terpopuler. Kamu bisa melakukan backup dan restore
                database MongoDB dengan antarmuka yang sama mudahnya
              </p>
              <div className="text-center mt-auto">
                <a
                  href={getLocalizedUrl("id", "mongodb-backup")}
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Selengkapnya →
                </a>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Proses</span>
            </div>
            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Bagaimana cara membuat backup?
            </h2>
            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Prioritas utama Databasus adalah kesederhanaan. Saat ini, Databasus merupakan alat
              backup PostgreSQL paling mudah di dunia. Untuk membuat backup, ikuti 4 langkah
              berikut. Setelah itu, restore bisa dilakukan dengan satu klik
            </p>
          </div>
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Langkah 1
              </span>
              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Pilih jadwal yang dibutuhkan
                </h3>
                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Kamu bisa memilih waktu sesuai kebutuhan: harian, mingguan, bulanan, waktu
                    tertentu (misalnya pukul 4 pagi), dan siklus cron
                  </p>
                  <p>
                    Untuk interval mingguan, tentukan harinya. Untuk interval bulanan, tentukan
                    tanggalnya
                  </p>
                  <p>Jika database besar, sebaiknya pilih waktu ketika traffic sedang rendah</p>
                </div>
              </div>
              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Langkah 1"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Langkah 2
              </span>
              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Masukkan informasi database
                </h3>
                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Masukkan kredensial database PostgreSQL, pilih versi dan database tujuan.
                    Tentukan juga apakah SSL diperlukan
                  </p>
                  <p>
                    Secara default, Databasus mengompresi backup pada tingkat seimbang agar proses
                    tidak terlalu lambat (~20% lebih lambat) dan menghemat ruang x4-x8 (sekaligus
                    mengurangi traffic jaringan)
                  </p>
                </div>
              </div>
              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-2.svg"
                  alt="Langkah 2"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Langkah 3
              </span>
              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Pilih penyimpanan backup
                </h3>
                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Kamu bisa menyimpan file backup secara lokal, di S3, Google Drive, NAS, Dropbox,
                    dan layanan lain
                  </p>
                  <p>Pastikan ruang penyimpanan yang tersedia mencukupi</p>
                </div>
              </div>
              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-3.svg"
                  alt="Langkah 3"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Langkah 4
              </span>
              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Pilih tempat menerima notifikasi (opsional)
                </h3>
                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Saat backup berhasil atau gagal, Databasus bisa mengirim notifikasi. Tujuannya
                    bisa berupa chat DevOps, email kamu, atau bahkan webhook tim
                  </p>
                  <p>Kami berencana mendukung sebagian besar aplikasi pesan dan platform populer</p>
                </div>
              </div>
              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-4.svg"
                  alt="Langkah 4"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
          </div>
          <div className="text-center mt-8 md:mt-12">
            <a
              href="#installation"
              className="inline-flex items-center gap-2 px-6 py-3 bg-white text-black rounded-lg text-[15px] font-medium hover:opacity-70 transition-colors"
            >
              Mulai memasang
              <svg
                aria-hidden={true}
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
          </div>
        </div>
      </section>
      <section id="installation" className="px-4 md:px-6 lg:px-0">
        <div className="max-w-[1000px] 2xl:max-w-[1200px] mx-auto border border-[#ffffff20] rounded-xl py-10 md:py-20 px-4 md:px-6">
          <div className="max-w-[1100px] mx-auto">
            <div className="text-center mb-8 md:mb-10">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Mulai menggunakan</span>
              </div>
              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                Bagaimana cara memasangnya?
              </h2>
              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus mendukung berbagai metode instalasi. Semuanya sederhana dan mudah
                digunakan, bahkan tanpa pengalaman administrasi sistem atau DevOps. Database lokal
                maupun cloud didukung.
              </p>
            </div>
            <InstallationComponent lang="id" />
          </div>
        </div>
      </section>
      <section id="faq" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-8 md:mb-12">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">FAQ</span>
            </div>
            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Pertanyaan umum
            </h2>
            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Databasus bertujuan membuat backup semudah mungkin bagi developer individu, DevOps,
              dan tim. Antarmukanya memudahkan pembuatan backup, menampilkan perkembangan proses,
              dan memungkinkan restore hanya dengan beberapa klik
            </p>
          </div>
        </div>
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Apa itu Databasus dan mengapa perlu menggunakannya alih-alih script buatan sendiri?"
              answer="Databasus adalah layanan self-hosted berlisensi Apache 2.0 untuk backup database. Berbeda dari shell script, Databasus menyediakan antarmuka untuk menjadwalkan tugas, mengompresi dan menyimpan arsip ke berbagai tujuan (disk lokal, S3, Google Drive, NAS, Dropbox, SFTP, rclone, dan lainnya), mengatur kebijakan retensi agar backup lama dihapus otomatis, serta memberi tahu tim saat tugas selesai atau gagal, tanpa menulis kode sendiri"
            />
            <FaqItem
              number="2"
              question="Bagaimana cara tercepat memasang Databasus?"
              answer="Databasus mendukung beberapa metode instalasi: script otomatis, Docker, Docker Compose, dan Kubernetes dengan Helm. Cara tercepat adalah menjalankan installer cURL satu baris, yang mengambil image Docker terbaru, membuat docker-compose.yml, dan menjalankan layanan agar otomatis aktif kembali setelah reboot. Untuk lingkungan Kubernetes, gunakan Helm chart resmi untuk deployment siap production. Seluruh proses biasanya kurang dari dua menit pada VPS umum."
            />
            <FaqItem
              number="3"
              question="Bagaimana cara kerja verifikasi restore?"
              answer="Databasus menjalankan agen verifikasi kecil di host yang kamu kendalikan. Pada setiap jadwal, agen mengunduh backup terbaru dan melakukan restore ke container database sementara. Setelah itu, agen memeriksa kewajaran hasil restore dengan membandingkannya terhadap sumber. Hasilnya dilaporkan kembali, termasuk exit code restore dan jumlah baris per tabel. Jadwal bisa dijalankan setelah backup, per jam, harian, mingguan, bulanan, atau menggunakan ekspresi cron UTC. Kegagalan bisa dilaporkan melalui kanal notifikasi mana pun yang terhubung ke database, seperti Slack, Teams, Discord, email, dan lainnya."
            />
            <FaqItem
              number="4"
              question="Bagaimana Databasus menjaga keamanan?"
              answer={
                <>
                  Databasus menerapkan keamanan pada tiga tingkat: (1) Enkripsi data sensitif: semua
                  password, token, dan kredensial dienkripsi dengan AES-256-GCM dan disimpan
                  terpisah dari database; (2) Enkripsi backup: setiap file backup dienkripsi dengan
                  key unik yang diturunkan dari master key, ID backup, dan salt acak, sehingga
                  backup tidak berguna tanpa key enkripsi kamu meskipun seseorang mendapat akses
                  penyimpanan; (3) Akses database read-only: Databasus hanya memerlukan izin SELECT
                  dan melakukan pemeriksaan menyeluruh untuk memastikan tidak ada hak tulis,
                  sehingga mencegah kerusakan data meskipun alat ini disusupi.
                  <br />
                  <br />
                  Login juga bisa mewajibkan faktor kedua: jika diaktifkan, password yang benar
                  diikuti kode enam digit yang dikirim ke email akun, sedangkan login melalui Google
                  atau GitHub tetap mengandalkan pemeriksaan penyedia tersebut.
                  <br />
                  <br />
                  Di luar runtime, keamanan dan keandalan diperiksa pada setiap commit dan PR
                  melalui analisis statis CodeQL, pemindaian secret gitleaks, aturan keamanan
                  semgrep, pemantauan CVE Dependabot, pemindaian image dan Dockerfile Trivy, serta
                  audit berkala Codex Security dari OpenAI. Integration test berjalan pada container
                  PostgreSQL, MySQL, MariaDB, dan MongoDB sungguhan serta memverifikasi siklus
                  backup lalu restore lengkap pada setiap PR. GitHub Actions dipatok ke SHA commit
                  dan workflow mengikuti prinsip hak akses minimum.
                  <br />
                  <br />
                  Lihat{" "}
                  <a
                    href={
                      getLocalizedUrl("id", "security") + "#security-and-reliability-engineering"
                    }
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Rekayasa keamanan dan keandalan
                  </a>{" "}
                  untuk pipeline lengkapnya.
                  <br />
                  <br />
                  Selain itu, semua log bisa diekspor ke sistem eksternal mana pun (VictoriaLogs,
                  SigNoz, Graylog, dan lainnya) melalui standar OpenTelemetry. Secara default, log
                  (termasuk log audit) juga ditulis ke file lokal agar log audit tidak hilang. Lihat{" "}
                  <a
                    href={getLocalizedUrl("id", "advanced-config") + "#logging"}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    konfigurasi lanjutan
                  </a>{" "}
                  di sini.
                </>
              }
            />
            <FaqItem
              number="5"
              question="Bagaimana cara menyiapkan dan menjalankan tugas backup pertama di Databasus?"
              answer={
                <>
                  Untuk memulai backup Databasus pertama, login ke dashboard, klik New Backup, lalu
                  pilih interval: per jam, harian, mingguan, bulanan, atau cron. Tentukan waktu
                  eksekusi yang tepat (misalnya 02:30 saat traffic rendah).
                  <br />
                  <br />
                  Lalu masukkan host PostgreSQL, nomor port, nama database, kredensial, dan pilihan
                  SSL. Pilih tujuan arsip (path lokal, bucket S3, folder Google Drive, Dropbox, dan
                  lainnya). <br />
                  <br />
                  Jika perlu, tambahkan kanal notifikasi seperti email, Slack, Telegram, atau
                  webhook, lalu klik Save. Databasus langsung memvalidasi informasi, mengaktifkan
                  jadwal, menjalankan tugas awal, dan mengirim status secara langsung. Setelah
                  backup selesai, restore bisa dilakukan dengan sekali tekan.
                </>
              }
            />
            <FaqItem
              number="6"
              question="Seberapa luas Databasus digunakan?"
              answer="Databasus saat ini merupakan alat backup PostgreSQL open source yang paling banyak digunakan. Per 17 Juni 2026, image Docker-nya telah ditarik lebih dari 1.800.000 kali oleh DBA, engineer DevOps, developer, dan tim di seluruh dunia. Dengan 8.500+ GitHub stars, Databasus melampaui pgBackRest (~4.200 stars, tersedia sejak 2014) dan WAL-G (~4.100 stars, tersedia sejak 2017). Databasus diluncurkan pada 2025 dan melampaui keduanya dalam tahun pertamanya. Tingkat adopsi ini mencerminkan kepercayaan dan pilihan komunitas profesional database."
            />
            <FaqItem
              number="7"
              question="Apa perbedaan Databasus dengan PgBackRest, Barman, atau pg_dump? Di mana saya bisa membaca perbandingannya?"
              answer={
                <>
                  Databasus mengutamakan kesederhanaan. Antarmuka web modern memudahkan pengelolaan
                  backup banyak database sekaligus, menggantikan file konfigurasi rumit dan alat
                  command-line. Berbeda dari script pg_dump biasa, Databasus menyediakan
                  penjadwalan, kompresi, beragam tujuan penyimpanan, pemantauan kesehatan, dan
                  notifikasi real-time bawaan yang semuanya dikelola melalui antarmuka web
                  sederhana.
                  <br />
                  <br />
                  Berbeda dari pgBackRest dan WAL-G, Databasus membuat backup fisik, incremental,
                  dan WAL menggunakan mekanisme native PostgreSQL 17, bukan membuat mesin backup
                  sendiri. Databasus terhubung ke database dari jarak jauh dan menjangkau jaringan
                  tertutup melalui SSH tunnel ke server atau bastion, sehingga database yang tidak
                  terbuka ke publik tetap bisa di-backup dan dikelola dari satu dashboard.{" "}
                  <a
                    href={getLocalizedUrl("id", "faq") + "#pitr"}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Baca penerapan backup fisik dan PITR
                  </a>
                  .
                  <br />
                  <br />
                  Kami menyediakan halaman perbandingan mendetail untuk alat backup populer:{" "}
                  <a
                    href={getLocalizedUrl("id", "pgdump-alternative")}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pg_dump
                  </a>
                  ,{" "}
                  <a
                    href={getLocalizedUrl("id", "databasus-vs-pgbackrest")}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackRest
                  </a>
                  ,{" "}
                  <a
                    href={getLocalizedUrl("id", "databasus-vs-barman")}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs Barman
                  </a>
                  ,{" "}
                  <a
                    href={getLocalizedUrl("id", "databasus-vs-wal-g")}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs WAL-G
                  </a>{" "}
                  dan{" "}
                  <a
                    href={getLocalizedUrl("id", "databasus-vs-pgbackweb")}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackWeb
                  </a>
                  . Setiap perbandingan menjelaskan perbedaan utama, kelebihan, serta kekurangan
                  untuk membantu memilih alat yang sesuai kebutuhan.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Apakah Databasus didukung program OSS Anthropic dan OpenAI?"
              answer={
                <>
                  Ya, kami bangga Databasus diakui sebagai proyek open source bernilai oleh dua
                  perusahaan AI terkemuka dunia. Pada Maret 2026, Databasus diterima dalam{" "}
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Claude for Open Source
                  </a>{" "}
                  dari Anthropic dan{" "}
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Codex for Open Source
                  </a>{" "}
                  dari OpenAI. Bagi kami, ini merupakan pengakuan independen atas keandalan proyek
                  yang telah dievaluasi dan diakui sebagai infrastruktur penting yang layak
                  didukung.{" "}
                  <a
                    href={getLocalizedUrl("id", "faq") + "#oss-programs"}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Selengkapnya →
                  </a>
                </>
              }
            />
            <FaqItem
              number="9"
              question="Apakah Databasus merupakan alternatif pg_dump?"
              answer="Tidak sepenuhnya. Databasus berfokus pada pemulihan bencana dengan RTO dan RPO rendah, sehingga lebih dekat dengan alternatif pgBackRest atau WAL-G. Databasus berusaha membuat pemulihan bencana semudah pg_dump. Namun, untuk backup logis, Databasus memang menjadi alternatif pg_dump dan menggunakannya di balik layar, dengan tambahan antarmuka web yang mudah digunakan, penjadwalan otomatis, beragam tujuan penyimpanan, notifikasi real-time, pemantauan kesehatan, dan enkripsi backup. Backup logis juga tersedia untuk MySQL, MariaDB, dan MongoDB."
            />
            <FaqItem
              number="10"
              question="Database apa saja yang didukung Databasus?"
              answer={
                <>
                  Databasus mendukung PostgreSQL, MySQL, MariaDB, dan MongoDB. Namun, Databasus
                  awalnya dibuat khusus untuk PostgreSQL dan tetap berfokus padanya, dengan dukungan
                  100% dan efisiensi maksimal untuk backup PostgreSQL.
                  <br />
                  <br />
                  Meskipun MySQL, MariaDB, dan MongoDB juga didukung, PostgreSQL tetap menjadi
                  prioritas utama dengan fitur paling optimal dan pengembangan berkelanjutan.
                  <br />
                  <br />
                  Misalnya, Databasus menyediakan dukungan native untuk backup fisik dan WAL bagi
                  pemulihan bencana PostgreSQL. Jadi, Databasus pada dasarnya adalah alat backup
                  PostgreSQL; database lain merupakan perluasan dukungannya.
                </>
              }
            />
            <FaqItem
              number="11"
              question="Jenis backup apa saja yang didukung Databasus?"
              answer={
                <>
                  Databasus mendukung backup fisik, penuh, incremental, WAL, dan logis. Jadi,
                  Databasus cocok bagi yang menginginkan dump logis sederhana maupun alat pemulihan
                  bencana yang andal.
                  <ul className="list-disc list-inside mt-3 space-y-2">
                    <li>
                      <strong>Fisik</strong>: salinan seluruh cluster database pada tingkat file.
                      Lebih cepat untuk backup dan restore dataset besar dibanding dump logis.
                      Dibangun di atas mekanisme backup native PostgreSQL 17, sehingga kami
                      mengandalkan alat PostgreSQL yang telah teruji, bukan membuatnya ulang
                    </li>
                    <li>
                      <strong>Penuh</strong>: salinan cluster lengkap yang mandiri dan menjadi dasar
                      setiap rantai backup
                    </li>
                    <li>
                      <strong>Incremental</strong>: hanya menyimpan perubahan sejak backup
                      sebelumnya, sehingga ukurannya kecil dan prosesnya cepat
                    </li>
                    <li>
                      <strong>WAL streaming</strong>: menangkap aliran penulisan database secara
                      terus-menerus untuk Point-in-time recovery (PITR). Dirancang untuk pemulihan
                      bencana dan kehilangan data mendekati nol
                    </li>
                    <li>
                      <strong>Logis</strong>: dump native database dalam format biner khusus
                      engine-nya. Dikompresi dan dialirkan langsung ke penyimpanan tanpa file
                      perantara
                    </li>
                  </ul>
                  <br />
                  Backup fisik, incremental, dan WAL menggunakan mekanisme native PostgreSQL 17,
                  sehingga membutuhkan PostgreSQL 17 atau lebih baru. Pada versi lama, hanya backup
                  logis yang tersedia. Ini memang disengaja: sebagian besar database production
                  sudah menggunakan PostgreSQL 17 atau lebih baru, dan dalam sekitar dua tahun versi
                  yang lebih lama mencapai akhir masa dukungan. Databasus bertujuan menjadi alat
                  backup standar untuk database mulai PostgreSQL 17.
                  <br />
                  <br />
                  Semua jenis backup ini bisa berjalan melalui SSH tunnel jika koneksi harus tetap
                  privat, sehingga database tidak perlu dibuka ke publik. SSH tunneling sudah
                  tersedia secara bawaan.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Bagaimana cara bergabung dengan komunitas Databasus?"
              answer={
                <>
                  Kamu bisa bergabung dengan komunitas besar developer, DBA, dan engineer DevOps
                  kami di{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    t.me/databasus_community
                  </a>
                  . Komunitas adalah tempat untuk bertanya, berbagi pengalaman, mendapatkan bantuan
                  konfigurasi, dan mengikuti fitur serta rilis terbaru.
                </>
              }
            />
            <FaqItem
              number="13"
              question="Seberapa besar adopsi Databasus?"
              answer={
                <>
                  Databasus memiliki lebih dari 1,8 juta Docker pulls dan 8,5k GitHub stars. Sebagai
                  perbandingan, pgBackRest dan WAL-G masing-masing memiliki sekitar 4,2k stars,
                  sedangkan Barman sekitar 3,1k. Ini menjadikan Databasus alat backup database
                  terpopuler di GitHub.
                  <br />
                  <br />
                  Databasus diterima dalam program open source Anthropic dan OpenAI sebagai proyek
                  yang penting dan kritis. Kini Databasus digunakan oleh perusahaan, tim, dan
                  engineer DevOps, dengan dukungan komunitas yang besar dan aktif.
                  <br />
                  <br />
                  Databasus telah dikembangkan dan digunakan sejak 2023, serta digunakan luas
                  sebagai open source sejak awal 2025. Databasus sudah cukup lama digunakan dalam
                  production sungguhan, sehingga teruji dalam berbagai kasus khusus. Yang
                  terpenting, Databasus tidak menciptakan cara sendiri untuk melakukan backup data.
                  Databasus mengandalkan implementasi native PostgreSQL yang telah teruji, bukan
                  membuat penanganan khusus sendiri untuk setiap kasus.
                  <br />
                  <br />
                  Tujuan kami adalah menjadi alat backup standar untuk PostgreSQL mulai versi 17.
                  Databasus adalah alat backup pertama yang dibangun di atas protokol backup native
                  PostgreSQL yang efisien dan kini menjadi standar, bukan menulis implementasinya
                  sendiri.
                </>
              }
            />
            <FaqItem
              number="14"
              question="Saya lupa email atau password admin"
              answer={
                <>
                  Akun pertama yang dibuat pada instance bertugas mengelolanya. Jika lupa alamatnya,
                  jalankan docker exec -it databasus ./main --list-admins pada server tempat
                  Databasus berjalan. Perintah ini menampilkan setiap akun administrator beserta
                  alamat email, nama tampilan, tanggal pembuatan, dan status aktifnya, tanpa
                  menampilkan informasi password. Untuk mengatur password baru akun tersebut,
                  jalankan docker exec -it databasus ./main
                  --new-password=&quot;YourNewSecurePassword123&quot;
                  --email=&quot;owner@example.com&quot;. Instance yang dibuat sebelum akun pertama
                  bertugas mengelolanya masih menggunakan alamat placeholder admin sampai pemilik
                  menggantinya. Jika instance juga mewajibkan kode email saat login dan mail server
                  tidak lagi mengirimkannya, jalankan docker exec -it databasus ./main --disable-2fa
                  untuk menonaktifkan kewajiban kode. Ketiga perintah didokumentasikan di{" "}
                  <a
                    href={getLocalizedUrl("id", "password")}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    halaman password
                  </a>
                  .
                </>
              }
            />
          </div>
        </div>
      </section>
      <LandingFooterComponent lang="id" />
    </div>
  );
}

function FaqItem({
  number,
  question,
  answer,
}: {
  number: string;
  question: string;
  answer: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-[#ffffff20] p-4 md:p-6">
      <div className="flex items-center justify-center w-6 h-6 rounded border border-[#ffffff20] text-sm font-semibold mb-3 md:mb-4">
        {number}
      </div>
      <h3 className="text-base md:text-lg font-bold mb-2 md:mb-3">{question}</h3>
      <div className="text-gray-400 text-sm md:text-base">{answer}</div>
    </div>
  );
}
