import type { Metadata } from "next";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "How to add new notifier to Databasus | Contribution Guide",
  description:
    "Developer guide for contributing new notification integrations to Databasus. Learn how to implement backend models, create migrations and build frontend UI components.",
  keywords: [
    "Databasus contribution",
    "add notifier",
    "developer guide",
    "notification integration",
    "PostgreSQL backup",
    "open source contribution",
    "GORM",
    "React components",
  ],
  openGraph: {
    title: "How to add new notifier to Databasus | Contribution Guide",
    description:
      "Developer guide for contributing new notification integrations to Databasus. Learn how to implement backend models, create migrations and build frontend UI components.",
    type: "article",
    url: "https://databasus.com/contribute/how-to-add-notifier",
  },
  twitter: {
    card: "summary",
    title: "How to add new notifier to Databasus | Contribution Guide",
    description:
      "Developer guide for contributing new notification integrations to Databasus. Learn how to implement backend models, create migrations and build frontend UI components.",
  },
  alternates: {
    canonical: "https://databasus.com/contribute/how-to-add-notifier",
  },
  robots: "index, follow",
};

export default function HowToAddNotifierPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "How to add new notifier to Databasus",
            description:
              "Developer guide for contributing new notification integrations to Databasus",
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
              <h1 id="how-to-add-notifier">
                How to add new notifier to Databasus
              </h1>

              <p className="text-lg text-gray-400">
                This guide will walk you through the process of contributing a
                new notification integration to Databasus. You&apos;ll learn how
                to implement the backend logic, create database migrations and
                build the frontend UI components.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Note:</strong> This is a
                  contribution guide for developers who want to add new
                  notification integrations to the Databasus project. If you
                  only want to use existing notifiers, check out the{" "}
                  <a
                    href="/notifiers"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Notifiers documentation
                  </a>
                  .
                </p>
              </div>

              <h2 id="backend-implementation">Backend implementation</h2>

              <p>
                The backend implementation involves creating models, updating
                enums and setting up database migrations.
              </p>

              <h3 id="step-1-create-model">1. Create new notifier model</h3>

              <p>
                Create a new model in{" "}
                <code>
                  backend/internal/features/notifiers/models/&#123;notifier_name&#125;/
                </code>{" "}
                folder. Your model must implement the{" "}
                <code>NotificationSender</code> interface from the parent
                folder.
              </p>

              <p>The interface requires the following methods:</p>

              <ul>
                <li>
                  <code>
                    Send(encryptor encryption.FieldEncryptor, logger
                    *slog.Logger, heading string, message string) error
                  </code>{" "}
                  - sends the notification
                </li>
                <li>
                  <code>
                    Validate(encryptor encryption.FieldEncryptor) error
                  </code>{" "}
                  - validates the configuration
                </li>
                <li>
                  <code>HideSensitiveData()</code> - masks sensitive fields
                  before logging/display
                </li>
                <li>
                  <code>
                    EncryptSensitiveData(encryptor encryption.FieldEncryptor)
                    error
                  </code>{" "}
                  - encrypts sensitive fields before saving
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-green-400">
                    🔐 Encryption requirement:
                  </strong>{" "}
                  All sensitive fields (bot tokens, webhook URLs, API keys,
                  passwords) must be encrypted using the{" "}
                  <code className="bg-[#374151] text-gray-200">
                    EncryptSensitiveData()
                  </code>{" "}
                  method before saving to the database. Decrypt credentials in
                  the <code className="bg-[#374151] text-gray-200">Send()</code>{" "}
                  method before sending notifications. See{" "}
                  <code className="bg-[#374151] text-gray-200">discord</code>,{" "}
                  <code className="bg-[#374151] text-gray-200">telegram</code>,{" "}
                  <code className="bg-[#374151] text-gray-200">slack</code>, or{" "}
                  <code className="bg-[#374151] text-gray-200">teams</code>{" "}
                  notifier models for implementation examples.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong>🔑 Important:</strong> Use UUID primary key as{" "}
                  <code>NotifierID</code> that references the main notifiers
                  table. This maintains referential integrity across the
                  database.
                </p>
              </div>

              <h3 id="step-2-update-enums">2. Add notifier type to enums</h3>

              <p>
                Add your new notifier type to{" "}
                <code>backend/internal/features/notifiers/enums.go</code> in the{" "}
                <code>NotifierType</code> constants section. This enum is used
                throughout the application to identify different notifier types.
              </p>

              <h3 id="step-3-update-main-model">
                3. Update the main Notifier model
              </h3>

              <p>
                Modify <code>backend/internal/features/notifiers/model.go</code>{" "}
                to integrate your new notifier:
              </p>

              <ul>
                <li>
                  Add a new field with GORM foreign key relation to your
                  notifier model
                </li>
                <li>
                  Update the <code>getSpecificNotifier()</code> method to handle
                  your new type
                </li>
                <li>
                  Update the <code>Send()</code> method to route notifications
                  to your implementation
                </li>
              </ul>

              <h3 id="step-4-add-env-variables">
                4. Add environment variables (if needed)
              </h3>

              <p>
                If your notifier requires environment variables for testing, add
                them to <code>backend/internal/config/config.go</code>. This
                allows the test suite to access necessary configuration values.
              </p>

              <h3 id="step-5-docker-setup">
                5. Configure Docker containers (if needed)
              </h3>

              <p>
                If your notifier needs external services for testing, add them
                to <code>backend/docker-compose.yml.example</code>. Keep
                sensitive data blank in the example file.
              </p>

              <h3 id="step-6-github-actions">
                6. Add CI/CD secrets (if needed)
              </h3>

              <p>
                For sensitive environment variables needed in the GitHub Actions
                pipeline (like API keys or credentials), contact{" "}
                <strong>@rostislav_dugin</strong> to add them securely to the
                CI/CD environment.
              </p>

              <h3 id="step-7-create-migration">7. Create database migration</h3>

              <p>
                Create a new migration file in the{" "}
                <code>backend/migrations</code> folder:
              </p>

              <ul>
                <li>
                  Create a table with <code>notifier_id</code> as UUID primary
                  key
                </li>
                <li>
                  Add a foreign key constraint to the <code>notifiers</code>{" "}
                  table with <code>CASCADE DELETE</code>
                </li>
                <li>
                  Include all necessary columns for your notifier configuration
                </li>
                <li>
                  Look at existing notifier migrations (e.g., Slack, Teams) for
                  reference
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong>💡 Tip:</strong> The <code>CASCADE DELETE</code>{" "}
                  ensures that when a notifier is deleted from the main
                  notifiers table, the related configuration is automatically
                  removed.
                </p>
              </div>

              <h3 id="step-8-run-tests">8. Update and run tests</h3>

              <p>
                Update{" "}
                <code>
                  backend/internal/features/notifiers/controller_test.go
                </code>{" "}
                to add encryption verification tests for your notifier:
              </p>

              <ul>
                <li>
                  Add a test case to{" "}
                  <code>Test_NotifierSensitiveDataLifecycle_AllTypes</code> for
                  your notifier type
                </li>
                <li>
                  Add a test case to{" "}
                  <code>
                    Test_CreateNotifier_AllSensitiveFieldsEncryptedInDB
                  </code>
                </li>
                <li>
                  Verify that sensitive fields are encrypted with{" "}
                  <code>enc:</code> prefix in the database
                </li>
                <li>
                  Verify that decryption returns the original plaintext values
                </li>
                <li>
                  Verify that sensitive data is hidden when retrieved via API
                </li>
              </ul>

              <p>
                Before submitting your pull request, make sure all existing
                tests pass and your new encryption tests are working correctly.
              </p>

              <h2 id="frontend-implementation">Frontend implementation</h2>

              <p>
                The frontend work involves creating TypeScript models,
                validators and React components for managing your notifier.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ℹ️ Backend-only contributions:
                  </strong>{" "}
                  If you&apos;re only comfortable with backend development,
                  that&apos;s perfectly fine! Complete the backend part and
                  contact{" "}
                  <strong className="text-white">@rostislav_dugin</strong> to
                  help with the UI implementation.
                </p>
              </div>

              <h3 id="step-9-create-models">
                1. Add TypeScript models and validators
              </h3>

              <p>
                Create your notifier model and validator in{" "}
                <code>
                  frontend/src/entities/notifiers/models/&#123;notifier_name&#125;/
                </code>
                . Update the <code>index.ts</code> file to export your new model
                for use throughout the application.
              </p>

              <h3 id="step-10-add-icon">2. Upload icon and update utilities</h3>

              <p>Complete the following steps to add visual assets:</p>

              <ol>
                <li>
                  Upload an SVG icon to <code>public/icons/notifiers/</code>
                </li>
                <li>
                  Update{" "}
                  <code>
                    src/entities/notifiers/models/getNotifierLogoFromType.ts
                  </code>{" "}
                  to return your icon path
                </li>
                <li>
                  Update{" "}
                  <code>src/entities/notifiers/models/NotifierType.ts</code> to
                  include your new notifier type constant
                </li>
                <li>
                  Update{" "}
                  <code>
                    src/entities/notifiers/models/getNotifierNameFromType.ts
                  </code>{" "}
                  to return the display name for your notifier
                </li>
              </ol>

              <h3 id="step-11-create-components">3. Build UI components</h3>

              <p>Create two main React components:</p>

              <ul>
                <li>
                  <code>
                    src/features/edit-notifier/ui/edit/notifiers/Edit&#123;NotifierName&#125;Component.tsx
                  </code>{" "}
                  - for creating and editing notifier configuration
                </li>
                <li>
                  <code>
                    src/pages/notifiers/ui/show/notifier/Show&#123;NotifierName&#125;Component.tsx
                  </code>{" "}
                  - for displaying notifier details in read-only mode
                </li>
              </ul>

              <p>
                These components should handle form inputs, validation and data
                formatting specific to your notifier.
              </p>

              <h3 id="step-12-integrate-components">
                4. Integrate with main components
              </h3>

              <p>
                Update the main notifier management components to handle your
                new type:
              </p>

              <ul>
                <li>
                  <code>EditNotifierComponent.tsx</code> - add import,
                  validation function and component rendering for your notifier
                </li>
                <li>
                  <code>ShowNotifierComponent.tsx</code> - add import and
                  component rendering to display your notifier
                </li>
              </ul>

              <p>
                These updates enable the application to dynamically render the
                correct component based on the notifier type selected.
              </p>

              <h3 id="step-13-test-frontend">5. Test the implementation</h3>

              <p>
                Thoroughly test your implementation to ensure everything works
                as expected:
              </p>

              <ul>
                <li>Create a new notifier of your type</li>
                <li>Edit existing notifier configurations</li>
                <li>Send test notifications</li>
                <li>Verify validation works correctly</li>
                <li>Check that sensitive data is properly masked</li>
                <li>Test the delete functionality</li>
              </ul>

              <h2 id="submission-guidelines">Submission guidelines</h2>

              <p>When you&apos;re ready to contribute your work:</p>

              <ol>
                <li>
                  Fork the{" "}
                  <a
                    href="https://github.com/databasus/databasus"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Databasus repository
                  </a>
                </li>
                <li>Create a feature branch with a descriptive name</li>
                <li>Commit your changes with clear commit messages</li>
                <li>Ensure all tests pass</li>
                <li>
                  Submit a pull request with a description of your changes
                </li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-green-400">🎉 Thank you!</strong> Your
                  contributions help make Databasus more versatile and valuable
                  for the community. We appreciate your effort in expanding the
                  notification capabilities!
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/contribute"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Back to contribution guides
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
