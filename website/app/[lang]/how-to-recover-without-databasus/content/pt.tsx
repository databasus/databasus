import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Como restaurar PostgreSQL a partir de backup sem o Databasus?",
  description:
    "Aprenda a restaurar manualmente os backups da sua base de dados sem o Databasus. Sem vendor lock-in: descriptografe e restaure as suas cópias de segurança com ferramentas padrão e a sua chave secreta.",
  keywords: [
    "recuperação de backup",
    "restauração manual",
    "descriptografar backup",
    "sem vendor lock-in",
    "descriptografia AES-256-GCM",
    "restaurar PostgreSQL",
    "restaurar MySQL",
    "restaurar MariaDB",
    "restaurar MongoDB",
    "backup sem Databasus",
  ],
  openGraph: {
    title: "Como restaurar PostgreSQL a partir de backup sem o Databasus?",
    description:
      "Aprenda a restaurar manualmente os backups da sua base de dados sem o Databasus. Sem vendor lock-in: descriptografe e restaure as suas cópias de segurança com ferramentas padrão e a sua chave secreta.",
    type: "article",
    url: getLocalizedUrl("pt", "how-to-recover-without-databasus"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Como restaurar PostgreSQL a partir de backup sem o Databasus?",
    description:
      "Aprenda a restaurar manualmente os backups da sua base de dados sem o Databasus. Sem vendor lock-in: descriptografe e restaure as suas cópias de segurança com ferramentas padrão e a sua chave secreta.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "how-to-recover-without-databasus"),
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
            headline:
              "Como restaurar PostgreSQL a partir de backup sem o Databasus?",
            description:
              "Aprenda a restaurar manualmente os backups da sua base de dados sem o Databasus. Sem vendor lock-in: descriptografe e restaure as suas cópias de segurança com ferramentas padrão e a sua chave secreta.",
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
            name: "Como restaurar PostgreSQL a partir de backup sem o Databasus?",
            description:
              "Guia passo a passo para restaurar manualmente backups de bases de dados sem o Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Baixar os arquivos de backup",
                text: "Baixe o arquivo de backup e o arquivo de metadados do seu armazenamento",
              },
              {
                "@type": "HowToStep",
                name: "Descriptografar o backup",
                text: "Use o script Python para descriptografar o arquivo de backup com a sua chave mestra",
              },
              {
                "@type": "HowToStep",
                name: "Descomprimir se necessário",
                text: "Para MySQL e MariaDB, descomprima o arquivo de backup com zstd",
              },
              {
                "@type": "HowToStep",
                name: "Restaurar na base de dados",
                text: "Use as ferramentas próprias de cada base de dados para restaurar o backup descriptografado",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="pt" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="pt" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="manual-recovery">
                Restaurar PostgreSQL a partir de backup sem o Databasus
              </h1>

              <p className="text-lg text-gray-400">
                Fazer backup não é só proteger os dados. É também poder
                recuperá-los. O Databasus garante que as suas cópias de
                segurança continuam recuperáveis mesmo que o VPS com o Databasus
                seja apagado, que você perca o acesso ou não consiga abrir a
                interface por alguma razão. Você não precisa do Databasus para
                recuperar os backups: eles são guardados em formato padrão e não
                há vendor lock-in.
              </p>

              <h2 id="what-you-need">O que você precisa</h2>

              <p>Para recuperar um backup manualmente, você precisa de:</p>

              <ul>
                <li>
                  <strong>Arquivo de backup</strong> do seu armazenamento
                  (armazenamento local, S3, Azure Blob Storage, etc.)
                </li>
                <li>
                  <strong>Arquivo de metadados</strong> do mesmo armazenamento.
                  Tem o mesmo nome do arquivo de backup, mas com a extensão{" "}
                  <code>.metadata</code>.
                </li>
                <li>
                  <strong>Chave secreta</strong> de{" "}
                  <code>./databasus-data/secret.key</code> (localizada no mesmo
                  diretório dos arquivos de backup, normalmente{" "}
                  <code>/opt/databasus/</code>)
                </li>
              </ul>

              <h2 id="file-structure">Estrutura dos arquivos</h2>

              <p>
                Cada backup consiste em dois arquivos guardados no seu
                armazenamento (local ou na nuvem):
              </p>

              <ul>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}`}</code> -
                  Dados do backup, criptografados e comprimidos
                </li>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}.metadata`}</code>{" "}
                  - Arquivo JSON com os detalhes da criptografia
                </li>
              </ul>

              <p>
                O arquivo de metadados contém o salt de criptografia e o IV
                (nonce) em formato Base64:
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

              <h2 id="decryption">Descriptografia</h2>

              <p>
                O Databasus usa criptografia <strong>AES-256-GCM</strong> com
                derivação de chave <strong>PBKDF2</strong>. Cada backup tem uma
                chave de criptografia única derivada de:
              </p>

              <ul>
                <li>Chave mestra (do arquivo secret.key)</li>
                <li>ID do backup</li>
                <li>Salt aleatório (guardado nos metadados)</li>
              </ul>

              <p>Use este script Python para descriptografar o seu backup:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pythonScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={pythonScript} />
                </div>
              </div>

              <p>Instale as dependências necessárias:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pip install pycryptodome</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text="pip install pycryptodome" />
                </div>
              </div>

              <p>
                <strong>Como usar o script:</strong>
              </p>

              <ol>
                <li>
                  Salve o script acima num arquivo (por exemplo,{" "}
                  <code>decrypt_backup.py</code>)
                </li>
                <li>
                  Atualize os parâmetros na seção de exemplo de uso, no final do
                  script
                </li>
                <li>Execute o script:</li>
              </ol>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>python decrypt_backup.py</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text="python decrypt_backup.py" />
                </div>
              </div>

              <p>
                O script cria automaticamente o arquivo de saída com o prefixo{" "}
                <code>decrypted_</code>. Por exemplo, se o seu arquivo de backup
                for <code>backup-id.dump</code>, o arquivo descriptografado será{" "}
                <code>decrypted_backup-id.dump</code>.
              </p>

              <h2 id="restore">Restaurar na base de dados</h2>

              <p>
                Depois de descriptografar, restaure com as ferramentas próprias
                de cada base de dados:
              </p>

              <h3 id="postgresql-restore">PostgreSQL</h3>

              <p>
                Os backups de PostgreSQL usam compressão embutida e podem ser
                restaurados diretamente:
              </p>

              <p>
                <strong>Base de dados local:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={postgresqlRestore} />
                </div>
              </div>

              <p>
                <strong>Base de dados remota:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={postgresqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mysql-restore">MySQL</h3>

              <p>
                Os backups de MySQL são comprimidos com zstd nível 5 e precisam
                ser descomprimidos antes de restaurar.
              </p>

              <p>
                <strong>Passo 1: descomprimir o backup</strong>
              </p>

              <p>
                Use a ferramenta de linha de comando zstd ou qualquer ferramenta
                de descompressão compatível (7-Zip, PeaZip, WinRAR, etc.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mysqlDecompress} />
                </div>
              </div>

              <p>
                <strong>Passo 2: restaurar na base de dados</strong>
              </p>

              <p>Base de dados local:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mysqlRestore} />
                </div>
              </div>

              <p>Base de dados remota:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mysqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mariadb-restore">MariaDB</h3>

              <p>
                Os backups de MariaDB são comprimidos com zstd nível 5 e
                precisam ser descomprimidos antes de restaurar.
              </p>

              <p>
                <strong>Passo 1: descomprimir o backup</strong>
              </p>

              <p>
                Use a ferramenta de linha de comando zstd ou qualquer ferramenta
                de descompressão compatível (7-Zip, PeaZip, WinRAR, etc.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mariadbDecompress} />
                </div>
              </div>

              <p>
                <strong>Passo 2: restaurar na base de dados</strong>
              </p>

              <p>Base de dados local:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mariadbRestore} />
                </div>
              </div>

              <p>Base de dados remota:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mariadbRestoreRemote} />
                </div>
              </div>

              <h3 id="mongodb-restore">MongoDB</h3>

              <p>
                Os backups de MongoDB usam compressão gzip embutida e podem ser
                restaurados diretamente:
              </p>

              <p>
                <strong>Base de dados local:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mongodbRestore} />
                </div>
              </div>

              <p>
                <strong>Base de dados remota:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={mongodbRestoreRemote} />
                </div>
              </div>

              <h2 id="what-if-i-have-issues">E se tiver problemas?</h2>

              <p>
                Se você encontrar algum problema durante o processo de
                recuperação:
              </p>

              <ul>
                <li>
                  <strong>Peça ajuda a uma IA</strong>. Assistentes como
                  ChatGPT, Claude ou Gemini são excelentes para ajudar com
                  ferramentas de compressão e procedimentos de restauração de
                  bases de dados. Basta descrever o problema, e eles guiam você
                  pelo processo.
                </li>
                <li>
                  <strong>
                    Junte-se à nossa{" "}
                    <a
                      href="https://t.me/databasus_community"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      comunidade
                    </a>
                  </strong>
                  . Nossos desenvolvedores e membros da comunidade podem ajudar
                  com o seu caso específico.
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
