import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs WAL-G : comparaison d'outils de sauvegarde PostgreSQL",
  description:
    "Comparez les outils de sauvegarde PostgreSQL Databasus et WAL-G : approche de backup, support multi-bases, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
  keywords: [
    "Databasus vs WAL-G",
    "comparaison sauvegarde PostgreSQL",
    "alternative à WAL-G",
    "outils de sauvegarde PostgreSQL",
    "comparaison de sauvegarde de bases de données",
    "pg_dump vs archivage WAL",
    "backup auto-hébergé",
    "PITR PostgreSQL",
    "archivage WAL",
    "sauvegarde multi-bases",
  ],
  openGraph: {
    title: "Databasus vs WAL-G : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et WAL-G : approche de backup, support multi-bases, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
    type: "article",
    url: getLocalizedUrl("fr", "databasus-vs-wal-g"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs WAL-G : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et WAL-G : approche de backup, support multi-bases, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "databasus-vs-wal-g"),
    languages: getLanguageAlternates("databasus-vs-wal-g"),
  },
  robots: "index, follow",
};

export default function DatabasusVsWalGPage() {
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
              "Databasus vs WAL-G : comparaison d'outils de sauvegarde PostgreSQL",
            description:
              "Une comparaison détaillée des outils de sauvegarde PostgreSQL Databasus et WAL-G : approche de backup, support multi-bases, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
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
              <h1 id="databasus-vs-wal-g">Databasus vs WAL-G</h1>

              <p className="text-lg text-gray-400">
                Databasus et WAL-G sont tous deux conçus pour la reprise après
                sinistre avec un RTO et un RPO minimaux, et tous deux prennent
                en charge les sauvegardes physiques PostgreSQL, l&apos;archivage
                WAL et la récupération à un instant donné (PITR). Databasus
                exécute ces backups à distance sur la pile native de PostgreSQL
                17 : il réutilise l&apos;outillage éprouvé de PostgreSQL au lieu
                de le réinventer, le tout derrière une interface web intuitive.
                Il convient aux bases de données de toute taille et de toute
                complexité. Les sauvegardes physiques exigent PostgreSQL 17 ou
                plus récent ; sur les versions antérieures, seuls les backups
                logiques <code>pg_dump</code> sont disponibles. WAL-G est un
                outil en ligne de commande qui embarque son propre moteur : il
                couvre les sauvegardes physiques sur des versions de PostgreSQL
                bien plus anciennes, utilise un protocole de streaming
                personnalisé pour des performances légèrement meilleures, prend
                en charge les backups delta (pages modifiées uniquement) et
                couvre davantage de moteurs de bases de données, dont MS SQL,
                FoundationDB et Greenplum.
              </p>

              <h2 id="quick-comparison">Comparaison rapide</h2>

              <p>
                Voici un aperçu rapide des principales différences entre
                Databasus et WAL-G :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fonctionnalité</th>
                    <th>Databasus</th>
                    <th>WAL-G</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Gestion des sauvegardes</td>
                    <td data-label="Databasus">✅ Oui (plusieurs bases)</td>
                    <td data-label="WAL-G">❌ Non (une seule base)</td>
                  </tr>
                  <tr>
                    <td>Support d&apos;autres bases de données</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="WAL-G">✅ PostgreSQL, MySQL, MS SQL</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Databasus">Interface web</td>
                    <td data-label="WAL-G">Ligne de commande uniquement</td>
                  </tr>
                  <tr>
                    <td>Type de sauvegarde</td>
                    <td data-label="Databasus">Logique + physique</td>
                    <td data-label="WAL-G">Physique (archivage WAL)</td>
                  </tr>
                  <tr>
                    <td>Version PostgreSQL pour les sauvegardes physiques</td>
                    <td data-label="Databasus">17+ (natif)</td>
                    <td data-label="WAL-G">9.x+ (moteur propre)</td>
                  </tr>
                  <tr>
                    <td>Planification des sauvegardes</td>
                    <td data-label="Databasus">✅ Planificateur intégré</td>
                    <td data-label="WAL-G">Outil externe requis (cron)</td>
                  </tr>
                  <tr>
                    <td>Options de restauration</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="WAL-G">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes incrémentales</td>
                    <td data-label="Databasus">✅ Au niveau bloc (PG 17+)</td>
                    <td data-label="WAL-G">
                      Backups delta (pages modifiées uniquement)
                    </td>
                  </tr>
                  <tr>
                    <td>Sauvegardes à distance</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="WAL-G">❌ Non (exécution locale)</td>
                  </tr>
                  <tr>
                    <td>Fonctionnalités d&apos;équipe</td>
                    <td data-label="Databasus">
                      ✅ Espaces de travail, RBAC, journaux d&apos;audit
                    </td>
                    <td data-label="WAL-G">
                      ❌ Permissions au niveau OS uniquement
                    </td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, e-mail
                    </td>
                    <td data-label="WAL-G">❌ Scripts personnalisés requis</td>
                  </tr>
                  <tr>
                    <td>Chiffrement</td>
                    <td data-label="Databasus">AES-256-GCM intégré</td>
                    <td data-label="WAL-G">GPG ou libsodium</td>
                  </tr>
                  <tr>
                    <td>Courbe d&apos;apprentissage</td>
                    <td data-label="Databasus">Minimale</td>
                    <td data-label="WAL-G">Maîtrise de la CLI requise</td>
                  </tr>
                  <tr>
                    <td>Installation</td>
                    <td data-label="Databasus">
                      Script en une ligne ou Docker
                    </td>
                    <td data-label="WAL-G">
                      Téléchargement du binaire + configuration
                    </td>
                  </tr>
                  <tr>
                    <td>Adapté aux bases auto-hébergées</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="WAL-G">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Adapté aux bases cloud</td>
                    <td data-label="Databasus">
                      ✅ Oui (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="WAL-G">
                      ❌ Backup uniquement (pas de restauration vers le cloud)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="database-focus">Bases de données ciblées</h2>

              <p>
                L&apos;une des différences les plus importantes entre ces outils
                est leur périmètre de bases de données :
              </p>

              <h3 id="focus-databasus">
                Databasus : gestion complète des sauvegardes
              </h3>

              <p>
                Databasus est conçu pour une gestion complète des sauvegardes
                sur plusieurs systèmes de bases de données, avec un accent mis
                sur la facilité d&apos;utilisation :
              </p>

              <ul>
                <li>
                  <strong>Support multi-bases</strong> : gérez les backups de
                  PostgreSQL, MySQL, MariaDB et MongoDB depuis une seule
                  interface.
                </li>
                <li>
                  <strong>Expérience unifiée</strong> : l&apos;interface, les
                  flux de travail et les fonctionnalités se comportent de la
                  même façon sur toutes les bases prises en charge.
                </li>
                <li>
                  <strong>Versions prises en charge</strong> : PostgreSQL des
                  versions 12 à 18, avec des optimisations propres à chaque
                  version.
                </li>
                <li>
                  <strong>Gestion simplifiée</strong> : tout l&apos;effort de
                  développement porte sur l&apos;amélioration de
                  l&apos;expérience de gestion des sauvegardes.
                </li>
              </ul>

              <h3 id="focus-wal-g">WAL-G : support multi-bases</h3>

              <p>
                WAL-G a commencé comme outil de sauvegarde PostgreSQL, puis
                s&apos;est étendu à plusieurs systèmes de bases de données :
              </p>

              <ul>
                <li>
                  <strong>PostgreSQL</strong> : l&apos;implémentation
                  d&apos;origine et la plus mature.
                </li>
                <li>
                  <strong>MySQL/MariaDB</strong> : sauvegardes basées sur les
                  binlogs.
                </li>
                <li>
                  <strong>MS SQL Server</strong> : sauvegardes de SQL Server
                  sous Windows.
                </li>
                <li>
                  <strong>MongoDB</strong> : sauvegarde de bases de données
                  documentaires.
                </li>
                <li>
                  <strong>FoundationDB</strong> : support de bases de données
                  distribuées.
                </li>
                <li>
                  <strong>Greenplum</strong> : sauvegarde d&apos;entrepôts de
                  données.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">
                    Quand une gestion complète compte :
                  </strong>{" "}
                  si vous devez gérer les backups de plusieurs bases de données
                  avec une interface unifiée, Databasus offre une expérience
                  simplifiée. Vous obtenez une gestion centralisée des
                  sauvegardes, avec des fonctionnalités d&apos;équipe, sans
                  jongler entre des outils différents pour chaque base.
                </p>
              </div>

              <h2 id="target-audience">Public cible</h2>

              <p>
                Les deux outils s&apos;adressent à des profils différents, en
                fonction de leur philosophie de conception :
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
                  <strong>Environnements multi-bases</strong> : les
                  organisations qui exploitent PostgreSQL, MySQL, MariaDB ou
                  MongoDB profitent d&apos;une gestion centralisée des
                  sauvegardes.
                </li>
                <li>
                  <strong>DBA et reprise après sinistre</strong> : sauvegardes
                  physiques, archivage WAL et PITR pour les systèmes critiques
                  exigeant une perte de données quasi nulle.
                </li>
                <li>
                  <strong>Ingénieurs DevOps</strong> : le mode agent
                  s&apos;intègre à l&apos;infrastructure existante, tandis que
                  l&apos;interface web et l&apos;API offrent visibilité et
                  contrôle sans scripts personnalisés.
                </li>
              </ul>

              <h3 id="audience-wal-g">Public de WAL-G</h3>

              <p>
                WAL-G est conçu pour les utilisateurs à l&apos;aise avec les
                outils en ligne de commande :
              </p>

              <ul>
                <li>
                  <strong>Ingénieurs DevOps</strong> : ceux qui préfèrent
                  l&apos;infrastructure-as-code et les flux de travail en CLI.
                </li>
                <li>
                  <strong>Environnements multi-bases</strong> : les
                  organisations qui exploitent PostgreSQL aux côtés de MySQL,
                  MongoDB ou d&apos;autres bases prises en charge.
                </li>
                <li>
                  <strong>Déploiements cloud-native</strong> : les équipes qui
                  utilisent Kubernetes ou des environnements conteneurisés, où
                  les outils CLI s&apos;intègrent bien.
                </li>
                <li>
                  <strong>Support étendu de bases de données</strong> : les
                  équipes ayant besoin de sauvegarder MS SQL, FoundationDB ou
                  Greenplum en plus de PostgreSQL.
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
                  sur d&apos;autres versions de PostgreSQL. C&apos;est aussi le
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

              <h3 id="backup-wal-g">
                WAL-G : sauvegardes physiques avec archivage WAL
              </h3>

              <p>
                WAL-G effectue des sauvegardes au niveau fichier (physiques)
                avec archivage WAL continu :
              </p>

              <ul>
                <li>
                  <strong>Sauvegardes de base</strong> : copies complètes au
                  niveau fichier du répertoire de données PostgreSQL.
                </li>
                <li>
                  <strong>Backups delta</strong> : seules les pages modifiées
                  sont sauvegardées, ce qui réduit le stockage et le temps de
                  transfert.
                </li>
                <li>
                  <strong>Archivage WAL</strong> : l&apos;archivage continu des
                  journaux Write-Ahead permet la récupération à un instant
                  donné.
                </li>
                <li>
                  <strong>Optimisation copy-on-write</strong> : gestion efficace
                  des blocs de données inchangés.
                </li>
              </ul>

              <h2 id="recovery-options">Options de restauration</h2>

              <p>
                Les deux outils offrent des capacités de restauration, mais avec
                une granularité différente :
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

              <h3 id="recovery-wal-g">Restauration avec WAL-G</h3>

              <ul>
                <li>
                  <strong>Récupération à un instant donné (PITR)</strong> :
                  restaurez à la seconde près grâce au rejeu des WAL, en
                  minimisant la perte de données.
                </li>
                <li>
                  <strong>Restauration complète du cluster</strong> : restaurez
                  l&apos;ensemble du cluster de bases de données à un instant
                  précis.
                </li>
                <li>
                  <strong>Restauration delta</strong> : seules les pages
                  modifiées sont récupérées, ce qui accélère la restauration.
                </li>
                <li>
                  <strong>Création de standby</strong> : créez des réplicas
                  PostgreSQL à partir des sauvegardes pour des configurations à
                  haute disponibilité.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Remarque :</strong> les
                  deux outils prennent en charge la PITR. WAL-G propose en plus
                  la restauration delta (récupération des seules pages
                  modifiées) et utilise un protocole de streaming personnalisé
                  pour des performances légèrement meilleures à grande échelle.{" "}
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
                Les deux outils diffèrent nettement dans leur approche de
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

              <h3 id="ease-wal-g">Expérience utilisateur de WAL-G</h3>

              <ul>
                <li>
                  <strong>Interface en ligne de commande</strong> : toutes les
                  opérations passent par des commandes de terminal comme{" "}
                  <code>wal-g backup-push</code>,{" "}
                  <code>wal-g backup-fetch</code>.
                </li>
                <li>
                  <strong>Variables d&apos;environnement</strong> :
                  configuration principalement via des variables
                  d&apos;environnement plutôt que des fichiers de configuration.
                </li>
                <li>
                  <strong>Planification externe</strong> : nécessite des tâches
                  cron ou une orchestration externe pour automatiser les
                  sauvegardes.
                </li>
                <li>
                  <strong>Configuration de l&apos;archivage WAL</strong> : il
                  faut configurer l&apos;<code>archive_command</code> de
                  PostgreSQL pour l&apos;intégrer à WAL-G.
                </li>
                <li>
                  <strong>Maîtrise de la CLI attendue</strong> : la
                  documentation suppose une bonne connaissance des outils en
                  ligne de commande et des scripts shell.
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

              <h3 id="team-wal-g">Capacités d&apos;équipe de WAL-G</h3>

              <p>
                WAL-G est un outil en ligne de commande sans fonctionnalités
                d&apos;équipe intégrées :
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
                  Accès contrôlé via les permissions au niveau OS et les
                  politiques IAM du cloud
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
                Les deux outils offrent des fonctionnalités de sécurité, mais
                avec des approches différentes :
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

              <h3 id="security-wal-g">Sécurité de WAL-G</h3>

              <ul>
                <li>
                  <strong>Chiffrement GPG</strong> : prend en charge le
                  chiffrement des fichiers de sauvegarde via GPG.
                </li>
                <li>
                  <strong>Chiffrement libsodium</strong> : chiffrement
                  alternatif via la bibliothèque libsodium.
                </li>
                <li>
                  <strong>Intégration IAM cloud</strong> : s&apos;appuie sur
                  l&apos;IAM du fournisseur cloud pour contrôler l&apos;accès au
                  stockage.
                </li>
                <li>
                  <strong>Pas de gestion intégrée des identifiants</strong> :
                  repose sur des variables d&apos;environnement ou une gestion
                  de secrets externe.
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
                Les deux outils prennent en charge le stockage cloud, avec des
                priorités différentes :
              </p>

              <h3 id="storage-databasus">Stockage avec Databasus</h3>

              <p>Des options accessibles pour des cas d&apos;usage variés :</p>

              <ul>
                <li>Stockage local</li>
                <li>Amazon S3 et services compatibles S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (stockage en réseau)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-wal-g">Stockage avec WAL-G</h3>

              <p>Des options de stockage cloud-native :</p>

              <ul>
                <li>Amazon S3</li>
                <li>Google Cloud Storage (GCS)</li>
                <li>Azure Blob Storage</li>
                <li>Swift (OpenStack)</li>
                <li>Système de fichiers local</li>
                <li>SSH/SFTP</li>
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

              <h3 id="notifications-wal-g">Notifications avec WAL-G</h3>

              <p>
                WAL-G n&apos;a pas de notifications intégrées. Elles nécessitent
                :
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
                <li>
                  L&apos;intégration avec des outils comme Prometheus, Grafana
                  ou des solutions maison
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

              <h2 id="compression">Compression</h2>

              <p>
                Les deux outils offrent la compression pour réduire la taille
                des sauvegardes :
              </p>

              <h3 id="compression-databasus">Compression avec Databasus</h3>

              <ul>
                <li>
                  <strong>Compression zstd</strong> : utilise zstd au niveau 5
                  pour un bon équilibre entre vitesse et taux de compression.
                </li>
                <li>
                  <strong>Taille réduite de 4 à 8 fois</strong> : taux de
                  compression typiques avec seulement ~20 % de surcoût
                  d&apos;exécution.
                </li>
                <li>
                  <strong>Automatique</strong> : la compression est activée par
                  défaut, sans configuration.
                </li>
              </ul>

              <h3 id="compression-wal-g">Compression avec WAL-G</h3>

              <ul>
                <li>
                  <strong>Plusieurs algorithmes</strong> : prend en charge LZ4,
                  LZMA, Brotli et zstd.
                </li>
                <li>
                  <strong>Niveaux configurables</strong> : ajustez finement le
                  compromis entre taux de compression et vitesse.
                </li>
                <li>
                  <strong>Compression par fichier</strong> : les fichiers WAL et
                  les sauvegardes de base peuvent utiliser des réglages
                  différents.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Databasus et WAL-G répondent à des besoins différents dans
                l&apos;écosystème de sauvegarde PostgreSQL. Le bon choix dépend
                de votre environnement de bases de données, de la structure de
                votre équipe et de vos préférences opérationnelles.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Choisissez Databasus si :
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Vous avez besoin d&apos;une gestion complète des sauvegardes
                    PostgreSQL depuis une seule interface
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
                    Vous voulez une planification intégrée sans configuration
                    cron externe
                  </li>
                  <li>
                    Vous voulez gérer les backups de plusieurs bases de données
                    depuis un seul tableau de bord, avec planification,
                    notifications et fonctionnalités d&apos;équipe
                  </li>
                  <li>
                    Vous voulez une mise en route rapide sans expertise poussée
                    en bases de données
                  </li>
                  <li>
                    Le chiffrement intégré des sauvegardes est important pour
                    vous
                  </li>
                  <li>
                    Vous utilisez des bases gérées dans le cloud (AWS RDS,
                    Google Cloud SQL, Azure) ou auto-hébergées
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Choisissez WAL-G si :</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Vous avez besoin de sauvegardes physiques ou incrémentales
                    sur PostgreSQL antérieur à 17 (WAL-G embarque son propre
                    moteur de backup)
                  </li>
                  <li>
                    Vous avez besoin de backups delta (pages modifiées
                    uniquement) pour réduire le stockage et le temps de
                    transfert
                  </li>
                  <li>
                    Vous avez besoin du support de MS SQL, FoundationDB ou
                    Greenplum
                  </li>
                  <li>
                    Vous préférez les outils en ligne de commande et les flux de
                    travail infrastructure-as-code
                  </li>
                  <li>
                    Vous voulez plusieurs algorithmes de compression (LZ4, LZMA,
                    Brotli, zstd) avec un contrôle fin
                  </li>
                  <li>
                    Votre équipe a l&apos;expertise DevOps pour gérer des outils
                    en CLI
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
                WAL-G reste un excellent choix pour les équipes qui préfèrent
                les flux de travail en CLI et ont besoin de ses atouts
                spécifiques : backups delta (pages modifiées uniquement),
                protocole de streaming personnalisé pour des performances
                légèrement meilleures et support de moteurs de bases de données
                supplémentaires au-delà de PostgreSQL.
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
