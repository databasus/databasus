import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Notifiers - Databasus Documentation",
  description:
    "List of supported notification channels for Databasus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
  keywords: [
    "Databasus notifiers",
    "backup notifications",
    "Slack notifications",
    "Discord alerts",
    "Telegram notifications",
    "Teams notifications",
    "Email alerts",
    "Webhook notifications",
  ],
  openGraph: {
    title: "Notifiers - Databasus Documentation",
    description:
      "List of supported notification channels for Databasus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/notifiers",
  },
  twitter: {
    card: "summary",
    title: "Notifiers - Databasus Documentation",
    description:
      "List of supported notification channels for Databasus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
  },
  alternates: {
    canonical: "https://databasus.com/notifiers",
    languages: getLanguageAlternates("notifiers"),
  },
  robots: "index, follow",
};

export default function NotifiersPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Notifiers - Databasus Documentation",
            description:
              "List of supported notification channels for Databasus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
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

      <DocsNavbarComponent />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="notifiers">Notifiers</h1>

              <p className="text-lg text-gray-400">
                Databasus supports multiple notification channels to keep you
                informed about your PostgreSQL backup status. Get instant alerts
                when backups succeed, fail or encounter issues.
              </p>

              <h2 id="supported-notifiers">Supported notifiers</h2>

              <ul>
                <li>
                  <a
                    href="/notifiers/slack"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </a>{" "}
                  - Send notifications to Slack channels via webhooks
                </li>
                <li>
                  <strong>Discord</strong> - Post backup alerts to Discord
                  channels
                </li>
                <li>
                  <strong>Telegram</strong> - Receive notifications through
                  Telegram bots
                </li>
                <li>
                  <a
                    href="/notifiers/teams"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </a>{" "}
                  - Notify your team via Microsoft Teams channels
                </li>
                <li>
                  <strong>Email</strong> - Send email notifications for backup
                  events
                </li>
                <li>
                  <strong>Webhook</strong> - Custom webhook integration for any
                  service
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
