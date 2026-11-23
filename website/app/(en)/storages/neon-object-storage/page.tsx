import type { Metadata } from "next";
import { getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

const PAGE_URL = getLocalizedUrl("en", "storages/neon-object-storage");

export const metadata: Metadata = {
  title: "How to back up PostgreSQL to Neon Object Storage | Databasus",
  description:
    "Step-by-step guide with screenshots: create a Neon Object Storage bucket and credential, add it to Databasus as an S3 storage and keep database backups at Neon on a schedule.",
  keywords: [
    "Databasus",
    "Neon Object Storage backup",
    "Neon Object Storage",
    "Neon S3",
    "Neon bucket",
    "PostgreSQL backup to S3",
    "S3-compatible storage",
    "database backup",
  ],
  openGraph: {
    title: "How to back up PostgreSQL to Neon Object Storage | Databasus",
    description:
      "Step-by-step guide with screenshots: create a Neon Object Storage bucket and credential, add it to Databasus as an S3 storage and keep database backups at Neon on a schedule.",
    type: "article",
    locale: "en_US",
    url: PAGE_URL,
  },
  twitter: {
    card: "summary",
    title: "How to back up PostgreSQL to Neon Object Storage | Databasus",
    description:
      "Step-by-step guide with screenshots: create a Neon Object Storage bucket and credential, add it to Databasus as an S3 storage and keep database backups at Neon on a schedule.",
  },
  alternates: {
    canonical: PAGE_URL,
    languages: getLanguageAlternates("storages/neon-object-storage"),
  },
  robots: "index, follow",
};

export default function NeonObjectStoragePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to back up to Neon Object Storage with Databasus",
            description:
              "Step-by-step guide to create a Neon Object Storage bucket and credential and add it to Databasus as a backup storage",
            step: [
              {
                "@type": "HowToStep",
                name: "Create a bucket in Neon",
                text: "In Neon Console, open the project for backups, click Object storage in the sidebar and click Create bucket.",
                url: `${PAGE_URL}#create-bucket`,
              },
              {
                "@type": "HowToStep",
                name: "Copy the storage credentials",
                text: "Click Connect, open the Storage tab, choose Parameters only and click Copy credentials.",
                url: `${PAGE_URL}#copy-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Choose Neon Object Storage in Databasus",
                text: "In Databasus, open Storages, click Add storage and choose Neon Object Storage in the type list.",
                url: `${PAGE_URL}#choose-neon-object-storage`,
              },
              {
                "@type": "HowToStep",
                name: "Paste the credentials and type the bucket name",
                text: "Click Paste from clipboard to fill in the endpoint, region, access key and secret key, then type the bucket name.",
                url: `${PAGE_URL}#paste-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Test the connection and save",
                text: "Click Test connection and, when it passes, click Save.",
                url: `${PAGE_URL}#test-connection`,
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
              <h1 id="neon-object-storage-backup">
                How to back up to Neon Object Storage
              </h1>

              <p className="text-lg text-gray-400">
                <a href="https://neon.com/docs/storage/overview">
                  Neon Object Storage
                </a>{" "}
                is S3-compatible object storage built into every Neon project.
                To keep your backups there, create a bucket and a credential in
                Neon Console, then add the bucket to Databasus as a storage.
                Databasus uploads every backup of the databases that use this
                storage, whether they run on Neon or anywhere else.
              </p>

              <p>
                All you need is a running Databasus. If you don&apos;t have one
                yet, follow the{" "}
                <a href={getLocalizedHref("en", "installation")}>
                  installation guide
                </a>
                . The{" "}
                <a href="https://neon.com/docs/storage/s3-compatibility">
                  S3 compatibility page
                </a>{" "}
                of the Neon documentation lists what it supports.
              </p>

              <h2 id="create-bucket">Step 1. Create a bucket in Neon</h2>

              <p>
                Open the project that will hold the backups in Neon Console.
                Prefer a different project than the databases you back up:
                deleting a project removes its buckets too. Click{" "}
                <strong>Object storage</strong> in the sidebar, then{" "}
                <strong>Create bucket</strong> and enter a name for the bucket.
                A new bucket is private, and it belongs to the current branch
                of the project.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-object-storage-buckets.png"
                alt="Object storage page of a Neon project with a bucket and the Create bucket button"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
              />

              <h2 id="copy-credentials">
                Step 2. Copy the storage credentials
              </h2>

              <p>
                Click <strong>Connect</strong> and open the{" "}
                <strong>Storage</strong> tab. With{" "}
                <strong>Parameters only</strong> selected, Neon shows the
                endpoint, the access key, the secret key and the region. If the
                block shows placeholders instead of keys, generate a credential
                first, as its first line asks. Then click{" "}
                <strong>Copy credentials</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-connect-storage-credentials.png"
                alt="Storage tab of the Neon Connect window with the endpoint, keys and region and the Copy credentials button"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                The block looks like this. It does not hold the bucket name, so
                remember the name from step 1.
              </p>

              <pre>
                <code>{`AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.c-9.us-east-2.aws.neon.tech"
AWS_ACCESS_KEY_ID="nak_live_..."
AWS_SECRET_ACCESS_KEY="..."
AWS_REGION="us-east-2"`}</code>
              </pre>

              <h2 id="add-storage">Step 3. Add the storage to Databasus</h2>

              <h3 id="choose-neon-object-storage">
                Choose Neon Object Storage in Databasus
              </h3>

              <p>
                In Databasus, open <strong>Storages</strong> and click{" "}
                <strong>Add storage</strong>. Give the storage a name and choose{" "}
                <strong>Neon Object Storage</strong> in the type list. Databasus
                saves it as an ordinary S3 storage, so the storage list shows
                its type as S3.
              </p>

              <h3 id="paste-credentials">
                Paste the credentials and type the bucket name
              </h3>

              <p>
                Click <strong>Paste from clipboard</strong>. Databasus reads the
                block you copied in Neon and fills in the endpoint, region,
                access key and secret key. The block stays in your browser: the
                keys leave it only when you test or save the storage. Type the
                bucket name into <strong>S3 Bucket</strong> yourself. You can
                also fill in every field by hand below the{" "}
                <strong>or enter the details manually</strong> divider.
              </p>

              <img
                src="/images/storages/neon-object-storage/databasus-neon-object-storage-form.png"
                alt="Neon Object Storage form of Databasus with the pasted endpoint, region and keys and the typed bucket name"
                width={1435}
                height={897}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="test-connection">Test the connection and save</h3>

              <p>
                Click <strong>Test connection</strong>. When the test passes,
                click <strong>Save</strong>. Now choose this storage in the
                backup settings of any database. If the database itself runs on
                Neon, the{" "}
                <a href={getLocalizedHref("en", "faq/neon")}>
                  Neon backup guide
                </a>{" "}
                walks through the rest of the setup.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("en", "storages")}
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
