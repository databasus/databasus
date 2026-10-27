import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Reset password - Databasus Documentation",
  description:
    "Learn how to reset user passwords, list administrators and create accounts in Databasus using the built-in command-line tool. Quick and secure password recovery for your PostgreSQL backup system.",
  keywords: [
    "Databasus password reset",
    "reset user password",
    "PostgreSQL backup password",
    "Docker password recovery",
    "password recovery",
    "Databasus authentication",
  ],
  openGraph: {
    title: "Reset Password - Databasus Documentation",
    description:
      "Learn how to reset user passwords, list administrators and create accounts in Databasus using the built-in command-line tool. Quick and secure password recovery for your PostgreSQL backup system.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/password",
  },
  twitter: {
    card: "summary",
    title: "Reset Password - Databasus Documentation",
    description:
      "Learn how to reset user passwords, list administrators and create accounts in Databasus using the built-in command-line tool. Quick and secure password recovery for your PostgreSQL backup system.",
  },
  alternates: {
    canonical: "https://databasus.com/password",
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"`;
  const listAdminsCommand = `docker exec -it databasus ./main --list-admins`;
  const createUserCommand = `docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Reset Password - Databasus Documentation",
            description:
              "Learn how to reset user passwords in Databasus using the built-in command-line tool.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to reset Databasus user password",
            description:
              "Step-by-step guide to reset user passwords in Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Run password reset command",
                text: "Execute the docker exec command with your new password and user email.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Use docker exec to run the password reset command inside the Databasus container",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Verify password change",
                text: "Log in to Databasus with your new password to confirm the change was successful.",
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
              <h1 id="reset-password">Reset user password</h1>

              <h2 id="reset-password-command">Reset password command</h2>

              <p>
                To reset a user&apos;s password, use the following command on
                the server where Databasus is running:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">Parameters</h2>

              <p>The command accepts the following parameters:</p>

              <ul>
                <li>
                  <strong>--new-password</strong>: The new password, at least 8
                  characters long. Make sure it&apos;s secure and contains a mix
                  of letters, numbers and special characters.
                </li>
                <li>
                  <strong>--email</strong>: The email address of the account
                  whose password you want to reset (e.g.,{" "}
                  <code>owner@example.com</code>). An instance created before
                  the first account administered it still carries the
                  placeholder address <code>admin</code> until its owner
                  replaces it, so pass <code>admin</code> there.
                </li>
              </ul>

              <h2 id="list-admins">List administrator accounts</h2>

              <p>
                If you do not remember which address administers the instance,
                list the administrator accounts on the server where Databasus is
                running:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{listAdminsCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={listAdminsCommand} />
                </div>
              </div>

              <p>
                The output names every administrator account with its email
                address, display name, creation date and active state, and marks
                the one the instance recognizes as its administrator. It prints
                no password, no password hash and no token. On an instance where
                nobody has created an account yet, it reports that the instance
                has no administrator.
              </p>

              <h2 id="create-user">Create an account from the console</h2>

              <p>
                To provision an account without signing up in the browser, for
                example from a deployment script, create it on the server where
                Databasus is running:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createUserCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={createUserCommand} />
                </div>
              </div>

              <p>The command accepts the following parameters:</p>

              <ul>
                <li>
                  <strong>--email</strong>: The email address of the new
                  account. It must be a valid email address.
                </li>
                <li>
                  <strong>--password</strong>: The password of the new account,
                  at least 8 characters long.
                </li>
                <li>
                  <strong>--role</strong>: <code>admin</code> or{" "}
                  <code>member</code>.
                </li>
                <li>
                  <strong>--name</strong>: An optional display name. Without it,
                  the account is named after the part of the address before the{" "}
                  <code>@</code>.
                </li>
              </ul>

              <p>
                The account is active at once and signs in with that address and
                password, whether or not open registration is enabled. On an
                instance that holds no account yet, the account you create
                administers the instance, so the first one must use{" "}
                <code>--role=admin</code>. An administrator created later this
                way is an ordinary administrator.
              </p>

              <p>
                Running the command again for an existing account with the same
                role succeeds and changes nothing, so a deployment can run it on
                every rollout. It does not change the password then: rotate the
                password with the <code>--new-password</code> reset command
                above. If the address belongs to an account with another role, a
                deactivated or invited account, or an account that signs in
                through Google or GitHub, the command fails and changes nothing.
              </p>

              <h2 id="disable-two-factor">Stop requiring a sign-in code</h2>

              <p>
                An instance can require a six-digit code emailed at sign-in on
                top of the password. If the mail server stops delivering those
                codes, nobody gets in with a password, and a password reset
                alone does not help. Switch the second factor off on the server
                where Databasus is running:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{disableTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={disableTwoFactorCommand} />
                </div>
              </div>

              <p>
                The command reports whether it changed anything, succeeds
                without complaint when the second factor is already off, and
                records the change in the audit log. The next password sign-in
                then needs no code, and an administrator can turn the setting
                back on once mail works again.
              </p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
