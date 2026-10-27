import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "How to configure Slack notifications for Databasus | Databasus",
  description:
    "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Databasus. Learn how to create a Slack bot app and configure notifications.",
  keywords: [
    "Databasus",
    "Slack notifications",
    "PostgreSQL backup",
    "Slack bot token",
    "Slack API",
    "backup alerts",
    "database notifications",
  ],
  openGraph: {
    title: "How to configure Slack notifications for Databasus | Databasus",
    description:
      "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Databasus. Learn how to create a Slack bot app and configure notifications.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/notifiers/slack",
  },
  twitter: {
    card: "summary",
    title: "How to configure Slack notifications for Databasus | Databasus",
    description:
      "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Databasus. Learn how to create a Slack bot app and configure notifications.",
  },
  alternates: {
    canonical: "https://databasus.com/notifiers/slack",
    languages: getLanguageAlternates("notifiers/slack"),
  },
  robots: "index, follow",
};

export default function SlackPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to configure Slack notifications for Databasus",
            description:
              "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Go to Slack API",
                text: "Navigate to https://api.slack.com/apps and sign in to your Slack workspace.",
              },
              {
                "@type": "HowToStep",
                name: "Create New App",
                text: "Click on 'Create New App' button and choose 'From scratch'.",
              },
              {
                "@type": "HowToStep",
                name: "Configure Bot Permissions",
                text: "Navigate to OAuth & Permissions and add the required scopes: chat:write, channels:join, im:write, and groups:write under Bot Token Scopes.",
              },
              {
                "@type": "HowToStep",
                name: "Install to Workspace",
                text: "Install the app to your workspace and authorize it.",
              },
              {
                "@type": "HowToStep",
                name: "Copy Bot Token",
                text: "Copy the Bot User OAuth Token that starts with 'xoxb-'.",
              },
              {
                "@type": "HowToStep",
                name: "Get Channel ID",
                text: "Open your target channel and get the Channel ID from the channel details.",
              },
              {
                "@type": "HowToStep",
                name: "Add bot to private channel",
                text: "If using a private channel, invite the bot to the channel by mentioning it.",
              },
              {
                "@type": "HowToStep",
                name: "Configure in Databasus",
                text: "In Databasus, add the Bot Token and Channel ID to the Slack notifier configuration.",
              },
              {
                "@type": "HowToStep",
                name: "Test the notification",
                text: "Test the notification to ensure it's working correctly.",
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
              <h1 id="slack-notifications">Slack notifications</h1>

              <p className="text-lg text-gray-400">
                Configure Slack to receive instant notifications about your
                PostgreSQL backup status. Get alerts for successful backups,
                failures and warnings directly in your Slack channels.
              </p>

              <h2 id="create-slack-app">Create a Slack App</h2>

              <h3 id="go-to-slack-api">1. Go to Slack API</h3>

              <p>
                Navigate to{" "}
                <a
                  href="https://api.slack.com/apps"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://api.slack.com/apps
                </a>{" "}
                and sign in to your Slack workspace.
              </p>

              <h3 id="create-new-app">2. Create New App</h3>

              <p>
                Click on <strong>&quot;Create New App&quot;</strong> button.
              </p>

              <h3 id="choose-from-scratch">
                3. Choose &quot;From scratch&quot;
              </h3>

              <p>
                Select <strong>&quot;From scratch&quot;</strong> option when
                prompted.
              </p>

              <h3 id="name-your-app">4. Name your app</h3>

              <p>
                Enter a name for your app (e.g., &quot;Databasus
                Notifications&quot;) and select the workspace where you want to
                install it. Click <strong>&quot;Create App&quot;</strong>.
              </p>

              <h2 id="configure-bot-permissions">Configure Bot Permissions</h2>

              <h3 id="navigate-to-oauth">
                5. Navigate to OAuth &amp; Permissions
              </h3>

              <p>
                In the left sidebar, click on{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-1.png"
                alt="Navigate to OAuth &amp; Permissions"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="add-bot-scopes">6. Add Bot Token Scopes (Required)</h3>

              <p>
                Scroll down to the <strong>&quot;Scopes&quot;</strong> section
                and under <strong>&quot;Bot Token Scopes&quot;</strong>, click{" "}
                <strong>&quot;Add an OAuth Scope&quot;</strong>.
              </p>

              <p>Add all of the following required scopes:</p>

              <ul>
                <li>
                  <code>chat:write</code> - to send messages to channels
                </li>
                <li>
                  <code>channels:join</code> - to allow the bot to join public
                  channels automatically
                </li>
                <li>
                  <code>im:write</code> - to send direct messages to users
                </li>
                <li>
                  <code>groups:write</code> - to send messages to private
                  channels
                </li>
                <li>
                  <code>channels:history</code> - to read channel history
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-2.png"
                alt="Add Bot Token Scopes"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h2 id="install-app">Install App to Workspace</h2>

              <h3 id="install-to-workspace">7. Install to Workspace</h3>

              <p>
                Scroll to the top of the{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong> page and
                click <strong>&quot;Install to Workspace&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-3.png"
                alt="Install to Workspace"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="authorize-app">8. Authorize the app</h3>

              <p>
                Review the permissions and click{" "}
                <strong>&quot;Allow&quot;</strong> to authorize the app.
              </p>

              <h3 id="copy-bot-token">9. Copy Bot User OAuth Token</h3>

              <p>
                After installation, you&apos;ll see the{" "}
                <strong>&quot;Bot User OAuth Token&quot;</strong>. It starts
                with <code>xoxb-</code>. Copy this token - you&apos;ll need it
                for Databasus configuration.
              </p>

              <h2 id="get-channel-id">Get Channel ID</h2>

              <h3 id="open-channel">10. Open your target channel</h3>

              <p>
                In your Slack workspace, open the channel where you want to
                receive backup notifications.
              </p>

              <h3 id="get-channel-info">11. Get the channel ID</h3>

              <p>
                Click on the channel name at the top, then scroll down in the
                channel details. You&apos;ll find the{" "}
                <strong>Channel ID</strong> at the bottom of the
                &quot;About&quot; section. It starts with <code>C</code> (for
                public channels) or <code>G</code> (for private channels).
              </p>

              <p>Copy this Channel ID.</p>

              <img
                src="/images/notifier-slack/image-4.png"
                alt="Get Channel ID"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[500px]"
                loading="lazy"
              />

              <h3 id="add-bot-to-channel">
                12. Add bot to channel (required for private channels)
              </h3>

              <p>
                <strong>
                  If you&apos;re using a private channel, you must manually
                  invite the bot to the channel:
                </strong>
              </p>

              <ol>
                <li>
                  In the private channel, type{" "}
                  <code>@Databasus Notifications</code> (or whatever name you
                  gave your app)
                </li>
                <li>
                  Click on the bot name when it appears and select{" "}
                  <strong>&quot;Add to Channel&quot;</strong> or{" "}
                  <strong>&quot;Invite to Channel&quot;</strong>
                </li>
              </ol>

              <p>
                For <strong>public channels</strong>, the bot will automatically
                join when sending the first message (thanks to the{" "}
                <code>channels:join</code> permission), so this step is not
                necessary.
              </p>

              <h2 id="configure-databasus">Configure in Databasus</h2>

              <h3 id="add-slack-notifier">13. Add Slack notifier</h3>

              <p>
                In Databasus, navigate to the notifiers settings and add a new
                Slack notifier:
              </p>

              <ul>
                <li>
                  <strong>Bot Token:</strong> Paste the Bot User OAuth Token you
                  copied (starts with <code>xoxb-</code>)
                </li>
                <li>
                  <strong>Target Channel ID:</strong> Paste the Channel ID you
                  copied (starts with <code>C</code>, <code>G</code>,{" "}
                  <code>D</code>, or <code>U</code>)
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-5.png"
                alt="Add Slack notifier"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="test-notification">14. Test the notification</h3>

              <p>
                After configuring the notifier, test it to ensure it&apos;s
                working correctly. You should receive a test message in your
                selected Slack channel.
              </p>

              <p>
                That&apos;s it! Your Slack workspace is now configured to
                receive PostgreSQL backup notifications from Databasus.
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
