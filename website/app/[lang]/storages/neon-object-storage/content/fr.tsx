import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

const PAGE_URL = getLocalizedUrl("fr", "storages/neon-object-storage");

export const metadata: Metadata = {
  title: "Sauvegarde PostgreSQL vers Neon Object Storage : guide pas à pas | Databasus",
  description:
    "Guide pas à pas avec captures d'écran : créez un bucket et des identifiants dans Neon Object Storage, ajoutez-le à Databasus comme stockage S3 et conservez les sauvegardes de vos bases de données chez Neon selon une planification.",
  keywords: [
    "Databasus",
    "sauvegarde Neon Object Storage",
    "Neon Object Storage",
    "Neon S3",
    "bucket Neon",
    "sauvegarde PostgreSQL vers S3",
    "stockage compatible S3",
    "sauvegarde de base de données",
  ],
  openGraph: {
    title: "Sauvegarde PostgreSQL vers Neon Object Storage : guide pas à pas | Databasus",
    description:
      "Guide pas à pas avec captures d'écran : créez un bucket et des identifiants dans Neon Object Storage, ajoutez-le à Databasus comme stockage S3 et conservez les sauvegardes de vos bases de données chez Neon selon une planification.",
    type: "article",
    url: PAGE_URL,
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Sauvegarde PostgreSQL vers Neon Object Storage : guide pas à pas | Databasus",
    description:
      "Guide pas à pas avec captures d'écran : créez un bucket et des identifiants dans Neon Object Storage, ajoutez-le à Databasus comme stockage S3 et conservez les sauvegardes de vos bases de données chez Neon selon une planification.",
  },
  alternates: {
    canonical: PAGE_URL,
    languages: getLanguageAlternates("storages/neon-object-storage"),
  },
  robots: "index, follow",
};

export default function NeonObjectStoragePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Comment sauvegarder vers Neon Object Storage avec Databasus",
            description:
              "Guide pas à pas pour créer un bucket et des identifiants dans Neon Object Storage et l'ajouter à Databasus comme stockage de sauvegardes",
            step: [
              {
                "@type": "HowToStep",
                name: "Créez un bucket dans Neon",
                text: "Dans Neon Console, ouvrez le projet destiné aux sauvegardes, cliquez sur Object storage dans la barre latérale, puis sur Create bucket.",
                url: `${PAGE_URL}#create-bucket`,
              },
              {
                "@type": "HowToStep",
                name: "Copiez les identifiants du stockage",
                text: "Cliquez sur Connect, ouvrez l'onglet Storage, choisissez Parameters only et cliquez sur Copy credentials.",
                url: `${PAGE_URL}#copy-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Choisissez Neon Object Storage dans Databasus",
                text: "Dans Databasus, ouvrez Storages, cliquez sur Add storage et choisissez Neon Object Storage dans la liste des types.",
                url: `${PAGE_URL}#choose-neon-object-storage`,
              },
              {
                "@type": "HowToStep",
                name: "Collez les identifiants et saisissez le nom du bucket",
                text: "Cliquez sur Paste from clipboard pour remplir l'endpoint, la région, l'access key et la secret key, puis saisissez le nom du bucket.",
                url: `${PAGE_URL}#paste-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Testez la connexion et enregistrez",
                text: "Cliquez sur Test connection et, une fois le test réussi, sur Save.",
                url: `${PAGE_URL}#test-connection`,
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
              <h1 id="neon-object-storage-backup">Comment configurer la sauvegarde vers Neon Object Storage</h1>

              <p className="text-lg text-gray-400">
                <a href="https://neon.com/docs/storage/overview">Neon Object Storage</a> est un
                stockage objet compatible S3 intégré à chaque projet Neon. Pour y conserver vos
                sauvegardes, créez un bucket et des identifiants dans Neon Console, puis ajoutez
                le bucket à Databasus comme stockage. Databasus y envoie alors chaque sauvegarde
                des bases de données qui utilisent ce stockage, qu&apos;elles tournent sur Neon
                ou ailleurs.
              </p>

              <p>
                Il vous suffit d&apos;un Databasus en fonctionnement. Si vous n&apos;en avez pas
                encore, suivez le <a href={getLocalizedHref("fr", "installation")}>guide
                d&apos;installation</a>. Neon Object Storage est en bêta ;
                la <a href="https://neon.com/docs/storage/s3-compatibility">page sur la
                compatibilité S3</a> de la documentation Neon indique ce qu&apos;il prend en
                charge.
              </p>

              <h2 id="create-bucket">Étape 1. Créez un bucket dans Neon</h2>

              <p>
                Dans Neon Console, ouvrez le projet qui recevra les sauvegardes. Préférez un
                autre projet que celui des bases de données à sauvegarder : supprimer un projet
                supprime aussi ses buckets. Cliquez sur <strong>Object storage</strong> dans la
                barre latérale, puis sur <strong>Create bucket</strong>, et saisissez un nom
                pour le bucket. Un nouveau bucket est privé et appartient à la branche courante
                du projet.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-object-storage-buckets.png"
                alt="Page Object storage d'un projet Neon avec un bucket et le bouton Create bucket"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
              />

              <h2 id="copy-credentials">Étape 2. Copiez les identifiants du stockage</h2>

              <p>
                Cliquez sur <strong>Connect</strong> et ouvrez
                l&apos;onglet <strong>Storage</strong>. Avec <strong>Parameters
                only</strong> sélectionné, Neon affiche l&apos;endpoint, l&apos;access key, la
                secret key et la région. Si le bloc affiche des valeurs fictives à la place des
                clés, générez d&apos;abord des identifiants, comme le demande sa première ligne.
                Cliquez ensuite sur <strong>Copy credentials</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-connect-storage-credentials.png"
                alt="Onglet Storage de la fenêtre Connect de Neon avec l'endpoint, les clés, la région et le bouton Copy credentials"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Le bloc ressemble à ceci. Il ne contient pas le nom du bucket : retenez celui de
                l&apos;étape 1.
              </p>

              <pre>
                <code>{`AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.c-9.us-east-2.aws.neon.tech"
AWS_ACCESS_KEY_ID="nak_live_..."
AWS_SECRET_ACCESS_KEY="..."
AWS_REGION="us-east-2"`}</code>
              </pre>

              <h2 id="add-storage">Étape 3. Ajoutez le stockage à Databasus</h2>

              <h3 id="choose-neon-object-storage">Choisissez Neon Object Storage dans Databasus</h3>

              <p>
                Dans Databasus, ouvrez <strong>Storages</strong> et cliquez sur <strong>Add
                storage</strong>. Donnez un nom au stockage et choisissez <strong>Neon Object
                Storage</strong> dans la liste des types. Databasus l&apos;enregistre comme un
                stockage S3 ordinaire, la liste des stockages affiche donc le type S3.
              </p>

              <h3 id="paste-credentials">Collez les identifiants et saisissez le nom du bucket</h3>

              <p>
                Cliquez sur <strong>Paste from clipboard</strong>. Databasus lit le bloc copié
                dans Neon et remplit l&apos;endpoint, la région, l&apos;access key et la secret
                key. Le bloc reste dans votre navigateur : les clés n&apos;en sortent que
                lorsque vous testez ou enregistrez le stockage. Saisissez vous-même le nom du
                bucket dans <strong>S3 Bucket</strong>. Vous pouvez aussi remplir tous les
                champs à la main sous le séparateur <strong>or enter the details
                manually</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/databasus-neon-object-storage-form.png"
                alt="Formulaire Neon Object Storage de Databasus avec l'endpoint, la région et les clés collés et le nom du bucket saisi"
                width={1435}
                height={897}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="test-connection">Testez la connexion et enregistrez</h3>

              <p>
                Cliquez sur <strong>Test connection</strong>. Une fois le test réussi, cliquez
                sur <strong>Save</strong>. Vous pouvez maintenant choisir ce stockage dans les
                paramètres de sauvegarde de n&apos;importe quelle base de données. Si la base
                elle-même tourne sur Neon, le <a href={getLocalizedHref("fr", "faq/neon")}>guide
                de sauvegarde Neon</a> décrit la suite de la configuration.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("fr", "storages")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Retour aux stockages
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
