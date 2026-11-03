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
                notificateurs et des stockages (S3, Google Drive, Dropbox,
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
                l&apos;accès à votre stockage cloud (S3, Google Drive, etc.), il
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
                Databasus manipule des données sensibles, donc la prévention des
                vulnérabilités, des accès non autorisés et des fuites de données
                est une préoccupation de premier plan. Nous y investissons des
                deux côtés du système : dans le code lui-même (contrôles de
                permissions, chiffrement, gestion soigneuse des secrets) et dans
                l&apos;infrastructure qui l&apos;entoure (analyse des
                dépendances, réponse aux CVE, pratiques DevSecOps). Le pipeline
                ci-dessous s&apos;exécute automatiquement sur chaque commit et
                PR. Aucune couche ne suffit à elle seule, mais ensemble elles
                réduisent le risque qu&apos;un code vulnérable, des dépendances
                dangereuses, des images cassées ou des backups non restaurables
                atteignent une release.
              </p>

              <h3 id="static-analysis">Analyse statique</h3>

              <p>
                L&apos;analyse statique s&apos;exécute en plusieurs passes
                indépendantes. CodeQL analyse l&apos;ensemble du code à la
                recherche de problèmes de sécurité. Sur chaque PR,{" "}
                <strong>gitleaks</strong> recherche les secrets divulgués et{" "}
                <strong>semgrep</strong> vérifie le code selon des règles de
                sécurité. Les Dockerfiles et les workflows CI
                ont leurs propres règles supplémentaires (références
                d&apos;actions épinglées, permissions de moindre privilège,
                images de base suspectes), si bien que les schémas non sûrs sont
                signalés avant même d&apos;être fusionnés.
              </p>

              <p>
                En plus de ces contrôles par PR, <strong>Codex Security</strong>{" "}
                d&apos;OpenAI effectue des audits réguliers et plus approfondis
                de tout le code. C&apos;est un programme distinct qui détecte
                les problèmes architecturaux et transverses que les analyses
                limitées au moment de la PR laissent passer.
              </p>

              <h3 id="dependency-management">Gestion des dépendances</h3>

              <p>
                Dependabot surveille toutes nos dépendances par rapport à la
                GitHub Advisory Database et fait remonter les CVE quelques
                minutes après leur publication. Les mises à jour sont appliquées
                avec un délai, pour laisser aux versions fraîchement publiées le
                temps de mûrir avant que nous les adoptions : une défense
                délibérée contre les attaques de la chaîne
                d&apos;approvisionnement, comme les paquets compromis.
              </p>

              <p>
                La <strong>Dependency Review Action</strong> fait échouer le
                contrôle de toute PR qui introduit une nouvelle CVE{" "}
                <strong>HIGH</strong> ou <strong>CRITICAL</strong>.
              </p>

              <h3 id="container-and-ci-hardening">
                Durcissement des conteneurs et de la CI
              </h3>

              <ul>
                <li>
                  L&apos;image de l&apos;application est analysée avec <strong>Trivy</strong> sur
                  les deux architectures avant chaque release.
                </li>
                <li>
                  Une passe Trivy distincte sur le Dockerfile détecte les
                  erreurs de configuration avant qu&apos;elles n&apos;arrivent
                  dans une image.
                </li>
                <li>
                  Toutes les GitHub Actions sont épinglées sur des SHA de commit
                  complets plutôt que sur des tags flottants comme{" "}
                  <code>@v4</code> ou <code>@main</code>, qui ont été un vecteur
                  d&apos;attaque actif en 2025.
                </li>
                <li>
                  Les workflows utilisent par défaut des permissions de moindre
                  privilège et ne les élèvent par job que lorsque c&apos;est
                  réellement nécessaire.
                </li>
              </ul>

              <h3 id="testing-and-verification">Tests et vérification</h3>

              <p>
                Les chemins critiques sont couverts par des tests unitaires et
                d&apos;intégration, exécutés contre de vrais conteneurs de bases
                de données pour chaque moteur et version majeure pris en charge.
              </p>

              <p>
                La restauration est le chemin qui compte le plus pour un outil
                de sauvegarde, donc nous la testons explicitement : chaque PR
                exécute des cycles complets de sauvegarde puis restauration
                contre ces mêmes conteneurs réels, en vérifiant que les backups
                peuvent réellement être restaurés de bout en bout, et pas
                seulement écrits avec succès.
              </p>

              <p>
                Le reste du pipeline CI/CD exécute le lint, la vérification de
                types, la suite de tests complète, des smoke tests d&apos;image
                et des builds multi-architecture sur chaque PR. Une release ne
                sort qu&apos;une fois tout cela et les analyses de sécurité
                passés, sans alerte critique CodeQL ouverte.
              </p>

              <h3 id="reporting-a-vulnerability">Signaler une vulnérabilité</h3>

              <p>
                Vous avez trouvé une vulnérabilité ? Signalez-la via
                l&apos;onglet GitHub Security, voir{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . Les rapports de sécurité sont traités en priorité absolue.
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
