import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "¿Cómo restaurar PostgreSQL desde una copia de seguridad sin Databasus?",
  description:
    "Aprenda a restaurar manualmente sus copias de seguridad de bases de datos sin Databasus. Sin dependencia del proveedor: descifre y restaure sus respaldos con herramientas estándar y su clave secreta.",
  keywords: [
    "recuperación de copias de seguridad",
    "restauración manual",
    "descifrar copia de seguridad",
    "sin dependencia del proveedor",
    "descifrado AES-256-GCM",
    "restaurar PostgreSQL",
    "restaurar MySQL",
    "restaurar MariaDB",
    "restaurar MongoDB",
    "copia de seguridad sin Databasus",
  ],
  openGraph: {
    title:
      "¿Cómo restaurar PostgreSQL desde una copia de seguridad sin Databasus?",
    description:
      "Aprenda a restaurar manualmente sus copias de seguridad de bases de datos sin Databasus. Sin dependencia del proveedor: descifre y restaure sus respaldos con herramientas estándar y su clave secreta.",
    type: "article",
    url: getLocalizedUrl("es", "how-to-recover-without-databasus"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "¿Cómo restaurar PostgreSQL desde una copia de seguridad sin Databasus?",
    description:
      "Aprenda a restaurar manualmente sus copias de seguridad de bases de datos sin Databasus. Sin dependencia del proveedor: descifre y restaure sus respaldos con herramientas estándar y su clave secreta.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "how-to-recover-without-databasus"),
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
              "¿Cómo restaurar PostgreSQL desde una copia de seguridad sin Databasus?",
            description:
              "Aprenda a restaurar manualmente sus copias de seguridad de bases de datos sin Databasus. Sin dependencia del proveedor: descifre y restaure sus respaldos con herramientas estándar y su clave secreta.",
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
            name: "¿Cómo restaurar desde una copia de seguridad sin Databasus?",
            description:
              "Guía paso a paso para restaurar manualmente copias de seguridad de bases de datos sin Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Descargar los archivos de la copia de seguridad",
                text: "Descargue desde su almacenamiento tanto el archivo de la copia como el archivo de metadatos",
              },
              {
                "@type": "HowToStep",
                name: "Descifrar la copia de seguridad",
                text: "Use el script de Python para descifrar el archivo de la copia con su clave maestra",
              },
              {
                "@type": "HowToStep",
                name: "Descomprimir si es necesario",
                text: "Para MySQL y MariaDB, descomprima el archivo de la copia con zstd",
              },
              {
                "@type": "HowToStep",
                name: "Restaurar en la base de datos",
                text: "Use las herramientas propias de cada base de datos para restaurar la copia descifrada",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="es" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="es" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="manual-recovery">
                Restaurar PostgreSQL desde una copia de seguridad sin Databasus
              </h1>

              <p className="text-lg text-gray-400">
                Hacer copias de seguridad no es solo proteger los datos, sino
                también poder recuperarlos. Databasus se asegura de que sus
                respaldos sigan siendo recuperables incluso si el VPS con
                Databasus se elimina, pierde el acceso o no puede entrar en la
                interfaz por cualquier motivo. No necesita Databasus para
                recuperar las copias: se guardan en formato estándar y no hay
                dependencia del proveedor.
              </p>

              <h2 id="what-you-need">Qué necesita</h2>

              <p>Para recuperar una copia de seguridad manualmente necesita:</p>

              <ul>
                <li>
                  <strong>El archivo de la copia</strong> desde su
                  almacenamiento (almacenamiento local, S3, Azure Blob Storage, etc.)
                </li>
                <li>
                  <strong>El archivo de metadatos</strong> del mismo
                  almacenamiento. Se llama igual que el archivo de la copia,
                  pero con la extensión <code>.metadata</code>.
                </li>
                <li>
                  <strong>La clave secreta</strong> de{" "}
                  <code>./databasus-data/secret.key</code> (ubicada en el mismo
                  directorio que los archivos de copia, normalmente{" "}
                  <code>/opt/databasus/</code>)
                </li>
              </ul>

              <h2 id="file-structure">Estructura de archivos</h2>

              <p>
                Cada copia de seguridad consta de dos archivos guardados en su
                almacenamiento (local o en la nube):
              </p>

              <ul>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}`}</code> -
                  Datos de la copia cifrados y comprimidos
                </li>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}.metadata`}</code>{" "}
                  - Archivo JSON con los detalles del cifrado
                </li>
              </ul>

              <p>
                El archivo de metadatos contiene la sal de cifrado y el IV
                (nonce) en formato Base64:
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

              <h2 id="decryption">Descifrado</h2>

              <p>
                Databasus usa cifrado <strong>AES-256-GCM</strong> con
                derivación de clave <strong>PBKDF2</strong>. Cada copia de
                seguridad tiene una clave de cifrado única derivada de:
              </p>

              <ul>
                <li>La clave maestra (del archivo secret.key)</li>
                <li>El ID de la copia</li>
                <li>Una sal aleatoria (guardada en los metadatos)</li>
              </ul>

              <p>Use este script de Python para descifrar su copia:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pythonScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={pythonScript} />
                </div>
              </div>

              <p>Instale las dependencias necesarias:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pip install pycryptodome</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text="pip install pycryptodome" />
                </div>
              </div>

              <p>
                <strong>Cómo usar el script:</strong>
              </p>

              <ol>
                <li>
                  Guarde el script anterior en un archivo (por ejemplo,{" "}
                  <code>decrypt_backup.py</code>)
                </li>
                <li>
                  Actualice los parámetros en la sección de ejemplo de uso al
                  final
                </li>
                <li>Ejecute el script:</li>
              </ol>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>python decrypt_backup.py</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text="python decrypt_backup.py" />
                </div>
              </div>

              <p>
                El script creará automáticamente el archivo de salida con el
                prefijo <code>decrypted_</code>. Por ejemplo, si su archivo de
                copia es <code>backup-id.dump</code>, el archivo descifrado será{" "}
                <code>decrypted_backup-id.dump</code>.
              </p>

              <h2 id="restore">Restauración en la base de datos</h2>

              <p>
                Tras el descifrado, restaure con las herramientas propias de
                cada base de datos:
              </p>

              <h3 id="postgresql-restore">PostgreSQL</h3>

              <p>
                Las copias de PostgreSQL usan compresión integrada y se pueden
                restaurar directamente:
              </p>

              <p>
                <strong>Base de datos local:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={postgresqlRestore} />
                </div>
              </div>

              <p>
                <strong>Base de datos remota:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={postgresqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mysql-restore">MySQL</h3>

              <p>
                Las copias de MySQL se comprimen con zstd nivel 5 y deben
                descomprimirse antes de restaurar.
              </p>

              <p>
                <strong>Paso 1: descomprima la copia</strong>
              </p>

              <p>
                Use la herramienta de línea de comandos zstd o cualquier
                herramienta de descompresión compatible (7-Zip, PeaZip, WinRAR,
                etc.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mysqlDecompress} />
                </div>
              </div>

              <p>
                <strong>Paso 2: restaure en la base de datos</strong>
              </p>

              <p>Base de datos local:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mysqlRestore} />
                </div>
              </div>

              <p>Base de datos remota:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mysqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mariadb-restore">MariaDB</h3>

              <p>
                Las copias de MariaDB se comprimen con zstd nivel 5 y deben
                descomprimirse antes de restaurar.
              </p>

              <p>
                <strong>Paso 1: descomprima la copia</strong>
              </p>

              <p>
                Use la herramienta de línea de comandos zstd o cualquier
                herramienta de descompresión compatible (7-Zip, PeaZip, WinRAR,
                etc.):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mariadbDecompress} />
                </div>
              </div>

              <p>
                <strong>Paso 2: restaure en la base de datos</strong>
              </p>

              <p>Base de datos local:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mariadbRestore} />
                </div>
              </div>

              <p>Base de datos remota:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mariadbRestoreRemote} />
                </div>
              </div>

              <h3 id="mongodb-restore">MongoDB</h3>

              <p>
                Las copias de MongoDB usan compresión gzip integrada y se pueden
                restaurar directamente:
              </p>

              <p>
                <strong>Base de datos local:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mongodbRestore} />
                </div>
              </div>

              <p>
                <strong>Base de datos remota:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={mongodbRestoreRemote} />
                </div>
              </div>

              <h2 id="what-if-i-have-issues">¿Qué hago si tengo problemas?</h2>

              <p>
                Si encuentra algún problema durante el proceso de recuperación:
              </p>

              <ul>
                <li>
                  <strong>Pida ayuda a una IA</strong>. Los asistentes de IA
                  como ChatGPT, Claude o Gemini ayudan muy bien con herramientas
                  de compresión y procedimientos de restauración de bases de
                  datos. Describa su problema y le guiarán durante el proceso.
                </li>
                <li>
                  <strong>
                    Únase a nuestra{" "}
                    <a
                      href="https://t.me/databasus_community"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      comunidad
                    </a>
                  </strong>
                  . Nuestros desarrolladores y miembros de la comunidad pueden
                  ayudarle con su caso concreto.
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
