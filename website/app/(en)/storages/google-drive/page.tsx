import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "How to connect Google Drive to Databasus | Databasus",
  description:
    "Step-by-step guide to configure Google Drive storage for PostgreSQL backups with Databasus. Learn how to set up Google Cloud project and OAuth.",
  keywords: [
    "Databasus",
    "Google Drive",
    "PostgreSQL backup",
    "Google Cloud",
    "OAuth",
    "cloud storage",
    "database backup",
  ],
  openGraph: {
    title: "How to connect Google Drive to Databasus | Databasus",
    description:
      "Step-by-step guide to configure Google Drive storage for PostgreSQL backups with Databasus. Learn how to set up Google Cloud project and OAuth.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/storages/google-drive",
  },
  twitter: {
    card: "summary",
    title: "How to connect Google Drive to Databasus | Databasus",
    description:
      "Step-by-step guide to configure Google Drive storage for PostgreSQL backups with Databasus. Learn how to set up Google Cloud project and OAuth.",
  },
  alternates: {
    canonical: "https://databasus.com/storages/google-drive",
    languages: getLanguageAlternates("storages/google-drive"),
  },
  robots: "index, follow",
};

export default function GoogleDrivePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to connect Google Drive to Databasus",
            description:
              "Step-by-step guide to configure Google Drive storage for PostgreSQL backups with Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Create new project",
                text: "Go to Google Cloud Console and create a new project.",
              },
              {
                "@type": "HowToStep",
                name: "Enable Google Drive API",
                text: "Go to API & Services tab, then to API library and enable Google Drive API.",
              },
              {
                "@type": "HowToStep",
                name: "Configure consent screen",
                text: "Go to Credentials → Create credentials → Configure consent screen and fill required data.",
              },
              {
                "@type": "HowToStep",
                name: "Create OAuth client ID",
                text: "Go to Credentials → Create credentials → OAuth client ID.",
              },
              {
                "@type": "HowToStep",
                name: "Configure application settings",
                text: "Set application type to Web application and configure authorized origins and redirect URIs.",
              },
              {
                "@type": "HowToStep",
                name: "Add scope",
                text: 'Go to Data Access and add scope "/auth/drive.file".',
              },
              {
                "@type": "HowToStep",
                name: "Publish the app",
                text: "Go to Audience and publish the app.",
              },
              {
                "@type": "HowToStep",
                name: "Sign in via Google account",
                text: "Fill credentials data and sign in with your Google Account.",
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
              <h1 id="google-drive">Google Drive storage</h1>

              <p className="text-lg text-gray-400">
                To keep your backups in Google Drive, you need to create a
                Google Cloud project to access the Google Drive API, then sign
                in via your Google Account.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Important:</strong> To
                  connect Google Drive, your Databasus instance must be running
                  under HTTPS. For more information on setting up HTTPS, see the{" "}
                  <a
                    href="/installation#caddy-reverse-proxy"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Caddy reverse proxy setup guide →
                  </a>
                  <br />
                  For this guide, we&apos;ll assume your Databasus instance is
                  running at <code>databasus.yourdomain.com</code>.
                </p>
              </div>

              <h2 id="create-google-cloud-project">
                Create Google Cloud project
              </h2>

              <h3 id="create-new-project">1. Create new project</h3>

              <p>
                Go to{" "}
                <a
                  href="https://console.cloud.google.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://console.cloud.google.com/
                </a>{" "}
                and choose <strong>&quot;new project&quot;</strong> (top left).
              </p>

              <h3 id="enable-google-drive-api">2. Enable Google Drive API</h3>

              <p>
                Go to <strong>&quot;API &amp; Services&quot;</strong> tab, then
                to <strong>&quot;API library&quot;</strong>. Choose{" "}
                <strong>Google Drive API</strong> and enable it:
              </p>

              <Image
                src="/images/google-drive-storage/image-1.webp"
                alt="Enable Google Drive API"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-consent-screen">3. Configure consent screen</h3>

              <p>
                Go to <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;Configure consent screen&quot;</strong> and fill
                any data there:
              </p>

              <Image
                src="/images/google-drive-storage/image-2.webp"
                alt="Configure consent screen"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-oauth-client-id">4. Create OAuth client ID</h3>

              <p>
                Go to <strong>&quot;Credentials&quot;</strong> →{" "}
                <strong>&quot;Create credentials&quot;</strong> →{" "}
                <strong>&quot;OAuth client ID&quot;</strong>:
              </p>

              <Image
                src="/images/google-drive-storage/image-3.webp"
                alt="Create OAuth client ID"
                width={500}
                height={300}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-application-settings">
                5. Configure application settings
              </h3>

              <p>Fill the following data:</p>

              <ul>
                <li>
                  <strong>Application type:</strong> Web application
                </li>
                <li>
                  <strong>Authorized JavaScript origins:</strong>{" "}
                  <code>https://databasus.yourdomain.com</code>
                </li>
                <li>
                  <strong>Authorized redirect URIs:</strong>{" "}
                  <code>
                    https://databasus.yourdomain.com/storages/google-oauth
                  </code>
                </li>
              </ul>

              <p>
                <strong>Note:</strong> Replace{" "}
                <code>databasus.yourdomain.com</code> with your actual domain
                where Databasus is hosted (e.g., <code>backup.example.com</code>
                ).
              </p>

              <p>Then copy the credentials:</p>

              <Image
                src="/images/google-drive-storage/image-4.png"
                alt="Configure application settings - part 1"
                width={1000}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <Image
                src="/images/google-drive-storage/image-5.png"
                alt="Configure application settings - part 2"
                width={450}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="add-scope">6. Add scope</h3>

              <p>
                Go to <strong>&quot;Data Access&quot;</strong> and add scope{" "}
                <code>&quot;/auth/drive.file&quot;</code>:
              </p>

              <Image
                src="/images/google-drive-storage/image-6.png"
                alt="Add scope"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="publish-app">7. Publish the app</h3>

              <p>
                Go to <strong>&quot;Audience&quot;</strong> and publish the app:
              </p>

              <Image
                src="/images/google-drive-storage/image-7.png"
                alt="Publish the app"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="sign-in-google-account">Sign in via Google account</h2>

              <h3 id="fill-credentials">1. Fill credentials data</h3>

              <p>Fill the credentials from the previous steps in Databasus:</p>

              <Image
                src="/images/google-drive-storage/image-8.png"
                alt="Fill credentials data"
                width={600}
                height={600}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="choose-account">2. Choose your account</h3>

              <p>Choose your Google account to sign in.</p>

              <h3 id="handle-security-warning">3. Handle security warning</h3>

              <p>
                If you see a warning, click{" "}
                <strong>&quot;Advanced&quot;</strong> (left bottom corner) and
                choose <strong>&quot;Proceed anyway&quot;</strong>.
              </p>

              <p>
                <strong>Note:</strong> This warning appears because your app is
                not yet verified by Google. It&apos;s safe to proceed for your
                own application.
              </p>

              <p>
                That&apos;s it! Your Google Drive is now connected to Databasus
                and ready to store your PostgreSQL backups.
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
