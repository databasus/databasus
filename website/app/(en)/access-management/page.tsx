import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Access Management - Databasus Documentation",
  description:
    "Learn how to manage access, roles, and permissions in Databasus. Control who can sign up, create workspaces, and manage databases with workspace-level and system-level roles.",
  keywords: [
    "Databasus access management",
    "user roles",
    "workspace permissions",
    "audit logs",
    "PostgreSQL backup security",
    "team collaboration",
    "access control",
    "workspace management",
  ],
  openGraph: {
    title: "Access Management - Databasus Documentation",
    description:
      "Learn how to manage access, roles, and permissions in Databasus. Control who can sign up, create workspaces, and manage databases with workspace-level and system-level roles.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/access-management",
  },
  twitter: {
    card: "summary",
    title: "Access Management - Databasus Documentation",
    description:
      "Learn how to manage access, roles, and permissions in Databasus. Control who can sign up, create workspaces, and manage databases with workspace-level and system-level roles.",
  },
  alternates: {
    canonical: "https://databasus.com/access-management",
    languages: getLanguageAlternates("access-management"),
  },
  robots: "index, follow",
};

export default function AccessManagementPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Access Management - Databasus Documentation",
            description:
              "Learn how to manage access, roles, and permissions in Databasus. Control who can sign up, create workspaces, and manage databases with workspace-level and system-level roles.",
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
              <h1 id="settings">Settings</h1>

              <p>
                Databasus is suitable both for single users and teams. This
                section is dedicated to the access management for teams.{" "}
                <strong>
                  So if you are the only user in your Databasus instance
                </strong>
                , you can skip this section.
              </p>

              <p>
                Databasus does not have a lot of settings. Actually, it only
                allows you to control:
              </p>

              <ul>
                <li>Who can sign up in your Databasus instance</li>
                <li>Who can create workspaces</li>
                <li>
                  Who can manage databases, notifiers and storages within
                  workspaces
                </li>
              </ul>

              <h2 id="workspaces">Workspaces</h2>

              <p>
                Workspace is a place where you{" "}
                <strong>group databases, notifiers and storages</strong>. You
                can add members to workspaces (and create multiple workspaces).
              </p>

              <p>
                You can manage access management per workspace. For example:
              </p>

              <ul>
                <li>
                  you have a DevOps team responsible for 10 DBs of the project
                  (so a couple of users inside a workspace);
                </li>
                <li>
                  you have 3 different projects with different DBs and storages
                  (so a couple of workspaces with different users);
                </li>
                <li>
                  you have 5 independent DBs where different users can access
                  each one (so user A has access to DB1, user B has access to
                  DB2, user C has access to DB3, etc.).
                </li>
              </ul>

              <img
                src="/images/access-management/users.png"
                alt="Workspaces"
                width={550}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <p>
                If you allow users to sign up for your Databasus and create
                their own workspaces (see{" "}
                <a href="#global-settings">global settings</a>), they will be
                able to create their own workspaces.
              </p>

              <p>
                <strong>
                  Users never see other workspaces than their own until they are
                  invited to join.
                </strong>
              </p>

              <h2 id="audit-logs">Audit logs</h2>

              <p>
                Audit logs are messages about actions performed by users. They
                are needed to track changes and actions performed by users, as
                well as to detect any suspicious activity.
              </p>

              <p>For example:</p>

              <ul>
                <li>user created a new database</li>
                <li>user deleted a database</li>
                <li>user initiated a new backup</li>
                <li>user downloaded a backup</li>
                <li>user created a new notifier</li>
                <li>user created a workspace</li>
                <li>user deleted a workspace</li>
                <li>etc.</li>
              </ul>

              <p>You can view audit logs with filters:</p>

              <ul>
                <li>per workspace;</li>
                <li>per user (within multiple workspaces);</li>
              </ul>

              <img
                src="/images/access-management/audit-logs.png"
                alt="Audit logs"
                width={1000}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="user-roles">User roles</h2>

              <p>
                All users in Databasus have roles <u>within the system</u>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Feature</th>
                    <th>Admin</th>
                    <th>Member</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Manage all settings and users</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">❌</td>
                  </tr>
                  <tr>
                    <td>Create workspaces</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Member">✅ (if allowed by settings)</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Usually, there is only one administrator in the system: the
                first account created on the instance, which administers it.
              </p>

              <p>
                <u>Within a workspace</u> there are also roles:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Feature</th>
                    <th>Viewer</th>
                    <th>Member</th>
                    <th>Admin</th>
                    <th>Owner</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>View databases, notifiers, storages</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Initiate and download backups</td>
                    <td data-label="Viewer">✅</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Manage databases, notifiers, storages</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">✅</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Manage users</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">✅</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                  <tr>
                    <td>Manage admins</td>
                    <td data-label="Viewer">❌</td>
                    <td data-label="Member">❌</td>
                    <td data-label="Admin">❌</td>
                    <td data-label="Owner">✅</td>
                  </tr>
                </tbody>
              </table>

              <p>
                Keep in mind: <strong>sensitive data</strong> (passwords,
                tokens, etc.) of DBs, storages and notifiers{" "}
                <strong>is always hidden from any user</strong>. Nobody can see
                secrets after creation.
              </p>

              <h2 id="global-settings">Global settings</h2>

              <p>In global settings there are 3 properties:</p>

              <ol>
                <li>
                  <strong>Allow external registrations</strong> - by default,
                  all users can sign up for your Databasus (but they still do
                  not have access to any workspaces until they are invited or
                  create their own workspaces).
                  <br />
                  <br />
                  If you want to allow only invited users to sign up, you can
                  disable this option. In this case, the sign up form will be
                  closed until you invite the user to any of workspaces.
                  <br />
                  <br />
                  To invite users to the workspace, you need to click &quot;Add
                  user&quot; and enter an email. After this, the user with this
                  email will be able to complete sign up.
                </li>
                <li>
                  <strong>Allow member invitations</strong> - this setting is
                  needed when external registrations are disabled.
                  <br />
                  <br />
                  Imagine you already have some users and you know they are
                  reliable (for example, your team). You want to allow them to
                  invite other users to join Databasus. In this case, you can
                  enable this option and they will be able to invite other users
                  to join workspaces via invitations.
                  <br />
                  <br />
                  If it is disabled, only admins can invite users.
                </li>
                <li>
                  <strong>Allow member workspace creation</strong> - by default,
                  all members can create their own workspaces. If you want to
                  allow only admins to create workspaces, you can disable this
                  option.
                </li>
              </ol>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
