import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Réinitialiser le mot de passe - Documentation Databasus",
  description:
    "Apprenez à réinitialiser les mots de passe des utilisateurs, lister les administrateurs et créer des comptes dans Databasus avec l'outil en ligne de commande intégré. Récupération rapide et sécurisée pour votre système de sauvegarde PostgreSQL.",
  keywords: [
    "réinitialisation mot de passe Databasus",
    "réinitialiser mot de passe utilisateur",
    "mot de passe sauvegarde PostgreSQL",
    "récupération mot de passe Docker",
    "récupération de mot de passe",
    "authentification Databasus",
  ],
  openGraph: {
    title: "Réinitialiser le mot de passe - Documentation Databasus",
    description:
      "Apprenez à réinitialiser les mots de passe des utilisateurs, lister les administrateurs et créer des comptes dans Databasus avec l'outil en ligne de commande intégré. Récupération rapide et sécurisée pour votre système de sauvegarde PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("fr", "password"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Réinitialiser le mot de passe - Documentation Databasus",
    description:
      "Apprenez à réinitialiser les mots de passe des utilisateurs, lister les administrateurs et créer des comptes dans Databasus avec l'outil en ligne de commande intégré. Récupération rapide et sécurisée pour votre système de sauvegarde PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "password"),
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"`;
  const listAdminsCommand = `docker exec -it databasus ./main --list-admins`;
  const createUserCommand = `docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;
  const resetTwoFactorCommand = `docker exec -it databasus ./main --reset-2fa --email="owner@example.com"`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Réinitialiser le mot de passe - Documentation Databasus",
            description:
              "Apprenez à réinitialiser les mots de passe des utilisateurs dans Databasus avec l'outil en ligne de commande intégré.",
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
            name: "Comment réinitialiser le mot de passe d'un utilisateur Databasus",
            description:
              "Guide pas à pas pour réinitialiser les mots de passe des utilisateurs dans Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Exécuter la commande de réinitialisation",
                text: "Exécutez la commande docker exec avec votre nouveau mot de passe et l'e-mail de l'utilisateur.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Utilisez docker exec pour lancer la commande de réinitialisation du mot de passe dans le conteneur Databasus",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Vérifier le changement de mot de passe",
                text: "Connectez-vous à Databasus avec votre nouveau mot de passe pour confirmer que le changement a réussi.",
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
              <h1 id="reset-password">
                Réinitialiser le mot de passe d&apos;un utilisateur
              </h1>

              <h2 id="reset-password-command">
                Commande de réinitialisation du mot de passe
              </h2>

              <p>
                Pour réinitialiser le mot de passe d&apos;un utilisateur,
                utilisez la commande suivante sur le serveur où Databasus est en
                cours d&apos;exécution :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">Paramètres</h2>

              <p>La commande accepte les paramètres suivants :</p>

              <ul>
                <li>
                  <strong>--new-password</strong> : le nouveau mot de passe, d&apos;au
                  moins 8 caractères. Assurez-vous qu&apos;il est robuste et contient un mélange de
                  lettres, de chiffres et de caractères spéciaux.
                </li>
                <li>
                  <strong>--email</strong> : l&apos;adresse e-mail du compte
                  dont vous voulez réinitialiser le mot de passe (par exemple{" "}
                  <code>owner@example.com</code>). Une instance créée avant que
                  le premier compte ne l&apos;administre porte encore
                  l&apos;adresse provisoire <code>admin</code> tant que son
                  propriétaire ne l&apos;a pas remplacée : indiquez alors{" "}
                  <code>admin</code>.
                </li>
              </ul>

              <h2 id="list-admins">Lister les comptes administrateurs</h2>

              <p>
                Si vous ne savez plus quelle adresse administre l&apos;instance,
                listez les comptes administrateurs sur le serveur où Databasus
                tourne :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{listAdminsCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={listAdminsCommand} lang="fr" />
                </div>
              </div>

              <p>
                La sortie nomme chaque compte administrateur avec son adresse
                e-mail, son nom affiché, sa date de création et son état
                d&apos;activité, et marque celui que l&apos;instance reconnaît
                comme son administrateur. Aucun mot de passe, empreinte de mot
                de passe ni jeton n&apos;est affiché. Sur une instance où
                personne n&apos;a encore créé de compte, elle indique que
                l&apos;instance n&apos;a pas d&apos;administrateur.
              </p>

              <h2 id="create-user">Créer un compte depuis la console</h2>

              <p>
                Pour ouvrir un compte sans passer par l&apos;inscription dans le
                navigateur, par exemple depuis un script de déploiement,
                créez-le sur le serveur où Databasus tourne :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createUserCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={createUserCommand} lang="fr" />
                </div>
              </div>

              <p>La commande accepte les paramètres suivants :</p>

              <ul>
                <li>
                  <strong>--email</strong> : l&apos;adresse e-mail du nouveau
                  compte. Elle doit être valide.
                </li>
                <li>
                  <strong>--password</strong> : le mot de passe du nouveau
                  compte, d&apos;au moins 8 caractères.
                </li>
                <li>
                  <strong>--role</strong> : <code>admin</code> ou{" "}
                  <code>member</code>.
                </li>
                <li>
                  <strong>--name</strong> : un nom affiché, facultatif. Sans
                  lui, le compte prend le nom de la partie de l&apos;adresse
                  située avant le <code>@</code>.
                </li>
              </ul>

              <p>
                Le compte est actif tout de suite : il se connecte avec cette
                adresse et ce mot de passe, que l&apos;inscription libre soit
                activée ou non. Sur une instance qui n&apos;a encore aucun
                compte, le compte créé administre l&apos;instance, donc le
                premier doit utiliser <code>--role=admin</code>. Un
                administrateur créé plus tard de cette façon est un
                administrateur ordinaire.
              </p>

              <p>
                Relancer la commande pour un compte existant avec le même rôle
                réussit sans rien modifier : un déploiement peut donc
                l&apos;exécuter à chaque mise en production. Le mot de passe
                reste alors inchangé ; pour le renouveler, utilisez la commande
                de réinitialisation <code>--new-password</code> décrite plus
                haut. Si l&apos;adresse appartient à un compte d&apos;un autre
                rôle, à un compte désactivé ou invité, ou à un compte qui se
                connecte via Google ou GitHub, la commande échoue et ne change
                rien.
              </p>

              <h2 id="disable-two-factor">Ne plus exiger la 2FA</h2>

              <p>
                Une instance peut exiger, en plus du mot de passe, un code à six
                chiffres envoyé par e-mail à la connexion. Si le serveur de
                messagerie cesse de délivrer ces codes, plus personne
                n&apos;entre avec un mot de passe, et réinitialiser le mot de
                passe n&apos;y change rien. Désactivez le second facteur sur le
                serveur où Databasus tourne :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{disableTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={disableTwoFactorCommand} />
                </div>
              </div>

              <p>
                La commande indique si elle a changé quelque chose, se termine
                sans se plaindre quand le second facteur est déjà désactivé, et
                inscrit le changement au journal d&apos;audit. La prochaine
                connexion par mot de passe se passe alors de code, et un
                administrateur pourra réactiver le réglage dès que la messagerie
                refonctionnera.
              </p>

              <h2 id="reset-authenticator-app">Réinitialiser l&apos;application d&apos;authentification d&apos;un compte</h2>

              <p>
                Si quelqu&apos;un a perdu le téléphone qui porte son application d&apos;authentification, retirez l&apos;application de son compte sur le serveur où Databasus tourne :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetTwoFactorCommand} />
                </div>
              </div>

              <p>
                La prochaine connexion de ce compte suit le réglage 2FA de l&apos;instance. Un administrateur peut aussi réinitialiser l&apos;application d&apos;un utilisateur depuis la liste des utilisateurs.
              </p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
