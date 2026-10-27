import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Pengelolaan Akses - Dokumentasi Databasus",
  description:
    "Pelajari cara mengelola akses, peran, dan izin di Databasus. Atur siapa yang bisa mendaftar, membuat ruang kerja, dan mengelola database dengan peran di tingkat ruang kerja dan sistem.",
  keywords: [
    "pengelolaan akses Databasus",
    "peran pengguna",
    "izin ruang kerja",
    "log audit",
    "keamanan backup PostgreSQL",
    "kolaborasi tim",
    "kontrol akses",
    "pengelolaan ruang kerja",
  ],
  openGraph: {
    title: "Pengelolaan Akses - Dokumentasi Databasus",
    description:
      "Pelajari cara mengelola akses, peran, dan izin di Databasus. Atur siapa yang bisa mendaftar, membuat ruang kerja, dan mengelola database dengan peran di tingkat ruang kerja dan sistem.",
    type: "article",
    url: getLocalizedUrl("id", "access-management"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Pengelolaan Akses - Dokumentasi Databasus",
    description:
      "Pelajari cara mengelola akses, peran, dan izin di Databasus. Atur siapa yang bisa mendaftar, membuat ruang kerja, dan mengelola database dengan peran di tingkat ruang kerja dan sistem.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "access-management"),
    languages: getLanguageAlternates("access-management"),
  },
  robots: "index, follow",
};

export default function AccessManagementPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Pengelolaan Akses - Dokumentasi Databasus",
            description:
              "Pelajari cara mengelola akses, peran, dan izin di Databasus. Atur siapa yang bisa mendaftar, membuat ruang kerja, dan mengelola database dengan peran di tingkat ruang kerja dan sistem.",
            author: {
              "@type": "Organization",
              name: "Databasus",
            },
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

      <DocsNavbarComponent lang="id" />

      <div className="flex min-h-screen bg-[#0F1115]">
        <DocsSidebarComponent lang="id" />

        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="settings">Pengaturan</h1>

              <p>
                Databasus cocok untuk pengguna individu maupun tim. Bagian ini membahas pengelolaan
                akses untuk tim.{" "}
                <strong>Kalau kamu satu-satunya pengguna di instance Databasus ini</strong>, bagian
                ini bisa dilewati.
              </p>

              <p>Pengaturan Databasus tidak banyak. Yang bisa diatur hanya:</p>

              <ul>
                <li>Siapa yang bisa mendaftar di instance Databasus kamu</li>
                <li>Siapa yang bisa membuat ruang kerja</li>
                <li>
                  Siapa yang bisa mengelola database, kanal notifikasi, dan penyimpanan dalam ruang
                  kerja
                </li>
              </ul>

              <h2 id="workspaces">Ruang kerja</h2>

              <p>
                Ruang kerja adalah tempat untuk{" "}
                <strong>mengelompokkan database, kanal notifikasi, dan penyimpanan</strong>. Kamu
                bisa menambahkan anggota ke ruang kerja dan membuat lebih dari satu ruang kerja.
              </p>

              <p>Akses bisa diatur per ruang kerja. Misalnya:</p>

              <ul>
                <li>
                  tim DevOps bertanggung jawab atas 10 database proyek, sehingga beberapa pengguna
                  bergabung dalam satu ruang kerja;
                </li>
                <li>
                  ada 3 proyek dengan database dan penyimpanan berbeda, sehingga ada beberapa ruang
                  kerja dengan pengguna yang berbeda;
                </li>
                <li>
                  ada 5 database terpisah yang masing-masing diakses pengguna berbeda, sehingga
                  pengguna A punya akses ke DB1, pengguna B ke DB2, pengguna C ke DB3, dan
                  seterusnya.
                </li>
              </ul>

              <img
                src="/images/access-management/users.png"
                alt="Ruang kerja"
                width={550}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Kalau pengguna diizinkan mendaftar di Databasus dan membuat ruang kerja sendiri
                (lihat <a href="#global-settings">pengaturan global</a>), mereka bisa membuat ruang
                kerja masing-masing.
              </p>

              <p>
                <strong>
                  Pengguna tidak bisa melihat ruang kerja selain miliknya sampai diundang untuk
                  bergabung.
                </strong>
              </p>

              <h2 id="audit-logs">Log audit</h2>

              <p>
                Log audit mencatat tindakan pengguna. Catatan ini membantu melacak perubahan dan
                tindakan pengguna serta mendeteksi aktivitas mencurigakan.
              </p>

              <p>Misalnya:</p>

              <ul>
                <li>pengguna membuat database baru</li>
                <li>pengguna menghapus database</li>
                <li>pengguna memulai backup baru</li>
                <li>pengguna mengunduh backup</li>
                <li>pengguna membuat kanal notifikasi baru</li>
                <li>pengguna membuat ruang kerja</li>
                <li>pengguna menghapus ruang kerja</li>
                <li>dan sebagainya.</li>
              </ul>

              <p>Log audit bisa difilter:</p>

              <ul>
                <li>per ruang kerja;</li>
                <li>per pengguna di beberapa ruang kerja;</li>
              </ul>

              <img
                src="/images/access-management/audit-logs.png"
                alt="Log audit"
                width={1000}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="user-roles">Peran pengguna</h2>

              <p>
                Semua pengguna Databasus memiliki peran <u>di tingkat sistem</u>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fitur</th>
                    <th>Admin</th>
                    <th>Member</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Mengelola semua pengaturan dan pengguna</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">❌</td>
                  </tr>
                  <tr>
                    <td>Membuat ruang kerja</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">✅ (jika diizinkan di pengaturan)</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Biasanya sistem hanya memiliki satu administrator, yaitu akun pertama yang dibuat di
                instance tersebut dan bertugas mengelolanya.
              </p>

              <p>
                <u>Di dalam ruang kerja</u> juga ada beberapa peran:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fitur</th>
                    <th>Viewer</th>
                    <th>Member</th>
                    <th>Admin</th>
                    <th>Owner</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Melihat database, kanal notifikasi, penyimpanan</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Memulai dan mengunduh backup</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Mengelola database, kanal notifikasi, penyimpanan</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Mengelola pengguna</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Mengelola admin</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">❌</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Perlu diingat: <strong>data sensitif</strong> (password, token, dan sebagainya)
                milik database, penyimpanan, dan kanal notifikasi{" "}
                <strong>selalu disembunyikan dari semua pengguna</strong>. Tidak ada yang bisa
                melihat secret setelah dibuat.
              </p>

              <h2 id="global-settings">Pengaturan global</h2>

              <p>Pengaturan global memiliki 3 opsi:</p>

              <ol>
                <li>
                  <strong>Allow external registrations</strong> - secara default, siapa saja bisa
                  mendaftar di Databasus kamu. Namun, mereka belum punya akses ke ruang kerja mana
                  pun sampai diundang atau membuat ruang kerja sendiri.
                  <br />
                  <br />
                  Kalau hanya pengguna yang diundang yang boleh mendaftar, nonaktifkan opsi ini.
                  Formulir pendaftaran akan ditutup sampai pengguna diundang ke salah satu ruang
                  kerja.
                  <br />
                  <br />
                  Untuk mengundang pengguna ke ruang kerja, klik &quot;Add user&quot; lalu masukkan
                  alamat email. Setelah itu, pengguna dengan alamat tersebut bisa menyelesaikan
                  pendaftaran.
                </li>
                <li>
                  <strong>Allow member invitations</strong> - pengaturan ini diperlukan saat
                  pendaftaran dari luar dinonaktifkan.
                  <br />
                  <br />
                  Misalnya, sudah ada beberapa pengguna yang kamu percaya, seperti anggota tim.
                  Kalau mereka perlu mengundang pengguna lain ke Databasus, aktifkan opsi ini.
                  Mereka bisa mengundang pengguna lain untuk bergabung ke ruang kerja.
                  <br />
                  <br />
                  Jika dinonaktifkan, hanya admin yang bisa mengundang pengguna.
                </li>
                <li>
                  <strong>Allow member workspace creation</strong> - secara default, semua anggota
                  bisa membuat ruang kerja sendiri. Kalau hanya admin yang boleh membuat ruang
                  kerja, nonaktifkan opsi ini.
                </li>
              </ol>
            </article>
          </div>
        </main>

        <DocTableOfContentComponent />
      </div>
    </>
  );
}
