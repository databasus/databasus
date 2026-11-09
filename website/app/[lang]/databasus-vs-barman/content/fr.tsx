import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs Barman : comparaison d'outils de sauvegarde PostgreSQL",
  description:
    "Comparez les outils de sauvegarde PostgreSQL Databasus et Barman : approche de backup, capacités PITR, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
  keywords: [
    "Databasus vs Barman",
    "comparaison sauvegarde PostgreSQL",
    "alternative à Barman",
    "outils de sauvegarde PostgreSQL",
    "comparaison de sauvegarde de bases de données",
    "pg_dump vs sauvegarde physique",
    "backup auto-hébergé",
    "PITR PostgreSQL",
    "archivage WAL",
    "reprise après sinistre PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs Barman : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et Barman : approche de backup, capacités PITR, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
    type: "article",
    url: getLocalizedUrl("fr", "databasus-vs-barman"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs Barman : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et Barman : approche de backup, capacités PITR, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "databasus-vs-barman"),
    languages: getLanguageAlternates("databasus-vs-barman"),
  },
  robots: "index, follow",
};

export default function DatabasusVsBarmanPage() {
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
              "Databasus vs Barman : comparaison d'outils de sauvegarde PostgreSQL",
            description:
              "Une comparaison détaillée des outils de sauvegarde PostgreSQL Databasus et Barman : approche de backup, capacités PITR, facilité d'utilisation, fonctionnalités d'équipe et quand choisir chaque outil.",
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
              <h1 id="databasus-vs-barman">Databasus vs Barman</h1>

              <p className="text-lg text-gray-400">
                Databasus et Barman sont tous deux conçus pour la reprise après
                sinistre avec un RTO et un RPO minimaux, et tous deux prennent
                en charge les sauvegardes physiques, l&apos;archivage WAL et la
                récupération à un instant donné (PITR). Databasus exécute ces
                backups à distance sur la pile native de PostgreSQL 17 : il
                réutilise l&apos;outillage éprouvé de PostgreSQL au lieu de le
                réinventer, le tout derrière une interface web intuitive avec
                des fonctionnalités d&apos;équipe et le support de plusieurs
                moteurs de bases de données. Il convient aux bases de données de
                toute taille et de toute complexité. Les sauvegardes physiques
                exigent PostgreSQL 17 ou plus récent ; sur les versions
                antérieures, seuls les backups logiques <code>pg_dump</code>{" "}
                sont disponibles. Barman (Backup and Recovery Manager) embarque
                son propre moteur de sauvegarde : il couvre les sauvegardes
                physiques sur des versions de PostgreSQL bien plus anciennes et
                ajoute des fonctionnalités avancées comme les backups
                incrémentaux basés sur rsync, l&apos;intégration de la
                réplication en streaming et la géo-redondance Barman vers
                Barman.
              </p>

              <h2 id="quick-comparison">Comparaison rapide</h2>

              <p>
                Voici un aperçu rapide des principales différences entre
                Databasus et Barman :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fonctionnalité</th>
                    <th>Databasus</th>
                    <th>Barman</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Public cible</td>
                    <td data-label="Databasus">
                      Particuliers, équipes, DBA, entreprises
                    </td>
                    <td data-label="Barman">DBA, entreprises</td>
                  </tr>
                  <tr>
                    <td>Support d&apos;autres bases de données</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="Barman">❌ PostgreSQL uniquement</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Databasus">Interface web</td>
                    <td data-label="Barman">Ligne de commande uniquement</td>
                  </tr>
                  <tr>
                    <td>Type de sauvegarde</td>
                    <td data-label="Databasus">Logique + physique</td>
                    <td data-label="Barman">Physique (niveau fichier)</td>
                  </tr>
                  <tr>
                    <td>Version PostgreSQL pour les sauvegardes physiques</td>
                    <td data-label="Databasus">17+ (natif)</td>
                    <td data-label="Barman">9.x+ (moteur propre)</td>
                  </tr>
                  <tr>
                    <td>Options de restauration</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="Barman">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes incrémentales</td>
                    <td data-label="Databasus">✅ Au niveau bloc (PG 17+)</td>
                    <td data-label="Barman">Incrémentales basées sur rsync</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes à distance</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="Barman">
                      ❌ Non (accès au système de fichiers requis)
                    </td>
                  </tr>
                  <tr>
                    <td>Gestion multi-serveurs</td>
                    <td data-label="Databasus">Planification par base</td>
                    <td data-label="Barman">
                      Serveur de sauvegarde centralisé
                    </td>
                  </tr>
                  <tr>
                    <td>Fonctionnalités d&apos;équipe</td>
                    <td data-label="Databasus">
                      ✅ Espaces de travail, RBAC, journaux d&apos;audit
                    </td>
                    <td data-label="Barman">
                      ❌ Permissions au niveau OS uniquement
                    </td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, e-mail
                    </td>
                    <td data-label="Barman">❌ Scripts personnalisés requis</td>
                  </tr>
                  <tr>
                    <td>Courbe d&apos;apprentissage</td>
                    <td data-label="Databasus">Minimale</td>
                    <td data-label="Barman">Expertise DBA requise</td>
                  </tr>
                  <tr>
                    <td>Installation</td>
                    <td data-label="Databasus">
                      Script en une ligne ou Docker
                    </td>
                    <td data-label="Barman">Configuration manuelle requise</td>
                  </tr>
                  <tr>
                    <td>Gestion des sauvegardes</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="Barman">❌ Non</td>
                  </tr>
                  <tr>
                    <td>Adapté aux bases auto-hébergées</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="Barman">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Adapté aux bases cloud</td>
                    <td data-label="Databasus">
                      ✅ Oui (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="Barman">
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

              <h3 id="audience-barman">Public de Barman</h3>

              <p>
                Barman est spécifiquement conçu pour les administrateurs de
                bases de données (DBA) qui gèrent une infrastructure PostgreSQL
                d&apos;entreprise :
              </p>

              <ul>
                <li>
                  <strong>DBA en entreprise</strong> : les professionnels qui
                  ont besoin d&apos;une gestion centralisée des sauvegardes de
                  plusieurs serveurs PostgreSQL depuis un serveur de backup
                  dédié.
                </li>
                <li>
                  <strong>
                    Équipes ayant besoin d&apos;incrémental via rsync
                  </strong>{" "}
                  : la comparaison au niveau fichier réduit le temps de
                  sauvegarde et l&apos;usage réseau pour les gros clusters.
                </li>
                <li>
                  <strong>Exigences de géo-redondance</strong> : réplication
                  Barman vers Barman pour une redondance géographique entre
                  centres de données.
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
                  <strong>Compression efficace</strong> : utilise la compression
                  zstd (niveau 5) pour les sauvegardes logiques comme physiques.
                </li>
                <li>
                  <strong>Accès en lecture seule</strong> : les sauvegardes
                  logiques n&apos;exigent que des permissions SELECT, ce qui
                  minimise les risques de sécurité.
                </li>
              </ul>

              <h3 id="backup-barman">Barman : sauvegardes physiques</h3>

              <p>
                Barman effectue des sauvegardes au niveau fichier (physiques) du
                répertoire de données PostgreSQL :
              </p>

              <ul>
                <li>
                  <strong>Sauvegarde complète du cluster</strong> : capture
                  l&apos;ensemble du cluster de bases de données au niveau du
                  système de fichiers via rsync ou pg_basebackup.
                </li>
                <li>
                  <strong>Archivage WAL</strong> : archive en continu les
                  journaux Write-Ahead pour la récupération à un instant donné.
                </li>
                <li>
                  <strong>Incrémental avec rsync</strong> : utilise rsync pour
                  ne transférer que les fichiers modifiés, réduisant le temps de
                  sauvegarde et l&apos;usage réseau.
                </li>
                <li>
                  <strong>Intégration de la réplication en streaming</strong> :
                  peut recevoir les fichiers WAL via le protocole de réplication
                  en streaming pour un archivage en temps réel.
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

              <h3 id="recovery-barman">Restauration avec Barman</h3>

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
                  <strong>Restauration à distance</strong> : restaurez des bases
                  de données vers des serveurs distants via SSH.
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
                  deux outils prennent en charge la PITR. Barman propose en plus
                  la création de standby à partir des sauvegardes et la
                  restauration à distance via SSH vers d&apos;autres serveurs,
                  ce qui peut être précieux pour les configurations à haute
                  disponibilité.{" "}
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

              <h3 id="ease-barman">Expérience utilisateur de Barman</h3>

              <ul>
                <li>
                  <strong>Interface en ligne de commande</strong> : toutes les
                  opérations passent par des commandes de terminal comme{" "}
                  <code>barman backup</code>, <code>barman recover</code>.
                </li>
                <li>
                  <strong>Fichiers de configuration</strong> : nécessite
                  l&apos;édition manuelle de fichiers de configuration au format
                  INI pour chaque serveur.
                </li>
                <li>
                  <strong>Configuration de l&apos;archivage WAL</strong> : il
                  faut configurer l&apos;<code>archive_command</code> de
                  PostgreSQL ou les réglages de réplication en streaming.
                </li>
                <li>
                  <strong>Gestion des clés SSH</strong> : nécessite la mise en
                  place de clés SSH entre le serveur Barman et les serveurs
                  PostgreSQL.
                </li>
                <li>
                  <strong>Expertise DBA attendue</strong> : la documentation
                  suppose une bonne connaissance des mécanismes internes de
                  PostgreSQL et du fonctionnement des WAL.
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

              <h3 id="team-barman">Capacités d&apos;équipe de Barman</h3>

              <p>
                Barman est un outil en ligne de commande sans fonctionnalités
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
                  Accès contrôlé via les permissions au niveau OS et les clés
                  SSH
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

              <h3 id="security-barman">Sécurité de Barman</h3>

              <ul>
                <li>
                  <strong>Communication via SSH</strong> : utilise SSH pour des
                  échanges sécurisés entre le serveur Barman et les serveurs
                  PostgreSQL.
                </li>
                <li>
                  <strong>Pas de chiffrement intégré</strong> : Barman
                  n&apos;offre pas de chiffrement intégré des sauvegardes. Il
                  faut recourir à des outils externes ou à un stockage chiffré.
                </li>
                <li>
                  <strong>Sécurité au niveau OS</strong> : repose sur les
                  permissions du système de fichiers et la gestion des clés SSH
                  pour le contrôle d&apos;accès.
                </li>
                <li>
                  <strong>Vérification par sommes de contrôle</strong> : valide
                  l&apos;intégrité des sauvegardes à l&apos;aide de sommes de
                  contrôle.
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
                Les deux outils prennent en charge des destinations de stockage
                différentes :
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

              <h3 id="storage-barman">Stockage avec Barman</h3>

              <p>Des options de stockage orientées entreprise :</p>

              <ul>
                <li>Stockage local (systèmes de fichiers POSIX)</li>
                <li>Amazon S3 et stockage objet compatible S3</li>
                <li>
                  Redondance géographique via la réplication Barman vers Barman
                </li>
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

              <h3 id="notifications-barman">Notifications avec Barman</h3>

              <p>
                Barman n&apos;a pas de notifications intégrées. Elles
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
                <li>
                  L&apos;intégration avec des outils comme Nagios, Zabbix ou des
                  solutions maison
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

              <h2 id="multi-server-management">Gestion multi-serveurs</h2>

              <p>
                Les deux outils peuvent gérer les sauvegardes de plusieurs
                serveurs PostgreSQL, mais avec des approches différentes :
              </p>

              <h3 id="multi-databasus">Approche de Databasus</h3>

              <ul>
                <li>
                  <strong>Planification par base</strong> : chaque base de
                  données peut avoir son propre calendrier de sauvegarde et sa
                  propre destination de stockage.
                </li>
                <li>
                  <strong>Organisation en espaces de travail</strong> :
                  regroupez les bases liées dans des espaces de travail pour une
                  gestion plus simple.
                </li>
                <li>
                  <strong>Tableau de bord unifié</strong> : visualisez toutes
                  les sauvegardes et leur état dans une seule interface web.
                </li>
              </ul>

              <h3 id="multi-barman">Approche de Barman</h3>

              <ul>
                <li>
                  <strong>Serveur de sauvegarde centralisé</strong> : un serveur
                  Barman dédié gère les backups de plusieurs instances
                  PostgreSQL.
                </li>
                <li>
                  <strong>Configuration par serveur</strong> : chaque serveur
                  PostgreSQL nécessite son propre fichier de configuration sur
                  le serveur Barman.
                </li>
                <li>
                  <strong>Géo-redondance</strong> : les serveurs Barman peuvent
                  se répliquer vers d&apos;autres serveurs Barman pour une
                  redondance géographique.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Databasus et Barman répondent à des besoins différents dans
                l&apos;écosystème de sauvegarde PostgreSQL. Le bon choix dépend
                de vos exigences de restauration, de la structure de votre
                équipe et de votre expertise technique.
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
                    Le chiffrement intégré des sauvegardes est important pour
                    vous
                  </li>
                  <li>
                    Vous utilisez des bases gérées dans le cloud (AWS RDS,
                    Google Cloud SQL, Azure) ou du PostgreSQL auto-hébergé
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Choisissez Barman si :</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Vous avez besoin de sauvegardes physiques ou incrémentales
                    sur PostgreSQL antérieur à 17 (Barman embarque son propre
                    moteur de backup)
                  </li>
                  <li>
                    Vous avez besoin de backups incrémentaux basés sur rsync
                    (comparaison au niveau fichier) pour réduire le temps de
                    transfert
                  </li>
                  <li>
                    Il vous faut l&apos;intégration de la réplication en
                    streaming pour un archivage WAL en temps réel
                  </li>
                  <li>
                    Vous cherchez une redondance géographique Barman vers Barman
                  </li>
                  <li>
                    Vous voulez créer des standby à partir des sauvegardes pour
                    des configurations à haute disponibilité
                  </li>
                  <li>
                    Vous êtes à l&apos;aise avec les outils en ligne de commande
                    et les mécanismes internes de PostgreSQL
                  </li>
                  <li>
                    Votre organisation dispose d&apos;une expertise DBA dédiée
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
                cloud. Barman est le meilleur choix quand vous avez besoin de
                backups incrémentaux basés sur rsync, de l&apos;intégration de
                la réplication en streaming, de la géo-redondance Barman vers
                Barman ou de la création de standby à partir des sauvegardes.
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
