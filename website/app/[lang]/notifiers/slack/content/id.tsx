import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Cara mengatur notifikasi Slack untuk Databasus | Databasus",
  description:
    "Panduan langkah demi langkah menyiapkan notifikasi Slack untuk peringatan backup PostgreSQL dengan Databasus. Pelajari cara membuat aplikasi bot Slack dan mengatur notifikasi.",
  keywords: [
    "Databasus",
    "notifikasi Slack",
    "backup PostgreSQL",
    "token bot Slack",
    "Slack API",
    "peringatan backup",
    "notifikasi database",
  ],
  openGraph: {
    title: "Cara mengatur notifikasi Slack untuk Databasus | Databasus",
    description:
      "Panduan langkah demi langkah menyiapkan notifikasi Slack untuk peringatan backup PostgreSQL dengan Databasus. Pelajari cara membuat aplikasi bot Slack dan mengatur notifikasi.",
    type: "article",
    url: getLocalizedUrl("id", "notifiers/slack"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara mengatur notifikasi Slack untuk Databasus | Databasus",
    description:
      "Panduan langkah demi langkah menyiapkan notifikasi Slack untuk peringatan backup PostgreSQL dengan Databasus. Pelajari cara membuat aplikasi bot Slack dan mengatur notifikasi.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "notifiers/slack"),
    languages: getLanguageAlternates("notifiers/slack"),
  },
  robots: "index, follow",
};

export default function SlackPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara mengatur notifikasi Slack untuk Databasus",
            description:
              "Panduan langkah demi langkah menyiapkan notifikasi Slack untuk peringatan backup PostgreSQL dengan Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Buka Slack API",
                text: "Buka https://api.slack.com/apps dan masuk ke ruang kerja Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Buat aplikasi baru",
                text: "Klik tombol 'Create New App' dan pilih 'From scratch'.",
              },
              {
                "@type": "HowToStep",
                name: "Atur izin bot",
                text: "Buka OAuth & Permissions dan tambahkan scope yang diperlukan: chat:write, channels:join, im:write, dan groups:write di bagian Bot Token Scopes.",
              },
              {
                "@type": "HowToStep",
                name: "Instal ke ruang kerja",
                text: "Instal aplikasi ke ruang kerja dan berikan izin akses.",
              },
              {
                "@type": "HowToStep",
                name: "Salin token bot",
                text: "Salin Bot User OAuth Token yang diawali dengan 'xoxb-'.",
              },
              {
                "@type": "HowToStep",
                name: "Dapatkan Channel ID",
                text: "Buka kanal tujuan dan dapatkan Channel ID dari detail kanal.",
              },
              {
                "@type": "HowToStep",
                name: "Tambahkan bot ke kanal privat",
                text: "Jika menggunakan kanal privat, undang bot ke kanal dengan menyebutnya.",
              },
              {
                "@type": "HowToStep",
                name: "Atur di Databasus",
                text: "Di Databasus, masukkan Bot Token dan Channel ID ke konfigurasi kanal notifikasi Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Uji notifikasi",
                text: "Uji notifikasi untuk memastikan semuanya berfungsi dengan benar.",
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
              <h1 id="slack-notifications">Notifikasi Slack</h1>
              <p className="text-lg text-gray-400">
                Atur Slack untuk menerima notifikasi langsung tentang status backup PostgreSQL.
                Terima pemberitahuan backup berhasil, kegagalan, dan peringatan langsung di kanal
                Slack.
              </p>
              <h2 id="create-slack-app">Buat aplikasi Slack</h2>
              <h3 id="go-to-slack-api">1. Buka Slack API</h3>
              <p>
                Buka{" "}
                <a href="https://api.slack.com/apps" target="_blank" rel="noopener noreferrer">
                  https://api.slack.com/apps
                </a>{" "}
                dan masuk ke ruang kerja Slack.
              </p>
              <h3 id="create-new-app">2. Buat aplikasi baru</h3>
              <p>
                Klik tombol <strong>&quot;Create New App&quot;</strong>.
              </p>
              <h3 id="choose-from-scratch">3. Pilih &quot;From scratch&quot;</h3>
              <p>
                Pilih opsi <strong>&quot;From scratch&quot;</strong> saat diminta.
              </p>
              <h3 id="name-your-app">4. Beri nama aplikasi</h3>
              <p>
                Masukkan nama aplikasi (misalnya &quot;Databasus Notifications&quot;) dan pilih
                ruang kerja tempat aplikasi akan diinstal. Klik{" "}
                <strong>&quot;Create App&quot;</strong>.
              </p>
              <h2 id="configure-bot-permissions">Atur izin bot</h2>
              <h3 id="navigate-to-oauth">5. Buka OAuth &amp; Permissions</h3>
              <p>
                Di sidebar kiri, klik <strong>&quot;OAuth &amp; Permissions&quot;</strong>.
              </p>
              <img
                src="/images/notifier-slack/image-1.png"
                alt="Membuka OAuth &amp; Permissions"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />
              <h3 id="add-bot-scopes">6. Tambahkan Bot Token Scopes (wajib)</h3>
              <p>
                Gulir ke bagian <strong>&quot;Scopes&quot;</strong>. Di bawah{" "}
                <strong>&quot;Bot Token Scopes&quot;</strong>, klik{" "}
                <strong>&quot;Add an OAuth Scope&quot;</strong>.
              </p>
              <p>Tambahkan semua scope wajib berikut:</p>
              <ul>
                <li>
                  <code>chat:write</code> - untuk mengirim pesan ke kanal
                </li>
                <li>
                  <code>channels:join</code> - agar bot bisa bergabung ke kanal publik secara
                  otomatis
                </li>
                <li>
                  <code>im:write</code> - untuk mengirim pesan langsung ke pengguna
                </li>
                <li>
                  <code>groups:write</code> - untuk mengirim pesan ke kanal privat
                </li>
                <li>
                  <code>channels:history</code> - untuk membaca riwayat kanal
                </li>
              </ul>
              <img
                src="/images/notifier-slack/image-2.png"
                alt="Menambahkan Bot Token Scopes"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />
              <h2 id="install-app">Instal aplikasi ke ruang kerja</h2>
              <h3 id="install-to-workspace">7. Instal ke ruang kerja</h3>
              <p>
                Gulir ke bagian atas halaman <strong>&quot;OAuth &amp; Permissions&quot;</strong>{" "}
                dan klik <strong>&quot;Install to Workspace&quot;</strong>.
              </p>
              <img
                src="/images/notifier-slack/image-3.png"
                alt="Menginstal ke ruang kerja"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />
              <h3 id="authorize-app">8. Berikan izin akses aplikasi</h3>
              <p>
                Tinjau izinnya, lalu klik <strong>&quot;Allow&quot;</strong> untuk memberikan akses
                ke aplikasi.
              </p>
              <h3 id="copy-bot-token">9. Salin Bot User OAuth Token</h3>
              <p>
                Setelah instalasi, akan muncul <strong>&quot;Bot User OAuth Token&quot;</strong>{" "}
                yang diawali dengan <code>xoxb-</code>. Salin token ini untuk konfigurasi Databasus.
              </p>
              <h2 id="get-channel-id">Dapatkan Channel ID</h2>
              <h3 id="open-channel">10. Buka kanal tujuan</h3>
              <p>Di ruang kerja Slack, buka kanal tempat kamu ingin menerima notifikasi backup.</p>
              <h3 id="get-channel-info">11. Dapatkan ID kanal</h3>
              <p>
                Klik nama kanal di bagian atas, lalu gulir ke bawah di detail kanal.{" "}
                <strong>Channel ID</strong> ada di bagian bawah &quot;About&quot;. ID ini diawali
                dengan <code>C</code> (untuk kanal publik) atau <code>G</code> (untuk kanal privat).
              </p>
              <p>Salin Channel ID tersebut.</p>
              <img
                src="/images/notifier-slack/image-4.png"
                alt="Mendapatkan Channel ID"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[500px]"
                loading="lazy"
              />
              <h3 id="add-bot-to-channel">12. Tambahkan bot ke kanal (wajib untuk kanal privat)</h3>
              <p>
                <strong>
                  Jika menggunakan kanal privat, kamu harus mengundang bot ke kanal secara manual:
                </strong>
              </p>
              <ol>
                <li>
                  Di kanal privat, ketik <code>@Databasus Notifications</code> (atau nama yang kamu
                  berikan pada aplikasi)
                </li>
                <li>
                  Klik nama bot saat muncul dan pilih <strong>&quot;Add to Channel&quot;</strong>{" "}
                  atau <strong>&quot;Invite to Channel&quot;</strong>
                </li>
              </ol>
              <p>
                Untuk <strong>kanal publik</strong>, bot akan otomatis bergabung saat mengirim pesan
                pertama berkat izin <code>channels:join</code>, jadi langkah ini tidak diperlukan.
              </p>
              <h2 id="configure-databasus">Atur di Databasus</h2>
              <h3 id="add-slack-notifier">13. Tambahkan kanal notifikasi Slack</h3>
              <p>
                Di Databasus, buka pengaturan kanal notifikasi dan tambahkan kanal notifikasi Slack
                baru:
              </p>
              <ul>
                <li>
                  <strong>Bot Token:</strong> Tempel Bot User OAuth Token yang sudah disalin
                  (diawali dengan <code>xoxb-</code>)
                </li>
                <li>
                  <strong>Target Channel ID:</strong> Tempel Channel ID yang sudah disalin (diawali
                  dengan <code>C</code>, <code>G</code>, <code>D</code>, atau <code>U</code>)
                </li>
              </ul>
              <img
                src="/images/notifier-slack/image-5.png"
                alt="Menambahkan kanal notifikasi Slack"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />
              <h3 id="test-notification">14. Uji notifikasi</h3>
              <p>
                Setelah kanal notifikasi dikonfigurasi, uji untuk memastikan semuanya berfungsi
                dengan benar. Pesan uji akan diterima di kanal Slack yang dipilih.
              </p>
              <p>
                Ruang kerja Slack sekarang siap menerima notifikasi backup PostgreSQL dari
                Databasus.
              </p>
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Kembali ke kanal notifikasi
                </a>
              </div>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
