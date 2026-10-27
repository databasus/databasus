import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title:
    "How to configure Microsoft Teams notifications for Databasus | Databasus",
  description:
    "Step-by-step guide to set up Microsoft Teams notifications for PostgreSQL backup alerts with Databasus. Learn how to create Teams webhook and configure notifications.",
  keywords: [
    "Databasus",
    "Microsoft Teams notifications",
    "PostgreSQL backup",
    "Teams webhook",
    "backup alerts",
    "database notifications",
  ],
  openGraph: {
    title:
      "How to configure Microsoft Teams notifications for Databasus | Databasus",
    description:
      "Step-by-step guide to set up Microsoft Teams notifications for PostgreSQL backup alerts with Databasus. Learn how to create Teams webhook and configure notifications.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/notifiers/teams",
  },
  twitter: {
    card: "summary",
    title:
      "How to configure Microsoft Teams notifications for Databasus | Databasus",
    description:
      "Step-by-step guide to set up Microsoft Teams notifications for PostgreSQL backup alerts with Databasus. Learn how to create Teams webhook and configure notifications.",
  },
  alternates: {
    canonical: "https://databasus.com/notifiers/teams",
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
            name: "How to configure Microsoft Teams notifications for Databasus",
            description:
              "Step-by-step guide to set up Microsoft Teams notifications for PostgreSQL backup alerts with Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Open Teams channel",
                text: "Navigate to the Microsoft Teams channel where you want to receive notifications.",
              },
              {
                "@type": "HowToStep",
                name: "Access workflows",
                text: "Open the Workflows feature in your Teams channel.",
              },
              {
                "@type": "HowToStep",
                name: "Create new workflow",
                text: "Create a new workflow for incoming webhooks.",
              },
              {
                "@type": "HowToStep",
                name: "Select webhook template",
                text: "Choose the incoming webhook template from available options.",
              },
              {
                "@type": "HowToStep",
                name: "Configure webhook",
                text: "Set up the webhook name and channel.",
              },
              {
                "@type": "HowToStep",
                name: "Copy webhook URL",
                text: "Copy the generated webhook URL from Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Configure in Databasus",
                text: "Paste the webhook URL into Databasus notifier configuration.",
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
              <h1 id="teams-notifications">Microsoft Teams notifications</h1>

              <p className="text-lg text-gray-400">
                Configure Microsoft Teams to receive instant notifications about
                your PostgreSQL backup status. Get alerts for successful
                backups, failures and warnings directly in your Teams channels.
              </p>

              <h2 id="setup-teams-webhook">Setup Teams webhook</h2>

              <h3 id="open-teams-channel">1. Open your Teams channel</h3>

              <p>
                Navigate to the Microsoft Teams channel where you want to
                receive backup notifications. Click on the three dots (
                <strong>•••</strong>) next to the channel name.
              </p>

              <Image
                src="/images/notifier-teams/image-01.png"
                alt="Open Teams channel"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="access-workflows">2. Access workflows</h3>

              <p>
                From the channel menu, select{" "}
                <strong>&quot;Workflows&quot;</strong> to open the Power
                Automate integration.
              </p>

              <Image
                src="/images/notifier-teams/image-02.png"
                alt="Access Workflows"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-new-workflow">3. Create new workflow</h3>

              <p>
                In the Workflows panel, click on{" "}
                <strong>&quot;Create&quot;</strong> or search for{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                template.
              </p>

              <Image
                src="/images/notifier-teams/image-03.png"
                alt="Create new workflow"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="select-webhook-template">4. Select webhook template</h3>

              <p>
                Choose the{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                template from the available options.
              </p>

              <Image
                src="/images/notifier-teams/image-04.png"
                alt="Select webhook template"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-webhook">5. Configure webhook</h3>

              <p>
                Set up the webhook by providing a name (e.g.,{" "}
                <strong>&quot;Databasus Backup Notifications&quot;</strong>)
                and confirm the channel where notifications will be posted.
              </p>

              <Image
                src="/images/notifier-teams/image-05.png"
                alt="Configure webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="copy-webhook-url">6. Copy webhook URL</h3>

              <p>
                After creating the workflow, you&apos;ll see the{" "}
                <strong>HTTP POST URL</strong>. Copy this URL - you&apos;ll need
                it for Databasus configuration.
              </p>

              <Image
                src="/images/notifier-teams/image-06.png"
                alt="Copy webhook URL"
                width={500}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="configure-databasus">Configure in Databasus</h2>

              <h3 id="add-teams-notifier">1. Add Teams notifier</h3>

              <p>
                In Databasus, navigate to the notifiers settings and add a new
                Microsoft Teams notifier. Paste the webhook URL you copied from
                Teams.
              </p>

              <Image
                src="/images/notifier-teams/image-07.png"
                alt="Configure Teams in Databasus"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="test-notification">2. Test the notification</h3>

              <p>
                After configuring the webhook, test the notification to ensure
                it&apos;s working correctly. You should receive a test message
                in your selected Teams channel.
              </p>

              <p>
                That&apos;s it! Your Microsoft Teams channel is now configured
                to receive PostgreSQL backup notifications from Databasus.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/notifiers"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Back to notifiers
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
