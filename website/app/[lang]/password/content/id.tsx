import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Reset kata sandi - Dokumentasi Databasus",
  description:
    "Pelajari cara mereset kata sandi pengguna, melihat daftar administrator, dan membuat akun di Databasus melalui command-line bawaan. Pemulihan kata sandi yang cepat dan aman untuk sistem backup PostgreSQL.",
  keywords: [
    "reset kata sandi Databasus",
    "reset kata sandi pengguna",
    "kata sandi backup PostgreSQL",
    "pemulihan kata sandi Docker",
    "pemulihan kata sandi",
    "autentikasi Databasus",
  ],
  openGraph: {
    title: "Reset kata sandi - Dokumentasi Databasus",
    description:
      "Pelajari cara mereset kata sandi pengguna, melihat daftar administrator, dan membuat akun di Databasus melalui command-line bawaan. Pemulihan kata sandi yang cepat dan aman untuk sistem backup PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("id", "password"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Reset kata sandi - Dokumentasi Databasus",
    description:
      "Pelajari cara mereset kata sandi pengguna, melihat daftar administrator, dan membuat akun di Databasus melalui command-line bawaan. Pemulihan kata sandi yang cepat dan aman untuk sistem backup PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "password"),
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"`;
  const listAdminsCommand = `docker exec -it databasus ./main --list-admins`;
  const createUserCommand = `docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Reset kata sandi - Dokumentasi Databasus",
            description:
              "Pelajari cara mereset kata sandi pengguna di Databasus melalui command-line bawaan.",
            author: { "@type": "Organization", name: "Databasus" },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: { "@type": "ImageObject", url: "https://databasus.com/logo.svg" },
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
            name: "Cara mereset kata sandi pengguna Databasus",
            description:
              "Panduan langkah demi langkah untuk mereset kata sandi pengguna di Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Jalankan perintah reset kata sandi",
                text: "Jalankan perintah docker exec dengan kata sandi baru dan email pengguna.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Gunakan docker exec untuk menjalankan perintah reset kata sandi di dalam container Databasus",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Periksa perubahan kata sandi",
                text: "Masuk ke Databasus dengan kata sandi baru untuk memastikan perubahan berhasil.",
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
              <h1 id="reset-password">Reset kata sandi pengguna</h1>
              <h2 id="reset-password-command">Perintah reset kata sandi</h2>
              <p>
                Untuk mereset kata sandi pengguna, jalankan perintah berikut di server tempat
                Databasus berjalan:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetPasswordCommand} lang="id" />
                </div>
              </div>
              <h2 id="parameters">Parameter</h2>
              <p>Perintah ini menerima parameter berikut:</p>
              <ul>
                <li>
                  <strong>--new-password</strong>: Kata sandi baru, minimal 8 karakter. Pastikan
                  kata sandinya kuat dan berisi gabungan huruf, angka, dan karakter khusus.
                </li>
                <li>
                  <strong>--email</strong>: Alamat email akun yang kata sandinya ingin kamu reset
                  (misalnya, <code>owner@example.com</code>). Instance yang dibuat sebelum akun
                  pertama menjadi pengelolanya masih memakai alamat sementara <code>admin</code>{" "}
                  sampai pemiliknya menggantinya. Untuk instance seperti itu, gunakan{" "}
                  <code>admin</code>.
                </li>
              </ul>
              <h2 id="list-admins">Lihat daftar akun administrator</h2>
              <p>
                Kalau lupa alamat mana yang mengelola instance, tampilkan daftar akun administrator
                di server tempat Databasus berjalan:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{listAdminsCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={listAdminsCommand} lang="id" />
                </div>
              </div>
              <p>
                Output menampilkan semua akun administrator beserta alamat email, nama tampilan,
                tanggal pembuatan, dan status aktifnya, serta menandai akun yang dikenali instance
                sebagai pengelolanya. Kata sandi, hash kata sandi, dan token tidak ditampilkan. Jika
                belum ada akun yang dibuat, output menyatakan bahwa instance belum memiliki
                administrator.
              </p>
              <h2 id="create-user">Buat akun dari console</h2>
              <p>
                Untuk menyiapkan akun tanpa mendaftar lewat browser, misalnya dari script
                deployment, buat akun di server tempat Databasus berjalan:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createUserCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={createUserCommand} lang="id" />
                </div>
              </div>
              <p>Perintah ini menerima parameter berikut:</p>
              <ul>
                <li>
                  <strong>--email</strong>: Alamat email akun baru. Alamatnya harus valid.
                </li>
                <li>
                  <strong>--password</strong>: Kata sandi akun baru, minimal 8 karakter.
                </li>
                <li>
                  <strong>--role</strong>: <code>admin</code> atau <code>member</code>.
                </li>
                <li>
                  <strong>--name</strong>: Nama tampilan opsional. Jika tidak diisi, nama akun
                  diambil dari bagian alamat sebelum <code>@</code>.
                </li>
              </ul>
              <p>
                Akun langsung aktif dan bisa masuk dengan alamat serta kata sandi tersebut, baik
                pendaftaran terbuka diaktifkan maupun tidak. Pada instance yang belum memiliki akun,
                akun yang dibuat akan mengelola instance, jadi akun pertama harus memakai{" "}
                <code>--role=admin</code>. Administrator yang dibuat setelahnya dengan cara ini
                adalah administrator biasa.
              </p>
              <p>
                Menjalankan kembali perintah untuk akun yang sudah ada dengan peran yang sama akan
                berhasil tanpa mengubah apa pun, sehingga deployment bisa menjalankannya pada setiap
                rollout. Kata sandi juga tidak berubah. Untuk menggantinya, gunakan perintah reset{" "}
                <code>--new-password</code> di atas. Jika alamat itu milik akun dengan peran lain,
                akun nonaktif atau undangan, atau akun yang masuk lewat Google atau GitHub, perintah
                akan gagal tanpa mengubah apa pun.
              </p>
              <h2 id="disable-two-factor">Nonaktifkan kewajiban kode saat masuk</h2>
              <p>
                Instance bisa mewajibkan kode enam digit yang dikirim lewat email saat masuk, selain
                kata sandi. Jika server email berhenti mengirimkan kode, tidak ada yang bisa masuk
                dengan kata sandi. Mereset kata sandi saja tidak cukup. Nonaktifkan faktor kedua di
                server tempat Databasus berjalan:
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
                Perintah melaporkan apakah ada perubahan, tetap berhasil jika faktor kedua sudah
                nonaktif, dan mencatat perubahan di log audit. Setelah itu, proses masuk berikutnya
                dengan kata sandi tidak memerlukan kode. Administrator bisa mengaktifkan kembali
                pengaturan ini setelah email berfungsi lagi.
              </p>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
