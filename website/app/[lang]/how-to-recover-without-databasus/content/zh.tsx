import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "不使用 Databasus 如何从备份恢复？",
  description:
    "了解如何在不使用 Databasus 的情况下手动恢复数据库备份。没有供应商锁定：使用标准工具和你的密钥即可解密并恢复备份。",
  keywords: [
    "从备份恢复",
    "手动恢复",
    "解密备份",
    "无供应商锁定",
    "AES-256-GCM 解密",
    "PostgreSQL 恢复",
    "MySQL 恢复",
    "MariaDB 恢复",
    "MongoDB 恢复",
    "不使用 Databasus 的备份",
  ],
  openGraph: {
    title: "不使用 Databasus 如何从备份恢复？",
    description:
      "了解如何在不使用 Databasus 的情况下手动恢复数据库备份。没有供应商锁定：使用标准工具和你的密钥即可解密并恢复备份。",
    type: "article",
    url: getLocalizedUrl("zh", "how-to-recover-without-databasus"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "不使用 Databasus 如何从备份恢复？",
    description:
      "了解如何在不使用 Databasus 的情况下手动恢复数据库备份。没有供应商锁定：使用标准工具和你的密钥即可解密并恢复备份。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "how-to-recover-without-databasus"),
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
            headline: "不使用 Databasus 如何从备份恢复？",
            description:
              "了解如何在不使用 Databasus 的情况下手动恢复数据库备份。没有供应商锁定：使用标准工具和你的密钥即可解密并恢复备份。",
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
            name: "不使用 Databasus 如何从备份恢复？",
            description: "不依赖 Databasus 手动恢复数据库备份的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "下载备份文件",
                text: "从你的存储中下载备份文件和元数据文件",
              },
              {
                "@type": "HowToStep",
                name: "解密备份",
                text: "使用 Python 脚本和你的主密钥解密备份文件",
              },
              {
                "@type": "HowToStep",
                name: "按需解压",
                text: "MySQL 和 MariaDB 的备份需要先用 zstd 解压",
              },
              {
                "@type": "HowToStep",
                name: "恢复到数据库",
                text: "使用各数据库自带的工具恢复解密后的备份",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="manual-recovery">
                不使用 Databasus 从备份恢复 PostgreSQL
              </h1>

              <p className="text-lg text-gray-400">
                备份不只是保护数据，还包括把数据恢复出来。Databasus
                特别注意让你的备份始终可恢复：即使装有 Databasus 的 VPS
                被删除、你丢失了访问权限或因故打不开界面。备份以标准格式
                存储，没有供应商锁定，所以恢复备份并不需要 Databasus。
              </p>

              <h2 id="what-you-need">你需要什么</h2>

              <p>手动恢复备份需要：</p>

              <ul>
                <li>
                  <strong>备份文件</strong>，来自你的存储（本地存储、S3、 Azure Blob Storage 等）
                </li>
                <li>
                  <strong>元数据文件</strong>，来自同一存储。它与备份文件
                  同名，只是多了 <code>.metadata</code> 扩展名。
                </li>
                <li>
                  <strong>密钥文件</strong>，位于{" "}
                  <code>./databasus-data/secret.key</code>（与备份文件在
                  同一目录，通常是 <code>/opt/databasus/</code>）
                </li>
              </ul>

              <h2 id="file-structure">文件结构</h2>

              <p>每个备份由两个文件组成，存放在你的存储中（本地或云端）：</p>

              <ul>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}`}</code>
                  ：加密并压缩后的备份数据
                </li>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}.metadata`}</code>
                  ：包含加密信息的 JSON 文件
                </li>
              </ul>

              <p>元数据文件包含 Base64 格式的加密盐和 IV（nonce）：</p>

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

              <h2 id="decryption">解密</h2>

              <p>
                Databasus 使用 <strong>AES-256-GCM</strong> 加密，并通过{" "}
                <strong>PBKDF2</strong> 派生密钥。每个备份都有独立的加密
                密钥，由以下部分派生：
              </p>

              <ul>
                <li>主密钥（来自 secret.key 文件）</li>
                <li>备份 ID</li>
                <li>随机盐（保存在元数据中）</li>
              </ul>

              <p>使用这个 Python 脚本解密备份：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pythonScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={pythonScript} />
                </div>
              </div>

              <p>安装所需依赖：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pip install pycryptodome</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text="pip install pycryptodome" />
                </div>
              </div>

              <p>
                <strong>脚本用法：</strong>
              </p>

              <ol>
                <li>
                  把上面的脚本保存为文件（例如 <code>decrypt_backup.py</code>）
                </li>
                <li>修改文件末尾示例用法部分的参数</li>
                <li>运行脚本：</li>
              </ol>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>python decrypt_backup.py</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text="python decrypt_backup.py" />
                </div>
              </div>

              <p>
                脚本会自动生成带 <code>decrypted_</code>{" "}
                前缀的输出文件。例如，备份文件是 <code>backup-id.dump</code>
                ，解密后的文件就是 <code>decrypted_backup-id.dump</code>。
              </p>

              <h2 id="restore">恢复到数据库</h2>

              <p>解密之后，使用各数据库自带的工具恢复：</p>

              <h3 id="postgresql-restore">PostgreSQL</h3>

              <p>PostgreSQL 备份使用内置压缩，可以直接恢复：</p>

              <p>
                <strong>本地数据库：</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={postgresqlRestore} />
                </div>
              </div>

              <p>
                <strong>远程数据库：</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={postgresqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mysql-restore">MySQL</h3>

              <p>MySQL 备份使用 zstd 5 级压缩，恢复之前必须先解压。</p>

              <p>
                <strong>第 1 步：解压备份</strong>
              </p>

              <p>
                使用 zstd 命令行工具或任何兼容的解压工具（7-Zip、PeaZip、 WinRAR
                等）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mysqlDecompress} />
                </div>
              </div>

              <p>
                <strong>第 2 步：恢复到数据库</strong>
              </p>

              <p>本地数据库：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mysqlRestore} />
                </div>
              </div>

              <p>远程数据库：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mysqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mariadb-restore">MariaDB</h3>

              <p>MariaDB 备份使用 zstd 5 级压缩，恢复之前必须先解压。</p>

              <p>
                <strong>第 1 步：解压备份</strong>
              </p>

              <p>
                使用 zstd 命令行工具或任何兼容的解压工具（7-Zip、PeaZip、 WinRAR
                等）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mariadbDecompress} />
                </div>
              </div>

              <p>
                <strong>第 2 步：恢复到数据库</strong>
              </p>

              <p>本地数据库：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mariadbRestore} />
                </div>
              </div>

              <p>远程数据库：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mariadbRestoreRemote} />
                </div>
              </div>

              <h3 id="mongodb-restore">MongoDB</h3>

              <p>MongoDB 备份使用内置 gzip 压缩，可以直接恢复：</p>

              <p>
                <strong>本地数据库：</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mongodbRestore} />
                </div>
              </div>

              <p>
                <strong>远程数据库：</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={mongodbRestoreRemote} />
                </div>
              </div>

              <h2 id="what-if-i-have-issues">遇到问题怎么办？</h2>

              <p>如果恢复过程中遇到任何问题：</p>

              <ul>
                <li>
                  <strong>向 AI 求助</strong>。ChatGPT、Claude、Gemini 这类 AI
                  助手非常擅长解答压缩工具和数据库恢复方面的问题。描述一下
                  你的问题，它们就能一步步指导你。
                </li>
                <li>
                  <strong>
                    加入我们的{" "}
                    <a
                      href="https://t.me/databasus_community"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      社区
                    </a>
                  </strong>
                  。我们的开发者和社区成员可以针对你的具体情况提供帮助。
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
