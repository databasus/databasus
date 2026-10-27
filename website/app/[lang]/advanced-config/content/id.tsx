import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Konfigurasi Lanjutan - Dokumentasi Databasus",
  description:
    "Environment variable opsional untuk Databasus self-hosted: login Google dan GitHub, email SMTP, captcha Cloudflare Turnstile, telemetri, ekspor log OpenTelemetry, dan script analitik khusus. Tidak diperlukan untuk instalasi default.",
  keywords: [
    "environment variable Databasus",
    "konfigurasi lanjutan Databasus",
    "konfigurasi self-hosted",
    "GitHub OAuth",
    "Google OAuth",
    "pengaturan email SMTP",
    "Cloudflare Turnstile",
    "environment variable Docker",
    "log OpenTelemetry",
  ],
  openGraph: {
    title: "Konfigurasi Lanjutan - Dokumentasi Databasus",
    description:
      "Environment variable opsional untuk Databasus self-hosted: login Google dan GitHub, email SMTP, captcha Cloudflare Turnstile, telemetri, ekspor log OpenTelemetry, dan script analitik khusus. Tidak diperlukan untuk instalasi default.",
    type: "article",
    url: getLocalizedUrl("id", "advanced-config"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Konfigurasi Lanjutan - Dokumentasi Databasus",
    description:
      "Environment variable opsional untuk Databasus self-hosted: login Google dan GitHub, email SMTP, captcha Cloudflare Turnstile, telemetri, ekspor log OpenTelemetry, dan script analitik khusus. Tidak diperlukan untuk instalasi default.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "advanced-config"),
    languages: getLanguageAlternates("advanced-config"),
  },
  robots: "index, follow",
};

export default function AdvancedConfigPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Konfigurasi Lanjutan - Dokumentasi Databasus",
            description:
              "Environment variable opsional untuk Databasus self-hosted: login Google dan GitHub, email SMTP, captcha Cloudflare Turnstile, telemetri, ekspor log OpenTelemetry, dan script analitik khusus. Tidak diperlukan untuk instalasi default.",
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
              <h1 id="advanced-config">Konfigurasi lanjutan</h1>
              <p className="text-lg text-gray-400">
                Databasus bisa langsung berjalan dengan pengaturan default yang sesuai. Instalasi
                standar dengan satu container tidak membutuhkan konfigurasi apa pun. Semua variabel
                di halaman ini <strong>opsional</strong> dan tidak diperlukan pada 99% konfigurasi
                production
              </p>
              <h2 id="oauth">OAuth</h2>
              <p>
                Secara default, Databasus menggunakan email dan password untuk login. Kamu juga bisa
                mengizinkan pengguna login dengan akun Google atau GitHub. Tombol penyedia muncul
                begitu client ID diisi, tetapi login baru bisa diselesaikan jika{" "}
                <strong>keduanya</strong>, client ID dan client secret, sudah tersedia.
              </p>
              <p>
                Saat mendaftarkan aplikasi OAuth, atur URL redirect (callback) ke{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. Karena redirect ini, login
                OAuth membutuhkan instance yang bisa diakses melalui HTTPS pada domain publik. Lihat
                catatan berikut.
              </p>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    <strong>HTTPS diperlukan untuk login dan email.</strong> Login OAuth dan email
                    sama-sama membutuhkan instance yang bisa diakses melalui HTTPS pada domain
                    publik. Penyedia OAuth mengarahkan browser kembali ke{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, dan tautan di email
                    harus bisa dibuka oleh penerimanya. Instance yang hanya tersedia di localhost
                    atau menggunakan HTTP biasa tidak bisa memakai fitur ini. Cara termudah
                    mendapatkan HTTPS adalah melalui konfigurasi{" "}
                    <a
                      href={getLocalizedUrl("id", "installation") + "#caddy-reverse-proxy"}
                      className="text-blue-400 hover:text-blue-300"
                    >
                      reverse proxy Caddy
                    </a>
                    .
                  </p>
                </div>
              </div>
              <h3 id="oauth-google">Google</h3>
              <p>
                Buat client OAuth di{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth client ID, jenis
                aplikasi <em>Web application</em>), lalu tambahkan{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code> sebagai authorized redirect
                URI.
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Deskripsi</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Deskripsi">
                      Client ID dari client Google OAuth kamu. Jika diisi, tombol &quot;Sign in with
                      Google&quot; akan muncul.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Deskripsi">
                      Client secret dari client Google OAuth kamu. Harus diisi bersama ID agar login
                      berfungsi.
                    </td>
                  </tr>
                </tbody>
              </table>
              <h3 id="oauth-github">GitHub</h3>
              <p>
                Buat aplikasi OAuth di{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  GitHub Developer settings
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App), lalu atur
                authorization callback URL ke <code>https://&lt;your-domain&gt;/auth/callback</code>
                .
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Deskripsi</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Deskripsi">
                      Client ID dari aplikasi GitHub OAuth kamu. Jika diisi, tombol &quot;Sign in
                      with GitHub&quot; akan muncul.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Deskripsi">
                      Client secret dari aplikasi GitHub OAuth kamu. Harus diisi bersama ID agar
                      login berfungsi.
                    </td>
                  </tr>
                </tbody>
              </table>
              <h2 id="email-smtp">Email (SMTP)</h2>
              <p>
                Hubungkan mail server agar Databasus bisa mengirim undangan ruang kerja, kode reset
                password, dan kode login. Mail server dianggap sudah dikonfigurasi{" "}
                <strong>
                  begitu <code>SMTP_HOST</code> diisi
                </strong>
                . Tanpa variabel ini, layar pengaturan melaporkan mail server belum dikonfigurasi,
                dan layar login menyembunyikan tautan reset password.
              </p>
              <p>
                Ini adalah mail server untuk instance. Kanal notifikasi email yang melaporkan
                peristiwa backup memiliki pengaturan SMTP sendiri pada formulir masing-masing. Jadi,
                keberhasilan kanal notifikasi mengirim email tidak menunjukkan bahwa server ini
                berfungsi. Untuk memeriksa mail server instance, buka{" "}
                <strong>Databasus settings → Mail server</strong>, lalu tekan{" "}
                <strong>Send test email</strong>. Pesan dikirim ke alamat kamu sendiri, dan jika
                gagal, respons mail server akan ditampilkan.
              </p>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>
                      Jika relay kamu tidak menyediakan STARTTLS, atur{" "}
                      <code>SMTP_SECURITY=none</code> sebelum upgrade.
                    </strong>{" "}
                    Pada semua port selain 465, mode default adalah <code>starttls</code>, yang
                    tidak mau melanjutkan tanpa enkripsi. Relay semacam itu tidak akan menerima
                    pesan sampai variabel tersebut diatur. Jika instance mewajibkan kode login dari
                    email, tidak ada yang bisa login dengan password selama email tidak bisa
                    dikirim. Untuk mendapatkan akses kembali, nonaktifkan faktor kedua dengan{" "}
                    <code>docker exec -it databasus ./main --disable-2fa</code> (lihat{" "}
                    <a
                      href={getLocalizedUrl("id", "password") + "#disable-two-factor"}
                      className="text-blue-400 hover:text-blue-300"
                    >
                      Menonaktifkan kewajiban kode login
                    </a>
                    ).
                  </p>
                </div>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Deskripsi</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Deskripsi">
                      Nama host mail server (misalnya <code>smtp.gmail.com</code>). Mengisinya akan
                      mengaktifkan email instance.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Deskripsi">
                      Port mail server (misalnya <code>587</code>). Harus berupa bilangan bulat
                      positif jika <code>SMTP_HOST</code> diisi.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_SECURITY</code>
                    </td>
                    <td data-label="Deskripsi">
                      Keamanan koneksi: <code>tls</code>, <code>starttls</code>, atau{" "}
                      <code>none</code>, dijelaskan di bawah. Default-nya <code>tls</code> pada port
                      465 dan <code>starttls</code> pada port lainnya. Instance tidak akan berjalan
                      jika nilainya selain itu.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Deskripsi">Username untuk autentikasi SMTP.</td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Deskripsi">
                      Password untuk autentikasi SMTP. Untuk Gmail, gunakan App Password, bukan
                      password akun kamu.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Deskripsi">
                      Alamat pengirim, bisa tanpa nama (<code>noreply@example.com</code>) atau
                      dengan nama (<code>Acme Backups &lt;noreply@example.com&gt;</code>). Tanpa
                      nama, penerima akan melihat <code>Databasus</code>. Jika tidak diisi,{" "}
                      <code>SMTP_USER</code> digunakan jika berupa alamat email; jika bukan,
                      digunakan <code>noreply@</code> diikuti <code>SMTP_HOST</code>, sehingga
                      username seperti <code>apikey</code> tidak pernah menjadi pengirim. Instance
                      tidak akan berjalan jika nilainya tidak bisa dibaca sebagai alamat.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_HELO_NAME</code>
                    </td>
                    <td data-label="Deskripsi">
                      Nama yang digunakan instance saat menyapa mail server. Default-nya adalah host
                      dari <code>DATABASUS_URL</code>, lalu nama host mesin jika tidak tersedia.
                      Harus berupa nama host atau alamat literal seperti <code>[192.0.2.1]</code>{" "}
                      atau <code>[IPv6:2001:db8::1]</code>; jika tidak, instance tidak akan
                      berjalan.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Deskripsi">
                      Atur ke <code>true</code> untuk melewati verifikasi sertifikat TLS pada mode{" "}
                      <code>tls</code> dan <code>starttls</code>. Default-nya <code>false</code>.
                      Gunakan hanya untuk server bersertifikat self-signed di jaringan tepercaya,
                      karena pengaturan ini menonaktifkan perlindungan terhadap serangan
                      man-in-the-middle.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Deskripsi">
                      URL dasar publik instance kamu (misalnya{" "}
                      <code>https://backup.example.com</code>). Opsional: menambahkan tautan ke
                      email undangan dan menyediakan nama sapaan default. Email tetap berfungsi
                      tanpanya.
                    </td>
                  </tr>
                </tbody>
              </table>
              <p>
                <code>SMTP_SECURITY</code> memilih satu dari tiga mode. Kanal notifikasi email
                menyediakan pilihan yang sama di <strong>Advanced settings</strong>.
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Mode</th>
                    <th>Perilaku</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>tls</code>
                    </td>
                    <td data-label="Perilaku">
                      Koneksi dienkripsi sejak byte pertama. Umumnya pada port 465.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>starttls</code>
                    </td>
                    <td data-label="Perilaku">
                      Koneksi dimulai tanpa enkripsi, lalu ditingkatkan sebelum password atau pesan
                      apa pun dikirim. Jika server tidak menyediakan peningkatan ini, pengiriman
                      gagal alih-alih dilanjutkan tanpa enkripsi. Umumnya pada port 587.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>none</code>
                    </td>
                    <td data-label="Perilaku">
                      Koneksi tidak pernah dienkripsi. Gunakan untuk relay yang tidak menyediakan
                      STARTTLS, seperti Postfix lokal pada port 25.
                    </td>
                  </tr>
                </tbody>
              </table>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <code>none</code> mengirim password SMTP dan semua pesan tanpa enkripsi. Siapa
                    pun di antara Databasus dan relay bisa membacanya. Gunakan hanya untuk relay
                    pada host yang sama atau di jaringan privat tepercaya.
                  </p>
                </div>
              </div>
              <h2 id="signup-captcha">Captcha pendaftaran (Cloudflare Turnstile)</h2>
              <p>
                Jika instance kamu bisa diakses dari internet publik, kamu bisa memasang tantangan{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                pada formulir pendaftaran dan login untuk menghalau bot. Kedua key berasal dari
                dashboard Turnstile, dan tantangan hanya aktif jika keduanya diisi.
              </p>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    Untuk menghentikan pendaftaran dari luar sepenuhnya, bukan sekadar memberi
                    tantangan, kamu tidak membutuhkan captcha. Buka{" "}
                    <strong>Databasus settings → Allow sign up</strong> di antarmuka, lalu
                    nonaktifkan. Formulir pendaftaran akan ditutup sepenuhnya.
                  </p>
                </div>
              </div>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Deskripsi</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Deskripsi">
                      Site key publik Turnstile, digunakan untuk menampilkan widget di browser.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Deskripsi">
                      Secret key Turnstile, digunakan backend untuk memvalidasi respons tantangan.
                    </td>
                  </tr>
                </tbody>
              </table>
              <h2 id="docker-storage-permissions">Izin penyimpanan Docker</h2>
              <p>
                Biasanya Databasus memilih ID numerik pengguna dan grup dari direktori pgdata yang
                sudah ada, lalu dari mount backup atau direktori root data, dan terakhir menggunakan{" "}
                <code>999</code>. Atur <code>PUID</code> atau <code>PGID</code> hanya jika pilihan
                otomatis ini tidak sesuai dengan bind mount, CIFS share, atau NFS export kamu. Nilai
                harus berupa bilangan bulat basis 10 dari <code>1</code> sampai{" "}
                <code>4294967294</code>.
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Nilai otomatis</th>
                    <th>Akun</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>PUID</code>
                    </td>
                    <td>
                      Pemilik mount atau <code>999</code>
                    </td>
                    <td>
                      Pengguna <code>databasus</code>
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>PGID</code>
                    </td>
                    <td>
                      Grup mount atau <code>999</code>
                    </td>
                    <td>
                      Grup utama <code>databasus</code>
                    </td>
                  </tr>
                </tbody>
              </table>
              <p>
                Databasus dan PostgreSQL menggunakan akun sistem operasi non-root yang sama bernama{" "}
                <code>databasus</code>.
              </p>
              <p>
                Entrypoint dimulai sebagai root di dalam container untuk memilih ID dan mencoba{" "}
                <code>chown</code> serta <code>chmod</code>. Setelah itu, siklus hidup file yang
                sebenarnya diperiksa sebagai <code>databasus</code>. Mount tetap bisa berfungsi
                meskipun menolak perubahan metadata. Override Docker <code>user:</code> sembarangan
                tidak didukung.
              </p>
              <pre>
                <code>{`ERROR: Databasus cannot write to local storage paths /databasus-data/temp and /databasus-data/backups as UID 999 and GID 999.
Required operation: save a file through local storage.
Set PUID and PGID or fix the mounted directory permissions: https://databasus.com/advanced-config/#docker-storage-permissions
Details: permission denied`}</code>
              </pre>
              <p>
                Empat variabel identitas khusus layanan sebelumnya sengaja dihapus. Hapus variabel
                tersebut dari konfigurasi saat upgrade. Jika ID otomatis tidak bisa mengakses mount,
                startup berhenti dengan informasi operasi yang gagal dan tautan dokumentasi ini.
              </p>
              <h2 id="telemetry">Telemetri</h2>
              <p>
                Secara default, Databasus mengirim telemetri penggunaan anonim yang tidak
                mengidentifikasi pengguna. Telemetri tidak berisi data pribadi dan membantu kami
                memahami penggunaan proyek. Rincian data yang dikumpulkan tersedia di{" "}
                <a href="/privacy" className="text-blue-400 hover:text-blue-300">
                  kebijakan privasi
                </a>
                , dan kamu bisa menonaktifkannya sepenuhnya.
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Default</th>
                    <th>Deskripsi</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Default">
                      <code>false</code>
                    </td>
                    <td data-label="Deskripsi">
                      Atur ke <code>true</code> untuk menonaktifkan telemetri penggunaan anonim.
                    </td>
                  </tr>
                </tbody>
              </table>
              <h2 id="logging">Logging</h2>
              <p>
                Databasus menulis log ke stdout dan menyalinnya sebagai JSON ke{" "}
                <code>databasus.log</code> pada volume data. Atur <code>OPEN_TELEMETRY_URL</code>{" "}
                agar log juga diekspor melalui Open Telemetry ke backend seperti VictoriaLogs,
                Graylog, SigNoz, Grafana Loki, Datadog, atau Honeycomb, maupun ke OpenTelemetry
                Collector yang juga merupakan penerima OTLP.
              </p>
              <ul>
                <li>
                  <strong>Transport</strong> mengikuti skema. <code>http://</code> dan{" "}
                  <code>https://</code> mengirim OTLP/HTTP dan menggunakan URL apa adanya, termasuk
                  path; <code>grpc://</code> dan <code>grpcs://</code> mengirim OTLP/gRPC dan hanya
                  menggunakan host serta port.
                </li>
                <li>
                  <strong>Autentikasi</strong> dimasukkan ke <code>OPEN_TELEMETRY_HEADERS</code>{" "}
                  atau ke URL sebagai <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Secret</strong> (password, token, kredensial) di dalam URL disamarkan
                  sebelum catatan keluar dari proses.
                </li>
                <li>
                  <strong>Catatan audit</strong> dikirim bersama log aplikasi dengan tag{" "}
                  <code>log_type=audit</code> dan mengabaikan <code>LOG_LEVEL</code>, sehingga
                  menaikkan level tidak pernah menghilangkan jejak audit.
                </li>
              </ul>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Default</th>
                    <th>Deskripsi</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Default">-</td>
                    <td data-label="Deskripsi">
                      URL endpoint OTLP lengkap, termasuk path. Biarkan kosong agar log tetap di
                      container. Query string, host yang tidak ada, atau skema yang tidak dikenal
                      akan menghentikan container saat startup, alih-alih mengekspor tanpa tujuan.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Default">-</td>
                    <td data-label="Deskripsi">
                      Pasangan <code>key=value</code> yang dipisahkan koma dan dikirim pada setiap
                      ekspor, biasanya API key. Nilainya di-percent-decode, sesuai format standar{" "}
                      <code>OTEL_EXPORTER_OTLP_HEADERS</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="Default">
                      <code>info</code>
                    </td>
                    <td data-label="Deskripsi">
                      Salah satu dari <code>debug</code>, <code>info</code>, <code>warn</code>, atau{" "}
                      <code>error</code>. Nilai yang tidak dikenal menggunakan <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="Default">
                      <code>true</code>
                    </td>
                    <td data-label="Deskripsi">
                      Menulis <code>databasus.log</code> bersama data lainnya, merotasi file pada 5
                      MB dan menyimpan 3 file lama. Atur ke <code>false</code> jika platform kamu
                      sudah mengumpulkan stdout.
                    </td>
                  </tr>
                </tbody>
              </table>
              <p>
                Nilai untuk backend umum, masing-masing beserta header autentikasinya. Ganti host,
                region, dan key dengan milik kamu sendiri:
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Backend</th>
                    <th>
                      <code>OPEN_TELEMETRY_URL</code>
                    </th>
                    <th>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>VictoriaLogs</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>http://victoria-logs:9428/insert/opentelemetry/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code>: kredensial yang
                      diharapkan <code>vmauth</code> atau reverse proxy kamu, karena VictoriaLogs
                      sendiri tidak memiliki autentikasi pada path ingest.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code>: sesuai dengan extension{" "}
                      <code>bearertokenauth</code> atau <code>basicauth</code> pada penerima.
                      Collector yang hanya bisa diakses dari dalam jaringan biasanya tidak
                      memerlukannya.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code>: token yang diatur pada input
                      OpenTelemetry (gRPC). Input ini juga bisa menggunakan mTLS sebagai gantinya.
                    </td>
                  </tr>
                  <tr>
                    <td>SigNoz Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpcs://ingest.eu.signoz.cloud:443</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>signoz-ingestion-key=your-ingestion-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Grafana Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>https://otlp-gateway-prod-eu-west-0.grafana.net/otlp/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20&lt;base64&gt;</code>: base64 dari{" "}
                      <code>instance-id:api-token</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Honeycomb</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>https://api.honeycomb.io/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>x-honeycomb-team=your-api-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Datadog Agent</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://datadog-agent:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      Tidak ada. Agent menyimpan API key dan meneruskan data untuk kamu.
                    </td>
                  </tr>
                </tbody>
              </table>
              <p>
                Nilai header di-percent-decode, sehingga spasi setelah <code>Basic</code> atau{" "}
                <code>Bearer</code> ditulis sebagai <code>%20</code> dan koma di dalam nilai sebagai{" "}
                <code>%2C</code>. Basic auth juga bisa langsung dimasukkan ke URL sebagai{" "}
                <code>https://user:password@host/path</code>. Databasus mengubahnya menjadi header
                yang sama dan menyembunyikannya dari log. Melalui <code>http://</code> dan{" "}
                <code>grpc://</code>, key dan password dikirim tanpa enkripsi, jadi gunakan{" "}
                <code>https://</code> atau <code>grpcs://</code> di luar jaringan tepercaya.
              </p>
              <h2 id="analytics-script">Script analitik</h2>
              <p>
                Databasus bisa menyisipkan snippet analitik atau pelacakan kamu sendiri, seperti
                Google Analytics, Plausible, Umami, dan sejenisnya, ke aplikasi. Jika{" "}
                <code>ANALYTICS_SCRIPT</code> diisi, nilainya disisipkan ke{" "}
                <code>&lt;head&gt;</code> halaman saat startup.
              </p>
              <p>
                <strong>Peringatan keamanan:</strong> nilai ini disisipkan apa adanya sebagai HTML
                dan JavaScript mentah, lalu berjalan dengan akses penuh ke antarmuka Databasus di
                browser setiap pengunjung. Hanya isi dengan snippet yang sepenuhnya kamu kendalikan
                dan percayai.
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Variabel</th>
                    <th>Deskripsi</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Deskripsi">
                      Markup <code>&lt;script&gt;</code> khusus yang disisipkan sebelum tag penutup{" "}
                      <code>&lt;/head&gt;</code>. Biarkan kosong jika tidak ingin menambahkan
                      analitik.
                    </td>
                  </tr>
                </tbody>
              </table>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
