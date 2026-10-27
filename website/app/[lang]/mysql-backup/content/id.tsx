import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Backup MySQL dengan dukungan MariaDB",
  description:
    "Tool gratis dan open source untuk backup MySQL dan MariaDB terjadwal. Alternatif mysqldump dengan antarmuka web, penyimpanan cloud (S3, Google Drive), notifikasi (Slack, Discord, Telegram), dan enkripsi AES-256.",
  keywords:
    "backup MySQL, backup MariaDB, alternatif mysqldump, GUI mysqldump, otomatisasi backup MySQL, tool backup MySQL, tool backup MariaDB, backup MySQL terjadwal, backup MySQL cloud, backup MySQL S3, backup MySQL Docker, enkripsi backup MySQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("id", "mysql-backup"),
    languages: getLanguageAlternates("mysql-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("id", "mysql-backup"),
    title: "Backup MySQL dengan dukungan MariaDB",
    description:
      "Tool gratis dan open source untuk backup MySQL dan MariaDB terjadwal. Alternatif mysqldump dengan antarmuka web, penyimpanan cloud, notifikasi, dan enkripsi.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Antarmuka dashboard Databasus yang menampilkan pengelolaan backup MySQL",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary_large_image",
    title: "Backup MySQL dengan dukungan MariaDB",
    description:
      "Tool gratis dan open source untuk backup MySQL dan MariaDB terjadwal. Alternatif mysqldump dengan antarmuka web, penyimpanan cloud, notifikasi, dan enkripsi.",
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

export default function MysqlBackupPage() {
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
              "Tool gratis dan open source untuk backup MySQL dan MariaDB terjadwal. Alternatif mysqldump dengan antarmuka web, penyimpanan cloud, notifikasi, dan enkripsi.",
            url: getLocalizedUrl("id", "mysql-backup"),
            image: "https://databasus.com/images/index/dashboard.png",
            logo: "https://databasus.com/logo.svg",
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
            featureList: [
              "Backup MySQL dan MariaDB terjadwal",
              "Berbagai tujuan penyimpanan (S3, Google Drive, Dropbox, SFTP, rclone, dan lainnya)",
              "Notifikasi real-time (Slack, Telegram, Discord, Webhook, email, dan lainnya)",
              "Pemantauan ketersediaan database MySQL",
              "Self-hosted melalui Docker",
              "Open source dan gratis",
              "Dukungan MySQL 5.7, 8.0, 8.4, 9, 26",
              "Dukungan MariaDB 5.5, 10, 11, 12, 13",
              "Kompresi backup dan enkripsi AES-256-GCM",
              "Alternatif mysqldump dengan antarmuka web",
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
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Apa itu Databasus dan bagaimana cara melakukan backup database MySQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus adalah tool backup self-hosted berlisensi Apache 2.0 yang menggunakan mysqldump untuk membuat backup MySQL yang konsisten. Databasus melengkapi mysqldump dengan antarmuka web modern, penjadwalan otomatis, integrasi penyimpanan cloud, notifikasi real-time, dan enkripsi AES-256-GCM, sehingga tidak perlu menulis skrip shell sendiri.",
                },
              },
              {
                "@type": "Question",
                name: "Apakah Databasus alternatif atau pengganti mysqldump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus melengkapi mysqldump, bukan menggantikannya. Databasus menjalankan mysqldump dengan parameter yang dioptimalkan (--single-transaction, --routines, --triggers, --events), lalu menangani kompresi, enkripsi, dan upload ke penyimpanan yang dikonfigurasi. Keandalan mysqldump tetap tersedia, ditambah penjadwalan, penyimpanan cloud, notifikasi, dan fitur tim.",
                },
              },
              {
                "@type": "Question",
                name: "Versi MySQL apa saja yang didukung Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus mendukung MySQL 5.7, 8.0, 8.4, 9 dan 26. Untuk MySQL 8.0 dan yang lebih baru, Databasus menggunakan kompresi jaringan zstd agar transfer lebih cepat. Untuk MySQL 5.7, kompresi legacy digunakan secara otomatis. MySQL 5.7 hanya didukung pada x86-64.",
                },
              },
              {
                "@type": "Question",
                name: "Versi MariaDB apa saja yang didukung Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus mendukung MariaDB 5.5, 10, 11, 12 dan 13. Untuk backup MariaDB, Databasus menggunakan tool native mariadb-dump, bukan mysqldump, agar sepenuhnya kompatibel dengan fitur dan optimasi khusus MariaDB.",
                },
              },
              {
                "@type": "Question",
                name: "Bisakah melakukan backup database MySQL di AWS RDS, Google Cloud SQL atau Azure?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Ya, Databasus bekerja dengan database MySQL di cloud, termasuk AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL dan PlanetScale. Karena menggunakan backup logical melalui mysqldump, Databasus hanya memerlukan kredensial koneksi MySQL standar, tanpa izin cloud khusus.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana Databasus mengamankan kredensial dan backup database MySQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus menerapkan keamanan berlapis: (1) Semua password dan kredensial MySQL dienkripsi dengan AES-256-GCM sebelum disimpan; (2) Setiap file backup dienkripsi dengan kunci unik yang diturunkan dari master key, ID backup dan salt acak; (3) Kredensial diteruskan melalui file .my.cnf sementara dengan izin 0600, tanpa ditampilkan di command line atau log.",
                },
              },
            ],
          }),
        }}
      />
      <LandingHeaderComponent
        lang="id"
        sectionLinks={[
          { href: "#features", label: "Fitur" },
          { href: "#mysql-backup", label: "MySQL" },
          { href: "#mariadb-backup", label: "MariaDB" },
        ]}
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
            <h1 className="text-2xl sm:text-4xl sm:max-w-[400px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[500px]">
              Tool backup MySQL dengan dukungan MariaDB
            </h1>
            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus adalah tool gratis, open source, dan self-hosted untuk backup database MySQL
              dan MariaDB. Jadwalkan dump otomatis, simpan di S3, Google Drive, atau secara lokal.
              Terima notifikasi melalui Slack, Discord, atau Telegram saat backup selesai
            </p>
            <div className="flex flex-col sm:flex-row sm:flex-wrap items-center justify-center gap-2 sm:gap-2 max-w-[400px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
              <a
                href="#installation"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-3"
              >
                Self-host melalui Docker
              </a>
              <a
                href="https://github.com/databasus/databasus"
                target="_blank"
                rel="noopener noreferrer"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-4 sm:order-4"
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
          </div>
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Antarmuka dashboard Databasus untuk mengelola backup MySQL"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
            </div>
          </div>
        </div>
      </main>
      <section id="features" className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Ringkasan</span>
            </div>
            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Fitur backup MySQL
            </h2>
            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus melengkapi mysqldump dengan fitur untuk perusahaan: penjadwalan otomatis,
              integrasi penyimpanan cloud, notifikasi real-time, dan enkripsi AES-256-GCM. Cocok
              untuk developer, tim DevOps, dan organisasi yang mengelola database MySQL
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
                Dump MySQL terjadwal
              </h3>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Konfigurasi backup MySQL terjadwal"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Jadwalkan mysqldump pada waktu terbaik saat beban database rendah. Pilih interval
                per jam, harian, mingguan, bulanan, atau gunakan ekspresi cron untuk pengaturan yang
                lebih presisi
              </p>
            </div>
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Pemantauan ketersediaan MySQL
              </h3>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Pemeriksaan ketersediaan MySQL"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base mb-3">
                Pantau ketersediaan koneksi MySQL dengan pemeriksaan ketersediaan yang bisa diatur.
                Terima notifikasi saat database tidak bisa dijangkau atau kembali tersedia
              </p>
              <p className="text-gray-400 text-sm md:text-base">
                Atur interval pemeriksaan (setiap menit, 5 menit, dan lainnya) serta ambang
                kegagalan sebelum database ditandai tidak tersedia
              </p>
            </div>
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Simpan dump MySQL di mana saja
              </h3>
              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Simpan file backup MySQL secara lokal, di penyimpanan kompatibel S3, Google Drive,
                Dropbox, NAS, atau tujuan lain. Data tetap dalam kendalimu.{" "}
                <a
                  href={getLocalizedHref("id", "storages")}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Lihat semua →
                </a>
              </p>
              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="Tujuan penyimpanan backup MySQL"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                4
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Notifikasi backup
              </h3>
              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Terima pemberitahuan saat backup MySQL selesai atau gagal. Kirim notifikasi ke chat
                tim DevOps atau kanal pribadi.{" "}
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Lihat semua →
                </a>
              </p>
              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Notifikasi backup MySQL"
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
                Jalankan Databasus di infrastruktur sendiri. Semua kredensial MySQL dan data backup
                tetap berada di server yang kamu kendalikan. Deploy dalam sekitar 2 menit melalui
                skrip, Docker, atau Kubernetes
              </p>
              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Deployment Docker"
                  loading="lazy"
                />
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
                Databasus sepenuhnya open source dengan lisensi Apache 2.0. Periksa setiap baris
                kode, buat fork, dan ikut berkontribusi. Gratis untuk penggunaan pribadi maupun
                perusahaan
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="Open source di GitHub"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Versi MySQL dan MariaDB
                </h3>
                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Mendukung MySQL 5.7, 8.0, 8.4, 9 dan 26. MariaDB 5.5, 10, 11, 12 dan 13 didukung
                  menggunakan tool native mariadb-dump agar sepenuhnya kompatibel
                </p>
                <div>
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="Versi MySQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>
              <div className="hidden lg:block p-5 md:p-6">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  10
                </div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">Keamanan</h3>
                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Kredensial MySQL dienkripsi dengan AES-256-GCM sebelum disimpan. Setiap file
                  backup dienkripsi dengan kunci unik. Password diteruskan melalui file konfigurasi
                  sementara, tanpa ditampilkan di command line.{" "}
                  <a
                    href={getLocalizedHref("id", "security")}
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Baca selengkapnya →
                  </a>
                </p>
                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="Keamanan backup MySQL"
                    loading="lazy"
                  />
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
                  alt="Pengelolaan akses backup MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Atur siapa yang bisa melihat atau mengelola database MySQL. Buat ruang kerja untuk
                proyek yang berbeda. Tetapkan role viewer, editor, atau admin.{" "}
                <a
                  href={`${getLocalizedHref("id", "access-management")}#settings`}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Baca selengkapnya →
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
                  alt="Log audit backup MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Lacak semua aktivitas: unduhan backup, perubahan jadwal, dan pembaruan konfigurasi.
                Lihat siapa melakukan apa dan kapan untuk kepatuhan dan akuntabilitas.{" "}
                <a
                  href={`${getLocalizedHref("id", "access-management")}#audit-logs`}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Baca selengkapnya →
                </a>
              </p>
            </div>
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">Keamanan</h3>
              <p className="text-gray-400 text-sm md:text-base mb-4">
                Kredensial MySQL dienkripsi dengan AES-256-GCM sebelum disimpan. Setiap file backup
                dienkripsi dengan kunci unik. Password diteruskan melalui file konfigurasi
                sementara, tanpa ditampilkan di command line.{" "}
                <a
                  href={getLocalizedHref("id", "security")}
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Baca selengkapnya →
                </a>
              </p>
              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="Keamanan backup MySQL"
                  loading="lazy"
                />
              </div>
            </div>
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>
              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Bekerja dengan database MySQL di cloud
                </h3>
                <p className="text-gray-400 text-sm md:text-base">
                  Databasus terhubung ke database MySQL di cloud, termasuk AWS RDS for MySQL, Google
                  Cloud SQL, Azure Database for MySQL dan PlanetScale. Karena menggunakan backup
                  logical melalui mysqldump, cukup sediakan kredensial koneksi standar, tanpa izin
                  cloud khusus atau akses filesystem
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section id="mysql-backup" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Menggunakan mysqldump</span>
              </div>
              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Backup MySQL
              </h2>
              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus menggunakan <strong>mysqldump</strong>, utilitas backup resmi MySQL.
                  Saat backup dipicu, Databasus menjalankan mysqldump dengan parameter yang
                  dioptimalkan:
                </p>
                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --single-transaction
                    </code>{" "}
                    untuk snapshot konsisten tanpa mengunci tabel
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">--routines</code>{" "}
                    untuk menyertakan stored procedure dan fungsi
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">--triggers</code>{" "}
                    untuk menyertakan trigger database
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">--events</code>{" "}
                    untuk menyertakan event terjadwal
                  </li>
                </ul>
                <p className="text-gray-400">
                  Untuk MySQL 8.0 dan yang lebih baru, Databasus mengaktifkan{" "}
                  <strong>kompresi jaringan zstd</strong> agar transfer melalui koneksi lambat lebih
                  cepat. MySQL 5.7 otomatis menggunakan kompresi legacy.
                </p>
                <div className="pt-2">
                  <p className="text-white font-medium mb-2">Versi MySQL yang didukung:</p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 5.7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.0
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 9
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 26
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mysql.svg"
                  alt="Database MySQL"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Backup resmi MySQL melalui mysqldump dengan kompresi, enkripsi, dan penyimpanan
                  cloud
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />
      <section id="mariadb-backup" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row-reverse gap-8 lg:gap-16">
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Tool native mariadb-dump</span>
              </div>
              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Backup MariaDB
              </h2>
              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Untuk database MariaDB, Databasus menggunakan tool native{" "}
                  <strong>mariadb-dump</strong>, bukan mysqldump. Ini menjaga kompatibilitas penuh
                  dengan fitur dan optimasi khusus MariaDB.
                </p>
                <p className="text-gray-400">
                  mariadb-dump dijalankan dengan parameter optimal yang sama: mode
                  single-transaction untuk backup konsisten, dengan routine, trigger, dan event
                  disertakan secara default.
                </p>
                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>Kompatibilitas penuh dengan fitur MariaDB</li>
                  <li>Dukungan koneksi SSL/TLS</li>
                  <li>Kompresi jaringan untuk transfer lebih cepat</li>
                  <li>Pilihan enkripsi dan penyimpanan yang sama dengan MySQL</li>
                </ul>
                <div className="pt-2">
                  <p className="text-white font-medium mb-2">Versi MariaDB yang didukung:</p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 5.5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 10
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 11
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 12
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 13
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mariadb.svg"
                  alt="Database MariaDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Backup native MariaDB melalui mariadb-dump dengan kompatibilitas fitur penuh
                </p>
              </div>
            </div>
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
                Bagaimana cara instalasinya?
              </h2>
              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus mendukung beberapa metode instalasi. Deploy di VPS, mesin lokal, atau
                cluster Kubernetes dalam sekitar 2 menit. Instalasi yang sama bisa digunakan untuk
                backup MySQL, MariaDB, PostgreSQL dan MongoDB
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
              Pertanyaan tentang backup MySQL
            </h2>
            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Pertanyaan umum tentang backup database MySQL dan MariaDB dengan Databasus. Jika ada
              pertanyaan lain, bergabunglah dengan komunitas kami di Telegram
            </p>
          </div>
        </div>
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Apa itu Databasus dan bagaimana cara melakukan backup database MySQL?"
              answer="Databasus adalah tool backup self-hosted berlisensi Apache 2.0 yang menggunakan mysqldump untuk membuat backup MySQL yang konsisten. Databasus melengkapinya dengan antarmuka web modern, penjadwalan otomatis, integrasi penyimpanan cloud (S3, Google Drive, Dropbox), notifikasi real-time (Slack, Discord, Telegram), dan enkripsi AES-256-GCM, sehingga tidak perlu menulis skrip shell dan cron job sendiri."
            />
            <FaqItem
              number="2"
              question="Apakah Databasus alternatif atau pengganti mysqldump?"
              answer="Databasus melengkapi mysqldump, bukan menggantikannya. Databasus menjalankan mysqldump dengan parameter yang dioptimalkan (--single-transaction, --routines, --triggers, --events), lalu menangani kompresi, enkripsi, dan upload ke penyimpanan yang dikonfigurasi. Keandalan mysqldump tetap tersedia, ditambah penjadwalan, penyimpanan cloud, notifikasi, dan fitur kolaborasi tim melalui antarmuka web."
            />
            <FaqItem
              number="3"
              question="Versi MySQL apa saja yang didukung Databasus?"
              answer="Databasus mendukung MySQL 5.7, 8.0, 8.4, 9 dan 26. Untuk MySQL 8.0 dan yang lebih baru, Databasus menggunakan kompresi jaringan zstd agar transfer melalui koneksi lambat lebih cepat. Untuk MySQL 5.7, kompresi legacy digunakan otomatis. MySQL 5.7 hanya didukung pada x86-64. Semua versi mendapat fitur yang sama: penjadwalan, enkripsi, penyimpanan cloud, dan notifikasi."
            />
            <FaqItem
              number="4"
              question="Versi MariaDB apa saja yang didukung Databasus?"
              answer="Databasus mendukung MariaDB 5.5, 10, 11, 12 dan 13. Untuk backup MariaDB, Databasus menggunakan tool native mariadb-dump, bukan mysqldump, agar sepenuhnya kompatibel dengan fitur dan optimasi khusus MariaDB. Antarmuka web, penjadwalan, penyimpanan, dan fitur notifikasi yang sama berlaku untuk MySQL maupun MariaDB."
            />
            <FaqItem
              number="5"
              question="Bisakah melakukan backup database MySQL di AWS RDS, Google Cloud SQL atau Azure?"
              answer="Ya, Databasus bekerja dengan database MySQL di cloud, termasuk AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL dan PlanetScale. Karena menggunakan backup logical melalui mysqldump, cukup sediakan kredensial koneksi MySQL standar (host, port, username, password), tanpa izin cloud khusus, role IAM, atau akses filesystem."
            />
            <FaqItem
              number="6"
              question="Bagaimana penjadwalan backup MySQL di Databasus bekerja?"
              answer="Jadwalkan backup MySQL per jam, harian, mingguan, atau bulanan, atau gunakan ekspresi cron untuk pengaturan presisi. Pilih hari dan waktu tertentu, misalnya pukul 3 AM saat trafik rendah. Databasus menangani sisanya: menghubungkan ke MySQL, menjalankan mysqldump, mengompresi hasil, mengenkripsinya, dan meng-upload ke tujuan penyimpanan."
            />
            <FaqItem
              number="7"
              question="Di mana backup MySQL disimpan?"
              answer="Kamu menentukan tempat penyimpanan file backup MySQL: secara lokal di server, di penyimpanan kompatibel S3 (AWS S3, MinIO, Backblaze B2), Google Drive, Cloudflare R2, Dropbox, NAS melalui SFTP, atau tujuan apa pun yang didukung rclone. Setiap database bisa memiliki tujuan penyimpanan sendiri. File backup tetap dalam kendalimu."
            />
            <FaqItem
              number="8"
              question="Bagaimana Databasus mengamankan kredensial dan backup database MySQL?"
              answer={
                <>
                  Databasus menerapkan keamanan berlapis:
                  <br />
                  <br />
                  <strong>1. Enkripsi kredensial:</strong> Semua password, token, dan connection
                  string MySQL dienkripsi dengan AES-256-GCM sebelum disimpan. Kunci enkripsi
                  disimpan terpisah dari database.
                  <br />
                  <br />
                  <strong>2. Enkripsi backup:</strong> Setiap file backup dienkripsi dengan kunci
                  unik yang diturunkan dari master key, ID backup, dan salt acak.
                  <br />
                  <br />
                  <strong>3. Penanganan password yang aman:</strong> Password MySQL diteruskan
                  melalui file .my.cnf sementara dengan izin 0600, tanpa ditampilkan dalam argumen
                  command line atau log.
                </>
              }
            />
            <FaqItem
              number="9"
              question="Bisakah backup MySQL di-restore ke server atau penyedia cloud lain?"
              answer="Ya, karena Databasus membuat backup logical mysqldump standar, backup bisa di-restore ke server MySQL mana pun, baik versi berbeda, penyedia cloud lain, maupun mesin development lokal. Unduh backup dari Databasus (otomatis didekripsi), lalu gunakan perintah mysql untuk restore. Databasus menampilkan perintah restore yang tepat untuk setiap backup."
            />
            <FaqItem
              number="10"
              question="Apakah Databasus mendukung backup replication slave MySQL?"
              answer="Ya, kamu bisa melakukan backup server replica (slave) MySQL. Databasus menggunakan --single-transaction yang membuat snapshot konsisten tanpa menghentikan replikasi. Ini berguna untuk mengalihkan beban backup dari server primary. Cukup arahkan Databasus ke detail koneksi replica."
            />
            <FaqItem
              number="11"
              question="Apa perbedaan Databasus dengan MySQL Enterprise Backup atau Percona XtraBackup?"
              answer={
                <>
                  MySQL Enterprise Backup dan Percona XtraBackup membuat backup physical (biner)
                  yang memerlukan akses filesystem langsung ke direktori data MySQL. Keduanya lebih
                  cepat untuk database sangat besar, tetapi:
                  <br />
                  <br />• Tidak bisa melakukan backup MySQL di cloud (RDS, Cloud SQL, Azure)
                  <br />• Harus diinstal pada server database itu sendiri
                  <br />• Proses setup dan restore lebih rumit
                  <br />
                  <br />
                  Databasus menggunakan backup logical (mysqldump) yang bekerja dengan server MySQL
                  mana pun yang bisa dihubungi, termasuk semua penyedia cloud. Untuk sebagian besar
                  database di bawah 100GB, backup logical praktis dan jauh lebih mudah dikelola.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Bisakah melakukan backup database MySQL dan PostgreSQL dengan instance Databasus yang sama?"
              answer="Ya, Databasus mendukung MySQL, MariaDB, PostgreSQL dan MongoDB dalam satu instalasi. Kelola backup seluruh database dari satu antarmuka web, dengan jadwal, tujuan penyimpanan, dan kanal notifikasi berbeda untuk masing-masing database. Ini berguna bagi tim yang mengelola infrastruktur dengan berbagai jenis database."
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
