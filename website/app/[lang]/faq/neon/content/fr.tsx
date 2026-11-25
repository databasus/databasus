import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

export const metadata: Metadata = {
  title: "Sauvegarde PostgreSQL de Neon : guide pas à pas | Databasus",
  description:
    "Guide pas à pas avec captures d'écran : configurez une sauvegarde PostgreSQL planifiée de Neon avec Databasus, gardez les sauvegardes dans votre propre stockage et restaurez-les dans Neon.",
  keywords: [
    "Databasus",
    "sauvegarde Neon",
    "sauvegarde PostgreSQL Neon",
    "sauvegarder une base Neon",
    "restauration Neon",
    "utilisateur en lecture seule Neon",
    "sauvegarde PostgreSQL",
    "sauvegarde de base de données",
  ],
  openGraph: {
    title: "Sauvegarde PostgreSQL de Neon : guide pas à pas | Databasus",
    description:
      "Guide pas à pas avec captures d'écran : configurez une sauvegarde PostgreSQL planifiée de Neon avec Databasus, gardez les sauvegardes dans votre propre stockage et restaurez-les dans Neon.",
    type: "article",
    url: getLocalizedUrl("fr", "faq/neon"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Sauvegarde PostgreSQL de Neon : guide pas à pas | Databasus",
    description:
      "Guide pas à pas avec captures d'écran : configurez une sauvegarde PostgreSQL planifiée de Neon avec Databasus, gardez les sauvegardes dans votre propre stockage et restaurez-les dans Neon.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "faq/neon"),
    languages: getLanguageAlternates("faq/neon"),
  },
  robots: "index, follow",
};

export default function NeonPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Comment sauvegarder Neon PostgreSQL avec Databasus",
            description:
              "Guide pas à pas pour sauvegarder une base PostgreSQL Neon avec Databasus et la restaurer dans Neon",
            step: [
              {
                "@type": "HowToStep",
                name: "Copiez la chaîne de connexion dans Neon",
                text: "Dans la console Neon, ouvrez votre projet, cliquez sur Connect, puis sur Copy snippet.",
                url: `${getLocalizedUrl("fr", "faq/neon")}#copy-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Ajoutez la base Neon dans Databasus",
                text: "Cliquez sur Add database, choisissez Neon, collez la chaîne de connexion avec Paste from clipboard et cliquez sur Test connection.",
                url: `${getLocalizedUrl("fr", "faq/neon")}#paste-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Créez un utilisateur en lecture seule",
                text: "Cliquez sur Yes, create read-only user : Databasus conserve alors des identifiants qui ne peuvent pas modifier vos données.",
                url: `${getLocalizedUrl("fr", "faq/neon")}#read-only-user`,
              },
              {
                "@type": "HowToStep",
                name: "Réglez la planification, le stockage et la durée de conservation",
                text: "Choisissez la fréquence des sauvegardes, l'endroit où les stocker et combien de temps les garder.",
                url: `${getLocalizedUrl("fr", "faq/neon")}#schedule-storage-retention`,
              },
              {
                "@type": "HowToStep",
                name: "Choisissez les canaux de notification",
                text: "Sélectionnez les canaux qui recevront les notifications de sauvegarde et cliquez sur Complete. Databasus lance la première sauvegarde.",
                url: `${getLocalizedUrl("fr", "faq/neon")}#notifiers`,
              },
              {
                "@type": "HowToStep",
                name: "Restaurez une sauvegarde dans Neon",
                text: "Cliquez sur l'icône de restauration à côté d'une sauvegarde, collez la chaîne de connexion de la base Neon cible et cliquez sur Restore to this DB.",
                url: `${getLocalizedUrl("fr", "faq/neon")}#restore`,
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
              <h1 id="neon-backup">
                Comment configurer la sauvegarde PostgreSQL de Neon
              </h1>

              <p className="text-lg text-gray-400">
                Neon est un service PostgreSQL managé et serverless. Pour le
                sauvegarder avec Databasus, copiez la chaîne de connexion dans
                la console Neon, ajoutez la base dans Databasus avec le type
                Neon et choisissez où stocker les sauvegardes. Databasus fait
                ensuite des sauvegardes à la fréquence
                que vous avez choisie et sait les restaurer dans n&apos;importe
                quelle base Neon.
              </p>

              <p>
                Il vous faut seulement une instance Databasus en marche. Si vous n&apos;en avez pas encore, suivez le{" "}
                <a href={getLocalizedHref("fr", "installation")}>guide d&apos;installation</a>
                . Le stockage et les canaux de notification se créent en cours de configuration.
              </p>

              <h2 id="copy-connection-string">
                Étape 1. Copiez la chaîne de connexion dans Neon
              </h2>

              <p>
                Ouvrez votre projet dans la console Neon et cliquez sur{" "}
                <strong>Connect</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-connect-button.png"
                alt="Bouton Connect sur la page Computes d'un projet Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
              />

              <p>
                Dans la fenêtre <strong>Connection details</strong>, choisissez
                la branche, la base et le rôle à sauvegarder, puis cliquez sur{" "}
                <strong>Copy snippet</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-copy-connection-string.png"
                alt="Fenêtre Connection details de Neon avec le bouton Copy snippet"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="add-database">
                Étape 2. Ajoutez la base Neon dans Databasus
              </h2>

              <h3 id="paste-connection-string">
                Choisissez Neon et collez la chaîne de connexion
              </h3>

              <p>
                Dans Databasus, cliquez sur <strong>Add database</strong>,
                donnez un nom à la base, choisissez <strong>Neon</strong> et
                cliquez sur <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-neon.png"
                alt="Fenêtre Add database de Databasus avec le type Neon sélectionné"
                width={1440}
                height={898}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Cliquez sur <strong>Paste from clipboard</strong>. Databasus
                lit la chaîne de connexion copiée dans Neon et remplit
                l&apos;hôte, le port, l&apos;utilisateur, le mot de passe, le
                nom de la base et le mode SSL.
              </p>

              <img
                src="/images/faq/neon/databasus-paste-connection-string.png"
                alt="Formulaire de connexion Neon dans Databasus avec le bouton Paste from clipboard"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Cliquez sur <strong>Test connection</strong>. Une fois le test
                réussi, cliquez sur <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-test-connection.png"
                alt="Formulaire de connexion Neon rempli dans Databasus après un test de connexion réussi"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="read-only-user">Créez un utilisateur en lecture seule</h3>

              <p>
                Databasus propose de créer un{" "}
                <a
                  href={`${getLocalizedHref("fr", "security")}#level-3-read-only-access`}
                >
                  utilisateur en lecture seule
                </a>{" "}
                pour les sauvegardes. Cliquez sur{" "}
                <strong>Yes, create read-only user</strong>. Databasus ne
                conserve alors que des identifiants capables de lire vos
                données, sans pouvoir les modifier.
              </p>

              <img
                src="/images/faq/neon/databasus-create-read-only-user.png"
                alt="Databasus propose de créer un utilisateur en lecture seule pour les sauvegardes Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="schedule-storage-retention">
                Réglez la planification, le stockage et la durée de conservation
              </h3>

              <p>Choisissez :</p>

              <ul>
                <li>
                  <strong>Backup interval</strong> et l&apos;heure : à quelle
                  fréquence Databasus fait une sauvegarde, par exemple tous les
                  jours à 03:00.
                </li>
                <li>
                  <strong>Storage</strong> : où vont les sauvegardes. Vous
                  pouvez{" "}
                  <a href={getLocalizedHref("fr", "storages")}>
                    ajouter un stockage
                  </a>{" "}
                  directement depuis cette liste. Pour garder aussi les sauvegardes
                  chez Neon, voici{" "}
                  <a href={getLocalizedHref("fr", "storages/neon-object-storage")}>
                    comment sauvegarder vers Neon Object Storage
                  </a>
                  .
                </li>
                <li>
                  <strong>Encryption</strong> : si Databasus{" "}
                  <a
                    href={`${getLocalizedHref("fr", "security")}#level-2-backups-encryption`}
                  >
                    chiffre les fichiers de sauvegarde
                  </a>{" "}
                  avant de les envoyer.
                </li>
                <li>
                  <strong>Retention policy</strong> : combien de temps garder
                  les anciennes sauvegardes, par exemple 3 mois.
                </li>
                <li>
                  <strong>Notifications</strong> : quels événements signaler,
                  seulement les sauvegardes échouées ou aussi les sauvegardes
                  réussies.
                </li>
              </ul>

              <img
                src="/images/faq/neon/databasus-schedule-storage-retention.png"
                alt="Réglages de planification, de stockage, de chiffrement et de conservation des sauvegardes dans Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="notifiers">Choisissez les canaux de notification</h3>

              <p>
                Sélectionnez les{" "}
                <a href={getLocalizedHref("fr", "notifiers")}>
                  canaux de notification
                </a>{" "}
                qui recevront ces messages, par exemple{" "}
                <a href={getLocalizedHref("fr", "notifiers/slack")}>Slack</a>{" "}
                ou{" "}
                <a href={getLocalizedHref("fr", "notifiers/teams")}>
                  Microsoft Teams
                </a>
                . Vous pouvez en choisir plusieurs, ou aucun et les ajouter
                plus tard. Cliquez sur <strong>Complete</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-notifiers.png"
                alt="Choix des canaux de notification dans Databasus avant de terminer la configuration de la base"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="first-backup">Étape 3. Vérifiez la première sauvegarde</h2>

              <p>
                Databasus lance la première sauvegarde dès que vous cliquez sur{" "}
                <strong>Complete</strong>. Elle apparaît dans l&apos;onglet{" "}
                <strong>Backups</strong> avec sa taille et sa durée. Les
                sauvegardes suivantes se font selon la planification, et le
                bouton <strong>Make backup right now</strong> en lance une à
                tout moment.
              </p>

              <img
                src="/images/faq/neon/databasus-first-neon-backup.png"
                alt="Première sauvegarde Neon réussie dans l'onglet Backups de Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Une sauvegarde réussie ne garantit pas encore que la
                restauration fonctionnera. Pour le vérifier sur chaque
                sauvegarde, configurez la{" "}
                <a href={getLocalizedHref("fr", "restore-verification")}>
                  vérification de restauration
                </a>
                .
              </p>

              <h2 id="restore">Comment restaurer une sauvegarde Neon</h2>

              <p>
                Cliquez sur l&apos;icône de restauration (un nuage avec une
                flèche) à côté de la sauvegarde. Dans la console Neon, copiez
                la chaîne de connexion de la base cible comme à l&apos;
                <a href="#copy-connection-string">étape 1</a>, cliquez sur{" "}
                <strong>Paste from clipboard</strong>, puis sur{" "}
                <strong>Restore to this DB</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-restore-neon-backup.png"
                alt="Fenêtre de restauration depuis une sauvegarde dans Databasus avec la base Neon cible"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Databasus vide la base cible avant la restauration. La cible
                peut être :
              </p>

              <ul>
                <li>une nouvelle base dans le même projet Neon</li>
                <li>une base sur une branche Neon</li>
                <li>une base existante, y compris la base source elle-même</li>
              </ul>

              <p>
                Le plus sûr est de restaurer dans une nouvelle branche Neon.
                Vous y vérifiez les données restaurées, et la base de
                production reste intacte jusqu&apos;à ce que vous décidiez de
                basculer.
              </p>

              <p>Une fois la restauration terminée, son statut passe à Successful.</p>

              <img
                src="/images/faq/neon/databasus-restore-done.png"
                alt="Restauration terminée d'une sauvegarde Neon dans Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="health-check">
                La vérification de disponibilité garde-t-elle le compute Neon
                éveillé ?
              </h2>

              <p>
                Pas par défaut. Neon met un compute en veille quand personne
                n&apos;y est connecté et ne le facture pas pendant ce temps. La
                vérification de disponibilité se connecte toutes les minutes :
                le compute resterait éveillé et facturé jour et nuit. Databasus
                désactive donc la vérification de disponibilité pour les bases
                Neon. Si vous l&apos;activez, Databasus vous demande d&apos;abord
                de confirmer. Une sauvegarde planifiée ne réveille le compute
                que le temps de son exécution, puis Neon le remet en veille.
              </p>

              <p>
                Vous sauvegardez aussi Supabase ? Voici{" "}
                <a href={getLocalizedHref("fr", "faq/supabase")}>
                  comment sauvegarder Supabase
                </a>
                .
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("fr", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Retour à la FAQ
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
