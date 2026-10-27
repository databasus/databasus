import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Cara restore dari backup tanpa Databasus",
  description:
    "Pelajari cara restore backup database secara manual tanpa Databasus. Tanpa ketergantungan pada vendor: dekripsi dan restore backup dengan tool standar dan secret key milikmu.",
  keywords: [
    "pemulihan backup",
    "restore manual",
    "dekripsi backup",
    "tanpa ketergantungan vendor",
    "dekripsi AES-256-GCM",
    "restore PostgreSQL",
    "restore MySQL",
    "restore MariaDB",
    "restore MongoDB",
    "backup tanpa Databasus",
  ],
  openGraph: {
    title: "Cara restore dari backup tanpa Databasus",
    description:
      "Pelajari cara restore backup database secara manual tanpa Databasus. Tanpa ketergantungan pada vendor: dekripsi dan restore backup dengan tool standar dan secret key milikmu.",
    type: "article",
    url: getLocalizedUrl("id", "how-to-recover-without-databasus"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Cara restore dari backup tanpa Databasus",
    description:
      "Pelajari cara restore backup database secara manual tanpa Databasus. Tanpa ketergantungan pada vendor: dekripsi dan restore backup dengan tool standar dan secret key milikmu.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "how-to-recover-without-databasus"),
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Cara restore dari backup tanpa Databasus",
            description:
              "Pelajari cara restore backup database secara manual tanpa Databasus. Tanpa ketergantungan pada vendor: dekripsi dan restore backup dengan tool standar dan secret key milikmu.",
            author: { "@type": "Organization", name: "Databasus" },
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
            name: "Cara restore dari backup tanpa Databasus",
            description:
              "Panduan langkah demi langkah untuk restore backup database secara manual tanpa Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Unduh file backup",
                text: "Unduh file backup beserta file metadata dari penyimpanan",
              },
              {
                "@type": "HowToStep",
                name: "Dekripsi backup",
                text: "Gunakan skrip Python untuk mendekripsi file backup dengan master key milikmu",
              },
              {
                "@type": "HowToStep",
                name: "Dekompresi jika diperlukan",
                text: "Untuk MySQL dan MariaDB, dekompresi file backup menggunakan zstd",
              },
              {
                "@type": "HowToStep",
                name: "Restore ke database",
                text: "Gunakan tool yang sesuai dengan database untuk restore backup yang sudah didekripsi",
              },
            ],
          }),
        }}
      />
      <DocsNavbarComponent lang="id" />
      <div className="flex min-h-screen bg-[#0F1115]">
        <DocsSidebarComponent lang="id" />
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="manual-recovery">Pemulihan manual dari backup tanpa Databasus</h1>
              <p className="text-lg text-gray-400">
                Backup bukan hanya soal melindungi data, tetapi juga memulihkannya. Databasus
                dirancang agar backup tetap bisa dipulihkan meskipun VPS yang menjalankan Databasus
                terhapus, kamu kehilangan akses, atau antarmukanya tidak bisa dibuka. Kamu tidak
                memerlukan Databasus untuk memulihkan backup karena file disimpan dalam format
                standar, tanpa ketergantungan pada vendor.
              </p>
              <h2 id="what-you-need">Yang perlu disiapkan</h2>
              <p>Untuk memulihkan backup secara manual, siapkan:</p>
              <ul>
                <li>
                  <strong>File backup</strong> dari penyimpanan (penyimpanan lokal, S3, Google
                  Drive, dan lainnya)
                </li>
                <li>
                  <strong>File metadata</strong> dari penyimpanan yang sama. Namanya sama dengan
                  file backup, tetapi menggunakan ekstensi <code>.metadata</code>.
                </li>
                <li>
                  <strong>Secret key</strong> dari <code>./databasus-data/secret.key</code> (berada
                  di direktori yang sama dengan file backup, biasanya <code>/opt/databasus/</code>)
                </li>
              </ul>
              <h2 id="file-structure">Struktur file</h2>
              <p>Setiap backup terdiri dari dua file di penyimpanan lokal atau cloud:</p>
              <ul>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}`}</code> - Data backup yang
                  dienkripsi dan dikompresi
                </li>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}.metadata`}</code> - File JSON
                  berisi detail enkripsi
                </li>
              </ul>
              <p>File metadata berisi salt enkripsi dan IV (nonce) dalam format Base64:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`{
  "backupId": "550e8400-e29b-41d4-a716-446655440000",
  "encryptionSalt": "base64-encoded-salt",
  "encryptionIV": "base64-encoded-nonce",
  "encryption": "encrypted"
}`}</code>
                </pre>
              </div>
              <h2 id="decryption">Dekripsi</h2>
              <p>
                Databasus menggunakan enkripsi <strong>AES-256-GCM</strong> dengan derivasi kunci{" "}
                <strong>PBKDF2</strong>. Setiap backup memiliki kunci enkripsi unik yang diturunkan
                dari:
              </p>
              <ul>
                <li>Master key (dari file secret.key)</li>
                <li>ID backup</li>
                <li>Salt acak (disimpan dalam metadata)</li>
              </ul>
              <p>Gunakan skrip Python berikut untuk mendekripsi backup:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pythonScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={pythonScript} lang="id" />
                </div>
              </div>
              <p>Instal dependency yang diperlukan:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pip install pycryptodome</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="pip install pycryptodome" lang="id" />
                </div>
              </div>
              <p>
                <strong>Cara menggunakan skrip:</strong>
              </p>
              <ol>
                <li>
                  Simpan skrip di atas ke sebuah file (misalnya <code>decrypt_backup.py</code>)
                </li>
                <li>Ubah parameter pada bagian contoh penggunaan di paling bawah</li>
                <li>Jalankan skrip:</li>
              </ol>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>python decrypt_backup.py</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="python decrypt_backup.py" lang="id" />
                </div>
              </div>
              <p>
                Skrip otomatis membuat file hasil dengan awalan <code>decrypted_</code>. Misalnya,
                jika file backup bernama <code>backup-id.dump</code>, file yang sudah didekripsi
                akan bernama <code>decrypted_backup-id.dump</code>.
              </p>
              <h2 id="restore">Restore ke database</h2>
              <p>Setelah dekripsi, lakukan restore menggunakan tool yang sesuai dengan database:</p>
              <h3 id="postgresql-restore">PostgreSQL</h3>
              <p>Backup PostgreSQL menggunakan kompresi bawaan dan bisa langsung di-restore:</p>
              <p>
                <strong>Database lokal:</strong>
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlRestore} lang="id" />
                </div>
              </div>
              <p>
                <strong>Database remote:</strong>
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlRestoreRemote} lang="id" />
                </div>
              </div>
              <h3 id="mysql-restore">MySQL</h3>
              <p>
                Backup MySQL dikompresi dengan zstd level 5 dan wajib didekompresi sebelum restore.
              </p>
              <p>
                <strong>Langkah 1: Dekompresi backup</strong>
              </p>
              <p>
                Gunakan tool command-line zstd atau tool dekompresi yang kompatibel (7-Zip, PeaZip,
                WinRAR, dan lainnya):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlDecompress} lang="id" />
                </div>
              </div>
              <p>
                <strong>Langkah 2: Restore ke database</strong>
              </p>
              <p>Database lokal:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlRestore} lang="id" />
                </div>
              </div>
              <p>Database remote:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlRestoreRemote} lang="id" />
                </div>
              </div>
              <h3 id="mariadb-restore">MariaDB</h3>
              <p>
                Backup MariaDB dikompresi dengan zstd level 5 dan wajib didekompresi sebelum
                restore.
              </p>
              <p>
                <strong>Langkah 1: Dekompresi backup</strong>
              </p>
              <p>
                Gunakan tool command-line zstd atau tool dekompresi yang kompatibel (7-Zip, PeaZip,
                WinRAR, dan lainnya):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbDecompress} lang="id" />
                </div>
              </div>
              <p>
                <strong>Langkah 2: Restore ke database</strong>
              </p>
              <p>Database lokal:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbRestore} lang="id" />
                </div>
              </div>
              <p>Database remote:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbRestoreRemote} lang="id" />
                </div>
              </div>
              <h3 id="mongodb-restore">MongoDB</h3>
              <p>Backup MongoDB menggunakan kompresi gzip bawaan dan bisa langsung di-restore:</p>
              <p>
                <strong>Database lokal:</strong>
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mongodbRestore} lang="id" />
                </div>
              </div>
              <p>
                <strong>Database remote:</strong>
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mongodbRestoreRemote} lang="id" />
                </div>
              </div>
              <h2 id="what-if-i-have-issues">Bagaimana jika ada kendala?</h2>
              <p>Jika menemui masalah selama pemulihan:</p>
              <ul>
                <li>
                  <strong>Minta bantuan AI</strong>. Asisten AI seperti ChatGPT, Claude, atau Gemini
                  bisa membantu penggunaan tool kompresi dan prosedur restore database. Jelaskan
                  kendalanya agar mereka bisa memandu prosesnya.
                </li>
                <li>
                  <strong>
                    Bergabung dengan{" "}
                    <a
                      href="https://t.me/databasus_community"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      komunitas
                    </a>
                  </strong>
                  . Developer dan anggota komunitas kami bisa membantu menangani kasusmu.
                </li>
              </ul>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
