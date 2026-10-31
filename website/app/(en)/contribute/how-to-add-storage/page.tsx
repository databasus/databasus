import type { Metadata } from "next";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "How to add new storage to Databasus | Contribution Guide",
  description:
    "Developer guide for contributing new storage integrations to Databasus. Learn how to implement backend models, create migrations and build frontend UI components for storage providers.",
  keywords: [
    "Databasus contribution",
    "add storage",
    "developer guide",
    "storage integration",
    "PostgreSQL backup",
    "open source contribution",
    "GORM",
    "React components",
    "storage backend",
  ],
  openGraph: {
    title: "How to add new storage to Databasus | Contribution Guide",
    description:
      "Developer guide for contributing new storage integrations to Databasus. Learn how to implement backend models, create migrations and build frontend UI components for storage providers.",
    type: "article",
    url: "https://databasus.com/contribute/how-to-add-storage",
  },
  twitter: {
    card: "summary",
    title: "How to add new storage to Databasus | Contribution Guide",
    description:
      "Developer guide for contributing new storage integrations to Databasus. Learn how to implement backend models, create migrations and build frontend UI components for storage providers.",
  },
  alternates: {
    canonical: "https://databasus.com/contribute/how-to-add-storage",
  },
  robots: "index, follow",
};

export default function HowToAddStoragePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "How to add new storage to Databasus",
            description:
              "Developer guide for contributing new storage integrations to Databasus",
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
              <h1 id="how-to-add-storage">
                How to add new storage to Databasus
              </h1>

              <p className="text-lg text-gray-400">
                This guide will walk you through the process of contributing a
                new storage integration to Databasus. You&apos;ll learn how to
                implement the backend logic, create database migrations and
                build the frontend UI components for managing storage providers.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Note:</strong> This is a
                  contribution guide for developers who want to add new storage
                  integrations to the Databasus project. If you only want to use
                  existing storage providers, check out the{" "}
                  <a
                    href="/storages"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Storages documentation
                  </a>
                  .
                </p>
              </div>

              <h2 id="backend-implementation">Backend implementation</h2>

              <p>
                The backend implementation involves creating models, updating
                enums and setting up database migrations for your storage
                provider.
              </p>

              <h3 id="step-1-create-model">1. Create new storage model</h3>

              <p>
                Create a new model in{" "}
                <code>
                  backend/internal/features/storages/models/&#123;storage_name&#125;/
                </code>{" "}
                folder. Your model must implement the{" "}
                <code>StorageFileSaver</code> interface from the parent folder.
              </p>

              <p>The interface requires the following methods:</p>

              <ul>
                <li>
                  <code>
                    SaveFile(ctx context.Context, encryptor
                    encryption.FieldEncryptor, logger *slog.Logger, fileID
                    uuid.UUID, file io.Reader) error
                  </code>{" "}
                  - saves a backup file to the storage
                </li>
                <li>
                  <code>
                    GetFile(encryptor encryption.FieldEncryptor, fileID
                    uuid.UUID) (io.ReadCloser, error)
                  </code>{" "}
                  - retrieves a backup file from the storage
                </li>
                <li>
                  <code>
                    DeleteFile(encryptor encryption.FieldEncryptor, fileID
                    uuid.UUID) error
                  </code>{" "}
                  - deletes a backup file from the storage
                </li>
                <li>
                  <code>
                    Validate(encryptor encryption.FieldEncryptor) error
                  </code>{" "}
                  - validates the storage configuration
                </li>
                <li>
                  <code>
                    TestConnection(encryptor encryption.FieldEncryptor) error
                  </code>{" "}
                  - tests connectivity to the storage provider
                </li>
                <li>
                  <code>HideSensitiveData()</code> - masks sensitive fields like
                  API keys before logging/display
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
                  <strong className="text-amber-400">
                    ⚠️ Context cancellation:
                  </strong>{" "}
                  Your{" "}
                  <code className="bg-[#374151] text-gray-200">SaveFile</code>{" "}
                  implementation must respect the{" "}
                  <code className="bg-[#374151] text-gray-200">ctx</code>{" "}
                  context for cancellation. Check{" "}
                  <code className="bg-[#374151] text-gray-200">ctx.Done()</code>{" "}
                  periodically during upload operations and return early with an
                  appropriate error when the context is cancelled. This ensures
                  graceful shutdown and backup cancellation work correctly.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-purple-400">
                    📦 Backpressure handling:
                  </strong>{" "}
                  When implementing{" "}
                  <code className="bg-[#374151] text-gray-200">SaveFile</code>,
                  read from the{" "}
                  <code className="bg-[#374151] text-gray-200">io.Reader</code>{" "}
                  in chunks of up to{" "}
                  <strong className="text-white">32 MB</strong> and upload them
                  incrementally. This prevents memory exhaustion on large
                  backups and respects backpressure from the upstream pipe.
                  Avoid reading the entire file into memory at once.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-green-400">
                    🔐 Encryption requirement:
                  </strong>{" "}
                  All sensitive fields (API keys, passwords, access tokens,
                  secrets) must be encrypted using the{" "}
                  <code className="bg-[#374151] text-gray-200">
                    EncryptSensitiveData()
                  </code>{" "}
                  method before saving to the database. Decrypt credentials
                  before using them. See{" "}
                  <code className="bg-[#374151] text-gray-200">azure_blob</code>
                  ,{" "}
                  <code className="bg-[#374151] text-gray-200">
                    google_drive
                  </code>{" "}
                  or <code className="bg-[#374151] text-gray-200">s3</code>{" "}
                  storage models for implementation examples.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong>🔑 Important:</strong> Use UUID primary key as{" "}
                  <code>StorageID</code> that references the main storages
                  table. This maintains referential integrity across the
                  database. Also add a <code>TableName() string</code> method to
                  return the proper table name.
                </p>
              </div>

              <h3 id="step-2-update-enums">2. Add storage type to enums</h3>

              <p>
                Add your new storage type to{" "}
                <code>backend/internal/features/storages/enums.go</code> in the{" "}
                <code>StorageType</code> constants section. This enum is used
                throughout the application to identify different storage types.
              </p>

              <h3 id="step-3-update-main-model">
                3. Update the main Storage model
              </h3>

              <p>
                Modify <code>backend/internal/features/storages/model.go</code>{" "}
                to integrate your new storage:
              </p>

              <ul>
                <li>
                  Add a new field with GORM foreign key relation to your storage
                  model
                </li>
                <li>
                  Update the <code>getSpecificStorage()</code> method to handle
                  your new type
                </li>
                <li>
                  Update the <code>SaveFile()</code>, <code>GetFile()</code>,
                  and <code>DeleteFile()</code> methods to route to your storage
                  implementation
                </li>
                <li>
                  Update the <code>Validate()</code> method to include your
                  storage validation
                </li>
              </ul>

              <h3 id="step-4-add-env-variables">
                4. Add environment variables (if needed)
              </h3>

              <p>
                If your storage requires environment variables for testing, add
                them to <code>backend/internal/config/config.go</code>. This
                allows the test suite to access necessary configuration values
                such as API keys or credentials.
              </p>

              <h3 id="step-5-docker-setup">
                5. Configure Docker containers (if needed)
              </h3>

              <p>
                If your storage needs external services for testing (like MinIO
                for S3-compatible storage or FTP servers), add them to{" "}
                <code>backend/docker-compose.yml.example</code>. Keep sensitive
                data blank in the example file.
              </p>

              <h3 id="step-6-github-actions">
                6. Add CI/CD secrets (if needed)
              </h3>

              <p>
                For sensitive environment variables needed in the GitHub Actions
                pipeline (like cloud storage API keys or credentials), contact{" "}
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
                  Create a table with <code>storage_id</code> as UUID primary
                  key
                </li>
                <li>
                  Add a foreign key constraint to the <code>storages</code>{" "}
                  table with <code>CASCADE DELETE</code>
                </li>
                <li>
                  Include all necessary columns for your storage configuration
                  (endpoints, credentials, bucket names, etc.)
                </li>
                <li>
                  Look at existing storage migrations (e.g., Google Drive,
                  Cloudflare R2) for reference
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] pt-4 px-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong>💡 Tip:</strong> The <code>CASCADE DELETE</code>{" "}
                  ensures that when a storage is deleted from the main storages
                  table, the related configuration is automatically removed.
                </p>
              </div>

              <h3 id="step-8-update-tests">8. Update storage tests</h3>

              <p>
                Update tests in{" "}
                <code>backend/internal/features/storages/model_test.go</code> to
                include tests for your new storage. Your tests should verify:
              </p>

              <ul>
                <li>Successful file upload to storage</li>
                <li>File retrieval from storage</li>
                <li>File deletion from storage</li>
                <li>Connection testing</li>
                <li>Validation of configuration parameters</li>
              </ul>

              <p>
                Additionally, update{" "}
                <code>
                  backend/internal/features/storages/controller_test.go
                </code>{" "}
                to add encryption verification tests:
              </p>

              <ul>
                <li>
                  Add a test case to{" "}
                  <code>Test_StorageSensitiveDataLifecycle_AllTypes</code> for
                  your storage type
                </li>
                <li>
                  Add a test case to{" "}
                  <code>
                    Test_CreateStorage_AllSensitiveFieldsEncryptedInDB
                  </code>{" "}
                  (if not already present)
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

              <h3 id="step-9-run-tests">9. Run all tests</h3>

              <p>
                Before submitting your pull request, make sure all existing
                tests pass and your new tests are working correctly. This
                ensures your storage integration doesn&apos;t break existing
                functionality.
              </p>

              <h2 id="frontend-implementation">Frontend implementation</h2>

              <p>
                The frontend work involves creating TypeScript models,
                validators and React components for managing your storage
                provider.
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

              <h3 id="step-10-create-models">
                1. Add TypeScript models and API
              </h3>

              <p>
                Create your storage model and API integration in{" "}
                <code>
                  frontend/src/entities/storages/models/&#123;storage_name&#125;/
                </code>
                . Update the <code>index.ts</code> file to export your new model
                for use throughout the application.
              </p>

              <p>Your model should include:</p>

              <ul>
                <li>TypeScript interface matching your backend model</li>
                <li>
                  Validation function to ensure all required fields are present
                </li>
                <li>
                  Any helper functions specific to your storage configuration
                </li>
              </ul>

              <h3 id="step-11-add-icon">2. Upload icon and update utilities</h3>

              <p>Complete the following steps to add visual assets:</p>

              <ol>
                <li>
                  Upload an SVG icon to <code>public/icons/storages/</code>
                </li>
                <li>
                  Update{" "}
                  <code>
                    src/entities/storages/models/getStorageLogoFromType.ts
                  </code>{" "}
                  to return your icon path
                </li>
                <li>
                  Update{" "}
                  <code>src/entities/storages/models/StorageType.ts</code> to
                  include your new storage type constant
                </li>
                <li>
                  Update{" "}
                  <code>
                    src/entities/storages/models/getStorageNameFromType.ts
                  </code>{" "}
                  to return the display name for your storage
                </li>
              </ol>

              <h3 id="step-12-create-components">3. Build UI components</h3>

              <p>Create two main React components:</p>

              <ul>
                <li>
                  <code>
                    src/features/edit-storage/ui/edit/storages/Edit&#123;StorageName&#125;Component.tsx
                  </code>{" "}
                  - for creating and editing storage configuration
                </li>
                <li>
                  <code>
                    src/pages/storages/ui/show/storages/Show&#123;StorageName&#125;Component.tsx
                  </code>{" "}
                  - for displaying storage details in read-only mode
                </li>
              </ul>

              <p>
                These components should handle form inputs, validation and data
                formatting specific to your storage provider. Include features
                like:
              </p>

              <ul>
                <li>Input fields for all required configuration parameters</li>
                <li>Test connection button to verify credentials</li>
                <li>Clear error messages for validation failures</li>
                <li>Masked display of sensitive information</li>
              </ul>

              <h3 id="step-13-integrate-components">
                4. Integrate with main components
              </h3>

              <p>
                Update the main storage management components to handle your new
                type:
              </p>

              <ul>
                <li>
                  <code>EditStorageComponent.tsx</code> - add import and
                  component rendering for your storage
                </li>
                <li>
                  <code>ShowStorageComponent.tsx</code> - add import and
                  component rendering to display your storage
                </li>
              </ul>

              <p>
                These updates enable the application to dynamically render the
                correct component based on the storage type selected by the
                user.
              </p>

              <h3 id="step-14-test-frontend">5. Test the implementation</h3>

              <p>
                Thoroughly test your implementation to ensure everything works
                as expected:
              </p>

              <ul>
                <li>Create a new storage of your type</li>
                <li>Edit existing storage configurations</li>
                <li>Test connection to verify credentials</li>
                <li>Upload a backup file to the storage</li>
                <li>Download a backup file from the storage</li>
                <li>Delete a backup file from the storage</li>
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
                  Test the integration end-to-end with actual backup operations
                </li>
                <li>
                  Submit a pull request with a description of your changes and
                  any special considerations
                </li>
              </ol>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-green-400">🎉 Thank you!</strong> Your
                  contributions help make Databasus more versatile and valuable
                  for the community. We appreciate your effort in expanding the
                  storage capabilities!
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
