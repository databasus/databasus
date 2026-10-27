import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Kanal notifikasi - Dokumentasi Databasus",
  description:
    "Daftar kanal notifikasi yang didukung untuk peringatan backup Databasus, termasuk Slack, Discord, Telegram, Microsoft Teams, Email, dan Webhook.",
  keywords: [
    "kanal notifikasi Databasus",
    "notifikasi backup",
    "notifikasi Slack",
    "peringatan Discord",
    "notifikasi Telegram",
    "notifikasi Teams",
    "peringatan Email",
    "notifikasi Webhook",
  ],
  openGraph: {
    title: "Kanal notifikasi - Dokumentasi Databasus",
    description:
      "Daftar kanal notifikasi yang didukung untuk peringatan backup Databasus, termasuk Slack, Discord, Telegram, Microsoft Teams, Email, dan Webhook.",
    type: "article",
    url: getLocalizedUrl("id", "notifiers"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Kanal notifikasi - Dokumentasi Databasus",
    description:
      "Daftar kanal notifikasi yang didukung untuk peringatan backup Databasus, termasuk Slack, Discord, Telegram, Microsoft Teams, Email, dan Webhook.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "notifiers"),
    languages: getLanguageAlternates("notifiers"),
  },
  robots: "index, follow",
};

export default function NotifiersPage() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Kanal notifikasi - Dokumentasi Databasus",
            description:
              "Daftar kanal notifikasi yang didukung untuk peringatan backup Databasus, termasuk Slack, Discord, Telegram, Microsoft Teams, Email, dan Webhook.",
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
      <DocsNavbarComponent lang="id" />
      <div className="flex min-h-screen bg-[#0F1115]">
        <DocsSidebarComponent lang="id" />
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="notifiers">Kanal notifikasi</h1>
              <p className="text-lg text-gray-400">
                Databasus mendukung berbagai kanal notifikasi untuk memberi kabar tentang status
                backup PostgreSQL. Terima peringatan langsung saat backup berhasil, gagal, atau
                mengalami masalah.
              </p>
              <h2 id="supported-notifiers">Kanal notifikasi yang didukung</h2>
              <ul>
                <li>
                  <a
                    href={getLocalizedHref("id", "notifiers/slack")}
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </a>{" "}
                  - Kirim notifikasi ke kanal Slack melalui webhook
                </li>
                <li>
                  <strong>Discord</strong> - Kirim peringatan backup ke kanal Discord
                </li>
                <li>
                  <strong>Telegram</strong> - Terima notifikasi melalui bot Telegram
                </li>
                <li>
                  <a
                    href={getLocalizedHref("id", "notifiers/teams")}
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </a>{" "}
                  - Beri tahu tim melalui kanal Microsoft Teams
                </li>
                <li>
                  <strong>Email</strong> - Kirim notifikasi email tentang aktivitas backup
                </li>
                <li>
                  <strong>Webhook</strong> - Integrasi webhook khusus untuk layanan apa pun
                </li>
              </ul>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
