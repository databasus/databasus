import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "Cara mengatur notifikasi Microsoft Teams untuk Databasus | Databasus",
  description:
    "Panduan langkah demi langkah mengatur notifikasi Microsoft Teams untuk peringatan backup PostgreSQL dengan Databasus. Pelajari cara membuat webhook Teams dan mengatur notifikasi.",
  keywords: [
    "Databasus",
    "notifikasi Microsoft Teams",
    "backup PostgreSQL",
    "webhook Teams",
    "peringatan backup",
    "notifikasi database",
  ],
  openGraph: {
    title: "Cara mengatur notifikasi Microsoft Teams untuk Databasus | Databasus",
    description:
      "Panduan langkah demi langkah mengatur notifikasi Microsoft Teams untuk peringatan backup PostgreSQL dengan Databasus. Pelajari cara membuat webhook Teams dan mengatur notifikasi.",
    type: "article",
    url: getLocalizedUrl("id", "notifiers/teams"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara mengatur notifikasi Microsoft Teams untuk Databasus | Databasus",
    description:
      "Panduan langkah demi langkah mengatur notifikasi Microsoft Teams untuk peringatan backup PostgreSQL dengan Databasus. Pelajari cara membuat webhook Teams dan mengatur notifikasi.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "notifiers/teams"),
    languages: getLanguageAlternates("notifiers/teams"),
  },
  robots: "index, follow",
};

export default function TeamsPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara mengatur notifikasi Microsoft Teams untuk Databasus",
            description:
              "Panduan langkah demi langkah mengatur notifikasi Microsoft Teams untuk peringatan backup PostgreSQL dengan Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Buka kanal Teams",
                text: "Buka kanal Microsoft Teams tempat kamu ingin menerima notifikasi.",
              },
              {
                "@type": "HowToStep",
                name: "Buka Workflows",
                text: "Buka fitur Workflows di kanal Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Buat workflow baru",
                text: "Buat workflow baru untuk webhook masuk.",
              },
              {
                "@type": "HowToStep",
                name: "Pilih template webhook",
                text: "Pilih template webhook masuk dari pilihan yang tersedia.",
              },
              {
                "@type": "HowToStep",
                name: "Atur webhook",
                text: "Tentukan nama webhook dan kanalnya.",
              },
              {
                "@type": "HowToStep",
                name: "Salin URL webhook",
                text: "Salin URL webhook yang dibuat oleh Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Atur di Databasus",
                text: "Tempel URL webhook ke konfigurasi kanal notifikasi Databasus.",
              },
            ],
          }),
        }}
      />
      <DocsNavbarComponent lang="id" />
      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="id" />
        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="teams-notifications">Notifikasi Microsoft Teams</h1>
              <p className="text-lg text-gray-400">
                Atur Microsoft Teams untuk langsung menerima notifikasi status backup PostgreSQL.
                Peringatan backup berhasil, gagal, dan peringatan lainnya akan masuk langsung ke
                kanal Teams.
              </p>
              <h2 id="setup-teams-webhook">Menyiapkan webhook Teams</h2>
              <h3 id="open-teams-channel">1. Buka kanal Teams</h3>
              <p>
                Buka kanal Microsoft Teams tempat kamu ingin menerima notifikasi backup. Klik tiga
                titik (<strong>•••</strong>) di samping nama kanal.
              </p>
              <Image
                src="/images/notifier-teams/image-01.png"
                alt="Buka kanal Teams"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="access-workflows">2. Buka Workflows</h3>
              <p>
                Dari menu kanal, pilih <strong>&quot;Workflows&quot;</strong> untuk membuka
                integrasi Power Automate.
              </p>
              <Image
                src="/images/notifier-teams/image-02.png"
                alt="Buka Workflows"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="create-new-workflow">3. Buat workflow baru</h3>
              <p>
                Di panel Workflows, klik <strong>&quot;Create&quot;</strong> atau cari template{" "}
                <strong>&quot;Post to a channel when a webhook request is received&quot;</strong>.
              </p>
              <Image
                src="/images/notifier-teams/image-03.png"
                alt="Buat workflow baru"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="select-webhook-template">4. Pilih template webhook</h3>
              <p>
                Pilih template{" "}
                <strong>&quot;Post to a channel when a webhook request is received&quot;</strong>{" "}
                dari pilihan yang tersedia.
              </p>
              <Image
                src="/images/notifier-teams/image-04.png"
                alt="Pilih template webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="configure-webhook">5. Atur webhook</h3>
              <p>
                Atur webhook dengan memberikan nama (misalnya,{" "}
                <strong>&quot;Databasus Backup Notifications&quot;</strong>) dan konfirmasikan kanal
                tempat notifikasi akan dikirim.
              </p>
              <Image
                src="/images/notifier-teams/image-05.png"
                alt="Atur webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="copy-webhook-url">6. Salin URL webhook</h3>
              <p>
                Setelah workflow dibuat, <strong>HTTP POST URL</strong> akan muncul. Salin URL ini
                untuk konfigurasi Databasus.
              </p>
              <Image
                src="/images/notifier-teams/image-06.png"
                alt="Salin URL webhook"
                width={500}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h2 id="configure-databasus">Mengatur di Databasus</h2>
              <h3 id="add-teams-notifier">1. Tambahkan kanal notifikasi Teams</h3>
              <p>
                Di Databasus, buka pengaturan kanal notifikasi dan tambahkan kanal notifikasi
                Microsoft Teams baru. Tempel URL webhook yang disalin dari Teams.
              </p>
              <Image
                src="/images/notifier-teams/image-07.png"
                alt="Atur Teams di Databasus"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />
              <h3 id="test-notification">2. Uji notifikasi</h3>
              <p>
                Setelah webhook diatur, uji notifikasi untuk memastikan semuanya berfungsi dengan
                benar. Pesan pengujian seharusnya masuk ke kanal Teams yang dipilih.
              </p>
              <p>
                Selesai! Kanal Microsoft Teams sekarang siap menerima notifikasi backup PostgreSQL
                dari Databasus.
              </p>
              {/* Navigation */}
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
        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
