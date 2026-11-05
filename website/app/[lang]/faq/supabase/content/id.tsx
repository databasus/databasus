import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Cara backup Supabase dengan Databasus | Databasus",
  description:
    "Pelajari cara backup database PostgreSQL Supabase dengan Databasus. Panduan langkah demi langkah untuk mengatur session pooler atau alamat IPv4 bagi backup Supabase.",
  keywords: [
    "Databasus",
    "backup Supabase",
    "backup PostgreSQL Supabase",
    "backup database Supabase",
    "session pooler Supabase",
    "IPv4 Supabase",
    "backup PostgreSQL",
    "backup database",
  ],
  openGraph: {
    title: "Cara backup Supabase dengan Databasus | Databasus",
    description:
      "Pelajari cara backup database PostgreSQL Supabase dengan Databasus. Panduan langkah demi langkah untuk mengatur session pooler atau alamat IPv4 bagi backup Supabase.",
    type: "article",
    url: getLocalizedUrl("id", "faq/supabase"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara backup Supabase dengan Databasus | Databasus",
    description:
      "Pelajari cara backup database PostgreSQL Supabase dengan Databasus. Panduan langkah demi langkah untuk mengatur session pooler atau alamat IPv4 bagi backup Supabase.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "faq/supabase"),
    languages: getLanguageAlternates("faq/supabase"),
  },
  robots: "index, follow",
};

export default function SupabasePage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara backup Supabase dengan Databasus",
            description:
              "Panduan langkah demi langkah untuk backup database PostgreSQL Supabase dengan Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Ambil detail koneksi dari Supabase",
                text: "Buka pengaturan proyek Supabase dan cari detail koneksi database.",
              },
              {
                "@type": "HowToStep",
                name: "Gunakan Session Pooler dengan IPv4",
                text: "Salin connection string Session Pooler dan pastikan 'Use IPv4 Address' aktif.",
              },
              {
                "@type": "HowToStep",
                name: "Atur Databasus",
                text: "Masukkan detail koneksi Supabase di Databasus untuk mulai membuat backup database.",
              },
              {
                "@type": "HowToStep",
                name: "Pahami batasan schema",
                text: "Secara default, hanya schema public yang di-backup karena akses ke schema Supabase lainnya dibatasi.",
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
              <h1 id="supabase-backup">Cara backup Supabase</h1>
              <p className="text-lg text-gray-400">
                Databasus mendukung backup database PostgreSQL Supabase. Syarat utamanya adalah
                memakai alamat IPv4 untuk terhubung ke instance Supabase.
              </p>
              <h2 id="connection-options">Pilihan koneksi</h2>
              <p>Ada dua cara menghubungkan Databasus ke database Supabase:</p>
              <ol>
                <li>
                  <strong>Gunakan Session Pooler dengan IPv4</strong> (Disarankan): Opsi gratis yang
                  tersedia di semua proyek Supabase
                </li>
                <li>
                  <strong>Beli IPv4 Add-on</strong>: Opsi koneksi langsung dari Supabase
                </li>
              </ol>
              <h2 id="session-pooler">Opsi 1: Gunakan Session Pooler (Disarankan)</h2>
              <p>
                Session Pooler menyediakan alamat IPv4 untuk koneksi database Supabase tanpa biaya
                tambahan. Berikut cara mengaturnya:
              </p>
              <h3 id="step-1">1. Cari koneksi pooler</h3>
              <p>
                Buka proyek Supabase, lalu pilih <strong>Project Settings</strong> →{" "}
                <strong>Database</strong>. Gulir ke bagian <strong>Connection string</strong> dan
                pilih mode <strong>Session pooler</strong>.
              </p>
              <img
                src="/images/faq/supabase/image-1.png"
                alt="Pilih mode Session pooler di Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
              <h3 id="step-2">2. Salin detail koneksi</h3>
              <p>
                Salin detail koneksi dan gunakan di Databasus saat menambahkan database. Lihat
                screenshot untuk membedakan tiap detail koneksi.
              </p>
              <img
                src="/images/faq/supabase/image-2.png"
                alt="Aktifkan tombol IPv4 Address di Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
              <h2 id="ipv4-addon">Opsi 2: Beli IPv4 Add-on</h2>
              <p>
                Supabase menawarkan add-on IPv4 berbayar yang menyediakan alamat IPv4 khusus untuk
                database. Opsi ini memberikan koneksi langsung tanpa melewati connection pooler.
              </p>
              <p>Untuk mengaktifkan opsi ini:</p>
              <ol>
                <li>Buka dashboard proyek Supabase</li>
                <li>
                  Pilih <strong>Project Settings</strong> → <strong>Add-ons</strong>
                </li>
                <li>
                  Aktifkan add-on <strong>IPv4</strong>
                </li>
                <li>Gunakan detail koneksi database langsung di Databasus</li>
              </ol>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Tips:</strong> Untuk sebagian besar
                  kebutuhan, opsi Session Pooler gratis dengan IPv4 sudah bekerja dengan baik untuk
                  backup. Add-on IPv4 berbayar hanya diperlukan jika kamu membutuhkan koneksi
                  langsung untuk alasan lain.
                </p>
              </div>
              <h2 id="default-schema">Batasan schema default</h2>
              <p>
                Secara default, Databasus hanya membuat backup schema <code>public</code> saat
                memakai database Supabase. Ini karena Supabase membatasi akses ke schema lain
                (seperti <code>auth</code>, <code>storage</code>, dan <code>realtime</code>) demi
                keamanan.
              </p>
              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-blue-400">ℹ️ Catatan:</strong> Schema <code>public</code>{" "}
                  berisi data aplikasi dan tabel buatan kamu. Schema kelolaan Supabase seperti{" "}
                  <code>auth</code> dan <code>storage</code> dilindungi dan dikelola oleh Supabase
                  sendiri.
                </p>
              </div>
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("id", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Kembali ke FAQ
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
