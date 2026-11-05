import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "How to use Databasus with Cloudflare R2 | Databasus",
  description:
    "Step-by-step guide to configure Cloudflare R2 storage for PostgreSQL backups with Databasus. Learn how to set up S3-compatible storage with R2.",
  keywords: [
    "Databasus",
    "Cloudflare R2",
    "PostgreSQL backup",
    "S3 storage",
    "cloud storage",
    "database backup",
  ],
  openGraph: {
    title: "How to use Databasus with Cloudflare R2 | Databasus",
    description:
      "Step-by-step guide to configure Cloudflare R2 storage for PostgreSQL backups with Databasus. Learn how to set up S3-compatible storage with R2.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/storages/cloudflare-r2",
  },
  twitter: {
    card: "summary",
    title: "How to use Databasus with Cloudflare R2 | Databasus",
    description:
      "Step-by-step guide to configure Cloudflare R2 storage for PostgreSQL backups with Databasus. Learn how to set up S3-compatible storage with R2.",
  },
  alternates: {
    canonical: "https://databasus.com/storages/cloudflare-r2",
    languages: getLanguageAlternates("storages/cloudflare-r2"),
  },
  robots: "index, follow",
};

export default function CloudflareR2Page() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to use Databasus with Cloudflare R2",
            description:
              "Step-by-step guide to configure Cloudflare R2 storage for PostgreSQL backups with Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Fill your bucket name",
                text: "Enter your R2 bucket name in the storage configuration.",
              },
              {
                "@type": "HowToStep",
                name: "Set the region",
                text: 'In region field, fill "auto"',
              },
              {
                "@type": "HowToStep",
                name: "Generate an access key ID & secret access key",
                text: "In the Cloudflare dashboard, go to R2 → API → Manage API Tokens. Create the token and grant it the permissions you need.",
              },
              {
                "@type": "HowToStep",
                name: "Find your account ID",
                text: "On any R2 page in the dashboard, you'll see your Account ID near the top.",
              },
              {
                "@type": "HowToStep",
                name: "Construct the S3 endpoint",
                text: "Replace <ACCOUNT_ID> with the value from your dashboard in the format: https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
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
              <h1 id="cloudflare-r2">Cloudflare R2 storage</h1>

              <p className="text-lg text-gray-400">
                To use Cloudflare R2 as an S3-compatible storage for your
                PostgreSQL backups, you&apos;ll need to configure your R2 bucket
                credentials and endpoint.
              </p>

              <h2 id="configuration-steps">Configuration steps</h2>

              <h3 id="fill-bucket-name">1. Fill your bucket name</h3>

              <p>Enter your R2 bucket name in the storage configuration:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-1.webp"
                alt="Fill your bucket name in Cloudflare R2"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="set-region">2. Set the region</h3>

              <p>
                In the region field, fill <code>&quot;auto&quot;</code>
              </p>

              <h3 id="generate-access-key">
                3. Generate an Access Key ID &amp; Secret Access Key
              </h3>

              <p>
                In the Cloudflare dashboard, go to{" "}
                <strong>R2 → API → Manage API Tokens</strong>. Create a new
                token and grant it the permissions you need (e.g.{" "}
                <strong>&quot;Object Read &amp; Write&quot;</strong>).
              </p>

              <p>When the token is created, you&apos;ll see:</p>

              <ul>
                <li>
                  <strong>Access Key ID</strong> (the token&apos;s ID)
                </li>
                <li>
                  <strong>Secret Access Key</strong> (the SHA-256 hash of the
                  token value)
                </li>
              </ul>

              <p>Copy both values to Databasus:</p>

              <Image
                src="/images/cloudflare-r2-storage/image-2.gif"
                alt="Generate Access Key ID and Secret Access Key"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="find-account-id">4. Find your account ID</h3>

              <p>
                On any R2 page in the dashboard, you&apos;ll see your Account ID
                near the top (or in your account settings):
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-3.webp"
                alt="Find your Account ID in Cloudflare dashboard"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="construct-endpoint">5. Construct the S3 endpoint</h3>

              <p>Use the following format for your S3 endpoint:</p>

              <pre>
                <code>https://&lt;ACCOUNT_ID&gt;.r2.cloudflarestorage.com</code>
              </pre>

              <p>
                Replace <code>&lt;ACCOUNT_ID&gt;</code> with the value from your
                dashboard and enter it in Databasus.
              </p>

              <p>
                That&apos;s it! Your configuration should now look like this:
              </p>

              <Image
                src="/images/cloudflare-r2-storage/image-4.png"
                alt="Configuration complete"
                width={500}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                Your Databasus is now ready to use Cloudflare R2 as storage for
                your PostgreSQL backups.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/storages"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Back to storages
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
