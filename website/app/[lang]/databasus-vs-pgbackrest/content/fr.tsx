import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs pgBackRest : comparaison d'outils de sauvegarde PostgreSQL",
  description:
    "Comparez les outils de sauvegarde PostgreSQL Databasus et pgBackRest : approche de backup, public cible, facilité d'utilisation, options de restauration et quand choisir chaque outil.",
  keywords: [
    "Databasus vs pgBackRest",
    "comparaison sauvegarde PostgreSQL",
    "alternative à pgBackRest",
    "outils de sauvegarde PostgreSQL",
    "comparaison de sauvegarde de bases de données",
    "pg_dump vs sauvegarde physique",
    "backup auto-hébergé",
    "PITR PostgreSQL",
    "sauvegarde de grandes bases de données",
    "outils de backup pour DBA",
  ],
  openGraph: {
    title:
      "Databasus vs pgBackRest : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et pgBackRest : approche de backup, public cible, facilité d'utilisation, options de restauration et quand choisir chaque outil.",
    type: "article",
    url: getLocalizedUrl("fr", "databasus-vs-pgbackrest"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs pgBackRest : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et pgBackRest : approche de backup, public cible, facilité d'utilisation, options de restauration et quand choisir chaque outil.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "databasus-vs-pgbackrest"),
    languages: getLanguageAlternates("databasus-vs-pgbackrest"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackRestPage() {
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
              "Databasus vs pgBackRest : comparaison d'outils de sauvegarde PostgreSQL",
            description:
              "Une comparaison détaillée des outils de sauvegarde PostgreSQL Databasus et pgBackRest : approche de backup, public cible, facilité d'utilisation, options de restauration et quand choisir chaque outil.",
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
              <h1 id="databasus-vs-pgbackrest">Databasus vs pgBackRest</h1>

              <p className="text-lg text-gray-400">
                Databasus et pgBackRest sont tous deux conçus pour la reprise
                après sinistre avec un RTO et un RPO minimaux, et tous deux
                prennent en charge les sauvegardes physiques, l&apos;archivage
                WAL et la récupération à un instant donné (PITR). Databasus
                exécute ces backups à distance sur la pile native de PostgreSQL
                17 : il réutilise l&apos;outillage éprouvé de PostgreSQL au lieu
                de le réinventer, le tout derrière une interface web intuitive.
                Il convient aux bases de données de toute taille et de toute
                complexité. Les sauvegardes physiques exigent PostgreSQL 17 ou
                plus récent ; sur les versions antérieures, seuls les backups
                logiques <code>pg_dump</code> sont disponibles. pgBackRest
                embarque son propre moteur de sauvegarde : il couvre donc aussi
                les sauvegardes physiques et incrémentales sur des versions de
                PostgreSQL bien plus anciennes, et ajoute des fonctionnalités
                avancées comme les sauvegardes différentielles et la
                restauration delta.
              </p>

              <h2 id="quick-comparison">Comparaison rapide</h2>

              <p>
                Voici un aperçu rapide des principales différences entre
                Databasus et pgBackRest :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fonctionnalité</th>
                    <th>Databasus</th>
                    <th>pgBackRest</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Public cible</td>
                    <td data-label="Databasus">
                      Particuliers, équipes, DBA, entreprises
                    </td>
                    <td data-label="pgBackRest">
                      DBA, flux de travail CLI et IaC
                    </td>
                  </tr>
                  <tr>
                    <td>Gestion des sauvegardes</td>
                    <td data-label="Databasus">✅ Plusieurs bases</td>
                    <td data-label="pgBackRest">❌ Une seule base</td>
                  </tr>
                  <tr>
                    <td>Support d&apos;autres bases de données</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="pgBackRest">❌ PostgreSQL uniquement</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Databasus">Interface web</td>
                    <td data-label="pgBackRest">
                      Ligne de commande, fichiers de configuration
                    </td>
                  </tr>
                  <tr>
                    <td>Type de sauvegarde</td>
                    <td data-label="Databasus">Logique + physique</td>
                    <td data-label="pgBackRest">Physique (niveau fichier)</td>
                  </tr>
                  <tr>
                    <td>Version PostgreSQL pour les sauvegardes physiques</td>
                    <td data-label="Databasus">17+ (natif)</td>
                    <td data-label="pgBackRest">9.4+ (moteur propre)</td>
                  </tr>
                  <tr>
                    <td>Options de restauration</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="pgBackRest">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Opérations parallèles</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="pgBackRest">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes incrémentales</td>
                    <td data-label="Databasus">✅ Au niveau bloc (PG 17+)</td>
                    <td data-label="pgBackRest">
                      Incrémentales au niveau bloc
                    </td>
                  </tr>
                  <tr>
                    <td>Sauvegardes différentielles</td>
                    <td data-label="Databasus">❌ Non</td>
                    <td data-label="pgBackRest">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Restauration delta</td>
                    <td data-label="Databasus">❌ Non</td>
                    <td data-label="pgBackRest">
                      ✅ Oui (fichiers modifiés uniquement)
                    </td>
                  </tr>
                  <tr>
                    <td>Sauvegardes à distance</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="pgBackRest">
                      ❌ Non (accès au système de fichiers requis)
                    </td>
                  </tr>
                  <tr>
                    <td>Fonctionnalités d&apos;équipe</td>
                    <td data-label="Databasus">
                      ✅ Espaces de travail, RBAC, journaux d&apos;audit
                    </td>
                    <td data-label="pgBackRest">❌ Utilisateur unique</td>
                  </tr>
                  <tr>
                    <td>Courbe d&apos;apprentissage</td>
                    <td data-label="Databasus">Minimale</td>
                    <td data-label="pgBackRest">Expertise DBA requise</td>
                  </tr>
                  <tr>
                    <td>Installation</td>
                    <td data-label="Databasus">
                      Script en une ligne ou Docker
                    </td>
                    <td data-label="pgBackRest">
                      Configuration manuelle requise
                    </td>
                  </tr>
                  <tr>
                    <td>Adapté aux bases auto-hébergées</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="pgBackRest">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Adapté aux bases cloud</td>
                    <td data-label="Databasus">
                      ✅ Oui (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="pgBackRest">
                      ❌ Non (accès au système de fichiers requis)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="target-audience">Public cible</h2>

              <p>
                La différence la plus importante entre ces outils est le public
                auquel ils s&apos;adressent :
              </p>

              <h3 id="audience-databasus">Public de Databasus</h3>

              <p>
                Databasus est conçu pour un large public, du développeur
                individuel à la grande entreprise :
              </p>

              <ul>
                <li>
                  <strong>Développeurs individuels</strong> : une installation
                  simple et une interface intuitive permettent de protéger ses
                  projets personnels sans expertise PostgreSQL poussée.
                </li>
                <li>
                  <strong>Équipes de développement</strong> : les espaces de
                  travail, le contrôle d&apos;accès par rôles et les journaux
                  d&apos;audit permettent une collaboration sécurisée entre les
                  membres de l&apos;équipe.
                </li>
                <li>
                  <strong>Entreprises</strong> : répond aux besoins des
                  entreprises avec une sécurité complète, plusieurs destinations
                  de stockage et des canaux de notification.
                </li>
                <li>
                  <strong>DBA et reprise après sinistre</strong> : sauvegardes
                  physiques, archivage WAL et PITR pour les systèmes critiques
                  exigeant une perte de données quasi nulle.
                </li>
              </ul>

              <h3 id="audience-pgbackrest">Public de pgBackRest</h3>

              <p>
                pgBackRest est un outil en ligne de commande destiné aux équipes
                qui veulent exploiter elles-mêmes le moteur de sauvegarde :
              </p>

              <ul>
                <li>
                  <strong>Flux de travail CLI et IaC</strong> : les équipes qui
                  préfèrent configurer les backups via des fichiers et des
                  scripts plutôt qu&apos;une interface web.
                </li>
                <li>
                  <strong>Versions anciennes de PostgreSQL</strong> : pgBackRest
                  embarque son propre moteur, il peut donc exécuter des
                  sauvegardes physiques et incrémentales sur des versions de
                  PostgreSQL antérieures à 17.
                </li>
                <li>
                  <strong>Fonctionnalités avancées à grande échelle</strong> :
                  quand les sauvegardes différentielles, la restauration delta
                  et la création de standby justifient l&apos;effort de
                  configuration supplémentaire.
                </li>
              </ul>

              <h2 id="backup-approach">Approche de sauvegarde</h2>

              <p>
                Les deux outils utilisent des stratégies de backup
                fondamentalement différentes, chacune avec ses avantages :
              </p>

              <h3 id="backup-databasus">
                Databasus : sauvegardes logiques + physiques
              </h3>

              <p>
                Databasus prend en charge les deux stratégies, logique et
                physique :
              </p>

              <ul>
                <li>
                  <strong>Sauvegardes physiques, incrémentales et WAL</strong> :
                  exécutées à distance via le protocole de réplication
                  PostgreSQL, sur la pile native de PostgreSQL 17 (
                  <code>pg_basebackup</code>,{" "}
                  <code>pg_basebackup --incremental</code> au niveau bloc piloté
                  par les résumés WAL côté serveur, <code>pg_receivewal</code>{" "}
                  et <code>pg_combinebackup</code>). Databasus réutilise
                  l&apos;outillage éprouvé de PostgreSQL au lieu de le
                  réinventer. Nécessite PostgreSQL 17 ou plus récent.
                </li>
                <li>
                  <strong>Sauvegardes logiques</strong> : utilise{" "}
                  <code>pg_dump</code> pour des backups portables, restaurables
                  sur d&apos;autres versions de PostgreSQL, avec restauration
                  sélective de tables ou de schémas précis. C&apos;est aussi le
                  seul type de sauvegarde sur PostgreSQL antérieur à 17 et la
                  voie utilisée pour MySQL, MariaDB et MongoDB.
                </li>
                <li>
                  <strong>Rien à installer sur la base de données</strong> : les
                  backups se connectent à distance ; les réseaux fermés sont
                  atteints via un tunnel SSH vers un hôte interne ou un bastion,
                  si bien que la base n&apos;a jamais à être exposée
                  publiquement.
                </li>
                <li>
                  <strong>Compression efficace</strong> : utilise zstd (niveau
                  5) pour les deux types de sauvegarde ; les fichiers en
                  ressortent 4 à 8 fois plus petits.
                </li>
                <li>
                  <strong>Accès en lecture seule</strong> : les sauvegardes
                  logiques n&apos;exigent que des permissions SELECT, ce qui
                  minimise les risques de sécurité.
                </li>
              </ul>

              <h3 id="backup-pgbackrest">pgBackRest : sauvegardes physiques</h3>

              <p>
                pgBackRest effectue des sauvegardes au niveau fichier
                (physiques) du répertoire de données PostgreSQL :
              </p>

              <ul>
                <li>
                  <strong>Incrémental au niveau bloc</strong> : seuls les blocs
                  modifiés sont sauvegardés, ce qui réduit le temps de backup et
                  le stockage pour les très grandes bases.
                </li>
                <li>
                  <strong>Archivage WAL</strong> : l&apos;archivage continu des
                  journaux Write-Ahead permet une récupération précise à un
                  instant donné.
                </li>
                <li>
                  <strong>Complet, différentiel, incrémental</strong> :
                  plusieurs stratégies de sauvegarde pour différents scénarios
                  de restauration.
                </li>
                <li>
                  <strong>Optimisé pour la volumétrie</strong> : conçu pour les
                  bases où les sauvegardes logiques prendraient trop de temps.
                </li>
              </ul>

              <h2 id="recovery-options">Options de restauration</h2>

              <p>
                Les deux outils offrent des options de restauration flexibles,
                mais avec une granularité différente :
              </p>

              <h3 id="recovery-databasus">Restauration avec Databasus</h3>

              <ul>
                <li>
                  <strong>Récupération à un instant donné (PITR)</strong> :
                  restaurez à la seconde près grâce au rejeu des WAL.
                </li>
                <li>
                  <strong>Restauration complète du cluster</strong> : restaurez
                  l&apos;ensemble du cluster de bases de données à un instant
                  précis à partir des sauvegardes physiques.
                </li>
                <li>
                  <strong>Restauration logique</strong> : restaurez à
                  n&apos;importe quel point de sauvegarde à partir des backups
                  logiques planifiés.
                </li>
                <li>
                  <strong>Restauration en un clic</strong> : téléchargez et
                  restaurez les sauvegardes logiques directement depuis
                  l&apos;interface web.
                </li>
                <li>
                  <strong>Compatibilité entre versions</strong> : les
                  sauvegardes logiques peuvent être restaurées sur d&apos;autres
                  versions de PostgreSQL.
                </li>
              </ul>

              <h3 id="recovery-pgbackrest">Restauration avec pgBackRest</h3>

              <ul>
                <li>
                  <strong>Récupération à un instant donné (PITR)</strong> :
                  restaurez à la seconde près grâce au rejeu des WAL.
                </li>
                <li>
                  <strong>Restauration complète du cluster</strong> : restaurez
                  l&apos;ensemble du cluster de bases de données à un instant
                  précis à partir des sauvegardes physiques.
                </li>
                <li>
                  <strong>Restauration parallèle</strong> : restauration
                  multi-thread pour une récupération plus rapide des grandes
                  bases.
                </li>
                <li>
                  <strong>Restauration delta</strong> : ne restaure que les
                  fichiers modifiés, ce qui réduit le temps de récupération.
                </li>
                <li>
                  <strong>Création de standby</strong> : créez des réplicas
                  PostgreSQL à partir des sauvegardes.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Remarque :</strong> les
                  deux outils prennent en charge la PITR. pgBackRest propose en
                  plus la restauration delta (récupération des seuls fichiers
                  modifiés), les sauvegardes différentielles et la création de
                  standby à partir des sauvegardes.{" "}
                  <a
                    href="/fr/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Découvrez comment Databasus prend en charge la PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">Facilité d&apos;utilisation</h2>

              <p>
                Les deux outils diffèrent radicalement dans leur approche de
                l&apos;expérience utilisateur :
              </p>

              <h3 id="ease-databasus">Expérience utilisateur de Databasus</h3>

              <ul>
                <li>
                  <strong>Interface web</strong> : configuration de tous les
                  réglages de sauvegarde en quelques clics. Aucune ligne de
                  commande requise.
                </li>
                <li>
                  <strong>Installation en 2 minutes</strong> : un script cURL en
                  une ligne ou une simple commande Docker vous met en route
                  immédiatement.
                </li>
                <li>
                  <strong>Supervision visuelle</strong> : le tableau de bord
                  affiche en un coup d&apos;œil l&apos;état des sauvegardes, les
                  contrôles de santé et l&apos;historique.
                </li>
                <li>
                  <strong>Notifications intégrées</strong> : configurez des
                  alertes Slack, Teams, Telegram, e-mail ou webhook directement
                  dans l&apos;interface.
                </li>
                <li>
                  <strong>Aucune expertise PostgreSQL requise</strong> : conçu
                  pour les développeurs qui veulent des sauvegardes fiables sans
                  devenir experts en bases de données.
                </li>
              </ul>

              <h3 id="ease-pgbackrest">Expérience utilisateur de pgBackRest</h3>

              <ul>
                <li>
                  <strong>Interface en ligne de commande</strong> : toutes les
                  opérations passent par des commandes de terminal.
                </li>
                <li>
                  <strong>Fichiers de configuration</strong> : nécessite
                  l&apos;édition manuelle de fichiers de configuration au format
                  INI.
                </li>
                <li>
                  <strong>Configuration de l&apos;archivage WAL</strong> : il
                  faut configurer l&apos;<code>archive_command</code> de
                  PostgreSQL et les réglages associés.
                </li>
                <li>
                  <strong>Courbe d&apos;apprentissage raide</strong> : exige de
                  comprendre les mécanismes internes de PostgreSQL, le
                  fonctionnement des WAL et les stratégies de sauvegarde.
                </li>
                <li>
                  <strong>Expertise DBA attendue</strong> : la documentation
                  suppose une bonne connaissance des concepts
                  d&apos;administration de bases de données.
                </li>
              </ul>

              <p>
                <a
                  href="/fr/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Consulter le guide d&apos;installation de Databasus →
                </a>
              </p>

              <h2 id="team-features">Fonctionnalités d&apos;équipe</h2>

              <p>
                Pour les organisations où plusieurs membres gèrent les
                sauvegardes :
              </p>

              <h3 id="team-databasus">Capacités d&apos;équipe de Databasus</h3>

              <ul>
                <li>
                  <strong>Espaces de travail</strong> : organisez bases de
                  données, notificateurs et stockages par projet ou par équipe.
                  Les utilisateurs ne voient que les espaces auxquels ils sont
                  invités.
                </li>
                <li>
                  <strong>Contrôle d&apos;accès par rôles</strong> : attribuez
                  des permissions de lecteur, d&apos;éditeur ou
                  d&apos;administrateur pour contrôler ce que chaque membre peut
                  faire.
                </li>
                <li>
                  <strong>Journaux d&apos;audit</strong> : tracez toutes les
                  activités et modifications du système. Essentiel pour la
                  conformité et la traçabilité.
                </li>
                <li>
                  <strong>Notifications partagées</strong> : les canaux
                  d&apos;équipe reçoivent automatiquement l&apos;état des
                  sauvegardes.
                </li>
              </ul>

              <h3 id="team-pgbackrest">
                Capacités d&apos;équipe de pgBackRest
              </h3>

              <p>
                pgBackRest est un outil en ligne de commande sans
                fonctionnalités d&apos;équipe intégrées :
              </p>

              <ul>
                <li>
                  Pas de gestion des utilisateurs ni de contrôle d&apos;accès
                </li>
                <li>Pas de journalisation d&apos;audit des opérations</li>
                <li>
                  La coordination d&apos;équipe exige des outils et processus
                  externes
                </li>
                <li>
                  Accès contrôlé via les permissions au niveau OS sur les
                  fichiers de configuration
                </li>
              </ul>

              <p>
                <a
                  href="/fr/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  En savoir plus sur la gestion des accès dans Databasus →
                </a>
              </p>

              <h2 id="security">Sécurité</h2>

              <p>
                Les deux outils offrent des fonctionnalités de sécurité solides
                :
              </p>

              <h3 id="security-databasus">Sécurité de Databasus</h3>

              <ul>
                <li>
                  <strong>Chiffrement AES-256-GCM</strong> : tous les mots de
                  passe, jetons et identifiants sont chiffrés. La clé de
                  chiffrement est stockée séparément de la base de données.
                </li>
                <li>
                  <strong>Chiffrement unique par sauvegarde</strong> : chaque
                  fichier de backup est chiffré avec une clé unique dérivée de
                  la clé maîtresse, de l&apos;identifiant de la sauvegarde et
                  d&apos;un sel aléatoire.
                </li>
                <li>
                  <strong>Accès en lecture seule à la base</strong> : seules les
                  permissions SELECT sont exigées, ce qui empêche toute
                  corruption des données même en cas de compromission.
                </li>
              </ul>

              <h3 id="security-pgbackrest">Sécurité de pgBackRest</h3>

              <ul>
                <li>
                  <strong>Chiffrement du dépôt</strong> : les dépôts de
                  sauvegarde peuvent être chiffrés en AES-256.
                </li>
                <li>
                  <strong>Transport TLS/SSH</strong> : communication sécurisée
                  pour les opérations à distance.
                </li>
                <li>
                  <strong>Vérification par sommes de contrôle</strong> : valide
                  l&apos;intégrité des sauvegardes lors de la création et de la
                  restauration.
                </li>
              </ul>

              <p>
                <a
                  href="/fr/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  En savoir plus sur la sécurité de Databasus →
                </a>
              </p>

              <h2 id="storage-options">Options de stockage</h2>

              <p>
                Les deux outils prennent en charge plusieurs destinations de
                stockage, Databasus offrant des options plus grand public :
              </p>

              <h3 id="storage-databasus">Stockage avec Databasus</h3>

              <ul>
                <li>Stockage local</li>
                <li>Amazon S3 et services compatibles S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (stockage en réseau)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-pgbackrest">Stockage avec pgBackRest</h3>

              <ul>
                <li>Stockage local (POSIX, CIFS)</li>
                <li>Amazon S3 et services compatibles S3</li>
                <li>Cloudflare R2 (compatible S3)</li>
                <li>Azure Blob Storage</li>
                <li>NAS (stockage en réseau)</li>
                <li>Google Cloud Storage</li>
                <li>SFTP</li>
              </ul>

              <p>
                <a
                  href="/fr/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Voir toutes les options de stockage de Databasus →
                </a>
              </p>

              <h2 id="notifications">Notifications</h2>

              <p>Rester informé de l&apos;état des sauvegardes :</p>

              <h3 id="notifications-databasus">Notifications avec Databasus</h3>

              <p>
                Prise en charge intégrée de plusieurs canaux de notification :
              </p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>E-mail</li>
                <li>Webhooks</li>
              </ul>

              <h3 id="notifications-pgbackrest">
                Notifications avec pgBackRest
              </h3>

              <p>
                pgBackRest n&apos;a pas de notifications intégrées. Elles
                nécessitent :
              </p>

              <ul>
                <li>
                  Des scripts personnalisés autour des commandes de sauvegarde
                </li>
                <li>
                  L&apos;intégration d&apos;outils de supervision externes
                </li>
                <li>
                  Une analyse manuelle des journaux et la mise en place
                  d&apos;alertes
                </li>
              </ul>

              <p>
                <a
                  href="/fr/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Voir tous les canaux de notification de Databasus →
                </a>
              </p>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Databasus et pgBackRest répondent à des besoins différents dans
                l&apos;écosystème de sauvegarde PostgreSQL. Le bon choix dépend
                de la taille de vos bases, de la structure de votre équipe et de
                vos exigences techniques.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Choisissez Databasus si :
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Vous êtes un développeur individuel, une équipe ou une
                    entreprise à la recherche d&apos;une solution de backup
                    intuitive
                  </li>
                  <li>
                    Vous préférez une interface web aux outils en ligne de
                    commande
                  </li>
                  <li>
                    Vous avez besoin de fonctionnalités de collaboration
                    (espaces de travail, RBAC, journaux d&apos;audit)
                  </li>
                  <li>
                    Vous voulez des notifications intégrées vers Slack, Teams,
                    Telegram, etc.
                  </li>
                  <li>
                    Vous voulez gérer les backups de plusieurs bases de données
                    depuis un seul tableau de bord, avec planification,
                    notifications et fonctionnalités d&apos;équipe
                  </li>
                  <li>
                    Vous voulez une mise en route rapide sans expertise
                    PostgreSQL poussée
                  </li>
                  <li>
                    Vous utilisez des bases gérées dans le cloud (AWS RDS,
                    Google Cloud SQL, Azure) ou du PostgreSQL auto-hébergé
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Choisissez pgBackRest si :</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Vous avez besoin de sauvegardes physiques ou incrémentales
                    sur PostgreSQL antérieur à 17 (pgBackRest embarque son
                    propre moteur de backup)
                  </li>
                  <li>
                    Vous avez besoin de sauvegardes différentielles ou de la
                    restauration delta (fichiers modifiés uniquement)
                  </li>
                  <li>
                    Vous avez besoin de créer des standby à partir des
                    sauvegardes pour la haute disponibilité
                  </li>
                  <li>
                    Vous préférez les outils en ligne de commande et les flux de
                    travail infrastructure-as-code
                  </li>
                  <li>
                    Votre équipe a l&apos;expertise PostgreSQL nécessaire pour
                    l&apos;exploiter et le régler
                  </li>
                </ul>
              </div>

              <p>
                Les deux outils prennent en charge les sauvegardes physiques,
                l&apos;archivage WAL et la PITR, et tous deux sont conçus pour
                la reprise après sinistre avec un RTO et un RPO minimaux.
                Databasus convient aux bases de données de toute taille et de
                toute complexité, et vous offre une interface web, des
                fonctionnalités d&apos;équipe et des backups à la fois logiques
                et physiques, pour les bases auto-hébergées comme gérées dans le
                cloud.
                <br />
                <br />
                pgBackRest est le meilleur choix quand vous voulez exploiter
                vous-même le moteur de sauvegarde, avez besoin de backups
                physiques sur PostgreSQL antérieur à 17, ou comptez sur ses
                sauvegardes différentielles et sa restauration delta.
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
