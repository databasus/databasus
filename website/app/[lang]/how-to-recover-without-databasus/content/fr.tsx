import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Comment restaurer PostgreSQL depuis une sauvegarde sans Databasus ?",
  description:
    "Découvrez comment restaurer manuellement vos sauvegardes de bases de données sans Databasus. Aucun verrouillage propriétaire : déchiffrez et restaurez vos backups avec des outils standards et votre clé secrète.",
  keywords: [
    "récupération de sauvegarde",
    "restauration manuelle",
    "déchiffrer une sauvegarde",
    "sans verrouillage propriétaire",
    "déchiffrement AES-256-GCM",
    "restaurer PostgreSQL",
    "restaurer MySQL",
    "restaurer MariaDB",
    "restaurer MongoDB",
    "sauvegarde sans Databasus",
  ],
  openGraph: {
    title:
      "Comment restaurer PostgreSQL depuis une sauvegarde sans Databasus ?",
    description:
      "Découvrez comment restaurer manuellement vos sauvegardes de bases de données sans Databasus. Aucun verrouillage propriétaire : déchiffrez et restaurez vos backups avec des outils standards et votre clé secrète.",
    type: "article",
    url: getLocalizedUrl("fr", "how-to-recover-without-databasus"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title:
      "Comment restaurer PostgreSQL depuis une sauvegarde sans Databasus ?",
    description:
      "Découvrez comment restaurer manuellement vos sauvegardes de bases de données sans Databasus. Aucun verrouillage propriétaire : déchiffrez et restaurez vos backups avec des outils standards et votre clé secrète.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "how-to-recover-without-databasus"),
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
              "Comment restaurer PostgreSQL depuis une sauvegarde sans Databasus ?",
            description:
              "Découvrez comment restaurer manuellement vos sauvegardes de bases de données sans Databasus. Aucun verrouillage propriétaire : déchiffrez et restaurez vos backups avec des outils standards et votre clé secrète.",
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
            name: "Comment restaurer PostgreSQL depuis une sauvegarde sans Databasus ?",
            description:
              "Guide pas à pas pour restaurer manuellement des sauvegardes de bases de données sans Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Télécharger les fichiers de sauvegarde",
                text: "Téléchargez le fichier de sauvegarde et le fichier de métadonnées depuis votre stockage",
              },
              {
                "@type": "HowToStep",
                name: "Déchiffrer la sauvegarde",
                text: "Utilisez le script Python pour déchiffrer le fichier de sauvegarde avec votre clé maîtresse",
              },
              {
                "@type": "HowToStep",
                name: "Décompresser si nécessaire",
                text: "Pour MySQL et MariaDB, décompressez le fichier de sauvegarde avec zstd",
              },
              {
                "@type": "HowToStep",
                name: "Restaurer dans la base de données",
                text: "Utilisez les outils propres à chaque base pour restaurer la sauvegarde déchiffrée",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="fr" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="fr" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="manual-recovery">
                Restaurer PostgreSQL depuis une sauvegarde sans Databasus
              </h1>

              <p className="text-lg text-gray-400">
                Sauvegarder, ce n&apos;est pas seulement protéger les données.
                C&apos;est aussi pouvoir les restaurer. Databasus veille à ce
                que vos backups restent récupérables même si le VPS qui héberge
                Databasus est supprimé, si vous avez perdu l&apos;accès ou si
                l&apos;interface est inaccessible pour une raison quelconque.
                Vous n&apos;avez donc pas besoin de Databasus pour restaurer vos
                backups : ils sont stockés dans un format standard et il
                n&apos;y a aucun verrouillage propriétaire.
              </p>

              <h2 id="what-you-need">Ce dont vous avez besoin</h2>

              <p>Pour restaurer manuellement une sauvegarde, il vous faut :</p>

              <ul>
                <li>
                  <strong>Le fichier de sauvegarde</strong> depuis votre
                  stockage (stockage local, S3, Azure Blob Storage, etc.)
                </li>
                <li>
                  <strong>Le fichier de métadonnées</strong> depuis le même
                  stockage. Il porte le même nom que le fichier de sauvegarde,
                  avec l&apos;extension <code>.metadata</code> en plus.
                </li>
                <li>
                  <strong>La clé secrète</strong> depuis{" "}
                  <code>./databasus-data/secret.key</code> (située dans le même
                  répertoire que les fichiers de sauvegarde, généralement{" "}
                  <code>/opt/databasus/</code>)
                </li>
              </ul>

              <h2 id="file-structure">Structure des fichiers</h2>

              <p>
                Chaque sauvegarde se compose de deux fichiers placés dans votre
                stockage (local ou cloud) :
              </p>

              <ul>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}`}</code> : les
                  données de sauvegarde chiffrées et compressées
                </li>
                <li>
                  <code>{`{database-name}-{timestamp}-{backup-id}.metadata`}</code>{" "}
                  : un fichier JSON avec les détails de chiffrement
                </li>
              </ul>

              <p>
                Le fichier de métadonnées contient le sel de chiffrement et
                l&apos;IV (nonce) au format Base64 :
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

              <h2 id="decryption">Déchiffrement</h2>

              <p>
                Databasus utilise le chiffrement <strong>AES-256-GCM</strong>{" "}
                avec dérivation de clé <strong>PBKDF2</strong>. Chaque
                sauvegarde a une clé de chiffrement unique dérivée de :
              </p>

              <ul>
                <li>La clé maîtresse (depuis le fichier secret.key)</li>
                <li>L&apos;ID de la sauvegarde</li>
                <li>Un sel aléatoire (stocké dans les métadonnées)</li>
              </ul>

              <p>
                Utilisez ce script Python pour déchiffrer votre sauvegarde :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pythonScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={pythonScript} />
                </div>
              </div>

              <p>Installez les dépendances requises :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pip install pycryptodome</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text="pip install pycryptodome" />
                </div>
              </div>

              <p>
                <strong>Comment utiliser le script :</strong>
              </p>

              <ol>
                <li>
                  Enregistrez le script ci-dessus dans un fichier (par exemple{" "}
                  <code>decrypt_backup.py</code>)
                </li>
                <li>
                  Modifiez les paramètres dans la section d&apos;exemple
                  d&apos;utilisation en bas du script
                </li>
                <li>Lancez le script :</li>
              </ol>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>python decrypt_backup.py</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text="python decrypt_backup.py" />
                </div>
              </div>

              <p>
                Le script créera automatiquement le fichier de sortie avec le
                préfixe <code>decrypted_</code>. Par exemple, si votre fichier
                de sauvegarde est <code>backup-id.dump</code>, le fichier
                déchiffré sera <code>decrypted_backup-id.dump</code>.
              </p>

              <h2 id="restore">Restaurer dans la base de données</h2>

              <p>
                Après le déchiffrement, restaurez avec les outils propres à
                chaque base :
              </p>

              <h3 id="postgresql-restore">PostgreSQL</h3>

              <p>
                Les backups PostgreSQL utilisent la compression intégrée et
                peuvent être restaurés directement :
              </p>

              <p>
                <strong>Base de données locale :</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={postgresqlRestore} />
                </div>
              </div>

              <p>
                <strong>Base de données distante :</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={postgresqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mysql-restore">MySQL</h3>

              <p>
                Les backups MySQL sont compressés avec zstd niveau 5 et doivent
                être décompressés avant la restauration.
              </p>

              <p>
                <strong>Étape 1 : décompresser la sauvegarde</strong>
              </p>

              <p>
                Utilisez l&apos;outil en ligne de commande zstd ou tout outil de
                décompression compatible (7-Zip, PeaZip, WinRAR, etc.) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mysqlDecompress} />
                </div>
              </div>

              <p>
                <strong>Étape 2 : restaurer dans la base de données</strong>
              </p>

              <p>Base de données locale :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mysqlRestore} />
                </div>
              </div>

              <p>Base de données distante :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mysqlRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mysqlRestoreRemote} />
                </div>
              </div>

              <h3 id="mariadb-restore">MariaDB</h3>

              <p>
                Les backups MariaDB sont compressés avec zstd niveau 5 et
                doivent être décompressés avant la restauration.
              </p>

              <p>
                <strong>Étape 1 : décompresser la sauvegarde</strong>
              </p>

              <p>
                Utilisez l&apos;outil en ligne de commande zstd ou tout outil de
                décompression compatible (7-Zip, PeaZip, WinRAR, etc.) :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbDecompress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mariadbDecompress} />
                </div>
              </div>

              <p>
                <strong>Étape 2 : restaurer dans la base de données</strong>
              </p>

              <p>Base de données locale :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mariadbRestore} />
                </div>
              </div>

              <p>Base de données distante :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mariadbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mariadbRestoreRemote} />
                </div>
              </div>

              <h3 id="mongodb-restore">MongoDB</h3>

              <p>
                Les backups MongoDB utilisent la compression gzip intégrée et
                peuvent être restaurés directement :
              </p>

              <p>
                <strong>Base de données locale :</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestore}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mongodbRestore} />
                </div>
              </div>

              <p>
                <strong>Base de données distante :</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{mongodbRestoreRemote}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={mongodbRestoreRemote} />
                </div>
              </div>

              <h2 id="what-if-i-have-issues">Que faire en cas de problème ?</h2>

              <p>Si vous rencontrez un problème pendant la restauration :</p>

              <ul>
                <li>
                  <strong>Demandez de l&apos;aide à une IA</strong>. Les
                  assistants IA comme ChatGPT, Claude ou Gemini sont très
                  efficaces pour aider avec les outils de compression et les
                  procédures de restauration de bases de données. Décrivez
                  simplement votre problème et ils vous guideront pas à pas.
                </li>
                <li>
                  <strong>
                    Rejoignez notre{" "}
                    <a
                      href="https://t.me/databasus_community"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      communauté
                    </a>
                  </strong>
                  . Nos développeurs et les membres de la communauté peuvent
                  vous aider dans votre cas particulier.
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
