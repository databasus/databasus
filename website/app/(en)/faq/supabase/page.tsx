import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "How to backup Supabase with Databasus | Databasus",
  description:
    "Learn how to backup your Supabase PostgreSQL database using Databasus. Step-by-step guide for configuring session pooler or IPv4 address for Supabase backups.",
  keywords: [
    "Databasus",
    "Supabase backup",
    "Supabase PostgreSQL backup",
    "backup Supabase database",
    "Supabase session pooler",
    "Supabase IPv4",
    "PostgreSQL backup",
    "database backup",
  ],
  openGraph: {
    title: "How to backup Supabase with Databasus | Databasus",
    description:
      "Learn how to backup your Supabase PostgreSQL database using Databasus. Step-by-step guide for configuring session pooler or IPv4 address for Supabase backups.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/faq/supabase",
  },
  twitter: {
    card: "summary",
    title: "How to backup Supabase with Databasus | Databasus",
    description:
      "Learn how to backup your Supabase PostgreSQL database using Databasus. Step-by-step guide for configuring session pooler or IPv4 address for Supabase backups.",
  },
  alternates: {
    canonical: "https://databasus.com/faq/supabase",
    languages: getLanguageAlternates("faq/supabase"),
  },
  robots: "index, follow",
};

export default function SupabasePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to backup Supabase with Databasus",
            description:
              "Step-by-step guide to backup your Supabase PostgreSQL database using Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Get connection details from Supabase",
                text: "Navigate to your Supabase project settings and find the database connection details.",
              },
              {
                "@type": "HowToStep",
                name: "Use Session Pooler with IPv4",
                text: "Copy the Session Pooler connection string and ensure 'Use IPv4 Address' is enabled.",
              },
              {
                "@type": "HowToStep",
                name: "Configure Databasus",
                text: "Enter the Supabase connection details in Databasus to start backing up your database.",
              },
              {
                "@type": "HowToStep",
                name: "Understand schema limitations",
                text: "By default, only the public schema is backed up as other Supabase schemas are restricted.",
              },
            ],
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
              <h1 id="supabase-backup">How to backup Supabase</h1>

              <p className="text-lg text-gray-400">
                Databasus supports backups for Supabase PostgreSQL databases.
                The main requirement is to use an IPv4 address to connect to
                your Supabase instance.
              </p>

              <h2 id="connection-options">Connection options</h2>

              <p>
                There are two ways to connect Databasus to your Supabase
                database:
              </p>

              <ol>
                <li>
                  <strong>Use Session Pooler with IPv4</strong> (Recommended) —
                  Free option available in all Supabase projects
                </li>
                <li>
                  <strong>Purchase IPv4 Add-on</strong> — Direct connection
                  option from Supabase
                </li>
              </ol>

              <h2 id="session-pooler">
                Option 1: Use Session Pooler (Recommended)
              </h2>

              <p>
                The Session Pooler provides an IPv4 address for your Supabase
                database connection at no additional cost. Here&apos;s how to
                configure it:
              </p>

              <h3 id="step-1">1. Find pooler connection</h3>

              <p>
                Navigate to your Supabase project, go to{" "}
                <strong>Project Settings</strong> → <strong>Database</strong>.
                Scroll down to the <strong>Connection string</strong> section
                and select <strong>Session pooler</strong> mode.
              </p>

              <img
                src="/images/faq/supabase/image-1.png"
                alt="Select Session pooler mode in Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h3 id="step-2">2. Copy connection details</h3>

              <p>
                Copy connection details and use them in Databasus when adding
                your database. See screenshot to differ connection details.
              </p>

              <img
                src="/images/faq/supabase/image-2.png"
                alt="Enable IPv4 Address toggle in Supabase"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <h2 id="ipv4-addon">Option 2: Purchase IPv4 Add-on</h2>

              <p>
                Supabase offers a paid IPv4 add-on that provides a dedicated
                IPv4 address for your database. This option gives you a direct
                connection without going through the connection pooler.
              </p>

              <p>To enable this option:</p>

              <ol>
                <li>Go to your Supabase project dashboard</li>
                <li>
                  Navigate to <strong>Project Settings</strong> →{" "}
                  <strong>Add-ons</strong>
                </li>
                <li>
                  Enable the <strong>IPv4</strong> add-on
                </li>
                <li>
                  Use the direct database connection details in Databasus
                </li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Tip:</strong> For most
                  use cases, the free Session Pooler with IPv4 option works
                  perfectly for backups. The paid IPv4 add-on is only necessary
                  if you need a direct connection for other reasons.
                </p>
              </div>

              <h2 id="default-schema">Default schema limitation</h2>

              <p>
                By default, Databasus backs up only the <code>public</code>{" "}
                schema when working with Supabase databases. This is because
                Supabase restricts access to other schemas (such as{" "}
                <code>auth</code>, <code>storage</code>, and{" "}
                <code>realtime</code>) for security reasons.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-blue-400">ℹ️ Note:</strong> The{" "}
                  <code>public</code> schema contains your application data and
                  custom tables. Supabase-managed schemas like <code>auth</code>{" "}
                  and <code>storage</code> are protected and managed by Supabase
                  itself.
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Back to FAQ
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
