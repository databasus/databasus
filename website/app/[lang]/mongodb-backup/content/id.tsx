import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Backup MongoDB",
  description:
    "Tool gratis dan open source untuk backup MongoDB terjadwal. Otomatiskan mongodump melalui UI web, simpan arsip di S3, Google Drive, atau secara lokal. Terima notifikasi melalui Slack, Discord, Telegram. Enkripsi AES-256 untuk data BSON.",
  keywords:
    "backup MongoDB, alternatif mongodump, otomatisasi backup MongoDB, tool backup MongoDB, backup MongoDB terjadwal, backup MongoDB cloud, backup MongoDB S3, backup MongoDB Docker, enkripsi backup MongoDB, backup MongoDB Atlas, backup replica set, backup database dokumen, backup BSON, backup NoSQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("id", "mongodb-backup"),
    languages: getLanguageAlternates("mongodb-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("id", "mongodb-backup"),
    title: "Backup MongoDB",
    description:
      "Tool gratis dan open source untuk backup MongoDB terjadwal. Otomatiskan mongodump melalui UI web, penyimpanan cloud, notifikasi, dan enkripsi.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Antarmuka dashboard Databasus yang menampilkan pengelolaan backup MongoDB",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary_large_image",
    title: "Backup MongoDB",
    description:
      "Tool gratis dan open source untuk backup MongoDB terjadwal. Otomatiskan mongodump melalui UI web, penyimpanan cloud, notifikasi, dan enkripsi.",
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

export default function MongodbBackupPage() {
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
              "Tool gratis dan open source untuk backup MongoDB terjadwal. Otomatiskan mongodump melalui UI web, penyimpanan cloud, notifikasi, dan enkripsi.",
            url: getLocalizedUrl("id", "mongodb-backup"),
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
              "Backup MongoDB terjadwal melalui mongodump",
              "Berbagai tujuan penyimpanan (S3, Google Drive, Dropbox, SFTP, rclone, dan lainnya)",
              "Notifikasi real-time (Slack, Telegram, Discord, Webhook, email, dan lainnya)",
              "Pemantauan ketersediaan koneksi MongoDB",
              "Self-hosted melalui Docker",
              "Open source dan gratis",
              "Dukungan MongoDB 4, 5, 6, 7, 8 dan 9",
              "Kompresi arsip BSON dengan gzip",
              "Enkripsi AES-256-GCM untuk file backup",
              "Dukungan MongoDB Atlas dan replica set",
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
                name: "Apa itu Databasus dan bagaimana cara melakukan backup database MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus adalah tool backup self-hosted berlisensi Apache 2.0 yang menggunakan mongodump untuk membuat backup MongoDB yang konsisten. Databasus melengkapinya dengan antarmuka web modern, penjadwalan otomatis, integrasi penyimpanan cloud, notifikasi real-time, dan enkripsi AES-256-GCM, sehingga tidak perlu menulis skrip shell sendiri.",
                },
              },
              {
                "@type": "Question",
                name: "Apakah Databasus mendukung replica set MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Ya, Databasus mendukung replica set MongoDB sepenuhnya. Hubungkan ke anggota replica set mana pun menggunakan format URI koneksi MongoDB standar. Databasus membaca dari node yang ditentukan, sehingga backup bisa dilakukan dari node secondary untuk mengurangi beban primary.",
                },
              },
              {
                "@type": "Question",
                name: "Bisakah melakukan backup database MongoDB Atlas dengan Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Ya, Databasus bekerja dengan MongoDB Atlas. Karena menggunakan backup logical melalui mongodump, cukup sediakan kredensial koneksi MongoDB standar, tanpa izin atau role Atlas khusus. Masukkan connection string Atlas dan Databasus menangani sisanya.",
                },
              },
              {
                "@type": "Question",
                name: "Versi MongoDB apa saja yang didukung Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus mendukung MongoDB versi 4, 5, 6, 7, 8 dan 9. Semua backup menggunakan tool native mongodump dengan flag --archive dan --gzip untuk membuat arsip BSON terkompresi yang efisien dan bisa di-restore dengan mongorestore.",
                },
              },
              {
                "@type": "Question",
                name: "Bagaimana Databasus mengamankan kredensial dan backup MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus menerapkan keamanan berlapis: (1) Semua password dan connection string MongoDB dienkripsi dengan AES-256-GCM sebelum disimpan; (2) Setiap file backup dienkripsi dengan kunci unik yang diturunkan dari master key, ID backup, dan salt acak; (3) URI koneksi diteruskan dengan aman ke mongodump, tanpa ditampilkan di log atau output command line.",
                },
              },
              {
                "@type": "Question",
                name: "Apakah Databasus mendukung backup incremental MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus berfokus pada backup logical penuh menggunakan mongodump, bukan backup incremental. Untuk sebagian besar kebutuhan, backup penuh terjadwal (per jam, harian, mingguan) menyediakan titik pemulihan yang cukup. MongoDB Atlas sudah menawarkan point-in-time recovery native, dan backup incremental eksternal tidak mudah di-restore ke cluster Atlas.",
                },
              },
            ],
          }),
        }}
      />
      <LandingHeaderComponent lang="id" sectionLinks={[{ href: "#features", label: "Fitur" }]} />
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
              Tool backup MongoDB
            </h1>
            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus adalah tool gratis, open source, dan self-hosted untuk backup database
              dokumen MongoDB. Otomatiskan mongodump dengan penjadwalan, simpan arsip BSON di S3,
              Google Drive, atau secara lokal. Terima notifikasi melalui Slack, Discord, atau
              Telegram saat backup selesai
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
                alt="Antarmuka dashboard Databasus untuk mengelola backup MongoDB"
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
              Fitur backup MongoDB
            </h2>
            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus melengkapi mongodump dengan fitur untuk perusahaan: penjadwalan otomatis,
              integrasi penyimpanan cloud, notifikasi real-time, dan enkripsi AES-256-GCM. Cocok
              untuk developer dan tim DevOps yang mengelola database dokumen dan collection MongoDB
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
                Dump MongoDB terjadwal
              </h3>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Konfigurasi backup MongoDB terjadwal"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Jadwalkan mongodump pada waktu terbaik saat beban aplikasi rendah. Pilih interval
                per jam, harian, mingguan, bulanan, atau gunakan ekspresi cron untuk pengaturan
                waktu yang presisi
              </p>
            </div>
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>
              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Pemantauan ketersediaan MongoDB
              </h3>
              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Pemeriksaan ketersediaan MongoDB"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base mb-3">
                Pantau ketersediaan koneksi MongoDB dengan pemeriksaan yang bisa diatur. Terima
                notifikasi saat database atau replica set tidak bisa dijangkau
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
                Simpan arsip BSON di mana saja
              </h3>
              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Simpan arsip backup MongoDB secara lokal, di penyimpanan kompatibel S3, Google
                Drive, Dropbox, NAS, atau tujuan lain. Data dokumen tetap dalam kendalimu.{" "}
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
                  alt="Tujuan penyimpanan backup MongoDB"
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
                Terima pemberitahuan saat backup MongoDB selesai atau gagal. Kirim notifikasi ke
                chat tim DevOps atau kanal pribadi.{" "}
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
                  alt="Notifikasi backup MongoDB"
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
                Jalankan Databasus di infrastruktur sendiri. Semua connection string MongoDB dan
                data backup tetap berada di server yang kamu kendalikan. Deploy dalam sekitar 2
                menit melalui skrip, Docker, atau Kubernetes
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
                  Versi MongoDB yang didukung
                </h3>
                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Mendukung MongoDB 4, 5, 6, 7, 8 dan 9. Databasus menggunakan tool native mongodump
                  untuk setiap versi agar sepenuhnya kompatibel dengan database dokumenmu
                </p>
                <div>
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="Versi MongoDB"
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
                  Connection string MongoDB dienkripsi dengan AES-256-GCM sebelum disimpan. Setiap
                  arsip BSON dienkripsi dengan kunci unik. Kredensial diteruskan dengan aman ke
                  mongodump, tanpa ditampilkan di log.{" "}
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
                    alt="Keamanan backup MongoDB"
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
                  alt="Pengelolaan akses backup MongoDB"
                  className="w-full"
                  loading="lazy"
                />
              </div>
              <p className="text-gray-400 text-sm md:text-base">
                Atur siapa yang bisa melihat atau mengelola database MongoDB. Buat ruang kerja untuk
                proyek berbeda. Tetapkan role viewer, editor, atau admin.{" "}
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
                  alt="Log audit backup MongoDB"
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
                Connection string MongoDB dienkripsi dengan AES-256-GCM sebelum disimpan. Setiap
                arsip BSON dienkripsi dengan kunci unik. Kredensial diteruskan dengan aman ke
                mongodump, tanpa ditampilkan di log.{" "}
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
                  alt="Keamanan backup MongoDB"
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
                  Bekerja dengan MongoDB Atlas dan self-hosted
                </h3>
                <p className="text-gray-400 text-sm md:text-base">
                  Databasus terhubung ke database MongoDB di cloud, termasuk MongoDB Atlas, AWS
                  DocumentDB, serta deployment self-hosted. Karena menggunakan backup logical
                  melalui mongodump, cukup sediakan kredensial koneksi standar, tanpa izin cloud
                  khusus atau akses filesystem
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section id="mongodump" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Menggunakan mongodump</span>
              </div>
              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Cara kerja backup MongoDB
              </h2>
              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus menggunakan <strong>mongodump</strong>, utilitas backup resmi MongoDB.
                  Saat backup dipicu, Databasus menjalankan mongodump dengan parameter yang
                  dioptimalkan:
                </p>
                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">--archive</code>{" "}
                    untuk output BSON dalam satu file, bukan struktur direktori
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">--gzip</code> untuk
                    arsip terkompresi yang mengurangi ukuran penyimpanan dan transfer
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">--db</code> untuk
                    backup database tertentu dari instance MongoDB
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">--uri</code> untuk
                    penanganan connection string yang aman dengan autentikasi
                  </li>
                </ul>
                <p className="text-gray-400">
                  Stream backup diteruskan langsung ke penyimpanan yang dikonfigurasi, dengan opsi
                  enkripsi AES-256-GCM sebelum ditulis. Pendekatan ini meminimalkan I/O disk dan
                  efisien untuk collection besar.
                </p>
                <div className="pt-2">
                  <p className="text-white font-medium mb-2">Versi MongoDB yang didukung:</p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 6
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 8
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 9
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mongodb.svg"
                  alt="Database MongoDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Backup resmi MongoDB melalui mongodump dengan kompresi gzip, enkripsi, dan
                  penyimpanan cloud
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
                backup MongoDB, PostgreSQL, MySQL, dan MariaDB
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
              Pertanyaan tentang backup MongoDB
            </h2>
            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Pertanyaan umum tentang backup database dokumen MongoDB dengan Databasus. Jika ada
              pertanyaan lain, bergabunglah dengan komunitas kami di Telegram
            </p>
          </div>
        </div>
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Apa itu Databasus dan bagaimana cara melakukan backup database MongoDB?"
              answer="Databasus adalah tool backup self-hosted berlisensi Apache 2.0 yang menggunakan mongodump untuk membuat backup MongoDB yang konsisten. Databasus melengkapinya dengan antarmuka web modern, penjadwalan otomatis, integrasi penyimpanan cloud (S3, Google Drive, Dropbox), notifikasi real-time (Slack, Discord, Telegram), dan enkripsi AES-256-GCM, sehingga tidak perlu menulis skrip shell dan cron job sendiri."
            />
            <FaqItem
              number="2"
              question="Apakah Databasus mendukung replica set MongoDB?"
              answer="Ya, Databasus mendukung replica set MongoDB sepenuhnya. Hubungkan ke anggota replica set mana pun menggunakan format URI koneksi MongoDB standar dengan opsi replica set. Databasus membaca dari node yang ditentukan, sehingga backup bisa dilakukan dari node secondary untuk mengurangi beban primary. Ini sangat berguna di lingkungan production agar backup tidak memengaruhi performa node primary."
            />
            <FaqItem
              number="3"
              question="Bisakah melakukan backup database MongoDB Atlas dengan Databasus?"
              answer="Ya, Databasus bekerja dengan MongoDB Atlas. Karena menggunakan backup logical melalui mongodump, cukup sediakan kredensial koneksi MongoDB standar, tanpa izin Atlas khusus, IP whitelisting selain server Databasus, atau role administratif. Masukkan connection string Atlas (tersedia di dashboard Atlas) dan Databasus menangani sisanya."
            />
            <FaqItem
              number="4"
              question="Versi MongoDB apa saja yang didukung Databasus?"
              answer="Databasus mendukung MongoDB versi 4, 5, 6, 7, 8 dan 9. Semua backup menggunakan tool native mongodump dengan flag --archive dan --gzip untuk membuat arsip BSON terkompresi yang efisien. Arsip bisa di-restore menggunakan mongorestore ke versi MongoDB yang kompatibel, sehingga memudahkan migrasi antarversi."
            />
            <FaqItem
              number="5"
              question="Bagaimana Databasus menangani collection MongoDB yang besar?"
              answer="Databasus mengalirkan output mongodump langsung ke tujuan penyimpanan, dengan opsi enkripsi selama stream diteruskan. Pendekatan ini menghindari penulisan file sementara ke disk sehingga efisien untuk database dengan collection besar. Flag --archive membuat satu file terkompresi, bukan struktur direktori, sehingga mengurangi overhead I/O dan mempermudah pengelolaan penyimpanan."
            />
            <FaqItem
              number="6"
              question="Bisakah melakukan backup cluster MongoDB sharded dengan Databasus?"
              answer={
                <>
                  Databasus saat ini berfokus pada backup database MongoDB individual, bukan backup
                  cluster sharded yang terkoordinasi.
                  <br />
                  <br />
                  Untuk cluster sharded, kamu bisa:
                  <br />
                  <br />• Melakukan backup setiap shard secara terpisah dengan menghubungkan ke
                  replica set shard
                  <br />• Melakukan backup melalui router mongos (meskipun bisa memengaruhi
                  performa)
                  <br />
                  <br />
                  Untuk cluster sharded production, pertimbangkan backup native MongoDB Atlas atau
                  mongodump dengan --oplog untuk konsistensi pada satu titik waktu di seluruh shard.
                </>
              }
            />
            <FaqItem
              number="7"
              question="Bagaimana Databasus mengamankan kredensial dan backup MongoDB?"
              answer={
                <>
                  Databasus menerapkan keamanan berlapis:
                  <br />
                  <br />
                  <strong>1. Enkripsi kredensial:</strong> Semua URI koneksi, password, dan detail
                  autentikasi MongoDB dienkripsi dengan AES-256-GCM sebelum disimpan.
                  <br />
                  <br />
                  <strong>2. Enkripsi backup:</strong> Setiap arsip BSON dienkripsi dengan kunci
                  unik yang diturunkan dari master key, ID backup, dan salt acak.
                  <br />
                  <br />
                  <strong>3. Penanganan kredensial yang aman:</strong> URI koneksi diteruskan
                  langsung ke mongodump melalui parameter aman, tanpa ditampilkan di log atau daftar
                  proses.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Apakah Databasus mendukung backup incremental MongoDB atau oplog tailing?"
              answer="Databasus berfokus pada backup logical penuh menggunakan mongodump, bukan backup incremental atau point-in-time recovery berbasis oplog. Untuk sebagian besar kebutuhan, backup penuh terjadwal (per jam, harian, mingguan) menyediakan titik pemulihan yang cukup tanpa kerumitan pengelolaan oplog. MongoDB Atlas sudah menawarkan backup berkelanjutan native dengan point-in-time recovery, dan backup incremental eksternal tidak mudah di-restore ke cluster Atlas."
            />
            <FaqItem
              number="9"
              question="Bisakah backup MongoDB di-restore ke versi atau cluster lain?"
              answer="Ya, karena Databasus membuat arsip mongodump standar dalam format BSON, backup bisa di-restore ke server MongoDB yang kompatibel, baik versi berbeda, penyedia cloud lain, maupun mesin development lokal. Unduh backup dari Databasus (otomatis didekripsi), lalu gunakan mongorestore dengan flag --archive dan --gzip. Databasus menampilkan perintah restore yang tepat untuk setiap backup."
            />
            <FaqItem
              number="10"
              question="Bagaimana kompresi mongodump bekerja di Databasus?"
              answer="Databasus menggunakan flag bawaan --gzip milik mongodump untuk mengompresi data BSON selama proses dump. Biasanya, ukuran arsip berkurang 60-80% dibandingkan BSON tanpa kompresi. Kompresi dilakukan dalam stream mongodump sebelum enkripsi opsional, sehingga arsip terkompresi maupun terenkripsi tetap efisien. Dekompresi dilakukan otomatis saat menggunakan mongorestore dengan flag --gzip."
            />
            <FaqItem
              number="11"
              question="Bisakah melakukan backup collection MongoDB tertentu, bukan seluruh database?"
              answer="Saat ini, Databasus melakukan backup seluruh database MongoDB, bukan collection individual. Dengan begitu, backup lengkap dan konsisten mencakup semua collection, indeks, dan metadata. Jika memerlukan backup tingkat collection, kamu bisa membuat database terpisah untuk domain data berbeda, masing-masing dengan jadwal backup sendiri di Databasus."
            />
            <FaqItem
              number="12"
              question="Apakah Databasus bekerja dengan MongoDB di Docker atau Kubernetes?"
              answer="Ya, Databasus terhubung ke MongoDB melalui jaringan menggunakan URI koneksi standar, sehingga bekerja di mana pun MongoDB di-deploy: container Docker, pod Kubernetes, VM, atau bare metal. Pastikan koneksi jaringan tersedia antara Databasus dan instance MongoDB. Untuk deployment Kubernetes, gunakan nama DNS service internal atau endpoint load balancer eksternal."
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
