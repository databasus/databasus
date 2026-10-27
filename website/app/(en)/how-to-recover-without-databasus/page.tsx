import type { Metadata } from "next";
import { getLanguageAlternates } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "How to restore from backup without Databasus?",
  description:
    "Learn how to manually restore your database backups without Databasus. No vendor lock-in: decrypt and restore your backups using standard tools and your secret key.",
  keywords: [
    "backup recovery",
    "manual restore",
    "decrypt backup",
    "no vendor lock-in",
    "AES-256-GCM decryption",
    "PostgreSQL restore",
    "MySQL restore",
    "MariaDB restore",
    "MongoDB restore",
    "backup without Databasus",
  ],
  openGraph: {
    title: "How to restore from backup without Databasus?",
    description:
      "Learn how to manually restore your database backups without Databasus. No vendor lock-in: decrypt and restore your backups using standard tools and your secret key.",
    type: "article",
    locale: "en_US",
    url: "https://databasus.com/how-to-recover-without-databasus",
  },
  twitter: {
    card: "summary",
    title: "How to restore from backup without Databasus?",
    description:
      "Learn how to manually restore your database backups without Databasus. No vendor lock-in: decrypt and restore your backups using standard tools and your secret key.",
  },
  alternates: {
    canonical: "https://databasus.com/how-to-recover-without-databasus",
    languages: getLanguageAlternates("how-to-recover-without-databasus"),
  },
  robots: "index, follow",
};

export default function ManualRecoveryPage() {
  const pythonScript = `import json
import base64
import struct
import os
from Crypto.Cipher import AES
from Crypto.Protocol.KDF import PBKDF2
from Crypto.Hash import SHA256

# Constants from Databasus encryption
MAGIC_BYTES = b"PGRSUS01"
HEADER_LENGTH = 64
CHUNK_SIZE = 1024 * 1024
PBKDF2_ITERATIONS = 100000


def decrypt_backup(backup_file, metadata_file, master_key):
    """
    Decrypt a Databasus backup file using metadata and master key.

    Args:
        backup_file: Path to encrypted backup file
        metadata_file: Path to metadata JSON file
        master_key: Master key from ./databasus-data/secret.key
    """
    # Validate files exist
    if not os.path.exists(backup_file):
        print(f"Error: Backup file not found: {backup_file}")
        return

    if not os.path.exists(metadata_file):
        print(f"Error: Metadata file not found: {metadata_file}")
        return

    # Read metadata
    with open(metadata_file, "r") as f:
        metadata = json.load(f)

    # Check if file is encrypted (case-insensitive check)
    encryption_status = metadata.get("encryption", "").upper()
    if encryption_status != "ENCRYPTED":
        print(
            f"Error: Backup is not encrypted (encryption status: {metadata.get('encryption')})"
        )
        print("No decryption needed. You can decompress/restore the file directly.")
        return

    backup_id = metadata["backupId"]
    salt = base64.b64decode(metadata["encryptionSalt"])
    iv = base64.b64decode(metadata["encryptionIV"])

    # Generate output filename with decrypted_ prefix
    backup_dir = os.path.dirname(backup_file) or "."
    backup_name = os.path.basename(backup_file)
    output_file = os.path.join(backup_dir, f"decrypted_{backup_name}")

    # Derive encryption key using PBKDF2
    key_material = (master_key + backup_id).encode("utf-8")
    derived_key = PBKDF2(
        key_material, salt, dkLen=32, count=PBKDF2_ITERATIONS, hmac_hash_module=SHA256
    )

    try:
        with open(backup_file, "rb") as f_in, open(output_file, "wb") as f_out:
            # Read and validate header
            header = f_in.read(HEADER_LENGTH)

            # Validate magic bytes
            magic = header[:8]
            if magic != MAGIC_BYTES:
                raise ValueError(
                    f"Invalid magic bytes: expected {MAGIC_BYTES}, got {magic}"
                )

            # Decrypt chunks
            chunk_index = 0
            while True:
                # Read chunk length (4 bytes)
                length_bytes = f_in.read(4)
                if not length_bytes:
                    break

                chunk_length = struct.unpack(">I", length_bytes)[0]

                # Read encrypted chunk
                encrypted_chunk = f_in.read(chunk_length)
                if not encrypted_chunk:
                    break

                # Generate chunk nonce (base IV + chunk index)
                chunk_nonce = bytearray(iv)
                chunk_nonce[4:12] = struct.pack(">Q", chunk_index)

                # Create cipher for this chunk
                chunk_cipher = AES.new(derived_key, AES.MODE_GCM, nonce=bytes(chunk_nonce))

                # Decrypt chunk
                try:
                    decrypted_chunk = chunk_cipher.decrypt_and_verify(
                        encrypted_chunk[:-16],  # ciphertext
                        encrypted_chunk[-16:],  # auth tag
                    )
                except ValueError as e:
                    if "MAC check failed" in str(e):
                        print("\\nError: Failed to decrypt backup (MAC check failed)")
                        print("This usually means:")
                        print("  - The master key is incorrect")
                        print("  - The backup file is corrupted")
                        print("  - The metadata doesn't match this backup file")
                        print(f"\\nFailed at chunk {chunk_index}")
                        raise
                    raise

                # Write decrypted data
                f_out.write(decrypted_chunk)
                chunk_index += 1

        print(f"Successfully decrypted {chunk_index} chunks to {output_file}")
        
    except ValueError as e:
        # Clean up partial output file after files are closed
        if "MAC check failed" in str(e) and os.path.exists(output_file):
            os.remove(output_file)
        return


# Example usage:
if __name__ == "__main__":
    decrypt_backup(
        backup_file="./your-backup-file",             # <--- change this to your backup file
        metadata_file="./your-backup-file.metadata",  # <--- change this to your metadata file
        master_key="your-master-key-here",            # <--- change this to your master key
    )`;

  const postgresqlRestore = `# Restore to local database
pg_restore -d your_database decrypted-backup.dump`;

  const postgresqlRestoreRemote = `# Restore to remote database
pg_restore -h hostname -p 5432 -U username -d database_name decrypted-backup.dump`;

  const mysqlDecompress = `# Decompress with zstd command-line tool
zstd -d decrypted-backup.sql.zst -o decrypted-backup.sql

# Or use graphical tools like 7-Zip, PeaZip, or WinRAR`;

  const mysqlRestore = `# Restore to local database
mysql your_database < decrypted-backup.sql`;

  const mysqlRestoreRemote = `# Restore to remote database
mysql -h hostname -P 3306 -u username -p database_name < decrypted-backup.sql`;

  const mariadbDecompress = `# Decompress with zstd command-line tool
zstd -d decrypted-backup.sql.zst -o decrypted-backup.sql

# Or use graphical tools like 7-Zip, PeaZip, or WinRAR`;

  const mariadbRestore = `# Restore to local database
mariadb your_database < decrypted-backup.sql`;

  const mariadbRestoreRemote = `# Restore to remote database
mariadb -h hostname -P 3306 -u username -p database_name < decrypted-backup.sql`;

  const mongodbRestore = `# Restore to local database
mongorestore --archive=decrypted-backup.archive --gzip --db your_database`;

  const mongodbRestoreRemote = `# Restore to remote database
mongorestore --host hostname:27017 --username username --password password \\
  --archive=decrypted-backup.archive --gzip --db database_name`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "How to restore from backup without Databasus?",
            description:
              "Learn how to manually restore your database backups without Databasus. No vendor lock-in: decrypt and restore your backups using standard tools and your secret key.",
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
            name: "How to restore from backup without Databasus?",
            description:
              "Step-by-step guide to manually restore database backups without Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Download backup files",
                text: "Download both the backup file and metadata file from your storage",
              },
              {
                "@type": "HowToStep",
                name: "Decrypt the backup",
                text: "Use the Python script to decrypt the backup file using your master key",
              },
              {
                "@type": "HowToStep",
                name: "Decompress if needed",
                text: "For MySQL and MariaDB, decompress the backup file using zstd",
              },
              {
                "@type": "HowToStep",
                name: "Restore to database",
                text: "Use database-specific tools to restore the decrypted backup",
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
              <h1 id="manual-recovery">Manual recovery from backup without Databasus</h1>

              <p className="text-lg text-gray-400">
                Backing up is not only about data protection. It&apos;s also
                about data recovery. Databasus draw attention to keep your
                backups recoverable even if VPS with Databasus is deleted, you
                lost access or cannot access UI for some reason. So you
                don&apos;t need Databasus to recover backups, because they
                stored in standard format and there is no vendor lock-in.
              </p>

              <h2 id="what-you-need">What you need</h2>

              <p>To manually recover a backup, you need:</p>

              <ul>
                <li>
                  <strong>Backup file</strong> from your storage (local storage,
                  S3, Google Drive, etc.)
                </li>
                <li>
                  <strong>Metadata file</strong> from the same storage. It named
                  in the same way as the backup file, but with the{" "}
                  <code>.metadata</code> extension.
                </li>
                <li>
                  <strong>Secret key</strong> from{" "}
                  <code>./databasus-data/secret.key</code> (located in the same
                  directory as the backup files, usually{" "}
                  <code>/opt/databasus/</code>)
                </li>
              </ul>

              <h2 id="file-structure">File structure</h2>

              <p>
                Each backup consists of two files stored in your storage (local
                or cloud):
              </p>

              <ul>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}`}</code> -
                  Encrypted and compressed backup data
                </li>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}.metadata`}</code>{" "}
                  - JSON file with encryption details
                </li>
              </ul>

              <p>
                The metadata file contains encryption salt and IV (nonce) in
                Base64 format:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    {`{
  "backupId": "550e8400-e29b-41d4-a716-446655440000",
  "encryptionSalt": "base64-encoded-salt",
  "encryptionIV": "base64-encoded-nonce",
  "encryption": "encrypted"
}`}
                  </code>
                </pre>
              </div>

              <h2 id="decryption">Decryption</h2>

              <p>
                Databasus uses <strong>AES-256-GCM</strong> encryption with{" "}
                <strong>PBKDF2</strong> key derivation. Each backup has a unique
                encryption key derived from:
              </p>

              <ul>
                <li>Master key (from secret.key file)</li>
                <li>Backup ID</li>
                <li>Random salt (stored in metadata)</li>
              </ul>

              <p>Use this Python script to decrypt your backup:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pythonScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={pythonScript} />
                </div>
              </div>

              <p>Install required dependencies:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pip install pycryptodome</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="pip install pycryptodome" />
                </div>
              </div>

              <p>
                <strong>How to use the script:</strong>
              </p>

              <ol>
                <li>
                  Save the script above to a file (e.g.,{" "}
                  <code>decrypt_backup.py</code>)
                </li>
                <li>
                  Update the parameters in the example usage section at the
                  bottom
                </li>
                <li>Run the script:</li>
              </ol>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>python decrypt_backup.py</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="python decrypt_backup.py" />
                </div>
              </div>

              <p>
                The script will automatically create the output file with a{" "}
                <code>decrypted_</code> prefix. For example, if your backup file
                is <code>backup-id.dump</code>, the decrypted file will be{" "}
                <code>decrypted_backup-id.dump</code>.
              </p>

              <h2 id="restore">Restore to database</h2>

              <p>After decryption, restore using database-specific tools:</p>

              <h3 id="postgresql-restore">PostgreSQL</h3>

              <p>
                PostgreSQL backups use built-in compression and can be restored
                directly:
              </p>

              <p>
                <strong>Local database:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlRestore} />
                </div>
              </div>

              <p>
                <strong>Remote database:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mysql-restore">MySQL</h3>

              <p>
                MySQL backups are compressed with zstd level 5 and must be
                decompressed before restoring.
              </p>

              <p>
                <strong>Step 1: Decompress the backup</strong>
              </p>

              <p>
                Use the zstd command-line tool or any compatible decompression
                tool (7-Zip, PeaZip, WinRAR, etc.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlDecompress} />
                </div>
              </div>

              <p>
                <strong>Step 2: Restore to database</strong>
              </p>

              <p>Local database:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlRestore} />
                </div>
              </div>

              <p>Remote database:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mariadb-restore">MariaDB</h3>

              <p>
                MariaDB backups are compressed with zstd level 5 and must be
                decompressed before restoring.
              </p>

              <p>
                <strong>Step 1: Decompress the backup</strong>
              </p>

              <p>
                Use the zstd command-line tool or any compatible decompression
                tool (7-Zip, PeaZip, WinRAR, etc.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbDecompress} />
                </div>
              </div>

              <p>
                <strong>Step 2: Restore to database</strong>
              </p>

              <p>Local database:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbRestore} />
                </div>
              </div>

              <p>Remote database:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbRestoreRemote} />
                </div>
              </div>

              <h3 id="mongodb-restore">MongoDB</h3>

              <p>
                MongoDB backups use built-in gzip compression and can be
                restored directly:
              </p>

              <p>
                <strong>Local database:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mongodbRestore} />
                </div>
              </div>

              <p>
                <strong>Remote database:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mongodbRestoreRemote} />
                </div>
              </div>

              <h2 id="what-if-i-have-issues">What if I have issues?</h2>

              <p>If you encounter any problems during the recovery process:</p>

              <ul>
                <li>
                  <strong>Ask AI for help</strong>. AI assistants like ChatGPT,
                  Claude or Gemini are excellent at helping with compression
                  tools and database restore procedures. Simply describe your
                  issue and they can guide you through the process.
                </li>
                <li>
                  <strong>
                    Join our{" "}
                    <a
                      href="https://t.me/databasus_community"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      community
                    </a>
                  </strong>
                  . Our developers and community members can help with your
                  particular case.
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
