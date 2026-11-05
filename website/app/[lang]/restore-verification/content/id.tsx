import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Verifikasi restore - Dokumentasi Databasus",
  description:
    "Pastikan backup database benar-benar bisa di-restore. Databasus mengambil backup terbaru, melakukan restore ke container database sementara, membandingkan hasilnya dengan sumber, dan melaporkan jumlah baris tiap tabel setiap kali dijalankan.",
  keywords: [
    "verifikasi restore",
    "restore database",
    "verifikasi backup",
    "pemulihan bencana",
    "pengujian backup database",
    "agen verifikasi Databasus",
    "integritas backup",
    "uji restore otomatis",
  ],
  openGraph: {
    title: "Verifikasi restore - Dokumentasi Databasus",
    description:
      "Pastikan backup database benar-benar bisa di-restore. Databasus mengambil backup terbaru, melakukan restore ke container database sementara, membandingkan hasilnya dengan sumber, dan melaporkan jumlah baris tiap tabel setiap kali dijalankan.",
    type: "article",
    url: getLocalizedUrl("id", "restore-verification"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Verifikasi restore - Dokumentasi Databasus",
    description:
      "Pastikan backup database benar-benar bisa di-restore. Databasus mengambil backup terbaru, melakukan restore ke container database sementara, membandingkan hasilnya dengan sumber, dan melaporkan jumlah baris tiap tabel setiap kali dijalankan.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "restore-verification"),
    languages: getLanguageAlternates("restore-verification"),
  },
  robots: "index, follow",
};

export default function RestoreVerificationPage() {
  const downloadAgent = `curl -L -o verification-agent "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" \\
  && chmod +x verification-agent`;
  const startAgent = `./verification-agent start \\
  --databasus-host=https://your-databasus-host \\
  --agent-id=<AGENT_ID> \\
  --token=<TOKEN> \\
  --max-cpu=2 \\
  --max-ram-mb=2048 \\
  --max-disk-gb=20 \\
  --max-concurrent-jobs=1`;

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Verifikasi restore - Dokumentasi Databasus",
            description:
              "Pastikan backup database benar-benar bisa di-restore. Databasus mengambil backup terbaru, melakukan restore ke container database sementara, membandingkan hasilnya dengan sumber, dan melaporkan jumlah baris tiap tabel setiap kali dijalankan.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara menyiapkan verifikasi restore di Databasus",
            description:
              "Panduan langkah demi langkah untuk mendaftarkan agen verifikasi, menjalankannya di server, dan mengatur verifikasi restore terjadwal.",
            step: [
              {
                "@type": "HowToStep",
                name: "Buat agen verifikasi di antarmuka",
                text: "Buka Settings → Verification agents, klik Create verification agent, beri nama, lalu salin token dan ID agen dari dialog.",
              },
              {
                "@type": "HowToStep",
                name: "Unduh binary agen",
                text: "Jalankan perintah curl pada host tempat verifikasi akan berjalan. Pilih amd64 atau arm64 sesuai arsitekturnya.",
              },
              {
                "@type": "HowToStep",
                name: "Jalankan agen",
                text: "Jalankan agen dengan --agent-id, --token, dan anggaran resource (--max-cpu, --max-ram-mb, --max-disk-gb, --max-concurrent-jobs).",
              },
              {
                "@type": "HowToStep",
                name: "Jadwalkan verifikasi",
                text: "Buka pengaturan verifikasi database, aktifkan Scheduled verification, lalu pilih interval (After backup, Hourly, Daily, Weekly, Monthly atau Cron).",
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
              <h1 id="restore-verification">Verifikasi restore</h1>
              <p className="text-lg text-gray-400">
                Backup yang selesai tanpa error belum tentu bisa di-restore. Satu-satunya bukti
                nyata adalah mencoba restore. Databasus melakukannya secara terjadwal:
              </p>
              <ul>
                <li>mengambil backup terbaru</li>
                <li>menjalankan restore ke container database sementara</li>
                <li>
                  memeriksa kewajaran hasil restore dengan membandingkannya terhadap database sumber
                </li>
                <li>menghapus container</li>
                <li>melaporkan hasilnya</li>
              </ul>
              <img
                src="/images/restore-verification/verified-backups.png"
                alt="Tab Verified backups"
                loading="lazy"
              />
              <img
                className="mt-5"
                src="/images/restore-verification/verifications.png"
                alt="Tab Verifications"
                loading="lazy"
              />
              <h2 id="what-is-verification-agent">Apa itu agen verifikasi?</h2>
              <p>
                Agen verifikasi adalah binary Go kecil yang dijalankan pada mesin yang kamu
                kendalikan. Mesin apa pun dengan CPU, RAM, dan ruang disk tersisa bisa digunakan.
                Agen mendaftar ke Databasus, mengambil tugas verifikasi dari antrean, menjalankannya
                secara lokal, lalu mengirimkan hasilnya kembali.
              </p>
              <h3 id="what-you-need">Yang perlu disiapkan</h3>
              <ul>
                <li>Host dengan akses HTTPS keluar ke URL Databasus milikmu.</li>
                <li>
                  Docker tersedia pada host tersebut. Agen membuat container database sementara
                  dengan versi mayor yang sesuai untuk setiap tugas.
                </li>
                <li>
                  Kapasitas disk untuk setiap tugas verifikasi yang mencakup{" "}
                  <strong>ukuran file backup</strong>, <strong>ukuran mentah database</strong>, dan{" "}
                  <strong>ruang cadangan</strong> tambahan.
                </li>
                <li>
                  Setidaknya 1 core CPU dan RAM 512 MB tersedia untuk setiap tugas yang berjalan
                  bersamaan.
                </li>
              </ul>
              <h3 id="why-not-just-checksums">Mengapa checksum saja tidak cukup?</h3>
              <p>
                Checksum dan exit code bisa mendeteksi beberapa jenis kegagalan, tetapi sama sekali
                tidak mendeteksi yang lain:
              </p>
              <ul>
                <li>
                  <strong>Checksum</strong> mendeteksi kerusakan bit pada file arsip, tetapi tidak
                  menunjukkan apakah dump itu sendiri lengkap atau valid secara semantik.
                </li>
                <li>
                  <strong>Exit code dump</strong> menunjukkan bahwa perintah dump berjalan. Ini
                  tidak mendeteksi role yang tidak memiliki izin baca atas objek tertentu, extension
                  yang hilang di sumber, atau ketidakcocokan tablespace. Semua itu bisa menyebabkan
                  objek dilewati atau dibuang tanpa pemberitahuan.
                </li>
                <li>
                  <strong>Verifikasi restore</strong> benar-benar memproses arsip menggunakan tool
                  restore bawaan database dan menghitung baris per tabel. Inilah satu-satunya
                  pemeriksaan yang mendeteksi semua masalah di atas. Jika backup tidak bisa
                  di-restore, kamu mengetahuinya sebelum membutuhkannya, bukan saat bencana terjadi.
                </li>
              </ul>
              <h2 id="configuration">Konfigurasi</h2>
              <h3 id="create-on-ui">Buat agen di antarmuka</h3>
              <p>
                Buka <strong>Settings → Verification agents</strong>, lalu klik{" "}
                <strong>Create verification agent</strong>. Pilih nama yang jelas seperti{" "}
                <code>staging-verifier</code> atau <code>eu-west-host-1</code>. Dialog berikutnya
                menampilkan <strong>token</strong> dan <strong>ID</strong> agen.
              </p>
              <p>
                Token ditampilkan <strong>hanya satu kali</strong>. Salin sebelum menutup dialog.
                Jika hilang, gunakan tindakan <strong>Rotate token</strong> pada baris agen untuk
                menerbitkan token baru. Token lama berhenti berfungsi pada heartbeat agen
                berikutnya. Dialog selanjutnya menampilkan perintah instalasi sesuai arsitektur
                server, sama dengan perintah yang dijelaskan di bawah.
              </p>
              <h3 id="launch">Jalankan agen di server</h3>
              <p>
                Masuk melalui SSH ke mesin yang akan menjalankan verifikasi. Unduh binary agen
                terlebih dahulu. Ganti <code>https://your-databasus-host</code> dengan URL Databasus
                milikmu, dan ganti <code>amd64</code> dengan <code>arm64</code> jika server
                menggunakan ARM:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={downloadAgent} lang="id" />
                </div>
              </div>
              <p>
                Lalu jalankan agen. ID dan token agen berasal dari dialog pada langkah sebelumnya:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startAgent} lang="id" />
                </div>
              </div>
              <p>
                <code>start</code> menjalankan agen sebagai daemon dan menyimpan flag ke{" "}
                <code>databasus-verification.json</code> di direktori kerja. Saat dijalankan ulang,
                cukup gunakan <code>./verification-agent start</code> tanpa flag. Log ditulis ke{" "}
                <code>databasus-verification.log</code> di samping binary.
              </p>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] mb-3 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    Host Databasus wajib menggunakan <code>https://</code>. HTTP biasa hanya
                    diizinkan jika menambahkan <code>--allow-insecure-http</code>, dan hanya
                    ditujukan untuk pengujian lokal. Jangan pernah mengekspos agen production
                    melalui HTTP tanpa enkripsi.
                  </p>
                </div>
              </div>
              <p>
                Keempat flag <code>--max-*</code> menetapkan <strong>anggaran</strong>, bukan
                alokasi per tugas. Agen melaporkannya ke Databasus pada setiap heartbeat, lalu
                Databasus membaginya ke seluruh tugas bersamaan yang diizinkan. Dengan{" "}
                <code>--max-cpu=2 --max-ram-mb=2048 --max-concurrent-jobs=1</code>, satu tugas
                mendapat seluruh 2 CPU dan RAM 2 GB. Dengan <code>--max-concurrent-jobs=2</code>,
                setiap tugas mendapat 1 CPU dan 1 GB. Batas minimumnya adalah 1 CPU dan 512 MB per
                tugas. Jika anggaran tidak mencukupi, agen melaporkan kapasitas tugas bersamaan yang
                lebih rendah. Anggaran disk paling mudah keliru: setiap tugas memerlukan ruang untuk{" "}
                <strong>ukuran file backup</strong>, <strong>ukuran mentah database</strong>, serta{" "}
                <strong>ruang cadangan hingga 5 GB</strong>. Jadi, atur <code>--max-disk-gb</code>{" "}
                cukup jauh di atas total kebutuhan itu untuk database terbesar.
              </p>
              <h3 id="manage">Mengelola agen</h3>
              <p>Binary yang sama menyediakan empat subcommand:</p>
              <ul>
                <li>
                  <code>./verification-agent status</code> menampilkan apakah daemon berjalan dan
                  tugas apa yang sedang dikerjakan.
                </li>
                <li>
                  <code>./verification-agent stop</code> menghentikan daemon. Verifikasi yang sedang
                  berjalan dilaporkan ke Databasus sebagai gagal dan dimasukkan kembali ke antrean.
                </li>
                <li>
                  <code>./verification-agent start</code> menjalankan ulang daemon. Flag disimpan
                  sejak pertama kali dijalankan. Sertakan <code>--token=&lt;NEW&gt;</code> setelah
                  rotasi untuk memperbarui token tersimpan.
                </li>
                <li>
                  <code>./verification-agent run</code> berjalan di foreground, bukan sebagai
                  daemon. Gunakan saat menjalankan agen dalam unit systemd atau container Docker
                  karena supervisor tersebut mengharapkan proses tidak melakukan fork ke background.
                </li>
              </ul>
              <p>
                Halaman Settings menampilkan tiga tindakan berikon pada setiap baris agen: melihat
                kembali perintah instalasi tanpa menampilkan token, merotasi token, dan menghapus
                agen. Penghapusan aman. Verifikasi yang sedang ditugaskan ke agen itu dikembalikan
                ke antrean dan diambil oleh agen lain jika tersedia.
              </p>
              <h2 id="schedules-and-notifications">Jadwal dan notifikasi</h2>
              <p>
                Verifikasi restore diatur per database. Buka pengaturan verifikasi database,
                aktifkan <strong>Scheduled verification</strong>, lalu pilih interval.
              </p>
              <h3 id="interval-options">Pilihan interval</h3>
              <ul>
                <li>
                  <strong>After backup</strong> memberikan jaminan terkuat: setiap backup yang
                  berhasil diverifikasi begitu selesai.
                </li>
                <li>
                  <strong>Hourly, daily, weekly, monthly</strong> memilih frekuensi dan waktu
                  pelaksanaan.
                </li>
                <li>
                  <strong>Cron</strong> menggunakan ekspresi cron UTC untuk jadwal di luar preset.
                  Contoh: <code>0 4 * * 0</code> (setiap Minggu pukul 4:00 AM UTC) dan{" "}
                  <code>0 */6 * * *</code> (setiap enam jam).
                </li>
              </ul>
              <h3 id="how-the-queue-works">Cara antrean menangani &quot;After backup&quot;</h3>
              <p>
                Verifikasi biasanya lebih lambat daripada backup yang diperiksanya. Jika backup baru
                masuk lebih cepat daripada verifikasi selesai, antrean akan terus bertambah.
                Databasus mencegahnya dengan{" "}
                <strong>
                  membatalkan verifikasi yang masih menunggu untuk database yang sama setiap kali
                  backup baru masuk
                </strong>
                . Hanya backup terbaru yang menunggu di antrean. Ini pilihan yang disengaja: lebih
                baik melewati verifikasi backup lama daripada menghabiskan berjam-jam memeriksa
                backup yang tidak akan dipakai untuk restore.
              </p>
              <h3 id="manual-runs">Menjalankan secara manual</h3>
              <p>
                Kamu juga bisa menjalankan verifikasi satu kali dari tab{" "}
                <strong>Restore verifications</strong> pada database tanpa mengubah jadwal. Ini
                berguna untuk memeriksa backup tertentu atau menguji agen baru secara end-to-end
                sebelum mengandalkannya untuk beban terjadwal.
              </p>
              <h3 id="notifications">Notifikasi</h3>
              <p>
                Hasil berhasil dan gagal bisa dikirim melalui kanal notifikasi apa pun yang sudah
                dikonfigurasi untuk database. Kedua checkbox, <strong>Verification success</strong>{" "}
                dan <strong>Verification failed</strong>, berdiri sendiri. Sebagian besar tim hanya
                mengaktifkan notifikasi kegagalan agar tidak kewalahan menerima notifikasi. Lihat{" "}
                <a
                  href={getLocalizedHref("id", "notifiers")}
                  className="text-blue-400 hover:text-blue-300"
                >
                  dokumentasi kanal notifikasi
                </a>{" "}
                untuk menghubungkan Slack, Microsoft Teams, Discord, email, dan lainnya.
              </p>
              <h3 id="results">Membaca hasil</h3>
              <p>
                Setiap percobaan verifikasi muncul sebagai satu baris pada tab{" "}
                <strong>Restore verifications</strong> di database. Statusnya berupa{" "}
                <strong>Pending</strong>, <strong>Running</strong>, <strong>Successful</strong>,{" "}
                <strong>Failed</strong>, atau <strong>Canceled</strong>. Klik baris untuk membuka
                panel berisi linimasa lengkap, exit code restore, ukuran database hasil restore,
                jumlah schema dan tabel, serta rincian jumlah baris tiap tabel. Untuk verifikasi
                gagal, pesan kegagalan ditampilkan di bagian atas panel.
              </p>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
