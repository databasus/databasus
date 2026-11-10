import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Sécurité - Comment Databasus protège vos données | Databasus",
  description:
    "Découvrez comment Databasus assure une sécurité de niveau entreprise : chiffrement AES-256-GCM des données sensibles et des sauvegardes, accès en lecture seule aux bases de données et journalisation d'audit complète.",
  keywords: [
    "sécurité Databasus",
    "sécurité sauvegarde PostgreSQL",
    "chiffrement AES-256-GCM",
    "chiffrement de base de données",
    "chiffrement des sauvegardes",
    "accès en lecture seule à la base de données",
    "sécurité entreprise",
    "protection des données",
    "sauvegardes sécurisées",
  ],
  openGraph: {
    title: "Sécurité - Comment Databasus protège vos données | Databasus",
    description:
      "Découvrez comment Databasus assure une sécurité de niveau entreprise : chiffrement AES-256-GCM des données sensibles et des sauvegardes, accès en lecture seule aux bases de données et journalisation d'audit complète.",
    type: "article",
    url: getLocalizedUrl("fr", "security"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Sécurité - Comment Databasus protège vos données | Databasus",
    description:
      "Découvrez comment Databasus assure une sécurité de niveau entreprise : chiffrement AES-256-GCM des données sensibles et des sauvegardes, accès en lecture seule aux bases de données et journalisation d'audit complète.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "security"),
    languages: getLanguageAlternates("security"),
  },
  robots: "index, follow",
};

export default function SecurityPage() {
  const encryptionPipeline = `PostgreSQL pg_dump → Compression → Encryption → Cloud Storage`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Sécurité - Comment Databasus protège vos données",
            description:
              "Découvrez comment Databasus assure une sécurité de niveau entreprise : chiffrement AES-256-GCM des données sensibles et des sauvegardes, accès en lecture seule aux bases de données et journalisation d'audit complète.",
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

      <DocsNavbarComponent lang="fr" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="fr" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="security">
                Comment Databasus applique-t-il la sécurité ?
              </h1>

              <p className="text-lg text-gray-400">
                Databasus est responsable de données sensibles :
              </p>

              <ul>
                <li>il accède à votre base de données ;</li>
                <li>
                  il la sauvegarde (c&apos;est-à-dire qu&apos;il fait une copie
                  des données) ;
                </li>
                <li>
                  il conserve des identifiants pour pouvoir accéder à votre base
                  régulièrement ;
                </li>
                <li>
                  il enregistre les backups dans votre S3 ou d&apos;autres
                  stockages cloud (si vous l&apos;activez) ;
                </li>
              </ul>

              <p>
                Par conséquent,{" "}
                <strong>
                  la priorité absolue de Databasus est d&apos;offrir une
                  sécurité et une fiabilité de niveau entreprise
                </strong>
                .
              </p>

              <p>Databasus garantit que :</p>

              <ul>
                <li>
                  les données sensibles ne sont jamais exposées et toujours
                  chiffrées ;
                </li>
                <li>
                  les backups sont chiffrés et inutilisables même si
                  quelqu&apos;un les voit dans le stockage cloud ;
                </li>
                <li>
                  Databasus ne reçoit même pas d&apos;accès en écriture ou en
                  modification à la base ;
                </li>
                <li>
                  toutes les actions sont journalisées et peuvent être auditées
                  ;
                </li>
              </ul>

              <p>
                Toutes ces mesures protègent vos données. Comme vous le savez,
                aucun système n&apos;est sûr à 100 %, mais nous faisons de notre
                mieux pour le rendre aussi sûr que possible. Même en cas de
                piratage, personne ne pourra corrompre vos données.
              </p>

              <p>Databasus applique la sécurité à trois niveaux :</p>

              <ol>
                <li>Chiffrement des données sensibles ;</li>
                <li>Chiffrement des sauvegardes ;</li>
                <li>Accès en lecture seule à la base de données.</li>
              </ol>

              <h2 id="level-1-sensitive-data-encryption">
                Niveau 1 : chiffrement des données sensibles
              </h2>

              <p>
                En interne, Databasus utilise une base PostgreSQL pour stocker
                les détails de connexion, les configurations, les paramètres des
                notificateurs et des stockages (S3, Azure Blob Storage, FTP,
                etc.).
              </p>

              <p>Toute donnée sensible est chiffrée. Par exemple :</p>

              <ul>
                <li>les mots de passe</li>
                <li>les jetons</li>
                <li>les webhooks avec secrets</li>
              </ul>

              <p>
                Databasus ne garde donc en base que des hachages ou des valeurs
                encodées. Le chiffrement utilise l&apos;algorithme{" "}
                <strong>AES-256-GCM</strong>. De plus, malgré le chiffrement,
                ces valeurs ne sont jamais exposées via l&apos;API ou
                l&apos;interface.
              </p>

              <p>
                La clé secrète utilisée pour le chiffrement est stockée sur le
                disque local (<code>./databasus-data/secret.key</code> par
                défaut) et n&apos;est pas présente dans la base elle-même. La
                compromission de la base ne donne donc pas accès aux données
                sensibles.
              </p>

              <h2 id="level-2-backups-encryption">
                Niveau 2 : chiffrement des sauvegardes
              </h2>

              <p>
                Chaque fichier de backup est chiffré à la volée pendant sa
                création. Databasus utilise l&apos;algorithme de chiffrement{" "}
                <strong>AES-256-GCM</strong>, qui garantit que les données de
                sauvegarde ne peuvent pas être lues sans la clé de chiffrement
                et que toute altération est détectée au déchiffrement.
              </p>

              <p>Les backups passent par ce pipeline :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{encryptionPipeline}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={encryptionPipeline} />
                </div>
              </div>

              <p>
                Chaque backup reçoit sa propre clé de chiffrement unique,
                dérivée de :
              </p>

              <ul>
                <li>
                  La clé maîtresse (stockée dans{" "}
                  <code>./databasus-data/secret.key</code>)
                </li>
                <li>L&apos;ID du backup</li>
                <li>Un sel aléatoire (unique par backup)</li>
              </ul>

              <p>
                <strong>Résultat</strong> : même si quelqu&apos;un obtient
                l&apos;accès à votre stockage cloud (S3, Azure Blob Storage, etc.), il
                ne peut pas lire les backups sans votre clé maîtresse.
              </p>

              <h2 id="level-3-read-only-access">
                Niveau 3 : accès en lecture seule à la base de données
              </h2>

              <p>
                Databasus applique le principe du moindre privilège : il
                n&apos;a besoin que d&apos;un accès en lecture pour créer les
                backups, jamais d&apos;un accès en écriture. Cela protège votre
                base contre toute corruption de données, accidentelle ou
                malveillante, via l&apos;outil de sauvegarde.
              </p>

              <p>
                Avant d&apos;accepter des identifiants de base de données,
                Databasus effectue des contrôles à trois niveaux :
              </p>

              <ol>
                <li>
                  <strong>Niveau rôle</strong> : vérifie que l&apos;utilisateur
                  n&apos;est PAS superutilisateur et ne peut créer ni rôles ni
                  bases de données
                </li>
                <li>
                  <strong>Niveau base</strong> : s&apos;assure de l&apos;absence
                  de privilèges CREATE ou TEMP
                </li>
                <li>
                  <strong>Niveau table</strong> : confirme l&apos;absence totale
                  de permissions d&apos;écriture (INSERT, UPDATE, DELETE,
                  TRUNCATE, etc.)
                </li>
              </ol>

              <p>
                L&apos;utilisateur de la base doit passer les trois contrôles
                pour être considéré en lecture seule. Si un privilège
                d&apos;écriture est détecté, Databasus vous en avertit.
              </p>

              <p>
                Databasus vous propose de créer des utilisateurs en lecture
                seule avec les bonnes permissions :
              </p>

              <ul>
                <li>
                  Accorde SELECT sur toutes les tables actuelles et futures
                </li>
                <li>Accorde USAGE sur les schémas (mais pas CREATE)</li>
                <li>
                  Révoque explicitement tous les privilèges d&apos;écriture
                </li>
              </ul>

              <p>
                <strong>Résultat</strong> : même si Databasus est compromis, le
                serveur piraté, la clé secrète volée et les identifiants
                déchiffrés, les attaquants ne peuvent pas corrompre votre base
                de données.
              </p>

              <h2 id="two-factor-authentication">
                Authentification à deux facteurs à la connexion
              </h2>

              <p>
                Un mot de passe seul ouvre tous les identifiants de bases, les
                clés de stockage et les sauvegardes que l&apos;instance
                conserve, donc un administrateur peut exiger un second facteur
                pour la connexion par mot de passe. Une fois activé, un mot de
                passe correct envoie un code à six chiffres à l&apos;adresse du
                compte, et la connexion n&apos;aboutit qu&apos;après la saisie
                de ce code. Le code cesse de fonctionner dix minutes après son
                envoi ou après cinq essais erronés, et en demander un autre
                annule le précédent. Activer le réglage protège la prochaine
                connexion et laisse fonctionner les sessions déjà ouvertes.
              </p>

              <p>
                Le réglage couvre la connexion par mot de passe. Se connecter
                par Google ou GitHub délivre toujours un jeton directement,
                parce que ces fournisseurs font leurs propres vérifications
                multifacteurs : une instance qui veut le second facteur pour
                tout le monde ne les configure tout simplement pas.
              </p>

              <p>
                Il ne peut être activé que si l&apos;instance dispose d&apos;un
                serveur de messagerie et que chaque administrateur actif porte
                une vraie adresse, pour que personne ne l&apos;active et ne se
                retrouve bloqué hors de l&apos;instance. Si la messagerie cesse
                ensuite de distribuer les e-mails, plus personne n&apos;entre
                avec un mot de passe, et c&apos;est délibéré : un serveur de
                messagerie en panne ne doit pas ramener discrètement la
                connexion à un seul facteur. Un opérateur ayant accès au serveur
                le désactive :
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
                La commande indique ce qu&apos;elle a changé et l&apos;inscrit
                au journal d&apos;audit, si bien qu&apos;une instance qui a
                cessé de demander des codes en explique la raison. Elle est
                documentée à côté de la récupération de mot de passe sur la{" "}
                <a
                  href="/fr/password#disable-two-factor"
                  className="text-blue-400 hover:text-blue-600"
                >
                  page mot de passe
                </a>
                .
              </p>

              <h2 id="security-and-reliability-engineering">
                🛡️ Ingénierie de sécurité et de fiabilité
              </h2>

              <p>
                Databasus traite des données sensibles. Avant chaque version, des contrôles
                automatiques recherchent les problèmes de sécurité et vérifient que les
                sauvegardes peuvent être restaurées. Ils réduisent les risques, mais ne
                garantissent pas l’absence de vulnérabilités.
              </p>

              <h3 id="static-analysis">Analyse statique</h3>

              <p>
                CodeQL analyse Go, JavaScript/TypeScript et les workflows GitHub Actions.
                gitleaks recherche les secrets exposés dans l’historique Git de la version.
                semgrep vérifie le code source, les Dockerfiles et les workflows CI à l’aide
                de règles de sécurité. Ces analyses sont exécutées avant chaque version et
                chaque semaine.
              </p>

              <p>
                Les alertes CodeQL de niveau <strong>CRITICAL</strong> ouvertes dans
                l’analyse de la version bloquent sa publication. Les alertes{" "}
                <strong>HIGH</strong> génèrent des avertissements.
              </p>

              <p>
                Codex Security d’OpenAI réalise aussi des audits réguliers plus approfondis.
                Ils peuvent révéler des problèmes d’architecture que des analyses plus
                ciblées risquent de manquer. Ces audits sont distincts des contrôles
                automatiques de publication.
              </p>

              <h3 id="dependency-management">Gestion des dépendances</h3>

              <p>
                Les mises à jour ordinaires attendent sept jours complets pour les paquets,
                les outils de développement et les GitHub Actions. Le délai est de quatorze
                jours pour les images, les clients de bases de données et les extensions de
                vérification autonomes. Les responsables vérifient les dates
                de publication et la compatibilité, y compris pour les versions exactes et
                les mises à jour de sécurité qui peuvent contourner les filtres
                d’ancienneté. Toute exception est signalée avec son motif.
              </p>

              <p>
                Dependency Review Action compare la version candidate à la précédente et
                bloque les nouvelles vulnérabilités <strong>HIGH</strong> ou{" "}
                <strong>CRITICAL</strong>. govulncheck fait échouer le contrôle Go si notre
                code peut appeler une fonction ayant une vulnérabilité connue. La commande{" "}
                <code>go mod verify</code> compare les modules Go téléchargés aux sommes de
                contrôle enregistrées.
              </p>

              <p>
                Avant la publication d’une version ou d’images de développement,{" "}
                <code>mise run verify-deps</code> vérifie les versions fixes des outils,
                images et GitHub Actions. Il contrôle aussi la cohérence des versions
                associées des environnements d’exécution, ainsi que les empreintes et les
                droits d’exécution des clients de bases de données inclus dans le dépôt. Ces
                vérifications couvrent une liste définie de déclarations et de fichiers
                locaux. Les gestionnaires de paquets vérifient séparément les versions
                verrouillées et les sommes de contrôle des paquets téléchargés.
              </p>

              <p>
                Le contrôle local des dépendances ne certifie pas la sécurité du code tiers
                et n’impose pas un délai minimal après publication à chaque dépendance. Le
                choix et la revue des versions restent à la charge des responsables du
                projet.
              </p>

              <h3 id="container-and-ci-hardening">Sécurité des conteneurs et de la CI</h3>

              <ul>
                <li>
                  Trivy analyse l’image de l’application sur amd64 et arm64. Les
                  vulnérabilités <strong>CRITICAL</strong> disposant d’un correctif font
                  échouer le contrôle. Les vulnérabilités <strong>HIGH</strong> avec correctif
                  sont signalées sans le bloquer. Les vulnérabilités sans correctif sont
                  exclues de ce contrôle.
                </li>
                <li>
                  Une analyse de configuration Trivy distincte bloque les problèmes{" "}
                  <strong>HIGH</strong> et <strong>CRITICAL</strong> dans les Dockerfiles de
                  l’application, de l’agent de vérification et de certaines images de test.
                </li>
                <li>
                  Les GitHub Actions tierces sont fixées à des SHA de commit complets.
                  Déplacer un tag ne peut donc pas modifier discrètement le code de l’action
                  exécutée.
                </li>
                <li>
                  Les workflows n’ont par défaut qu’un accès en lecture au dépôt. Chaque tâche
                  reçoit des droits supplémentaires uniquement si nécessaire.
                </li>
              </ul>

              <h3 id="testing-and-verification">Tests et vérification</h3>

              <p>
                Des tests unitaires et d’intégration couvrent les comportements critiques.
                Avant chaque version, des cycles complets de sauvegarde et de restauration
                sont exécutés dans de vrais conteneurs, pour chaque moteur et version
                majeure pris en charge. Ils vérifient la restauration des données, pas
                seulement la création d’un fichier de sauvegarde.
              </p>

              <p>
                Les contrôles incluent aussi le lint, la vérification des types, les tests
                dans le navigateur, les tests de démarrage des images et les compilations et
                tests natifs sur amd64 et arm64. La publication attend la réussite de toutes
                les tâches obligatoires, y compris les contrôles de sécurité décrits
                ci-dessus.
              </p>

              <h3 id="reporting-a-vulnerability">Signaler une vulnérabilité</h3>

              <p>
                Signalez les vulnérabilités en privé par les moyens de contact indiqués
                dans{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . N’ouvrez pas d’issue publique. Les signalements de sécurité sont
                prioritaires.
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
