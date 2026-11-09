import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Как восстановить PostgreSQL из бекапа без Databasus?",
  description:
    "Как вручную восстановить базу данных из бекапа без Databasus. Никакого vendor lock-in: расшифруйте и восстановите бекапы стандартными инструментами и своим секретным ключом.",
  keywords: [
    "восстановление из бекапа",
    "восстановление вручную",
    "расшифровка бекапа",
    "без vendor lock-in",
    "расшифровка AES-256-GCM",
    "восстановление PostgreSQL",
    "восстановление MySQL",
    "восстановление MariaDB",
    "восстановление MongoDB",
    "бекап без Databasus",
  ],
  openGraph: {
    title: "Как восстановить PostgreSQL из бекапа без Databasus?",
    description:
      "Как вручную восстановить базу данных из бекапа без Databasus. Никакого vendor lock-in: расшифруйте и восстановите бекапы стандартными инструментами и своим секретным ключом.",
    type: "article",
    url: getLocalizedUrl("ru", "how-to-recover-without-databasus"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Как восстановить PostgreSQL из бекапа без Databasus?",
    description:
      "Как вручную восстановить базу данных из бекапа без Databasus. Никакого vendor lock-in: расшифруйте и восстановите бекапы стандартными инструментами и своим секретным ключом.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "how-to-recover-without-databasus"),
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
            headline: "Как восстановить PostgreSQL из бекапа без Databasus?",
            description:
              "Как вручную восстановить базу данных из бекапа без Databasus. Никакого vendor lock-in: расшифруйте и восстановите бекапы стандартными инструментами и своим секретным ключом.",
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
            name: "Как восстановить PostgreSQL из бекапа без Databasus?",
            description:
              "Пошаговое руководство по восстановлению базы данных из бекапа вручную, без Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Скачайте файлы бекапа",
                text: "Скачайте из хранилища и сам файл бекапа, и файл метаданных",
              },
              {
                "@type": "HowToStep",
                name: "Расшифруйте бекап",
                text: "Расшифруйте файл бекапа Python-скриптом, используя мастер-ключ",
              },
              {
                "@type": "HowToStep",
                name: "Распакуйте при необходимости",
                text: "Для MySQL и MariaDB распакуйте файл бекапа утилитой zstd",
              },
              {
                "@type": "HowToStep",
                name: "Восстановите базу данных",
                text: "Восстановите расшифрованный бекап штатными инструментами вашей СУБД",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="ru" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="ru" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="manual-recovery">
                Восстановление PostgreSQL из бекапа без Databasus
              </h1>

              <p className="text-lg text-gray-400">
                Резервное копирование &mdash; это не только защита данных, но и
                их восстановление. Databasus устроен так, чтобы бекапы
                оставались восстановимыми, даже если VPS с Databasus удален, вы
                потеряли доступ или по какой-то причине не можете открыть
                интерфейс. Для восстановления бекапов Databasus не нужен: они
                хранятся в стандартном формате, и никакого vendor lock-in нет.
              </p>

              <h2 id="what-you-need">Что понадобится</h2>

              <p>Чтобы восстановить бекап вручную, вам нужны:</p>

              <ul>
                <li>
                  <strong>Файл бекапа</strong> из вашего хранилища (локальное
                  хранилище, S3, Azure Blob Storage и т.д.)
                </li>
                <li>
                  <strong>Файл метаданных</strong> из того же хранилища. Он
                  называется так же, как файл бекапа, но с расширением{" "}
                  <code>.metadata</code>.
                </li>
                <li>
                  <strong>Секретный ключ</strong> из{" "}
                  <code>./databasus-data/secret.key</code> (лежит в том же
                  каталоге, что и файлы бекапов, обычно{" "}
                  <code>/opt/databasus/</code>)
                </li>
              </ul>

              <h2 id="file-structure">Структура файлов</h2>

              <p>
                Каждый бекап состоит из двух файлов в вашем хранилище (локальном
                или облачном):
              </p>

              <ul>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}`}</code>{" "}
                  &mdash; зашифрованные и сжатые данные бекапа
                </li>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}.metadata`}</code>{" "}
                  &mdash; JSON-файл с параметрами шифрования
                </li>
              </ul>

              <p>
                Файл метаданных содержит соль шифрования и IV (nonce) в формате
                Base64:
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

              <h2 id="decryption">Расшифровка</h2>

              <p>
                Databasus использует шифрование <strong>AES-256-GCM</strong> с
                выведением ключа через <strong>PBKDF2</strong>. У каждого бекапа
                свой уникальный ключ шифрования, который выводится из:
              </p>

              <ul>
                <li>мастер-ключа (из файла secret.key)</li>
                <li>ID бекапа</li>
                <li>случайной соли (хранится в метаданных)</li>
              </ul>

              <p>Расшифруйте бекап этим Python-скриптом:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pythonScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={pythonScript} lang="ru" />
                </div>
              </div>

              <p>Установите необходимые зависимости:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pip install pycryptodome</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="pip install pycryptodome" lang="ru" />
                </div>
              </div>

              <p>
                <strong>Как пользоваться скриптом:</strong>
              </p>

              <ol>
                <li>
                  Сохраните скрипт выше в файл (например,{" "}
                  <code>decrypt_backup.py</code>)
                </li>
                <li>Поменяйте параметры в примере запуска в конце файла</li>
                <li>Запустите скрипт:</li>
              </ol>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>python decrypt_backup.py</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="python decrypt_backup.py" lang="ru" />
                </div>
              </div>

              <p>
                Скрипт сам создаст выходной файл с префиксом{" "}
                <code>decrypted_</code>. Например, если файл бекапа называется{" "}
                <code>backup-id.dump</code>, расшифрованный файл будет{" "}
                <code>decrypted_backup-id.dump</code>.
              </p>

              <h2 id="restore">Восстановление базы данных</h2>

              <p>
                После расшифровки восстановите бекап штатными инструментами
                вашей СУБД:
              </p>

              <h3 id="postgresql-restore">PostgreSQL</h3>

              <p>
                Дампы PostgreSQL используют встроенное сжатие, поэтому их можно
                восстанавливать сразу:
              </p>

              <p>
                <strong>Локальная база:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlRestore} lang="ru" />
                </div>
              </div>

              <p>
                <strong>Удаленная база:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlRestoreRemote} lang="ru" />
                </div>
              </div>

              <h3 id="mysql-restore">MySQL</h3>

              <p>
                Бекапы MySQL сжаты zstd (уровень 5), перед восстановлением их
                нужно распаковать.
              </p>

              <p>
                <strong>Шаг 1: распакуйте бекап</strong>
              </p>

              <p>
                Используйте консольную утилиту zstd или любой совместимый
                архиватор (7-Zip, PeaZip, WinRAR и т.д.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlDecompress} lang="ru" />
                </div>
              </div>

              <p>
                <strong>Шаг 2: восстановите базу данных</strong>
              </p>

              <p>Локальная база:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlRestore} lang="ru" />
                </div>
              </div>

              <p>Удаленная база:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mysqlRestoreRemote} lang="ru" />
                </div>
              </div>

              <h3 id="mariadb-restore">MariaDB</h3>

              <p>
                Бекапы MariaDB сжаты zstd (уровень 5), перед восстановлением их
                нужно распаковать.
              </p>

              <p>
                <strong>Шаг 1: распакуйте бекап</strong>
              </p>

              <p>
                Используйте консольную утилиту zstd или любой совместимый
                архиватор (7-Zip, PeaZip, WinRAR и т.д.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbDecompress} lang="ru" />
                </div>
              </div>

              <p>
                <strong>Шаг 2: восстановите базу данных</strong>
              </p>

              <p>Локальная база:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbRestore} lang="ru" />
                </div>
              </div>

              <p>Удаленная база:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mariadbRestoreRemote} lang="ru" />
                </div>
              </div>

              <h3 id="mongodb-restore">MongoDB</h3>

              <p>
                Бекапы MongoDB используют встроенное сжатие gzip, поэтому их
                можно восстанавливать сразу:
              </p>

              <p>
                <strong>Локальная база:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mongodbRestore} lang="ru" />
                </div>
              </div>

              <p>
                <strong>Удаленная база:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={mongodbRestoreRemote} lang="ru" />
                </div>
              </div>

              <h2 id="what-if-i-have-issues">
                Что делать, если возникли проблемы?
              </h2>

              <p>Если в процессе восстановления что-то пошло не так:</p>

              <ul>
                <li>
                  <strong>Спросите ИИ</strong>. ИИ-ассистенты вроде ChatGPT,
                  Claude или Gemini отлично помогают с архиваторами и
                  восстановлением баз данных. Просто опишите проблему &mdash; и
                  вам подскажут каждый шаг.
                </li>
                <li>
                  <strong>
                    Вступите в наше{" "}
                    <a
                      href="https://t.me/databasus_community"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      сообщество
                    </a>
                  </strong>
                  . Наши разработчики и участники сообщества помогут разобраться
                  с вашим конкретным случаем.
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
